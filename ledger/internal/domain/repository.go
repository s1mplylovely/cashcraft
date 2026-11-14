package domain

import "context"


type BudgetRepository interface {
	// добавляет или обновляет бюджет категории
	Upsert(ctx context.Context, b Budget) error
	// возвращает бюджет категории; found=false, если его нет
	GetByCategory(ctx context.Context, category string) (b Budget, found bool, err error)
	// возвращает все бюджеты
	List(ctx context.Context) ([]Budget, error)
}

type ExpenseRepository interface {
	// добавляет расход и записывает сгенерированный id в tx.ID
	Add(ctx context.Context, tx *Transaction) error
	// возвращает все расходы
	List(ctx context.Context) ([]Transaction, error)
	// возвращает сумму расходов по категории
	SumByCategory(ctx context.Context, category string) (float64, error)
	// возвращает суммы расходов по категориям за интервал [from, to]
	SummaryByPeriod(ctx context.Context, from, to string) ([]ReportItem, error)
}
