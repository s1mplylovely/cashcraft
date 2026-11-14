// Инфраструктурный адаптер кеша поверх Redis
package cache

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// клиент Redis, реализующий порт кеша
type RedisCache struct {
	client *redis.Client
}

// создает клиент Redis по конфигурации из окружения и проверяет Ping
func Connect(ctx context.Context) (*RedisCache, error) {
	addr := getenv("REDIS_ADDR", "localhost:6379")
	dbNum, _ := strconv.Atoi(getenv("REDIS_DB", "0"))
	password := os.Getenv("REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       dbNum,
		Password: password,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	log.Printf("Ledger: подключение к Redis установлено (addr=%s, db=%d)", addr, dbNum)
	return &RedisCache{client: client}, nil
}

// возвращает (nil, nil) при промахе кеша
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// кладет значение с TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// освобождает ресурсы клиента
func (c *RedisCache) Close() error {
	return c.client.Close()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
