package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"time"
)

type Validatable interface {
	Validate() error
}

type Transaction struct {
	ID int
	Amount float64
	Category string
	Description string
	Date string
}

type Budget struct {
	Category string
	Limit float64
	Period string // "месяц"/"год"
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
	ErrZeroAmount = errors.New("сумма транзакции не может быть 0")
	ErrNegativeAmount = errors.New("сумма транзакции не может быть отрицательной")
	ErrEmptyCategory = errors.New("категория не может быть пустой")
	ErrInvalidDate = errors.New("некорректный формат даты, ожидается YYYY-MM-DD")
	ErrNotPositiveLimit = errors.New("лимит бюджета должен быть положительным числом")
	ErrBudgetExceeded = errors.New("бюджет превышен")
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

// сумма добавленных транзакций по категории
func currentSumByCategory(category string) float64 {
	var sum float64
	for _, tx := range transactions {
		if tx.Category == category {
			sum += tx.Amount
		}
	}
	return sum
}

// не превысит ли новая транзакция лимит бюджета
func checkBudget(tx Transaction) error {
	categoryBudgets, ok := budgets[tx.Category]
	if !ok || len(categoryBudgets) == 0 {
		return nil
	}

	currentSum := currentSumByCategory(tx.Category)
	newSum := currentSum + tx.Amount

	for _, b := range categoryBudgets {
		if newSum > b.Limit {
			return fmt.Errorf("%w: категория %q, лимит %.2f (%s), сумма с новой транзакцией %.2f",
				ErrBudgetExceeded, tx.Category, b.Limit, b.Period, newSum)
		}
	}
	return nil
}

func AddTransaction(tx Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	if err := checkBudget(tx); err != nil {
		return err
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

func ListTransactions() []Transaction {
	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}

func removeByPeriod(categoryBudgets []Budget, period string) []Budget {
	return slices.DeleteFunc(categoryBudgets, func(budget Budget) bool {
		return budget.Period == period
	})
}

// добавляет бюджет
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}

	categoryBudgets := budgets[b.Category]
	categoryBudgets = removeByPeriod(categoryBudgets, b.Period)
	categoryBudgets = append(categoryBudgets, b)
	budgets[b.Category] = categoryBudgets
	return nil
}

// читает JSON-массив бюджетов и добавляет через SetBudget
func LoadBudgets(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("не удалось прочитать данные бюджетов: %w", err)
	}

	var loaded []Budget
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("не удалось распарсить JSON бюджетов: %w", err)
	}

	for _, b := range loaded {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("не удалось установить бюджет %+v: %w", b, err)
		}
		fmt.Printf("Добавлен бюджет: %+v\n", b)
	}
	return nil
}

func ListBudgets() []Budget {
	total := 0
	for _, list := range budgets {
		total += len(list)
	}
	result := make([]Budget, 0, total)
	for _, list := range budgets {
		result = append(result, list...)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Period < result[j].Period
	})
	return result
}
