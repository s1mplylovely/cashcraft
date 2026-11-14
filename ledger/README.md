# ledger

Доменная логика и работа с хранилищем. Данные в PostgreSQL, кеш отчетов и списка бюджетов — в Redis. Если Redis недоступен, домен работает напрямую с БД.

## Структура

```
ledger/
├── ledger.go                 Публичгый пакет: интерфейс Service, фабрика New,
│                             псевдонимы типов и ошибок. Единственный пакет,
│                             который импортирует Gateway.
├── cmd/ledger/main.go        CLI: собирает сервис через фабрику, грузит budgets.json
├── internal/
│   ├── domain/               Домен: сущности (Transaction, Budget), их Validate(),
│   │   ├── entity.go         и КОНТРАКТЫ репозиториев (интерфейсы)
│   │   └── repository.go     BudgetRepository, ExpenseRepository
│   ├── repository/pg/        Реализация контрактов поверх PostgreSQL (pgx)
│   │   ├── budget.go         upsert / get-by-category / list
│   │   └── expense.go        insert / list / sum-by-category / summary-by-period
│   ├── service/              Прикладной слой: интерфейс Service + реализация
│   │   ├── service.go        Service, ReportSummary, порт Cache
│   │   └── ledger_service.go валидация + бизнес-правило бюджета + отчет/кеш
│   ├── app/                  Сборка: env→sql.Open→репозитории→сервис
│   │   └── app.go            
│   └── cache/                Адаптер Redis (реализует порт service.Cache)
│       └── redis.go
├── migrations/               SQL-миграции
└── budgets.json              данные для первичного наполнения
```

## Запуск

```bash
cd cashcraft
go run ./ledger/cmd/ledger
```
