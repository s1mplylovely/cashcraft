package service

import (
	"context"
	"time"

	"ledger/internal/domain"
)

// агрегированный отчет по расходам за интервал
type ReportSummary struct {
	From string `json:"from"`
	To string `json:"to"`
	Total float64 `json:"total"`
	Items []domain.ReportItem `json:"items"`
}

type Service interface {
	SetBudget(ctx context.Context, b domain.Budget) error
	ListBudgets(ctx context.Context) ([]domain.Budget, error)
	AddTransaction(ctx context.Context, tx *domain.Transaction) error
	ListTransactions(ctx context.Context) ([]domain.Transaction, error)
	GetReportSummary(ctx context.Context, from, to string) (ReportSummary, error)
}

type Cache interface {
	// возвращает (nil, nil) при промахе
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
