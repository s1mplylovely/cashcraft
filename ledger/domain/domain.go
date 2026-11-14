package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ledger/internal/cache"
	"ledger/internal/db"
)

type Validatable interface {
	Validate() error
}

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

type Budget struct {
	Category string
	Limit    float64
	Period   string // "месяц"/"год"
}

var transactions []Transaction

var budgets = map[string][]Budget{
	"еда": {
		{Category: "еда", Limit: 5000, Period: "месяц"},
		{Category: "еда", Limit: 60000, Period: "год"},
	},
	"транспорт": {
		{Category: "транспорт", Limit: 3000, Period: "месяц"},
	},
}

var (
	ErrZeroAmount       = errors.New("сумма транзакции не может быть 0")
	ErrNegativeAmount   = errors.New("сумма транзакции не может быть отрицательной")
	ErrEmptyCategory    = errors.New("категория не может быть пустой")
	ErrInvalidDate      = errors.New("некорректный формат даты, ожидается YYYY-MM-DD")
	ErrNotPositiveLimit = errors.New("лимит бюджета должен быть положительным числом")
	ErrBudgetExceeded   = errors.New("бюджет превышен")
)

func (tx Transaction) Validate() error {
	if tx.Amount == 0 {
		return ErrZeroAmount
	}
	if tx.Amount < 0 {
		return fmt.Errorf("%w: %.2f", ErrNegativeAmount, tx.Amount)
	}
	if tx.Category == "" {
		return ErrEmptyCategory
	}
	if _, err := time.Parse("2006-01-02", tx.Date); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidDate, tx.Date)
	}
	return nil
}

func (b Budget) Validate() error {
	if b.Category == "" {
		return ErrEmptyCategory
	}
	if b.Limit <= 0 {
		return fmt.Errorf("%w: %.2f", ErrNotPositiveLimit, b.Limit)
	}
	return nil
}

func CheckValid(v Validatable) error {
	return v.Validate()
}

// подключает инфраструктуру Ledger: PostgreSQL (обязательно) и
// Redis (опционально)
func ConnectInfra() error {
	if _, err := db.Connect(); err != nil {
		return err
	}
	if _, err := cache.Connect(); err != nil {
		log.Printf("Ledger: Redis недоступен, кеширование отключено: %v", err)
	}
	return nil
}

// обертка для main: завершает процесс при ошибке подключения к БД
func MustConnectInfra() {
	if err := ConnectInfra(); err != nil {
		log.Fatalf("Ledger: не удалось подключиться к базе данных: %v", err)
	}
}

const budgetsCacheKey = "budgets:all"

// выполняет UPSERT бюджета категории (одна строка на категорию),
// после успешной записи инвалидирует кеш списка бюджетов
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}

	const q = `
INSERT INTO budgets (category, limit_amount, period)
VALUES ($1, $2, $3)
ON CONFLICT (category)
DO UPDATE SET limit_amount = EXCLUDED.limit_amount,
             period        = EXCLUDED.period`

	if _, err := db.DB().ExecContext(context.Background(), q, b.Category, b.Limit, b.Period); err != nil {
		return fmt.Errorf("не удалось сохранить бюджет: %w", err)
	}

	invalidateBudgetsCache()
	return nil
}

// читает все бюджеты, отсортированные по категории;
// результат кэшируется в Redis на короткий TTL (ключ budgets:all)
func ListBudgets() ([]Budget, error) {
	ctx := context.Background()

	// cache hit
	if c := cache.Client(); c != nil {
		if data, err := c.Get(ctx, budgetsCacheKey).Bytes(); err == nil {
			var cached []Budget
			if json.Unmarshal(data, &cached) == nil {
				return cached, nil
			}
		}
	}

	const q = `SELECT category, limit_amount::float8, period FROM budgets ORDER BY category`
	rows, err := db.DB().QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать бюджеты: %w", err)
	}
	defer rows.Close()

	result := make([]Budget, 0)
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.Category, &b.Limit, &b.Period); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// cache set
	if c := cache.Client(); c != nil {
		if data, err := json.Marshal(result); err == nil {
			_ = c.Set(ctx, budgetsCacheKey, data, 20*time.Second).Err()
		}
	}
	return result, nil
}

func invalidateBudgetsCache() {
	if c := cache.Client(); c != nil {
		_ = c.Del(context.Background(), budgetsCacheKey).Err()
	}
}

// проверяет лимит бюджета категории и, если он не превышен,
// вставляет расход в БД; сгенерированный id записывается в tx.ID
func AddTransaction(tx *Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	ctx := context.Background()
	d := db.DB()

	// лимит бюджета категории
	var limit float64
	err := d.QueryRowContext(ctx,
		`SELECT limit_amount::float8 FROM budgets WHERE category = $1`, tx.Category).Scan(&limit)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// бюджета нет — добавляем без ограничения
	case err != nil:
		return fmt.Errorf("не удалось прочитать бюджет: %w", err)
	default:
		// уже потрачено по категории
		var spent float64
		if err := d.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(amount), 0)::float8 FROM expenses WHERE category = $1`,
			tx.Category).Scan(&spent); err != nil {
			return fmt.Errorf("не удалось посчитать расходы: %w", err)
		}
		// проверка превышения
		if spent+tx.Amount > limit {
			return ErrBudgetExceeded
		}
	}

	// вставка (дата передаётся как time.Time — валидность уже проверена)
	date, _ := time.Parse("2006-01-02", tx.Date)
	if err := d.QueryRowContext(ctx,
		`INSERT INTO expenses (amount, category, description, date)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		tx.Amount, tx.Category, tx.Description, date).Scan(&tx.ID); err != nil {
		return fmt.Errorf("не удалось добавить транзакцию: %w", err)
	}
	return nil
}

// читает все расходы, отсортированные по дате и id по убыванию
func ListTransactions() ([]Transaction, error) {
	ctx := context.Background()

	const q = `SELECT id, amount::float8, category, description, date
	           FROM expenses ORDER BY date DESC, id DESC`
	rows, err := db.DB().QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать транзакции: %w", err)
	}
	defer rows.Close()

	result := make([]Transaction, 0)
	for rows.Next() {
		var (
			tx   Transaction
			desc sql.NullString
			date time.Time
		)
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Category, &desc, &date); err != nil {
			return nil, err
		}
		tx.Description = desc.String
		tx.Date = date.Format("2006-01-02")
		result = append(result, tx)
	}
	return result, rows.Err()
}
