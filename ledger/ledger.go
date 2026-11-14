// Единственный пакет, который импортирует Gateway
package ledger

import (
	"context"

	"ledger/internal/app"
	"ledger/internal/domain"
	"ledger/internal/service"
)

type (
	Transaction = domain.Transaction
	Budget = domain.Budget
	ReportItem = domain.ReportItem
	ReportSummary = service.ReportSummary
	Service = service.Service
)

var (
	ErrBudgetExceeded = domain.ErrBudgetExceeded
	ErrInvalidDate    = domain.ErrInvalidDate
)

// сообщает, что ошибка - нарушение валидации сущности (для ответа 400)
func IsValidationError(err error) bool { return domain.IsValidation(err) }

// собирает готовый сервис из конфигурации окружения
func New(ctx context.Context) (Service, func() error, error) {
	return app.New(ctx)
}
