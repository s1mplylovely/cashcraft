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
cd ledger
go run main.go
```

## Запуск Gateway

```bash
cd gateway
go run main.go
```

Сервер поднимется на `:8080`. В отдельном терминале проверить:

```bash
curl -i http://localhost:8080/ping
```
