// Слой сборки зависимостей
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"ledger/internal/cache"
	"ledger/internal/repository/pg"
	"ledger/internal/service"
)

// New читает конфигурацию из окружения, открывает подключение к БД, собирает
// репозитории и сервис и возвращает:
// - готовый service.Service
// - closeFn для освобождения ресурсов (закрывает БД и, если есть, Redis)
// - ошибку (например, если не удалось подключиться к БД)
func New(ctx context.Context) (service.Service, func() error, error) {
	dsn := dsnFromEnv()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("не удалось подключиться к PostgreSQL (%s): %w", safeDSN(dsn), err)
	}
	log.Printf("Ledger: подключение к PostgreSQL установлено (%s)", safeDSN(dsn))

	// Redis опционально
	var (
		cacheAdapter service.Cache
		redisCache   *cache.RedisCache
	)
	if rc, err := cache.Connect(ctx); err != nil {
		log.Printf("Ledger: Redis недоступен, кеширование отключено: %v", err)
	} else {
		redisCache = rc
		cacheAdapter = rc
	}

	budgetRepo := pg.NewBudgetRepository(db)
	expenseRepo := pg.NewExpenseRepository(db)
	svc := service.New(budgetRepo, expenseRepo, cacheAdapter)

	closeFn := func() error {
		if redisCache != nil {
			_ = redisCache.Close()
		}
		return db.Close()
	}
	return svc, closeFn, nil
}

// формирует строку подключения
func dsnFromEnv() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "postgres")
	pass := getenv("DB_PASS", "postgres")
	name := getenv("DB_NAME", "cashapp")

	u := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(user, pass),
		Host: net.JoinHostPort(host, port),
		Path: "/" + name,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// прячет пароль для логов
func safeDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	if _, hasPass := u.User.Password(); hasPass {
		u.User = url.UserPassword(u.User.Username(), "****")
	}
	return u.String()
}
