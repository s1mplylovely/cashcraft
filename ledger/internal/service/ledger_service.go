package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ledger/internal/domain"
)

const reportCacheTTL = 30 * time.Second

// реализация Service
type ledgerService struct {
	budgets  domain.BudgetRepository
	expenses domain.ExpenseRepository
	cache    Cache // может быть nil (кеш выключен)
}

var _ Service = (*ledgerService)(nil)

// собирает сервис из репозиториев и (опционально) кеша
func New(budgets domain.BudgetRepository, expenses domain.ExpenseRepository, cache Cache) Service {
	return &ledgerService{budgets: budgets, expenses: expenses, cache: cache}
}

// валидирует и сохраняет бюджет
func (s *ledgerService) SetBudget(ctx context.Context, b domain.Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	return s.budgets.Upsert(ctx, b)
}

// возвращает все бюджеты
func (s *ledgerService) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	return s.budgets.List(ctx)
}

// валидирует расход, проверяет бизнес-правило бюджета и добавляет запись
func (s *ledgerService) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	// сумма по категории + новая транзакция не превышает лимит
	budget, found, err := s.budgets.GetByCategory(ctx, tx.Category)
	if err != nil {
		return err
	}
	if found {
		spent, err := s.expenses.SumByCategory(ctx, tx.Category)
		if err != nil {
			return err
		}
		if spent+tx.Amount > budget.Limit {
			return domain.ErrBudgetExceeded
		}
	}

	return s.expenses.Add(ctx, tx)
}

// возвращает все расходы
func (s *ledgerService) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.expenses.List(ctx)
}

// возвращает агрегированный отчёт за интервал [from, to], при наличии кеша
// результат читается/пишется в Redis с TTL reportCacheTTL
func (s *ledgerService) GetReportSummary(ctx context.Context, from, to string) (ReportSummary, error) {
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return ReportSummary{}, fmt.Errorf("%w: from=%q", domain.ErrInvalidDate, from)
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return ReportSummary{}, fmt.Errorf("%w: to=%q", domain.ErrInvalidDate, to)
	}

	key := fmt.Sprintf("report:summary:%s:%s", from, to)

	// cache hit
	if s.cache != nil {
		if data, err := s.cache.Get(ctx, key); err == nil && data != nil {
			var cached ReportSummary
			if json.Unmarshal(data, &cached) == nil {
				log.Printf("Ledger: отчет %s — cache hit", key)
				return cached, nil
			}
		}
	}

	// cache miss -> считаем через репозиторий
	items, err := s.expenses.SummaryByPeriod(ctx, from, to)
	if err != nil {
		return ReportSummary{}, err
	}
	summary := ReportSummary{From: from, To: to, Items: items}
	if summary.Items == nil {
		summary.Items = []domain.ReportItem{}
	}
	for _, it := range items {
		summary.Total += it.Total
	}

	// cache set
	if s.cache != nil {
		if data, err := json.Marshal(summary); err == nil {
			if err := s.cache.Set(ctx, key, data, reportCacheTTL); err == nil {
				log.Printf("Ledger: отчет %s — cache miss → set (TTL %s)", key, reportCacheTTL)
			}
		}
	}
	return summary, nil
}
