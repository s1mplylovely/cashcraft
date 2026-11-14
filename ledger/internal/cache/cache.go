package cache

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// единственный на процесс клиент Redis (nil, если не подключились)
var client *redis.Client

// возвращает текущий клиент Redis или nil
func Client() *redis.Client { return client }

// создает клиент Redis по конфигурации из переменных окружения и проверяет
//	соединение через Ping; повторный вызов возвращает уже созданный клиент
func Connect() (*redis.Client, error) {
	if client != nil {
		return client, nil
	}

	addr := getenv("REDIS_ADDR", "localhost:6379")
	dbNum, _ := strconv.Atoi(getenv("REDIS_DB", "0"))
	password := os.Getenv("REDIS_PASSWORD")

	c := redis.NewClient(&redis.Options{
		Addr: addr,
		DB: dbNum,
		Password: password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, err
	}

	client = c
	log.Printf("Ledger: подключение к Redis установлено (addr=%s, db=%d)", addr, dbNum)
	return client, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
