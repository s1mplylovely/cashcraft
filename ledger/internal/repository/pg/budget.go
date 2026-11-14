package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ledger/internal/domain"
)

type BudgetRepository struct {
	db *sql.DB
}

// гарантируем на этапе компиляции, что контракт реализован
var _ domain.BudgetRepository = (*BudgetRepository)(nil)

func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

// добавляет или обновляет бюджет категории
func (r *BudgetRepository) Upsert(ctx context.Context, b domain.Budget) error {
	const q = `
INSERT INTO budgets (category, limit_amount, period)
VALUES ($1, $2, $3)
ON CONFLICT (category)
DO UPDATE SET limit_amount = EXCLUDED.limit_amount,
             period        = EXCLUDED.period`
	if _, err := r.db.ExecContext(ctx, q, b.Category, b.Limit, b.Period); err != nil {
		return fmt.Errorf("pg: upsert бюджета: %w", err)
	}
	return nil
}

// возвращает бюджет категории (found=false, если его нет)
func (r *BudgetRepository) GetByCategory(ctx context.Context, category string) (domain.Budget, bool, error) {
	const q = `SELECT category, limit_amount::float8, period FROM budgets WHERE category = $1`
	var b domain.Budget
	err := r.db.QueryRowContext(ctx, q, category).Scan(&b.Category, &b.Limit, &b.Period)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.Budget{}, false, nil
	case err != nil:
		return domain.Budget{}, false, fmt.Errorf("pg: выборка бюджета: %w", err)
	default:
		return b, true, nil
	}
}

// возвращает все бюджеты, отсортированные по категории
func (r *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	const q = `SELECT category, limit_amount::float8, period FROM budgets ORDER BY category`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pg: список бюджетов: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Budget, 0)
	for rows.Next() {
		var b domain.Budget
		if err := rows.Scan(&b.Category, &b.Limit, &b.Period); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}
