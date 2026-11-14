package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" в database/sql
)

// единственное на процесс подключение к БД
var pool *sql.DB

// возвращает текущий пул соединений
func DB() *sql.DB { return pool }

// открывает соединение с PostgreSQL и проверяет его через Ping;
// повторный вызов возвращает уже открытый пул
func Connect() (*sql.DB, error) {
	if pool != nil {
		return pool, nil
	}

	dsn := dsnFromEnv()

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// параметры пула соединений
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("не удалось подключиться к PostgreSQL (%s): %w", safeDSN(dsn), err)
	}

	pool = conn
	log.Printf("Ledger: подключение к PostgreSQL установлено (%s)", safeDSN(dsn))
	return pool, nil
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

// прячет пароль, чтобы не попал в логи
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
