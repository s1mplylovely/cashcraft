# CashCraft

Сервисы **Ledger** и **Gateway** в одном workspace (`go.work`):

- **ledger** — доменная логика и работа с хранилищем; данные хранятся в PostgreSQL, отчеты и список бюджетов кэшируются в Redis с коротким TTL
- **gateway** — HTTP API (`:8080`)

## Требования

- Go 1.22+ (проверить: `go version`)
- PostgreSQL 14+
- Redis 6+
- [goose](https://github.com/pressly/goose) для миграций:
  `go install github.com/pressly/goose/v3/cmd/goose@latest`

## Запуск

### 1. Зависимости Go

Внешние модули - `github.com/jackc/pgx/v5` (драйвер PostgreSQL) и `github.com/redis/go-redis/v9` (клиент Redis)

```bash
cd cashcraft
go work sync
```

### 2. Поднять PostgreSQL и Redis в Docker

```bash
cd cashcraft
docker compose up -d    # postgres (:5432) и redis (:6379)
docker compose ps       # проверить, что оба healthy
```

Остановка и очистка:

```bash
docker compose down            # остановить контейнеры
docker compose down -v         # остановить и удалить данные БД (том pgdata)
```

> Если PostgreSQL не в Docker - создать базу вручную (`createdb cashapp`) и задать переменные подключения (п. 4).

### 3. Применить миграции (goose)

```bash
cd cashcraft
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"

goose -dir ledger/migrations up      # накатить
goose -dir ledger/migrations status  # статус
goose -dir ledger/migrations down    # откатить
```

### 4. Переменные окружения

PostgreSQL - приоритетно `DATABASE_URL`, иначе сборка DSN из частей:

| Переменная     | По умолчанию | Назначение                       |
|----------------|--------------|----------------------------------|
| `DATABASE_URL` | —            | полный DSN (имеет приоритет)     |
| `DB_HOST`      | `localhost`  | хост PostgreSQL                  |
| `DB_PORT`      | `5432`       | порт                             |
| `DB_USER`      | `postgres`   | пользователь                     |
| `DB_PASS`      | `postgres`   | пароль                           |
| `DB_NAME`      | `cashapp`    | имя базы                         |

Redis:

| Переменная       | По умолчанию     | Назначение            |
|------------------|------------------|-----------------------|
| `REDIS_ADDR`     | `localhost:6379` | адрес Redis           |
| `REDIS_DB`       | `0`              | номер БД Redis        |
| `REDIS_PASSWORD` | —                | пароль (опционально)  |

Пример:

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export REDIS_DB=0
```

### 5. Запуск

```bash
cd cashcraft
go run ./ledger
go run ./gateway
```

Сервер поднимется на `:8080`. В отдельном терминале проверить:

```bash
curl -i http://localhost:8080/ping
```

При старте оба сервиса логируют подключение:

```
Ledger: подключение к PostgreSQL установлено (postgres://postgres:****@localhost:5432/cashapp?sslmode=disable)
Ledger: подключение к Redis установлено (addr=localhost:6379, db=0)
Gateway запущен на :8080
```

- Если PostgreSQL недоступен - процесс завершается с понятным сообщением.
- Если Redis недоступен - сервис продолжает работу без кеша.

## Доступные эндпоинты

- GET /ping — проверка доступности
- POST /api/transactions — создать транзакцию
- GET /api/transactions — список транзакций
- POST /api/budgets — создать/перезаписать бюджет
- GET /api/budgets — список бюджетов
- GET /api/reports/summary — отчет `SUM(amount)` по категориям; параметры `from`, `to` в формате `YYYY-MM-DD` (по умолчанию — текущий месяц)

## Примеры cURL для проверки работы API

```bash
BASE=http://localhost:8080
```

- создание бюджета с пустой категорией: 400 + ```{"error":"категория не может быть пустой"}```

```bash
curl -i -X POST $BASE/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"","limit":5000,"period":"месяц"}'
```

- список бюджетов: 200 + ```[...]```

```bash
curl -s $BASE/api/budgets
```

- успешная транзакция: 201 + ```{"id":1,...}```

```bash
curl -i -X POST $BASE/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":500,"category":"еда","description":"ресторан","date":"2026-09-25"}'
```

- транзакция с отрицательной суммой: 400 + ```{"error":"сумма транзакции не может быть отрицательной: -1.00"}```

```bash
curl -i -X POST $BASE/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":-1,"category":"еда","description":"кафе","date":"2026-09-25"}'
```

- транзакция с превышением бюджета: 409 + ```{"error":"бюджет превышен"}```

```bash
curl -i -X POST $BASE/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":99999,"category":"еда","description":"магазин","date":"2026-09-25"}'
```

- список транзакций: 200 + ```[...]```

```bash
curl -s $BASE/api/transactions
```

- успешное создание бюджета: 201 + ```{...}```
```bash
curl -i -X POST $BASE/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"еда","limit":5000,"period":"месяц"}'
```

- некорректный JSON: 400 + ```{"error":"некорректный JSON: ..."}```
```bash
curl -i -X POST $BASE/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":}'
```

### Эффект кеша Redis

Два одинаковых запроса отчета подряд: первый считается в БД и кладётся в Redis, второй возвращается из кеша (TTL 30 секунд).

```bash
curl -s "$BASE/api/reports/summary?from=2026-09-01&to=2026-09-30"   # cache miss -> set
curl -s "$BASE/api/reports/summary?from=2026-09-01&to=2026-09-30"   # cache hit
```

В логах Ledger:

```
Ledger: отчет report:summary:2026-09-01:2026-09-30 — cache miss -> set (TTL 30s)
Ledger: отчет report:summary:2026-09-01:2026-09-30 — cache hit
```

## Тесты

- `ledger/domain/domain_test.go` - unit-тесты домена `Transaction.Validate()`, `Budget.Validate()`, правило бюджета в `AddTransaction()`/`ListTransactions()`

- `gateway/internal/api/handlers_test.go` - интеграционные HTTP-тесты для `POST/GET /api/budgets` и `POST/GET /api/transactions` (через `httptest`)

Юнит-тесты валидации работают без инфраструктуры.
Тесты, которым нужна БД, автоматически пропускаются, если PostgreSQL недоступен.

## Инструкция по запуску тестов

```bash
cd app

# все тесты
go test ./ledger/... ./gateway/...

# подробный вывод
go test -v ./ledger/... ./gateway/...

# запустить тесты и сформировать покрытие
go test ./ledger/... ./gateway/... -cover -coverprofile=cover.out
go tool cover -html=cover.out -o cover.html
```

Ожидаемый вывод:

```
ok  	ledger/domain
ok  	gateway/internal/api
```

## Покрытие тестами

- unit-тесты (`ledger/domain/domain_test.go`) ~38.0%

- интеграционные тесты (`gateway/internal/api/handlers_test.go`) ~77.9%
