# gateway

HTTP API CashCraft на `:8080`. Зависит только от интерфейса сервиса `ledger.Service` и фабрики `ledger.New` (пакет `ledger`).

## Инициализация сервиса

На старте `main` получает готовый сервис через фабрику и передает его в конструктор HTTP-обработчиков:

```go
svc, closeFn, err := ledger.New(ctx)   // фабрика сама открывает БД/Redis
if err != nil {
    log.Fatalf("Gateway: не удалось инициализировать Ledger: %v", err)
}
defer closeFn()                        // освобождение ресурсов при выходе

http.ListenAndServe(":8080", api.NewRouter(svc))
```

`api.NewRouter(svc)` создаёт `Handlers{svc}`; каждый обработчик вызывает методы сервиса (`svc.SetBudget`, `svc.AddTransaction`, …). Детали хранения скрыты за интерфейсом.

## Запуск

```bash
cd app
go run ./gateway
```

## Эндпоинты

| Метод | Путь                     | Коды ответов |
|-------|--------------------------|--------------|
| GET   | `/ping`                  | 200 |
| POST  | `/api/budgets`           | 201 / 400 |
| GET   | `/api/budgets`           | 200 |
| POST  | `/api/transactions`      | 201 / 400 / 409 |
| GET   | `/api/transactions`      | 200 |
| GET   | `/api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD`   | 200 / 400 |

Маппинг ошибок:

- невалидный JSON -> `400` `{"error":"некорректный JSON: ..."}`
- провал валидации (пустая категория, нулевая/отрицательная сумма, дата) -> `400`
- превышение бюджета -> `409` `{"error":"бюджет превышен"}`
- некорректные `from`/`to` в отчете -> `400`

Все ответы — `Content-Type: application/json; charset=utf-8`
