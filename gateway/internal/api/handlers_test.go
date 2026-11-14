package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/internal/assert"
	"ledger/domain"
)

// прогоняет HTTP-запрос через роутер и возвращает записанный ответ,
// используется httptest.NewRecorder + httptest.NewRequest (без поднятия сети)
func doRequest(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// достает поле "error" из тела ответа формата {"error":"..."}
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("не удалось распарсить тело ошибки %q: %v", rec.Body.String(), err)
	}
	return payload["error"]
}

const contentTypeJSON = "application/json; charset=utf-8"

// подключает инфраструктуру и очищает таблицы перед интеграционным
// тестом; если PostgreSQL недоступен, тест пропускается
func requireDB(t *testing.T) {
	t.Helper()
	if err := domain.ConnectInfra(); err != nil {
		t.Skipf("пропуск: PostgreSQL недоступен (%v)", err)
	}
	if err := domain.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	t.Cleanup(func() { _ = domain.Reset() })
}

// Интеграционные тесты для POST/GET /api/budgets:
// используют общее глобальное состояние ledger, поэтому не параллельно,
// состояние изолируем через domain.Reset()
func TestAPI_Budgets(t *testing.T) {
	requireDB(t)

	router := NewRouter()

	t.Run("ok", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/budgets",
			`{"category":"еда","limit":5000,"period":"месяц"}`)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))

		var b BudgetResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &b))
		assert.Equal(t, "еда", b.Category)
		assert.Equal(t, 5000.0, b.Limit)
		assert.Equal(t, "месяц", b.Period)
	})

	t.Run("list", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodGet, "/api/budgets", "")

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))

		var list []BudgetResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Len(t, list, 1)
		assert.Equal(t, "еда", list[0].Category)
	})

	t.Run("bad_empty_category", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/budgets",
			`{"category":"","limit":5000,"period":"месяц"}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
		assert.True(t, decodeError(t, rec) != "", "тело ошибки должно содержать непустое поле error")
	})

	t.Run("bad_json", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/budgets", `{"limit":}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(decodeError(t, rec), "некорректный JSON"))
	})
}

// Интеграционные тесты для POST/GET /api/transactions:
// успешная цепочка (201 -> 200), превышение бюджета (409) и некорректный JSON (400)
func TestAPI_Transactions(t *testing.T) {
	requireDB(t)

	router := NewRouter()

	if rec := doRequest(t, router, http.MethodPost, "/api/budgets",
		`{"category":"еда","limit":5000,"period":"месяц"}`); rec.Code != http.StatusCreated {
		t.Fatalf("подготовка бюджета: код %d, тело %s", rec.Code, rec.Body.String())
	}

	t.Run("ok", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/transactions",
			`{"amount":1500.50,"category":"еда","description":"Пятёрочка","date":"2026-09-19"}`)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))

		var tx TransactionResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tx))
		assert.Equal(t, 1, tx.ID)
		assert.Equal(t, 1500.50, tx.Amount)
		assert.Equal(t, "еда", tx.Category)
		assert.Equal(t, "2026-09-19", tx.Date)
	})

	t.Run("list", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodGet, "/api/transactions", "")

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))

		var list []TransactionResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Len(t, list, 1)
		assert.Equal(t, "2026-09-19", list[0].Date)
	})

	t.Run("exceeded", func(t *testing.T) {
		// уже есть 1500.50 по "еда"; +4000 = 5500.50 > 5000 -> 409
		rec := doRequest(t, router, http.MethodPost, "/api/transactions",
			`{"amount":4000,"category":"еда","description":"Ашан","date":"2026-09-21"}`)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
		assert.Equal(t, "бюджет превышен", decodeError(t, rec))

		// отклоненная транзакция не должна попасть в список
		list := doRequest(t, router, http.MethodGet, "/api/transactions", "")
		var items []TransactionResponse
		assert.NoError(t, json.Unmarshal(list.Body.Bytes(), &items))
		assert.Len(t, items, 1)
	})

	t.Run("bad_json", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/transactions", `{"amount":}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(decodeError(t, rec), "некорректный JSON"))
	})

	t.Run("report_summary", func(t *testing.T) {
		// В БД одна успешная транзакция: "еда" на 1500.50
		// Два одинаковых запроса подряд - второй может прийти из кеша Redis
		var first domain.ReportSummary
		for i := 0; i < 2; i++ {
			rec := doRequest(t, router, http.MethodGet,
				"/api/reports/summary?from=2026-09-01&to=2026-09-30", "")
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))

			var summary domain.ReportSummary
			assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
			if i == 0 {
				first = summary
			} else {
				assert.Equal(t, first, summary) // одинаковый результат при повторе
			}
		}
		assert.Equal(t, 1500.50, first.Total)
		assert.Len(t, first.Items, 1)
		assert.Equal(t, "еда", first.Items[0].Category)
	})

	t.Run("report_bad_date", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodGet, "/api/reports/summary?from=2026-09-01&to=baddate", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
	})
}
