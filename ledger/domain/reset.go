package domain

import (
	"context"

	"ledger/internal/db"
)

// очищает таблицы expenses и budgets и сбрасывает счётчики id
func Reset() error {
	_, err := db.DB().ExecContext(context.Background(), `TRUNCATE expenses, budgets RESTART IDENTITY`)
	invalidateBudgetsCache()
	return err
}
