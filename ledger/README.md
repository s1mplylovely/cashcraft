# ledger

Доменная логика и работа с хранилищем. Данные в PostgreSQL, кеш отчетов и списка бюджетов — в Redis. Если Redis недоступен, домен работает напрямую с БД.

## Структура

```
ledger/
├── domain/              бизнес-логика
│   ├── domain.go        типы, валидация, UPSERT/SELECT бюджетов и транзакций
│   ├── report.go        GetReportSummary + кеширование в Redis
│   ├── loader.go        загрузка бюджетов из JSON (UPSERT)
│   └── reset.go         TRUNCATE для изоляции тестов
├── internal/db/         подключение к PostgreSQL (pgx stdlib, пул соединений)
├── internal/cache/      подключение к Redis (go-redis/v9)
├── migrations/          SQL-миграции (goose: up/down в одном файле)
└── budgets.json         данные для первичного наполнения
```

## Запуск

```bash
cd cashcraft
go run ./ledger
```
