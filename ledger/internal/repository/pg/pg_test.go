package pg

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"ledger/internal/assert"
	"ledger/internal/domain"
)

// Подключается к тестовой БД. Если PostgreSQL недоступен, тест пропускается
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("пропуск: sql.Open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Skipf("пропуск: PostgreSQL недоступен (%v)", err)
	}
	if _, err := db.Exec(`TRUNCATE expenses, budgets RESTART IDENTITY`); err != nil {
		t.Fatalf("TRUNCATE: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestBudgetRepository(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewBudgetRepository(db)

	// upsert + get
	assert.NoError(t, repo.Upsert(ctx, domain.Budget{Category: "еда", Limit: 5000, Period: "месяц"}))
	b, found, err := repo.GetByCategory(ctx, "еда")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 5000.0, b.Limit)

	// upsert обновляет
	assert.NoError(t, repo.Upsert(ctx, domain.Budget{Category: "еда", Limit: 7000, Period: "месяц"}))
	b, _, _ = repo.GetByCategory(ctx, "еда")
	assert.Equal(t, 7000.0, b.Limit)

	// отсутствующая категория
	_, found, err = repo.GetByCategory(ctx, "нет")
	assert.NoError(t, err)
	assert.True(t, !found)

	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestExpenseRepository(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewExpenseRepository(db)

	tx := domain.Transaction{Amount: 1500.50, Category: "еда", Description: "Пятёрочка", Date: "2026-09-19"}
	assert.NoError(t, repo.Add(ctx, &tx))
	assert.Equal(t, 1, tx.ID)

	assert.NoError(t, repo.Add(ctx, &domain.Transaction{Amount: 500, Category: "транспорт", Date: "2026-09-20"}))

	sum, err := repo.SumByCategory(ctx, "еда")
	assert.NoError(t, err)
	assert.Equal(t, 1500.50, sum)

	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "2026-09-20", list[0].Date) // ORDER BY date DESC

	items, err := repo.SummaryByPeriod(ctx, "2026-09-01", "2026-09-30")
	assert.NoError(t, err)
	assert.Len(t, items, 2)
}
