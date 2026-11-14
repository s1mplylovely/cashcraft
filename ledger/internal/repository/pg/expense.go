package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ledger/internal/domain"
)

type ExpenseRepository struct {
	db *sql.DB
}

var _ domain.ExpenseRepository = (*ExpenseRepository)(nil)

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// вставляет расход и записывает сгенерированный id в tx.ID
func (r *ExpenseRepository) Add(ctx context.Context, tx *domain.Transaction) error {
	// дата хранится как DATE; валидность формата уже проверена в сервисе.
	date, _ := time.Parse("2006-01-02", tx.Date)
	const q = `INSERT INTO expenses (amount, category, description, date)
	           VALUES ($1, $2, $3, $4) RETURNING id`
	if err := r.db.QueryRowContext(ctx, q, tx.Amount, tx.Category, tx.Description, date).Scan(&tx.ID); err != nil {
		return fmt.Errorf("pg: insert расхода: %w", err)
	}
	return nil
}

// возвращает все расходы, ORDER BY date DESC, id DESC
func (r *ExpenseRepository) List(ctx context.Context) ([]domain.Transaction, error) {
	const q = `SELECT id, amount::float8, category, description, date
	           FROM expenses ORDER BY date DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pg: список расходов: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Transaction, 0)
	for rows.Next() {
		var (
			tx   domain.Transaction
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

// возвращает сумму расходов по категории
func (r *ExpenseRepository) SumByCategory(ctx context.Context, category string) (float64, error) {
	const q = `SELECT COALESCE(SUM(amount), 0)::float8 FROM expenses WHERE category = $1`
	var sum float64
	if err := r.db.QueryRowContext(ctx, q, category).Scan(&sum); err != nil {
		return 0, fmt.Errorf("pg: сумма по категории: %w", err)
	}
	return sum, nil
}

// возвращает суммы по категориям за интервал [from, to]
func (r *ExpenseRepository) SummaryByPeriod(ctx context.Context, from, to string) ([]domain.ReportItem, error) {
	const q = `SELECT category, SUM(amount)::float8 AS total
	           FROM expenses
	           WHERE date BETWEEN $1 AND $2
	           GROUP BY category
	           ORDER BY category`
	rows, err := r.db.QueryContext(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("pg: отчёт по периоду: %w", err)
	}
	defer rows.Close()

	result := make([]domain.ReportItem, 0)
	for rows.Next() {
		var item domain.ReportItem
		if err := rows.Scan(&item.Category, &item.Total); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
