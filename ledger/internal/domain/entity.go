package domain

import (
	"errors"
	"fmt"
	"time"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string // формат YYYY-MM-DD
}

type Budget struct {
	Category string
	Limit    float64
	Period   string // "месяц"/"год"
}

type ReportItem struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

var (
	ErrZeroAmount = errors.New("сумма транзакции не может быть 0")
	ErrNegativeAmount = errors.New("сумма транзакции не может быть отрицательной")
	ErrEmptyCategory = errors.New("категория не может быть пустой")
	ErrInvalidDate = errors.New("некорректный формат даты, ожидается YYYY-MM-DD")
	ErrNotPositiveLimit = errors.New("лимит бюджета должен быть положительным числом")
	ErrBudgetExceeded = errors.New("budget exceeded")
)

// проверяет корректность расхода
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

// проверяет корректность бюджета
func (b Budget) Validate() error {
	if b.Category == "" {
		return ErrEmptyCategory
	}
	if b.Limit <= 0 {
		return fmt.Errorf("%w: %.2f", ErrNotPositiveLimit, b.Limit)
	}
	return nil
}

// используется HTTP-слоем для выбора кода ответа 400
func IsValidation(err error) bool {
	switch {
	case errors.Is(err, ErrZeroAmount),
		errors.Is(err, ErrNegativeAmount),
		errors.Is(err, ErrEmptyCategory),
		errors.Is(err, ErrInvalidDate),
		errors.Is(err, ErrNotPositiveLimit):
		return true
	default:
		return false
	}
}
