package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ledger/internal/cache"
	"ledger/internal/db"
)

// сумма расходов по одной категории за период
type ReportItem struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// агрегированный отчет по расходам за интервал [From, To]
type ReportSummary struct {
	From  string       `json:"from"`
	To    string       `json:"to"`
	Total float64      `json:"total"`
	Items []ReportItem `json:"items"`
}

// короткий TTL кеша отчета
const reportCacheTTL = 30 * time.Second

// возвращает агрегированный отчет SUM(amount) GROUP BY category
// за интервал [from, to] (даты в формате YYYY-MM-DD)
func GetReportSummary(ctx context.Context, from, to string) (ReportSummary, error) {
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return ReportSummary{}, fmt.Errorf("%w: from=%q", ErrInvalidDate, from)
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return ReportSummary{}, fmt.Errorf("%w: to=%q", ErrInvalidDate, to)
	}

	key := fmt.Sprintf("report:summary:%s:%s", from, to)

	// cache hit - значение берётся из Redis
	if c := cache.Client(); c != nil {
		if data, err := c.Get(ctx, key).Bytes(); err == nil {
			var cached ReportSummary
			if json.Unmarshal(data, &cached) == nil {
				log.Printf("Ledger: отчет %s — cache hit", key)
				return cached, nil
			}
		}
	}

	// cache miss - отчет считается в БД
	const q = `SELECT category, SUM(amount)::float8 AS total
	           FROM expenses
	           WHERE date BETWEEN $1 AND $2
	           GROUP BY category
	           ORDER BY category`
	rows, err := db.DB().QueryContext(ctx, q, from, to)
	if err != nil {
		return ReportSummary{}, fmt.Errorf("не удалось построить отчет: %w", err)
	}
	defer rows.Close()

	summary := ReportSummary{From: from, To: to, Items: make([]ReportItem, 0)}
	for rows.Next() {
		var item ReportItem
		if err := rows.Scan(&item.Category, &item.Total); err != nil {
			return ReportSummary{}, err
		}
		summary.Items = append(summary.Items, item)
		summary.Total += item.Total
	}
	if err := rows.Err(); err != nil {
		return ReportSummary{}, err
	}

	// cache miss - отчет пишется в Redis с TTL под ключом report:summary:<from>:<to>
	if c := cache.Client(); c != nil {
		if data, err := json.Marshal(summary); err == nil {
			if err := c.Set(ctx, key, data, reportCacheTTL).Err(); err == nil {
				log.Printf("Ledger: отчет %s — cache miss -> set (TTL %s)", key, reportCacheTTL)
			}
		}
	}
	return summary, nil
}
