package service

import (
	"context"
	"testing"

	"ledger/internal/assert"
	"ledger/internal/domain"
)

// in-memory фейки репозиториев (тест бизнес-логики без БД)

type fakeBudgetRepo struct {
	items map[string]domain.Budget
}

func newFakeBudgetRepo() *fakeBudgetRepo { return &fakeBudgetRepo{items: map[string]domain.Budget{}} }

func (r *fakeBudgetRepo) Upsert(_ context.Context, b domain.Budget) error {
	r.items[b.Category] = b
	return nil
}
func (r *fakeBudgetRepo) GetByCategory(_ context.Context, category string) (domain.Budget, bool, error) {
	b, ok := r.items[category]
	return b, ok, nil
}
func (r *fakeBudgetRepo) List(_ context.Context) ([]domain.Budget, error) {
	out := make([]domain.Budget, 0, len(r.items))
	for _, b := range r.items {
		out = append(out, b)
	}
	return out, nil
}

type fakeExpenseRepo struct {
	items  []domain.Transaction
	nextID int
}

func newFakeExpenseRepo() *fakeExpenseRepo { return &fakeExpenseRepo{nextID: 1} }

func (r *fakeExpenseRepo) Add(_ context.Context, tx *domain.Transaction) error {
	tx.ID = r.nextID
	r.nextID++
	r.items = append(r.items, *tx)
	return nil
}
func (r *fakeExpenseRepo) List(_ context.Context) ([]domain.Transaction, error) {
	return r.items, nil
}
func (r *fakeExpenseRepo) SumByCategory(_ context.Context, category string) (float64, error) {
	var sum float64
	for _, tx := range r.items {
		if tx.Category == category {
			sum += tx.Amount
		}
	}
	return sum, nil
}
func (r *fakeExpenseRepo) SummaryByPeriod(_ context.Context, from, to string) ([]domain.ReportItem, error) {
	byCat := map[string]float64{}
	for _, tx := range r.items {
		if tx.Date >= from && tx.Date <= to { // YYYY-MM-DD сравнивается лексикографически
			byCat[tx.Category] += tx.Amount
		}
	}
	out := make([]domain.ReportItem, 0, len(byCat))
	for cat, total := range byCat {
		out = append(out, domain.ReportItem{Category: cat, Total: total})
	}
	return out, nil
}

// тесты

// проверяет бизнес-правило бюджета на фейковых репозиториях
func TestService_BudgetRule(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := New(newFakeBudgetRepo(), newFakeExpenseRepo(), nil)

	if err := svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 5000, Period: "месяц"}); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}

	t.Run("в пределах лимита", func(t *testing.T) {
		tx := domain.Transaction{Amount: 1500.50, Category: "еда", Date: "2026-09-19"}
		assert.NoError(t, svc.AddTransaction(ctx, &tx))
		assert.Equal(t, 1, tx.ID) // id записан репозиторием

		tx2 := domain.Transaction{Amount: 1000, Category: "еда", Date: "2026-09-20"}
		assert.NoError(t, svc.AddTransaction(ctx, &tx2))

		list, _ := svc.ListTransactions(ctx)
		assert.Len(t, list, 2)
	})

	t.Run("превышение лимита", func(t *testing.T) {
		tx := domain.Transaction{Amount: 4000, Category: "еда", Date: "2026-09-21"}
		err := svc.AddTransaction(ctx, &tx)
		assert.ErrorIs(t, err, domain.ErrBudgetExceeded)

		list, _ := svc.ListTransactions(ctx)
		assert.Len(t, list, 2) // не добавлена
	})

	t.Run("категория без бюджета не ограничена", func(t *testing.T) {
		tx := domain.Transaction{Amount: 100000, Category: "другое", Date: "2026-09-22"}
		assert.NoError(t, svc.AddTransaction(ctx, &tx))
	})
}

// проверяет, что сервис валидирует сущность до обращения к репозиториям
func TestService_AddTransaction_Validation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := New(newFakeBudgetRepo(), newFakeExpenseRepo(), nil)

	err := svc.AddTransaction(ctx, &domain.Transaction{Amount: -5, Category: "еда", Date: "2026-09-19"})
	assert.ErrorIs(t, err, domain.ErrNegativeAmount)
}

// проверяет агрегацию отчета (кеш выключен, cache=nil)
func TestService_Report(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := New(newFakeBudgetRepo(), newFakeExpenseRepo(), nil)

	_ = svc.AddTransaction(ctx, &domain.Transaction{Amount: 1500.50, Category: "еда", Date: "2026-09-19"})
	_ = svc.AddTransaction(ctx, &domain.Transaction{Amount: 500, Category: "транспорт", Date: "2026-09-20"})

	summary, err := svc.GetReportSummary(ctx, "2026-09-01", "2026-09-30")
	assert.NoError(t, err)
	assert.Equal(t, 2000.50, summary.Total)
	assert.Len(t, summary.Items, 2)

	// невалидная дата
	_, err = svc.GetReportSummary(ctx, "2026-09-01", "baddate")
	assert.ErrorIs(t, err, domain.ErrInvalidDate)
}
