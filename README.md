# Инструкция по запуску сервисов

## Требования

Go 1.21+ (проверить: `go version`)

## Инициализация модулей (один раз)

```bash
cd gateway
go mod init github.com/yourname/gateway

cd ../ledger
go mod init github.com/yourname/ledger
```

## Запуск Ledger

```bash
go run ./ledger
```

## Запуск Gateway

```bash
go run ./gateway
```

Сервер поднимется на `:8080`. В отдельном терминале проверить:

```bash
curl -i http://localhost:8080/ping
```

## Доступные эндпоинты

- GET /ping — проверка доступности
- POST /api/transactions — создать транзакцию
- GET /api/transactions — список транзакций
- POST /api/budgets — создать/перезаписать бюджет
- GET /api/budgets — список бюджетов

## Примеры cURL для проверки работы API

- успешная транзакция: 201 + ```{"id":1,...}```

```bash
curl -i -X POST localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":100,"category":"еда","description":"ресторан","date":"2026-09-25"}'
```

- транзакция с отрицательной суммой: 400 + ```{"error":"сумма транзакции не может быть отрицательной: -1.00"}```

```bash
curl -i -X POST localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":-1,"category":"еда","description":"кафе","date":"2026-09-25"}'
```

- транзакция с превышением бюджета: 409 + ```{"error":"бюджет превышен"}```

```bash
curl -i -X POST localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":9999,"category":"еда","description":"магазин","date":"2026-09-25"}'
```

- список транзакций: 200 + ```[...]```

```bash
curl -s localhost:8080/api/transactions
```

- успешное создание бюджета: 201 + ```{...}```
```bash
curl -i -X POST localhost:8080/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"развлечения","limit":5000,"period":"месяц"}'
```

- создание бюджета с пустой категорией: 400 + ```{"error":"категория не может быть пустой"}```

```bash
curl -i -X POST localhost:8080/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"","limit":5000,"period":"месяц"}'
```

- список бюджетов: 200 + ```[...]```

```bash
curl -s localhost:8080/api/budgets
```

## Тесты

- `ledger/domain/domain_test.go` - unit-тесты домена `Transaction.Validate()`, `Budget.Validate()`, правило бюджета в `AddTransaction()`/`ListTransactions()`

- `gateway/internal/api/handlers_test.go` - интеграционные HTTP-тесты для `POST/GET /api/budgets` и `POST/GET /api/transactions` (через `httptest`)

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

- unit-тесты (`ledger/domain/domain_test.go`) ~58.3%

- интеграционные тесты (`gateway/internal/api/handlers_test.go`) ~83.3%
