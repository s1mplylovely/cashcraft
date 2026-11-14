package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/internal/assert"
	"ledger"
)

// in-memory реализация ledger.Service для HTTP-тестов
type fakeService struct {
	budgets map[string]ledger.Budget
	txs     []ledger.Transaction
	nextID  int
}

func newFakeService() *fakeService {
	return &fakeService{budgets: map[string]ledger.Budget{}, nextID: 1}
}

func (s *fakeService) SetBudget(_ context.Context, b ledger.Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	s.budgets[b.Category] = b
	return nil
}

func (s *fakeService) ListBudgets(_ context.Context) ([]ledger.Budget, error) {
	out := make([]ledger.Budget, 0, len(s.budgets))
	for _, b := range s.budgets {
		out = append(out, b)
	}
	return out, nil
}

func (s *fakeService) AddTransaction(_ context.Context, tx *ledger.Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}
	if b, ok := s.budgets[tx.Category]; ok {
		var spent float64
		for _, t := range s.txs {
			if t.Category == tx.Category {
				spent += t.Amount
			}
		}
		if spent+tx.Amount > b.Limit {
			return ledger.ErrBudgetExceeded
		}
	}
	tx.ID = s.nextID
	s.nextID++
	s.txs = append(s.txs, *tx)
	return nil
}

func (s *fakeService) ListTransactions(_ context.Context) ([]ledger.Transaction, error) {
	return s.txs, nil
}

func (s *fakeService) GetReportSummary(_ context.Context, from, to string) (ledger.ReportSummary, error) {
	byCat := map[string]float64{}
	for _, t := range s.txs {
		if t.Date >= from && t.Date <= to {
			byCat[t.Category] += t.Amount
		}
	}
	summary := ledger.ReportSummary{From: from, To: to, Items: []ledger.ReportItem{}}
	for cat, total := range byCat {
		summary.Items = append(summary.Items, ledger.ReportItem{Category: cat, Total: total})
		summary.Total += total
	}
	return summary, nil
}

// хелперы

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

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("не удалось распарсить тело ошибки %q: %v", rec.Body.String(), err)
	}
	return payload["error"]
}

const contentTypeJSON = "application/json; charset=utf-8"

func TestAPI_Budgets(t *testing.T) {
	t.Parallel()
	router := NewRouter(newFakeService())

	t.Run("ok", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/budgets",
			`{"category":"еда","limit":5000,"period":"месяц"}`)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
		var b BudgetResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &b))
		assert.Equal(t, "еда", b.Category)
		assert.Equal(t, 5000.0, b.Limit)
	})

	t.Run("list", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodGet, "/api/budgets", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		var list []BudgetResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Len(t, list, 1)
	})

	t.Run("bad_empty_category", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/budgets",
			`{"category":"","limit":5000,"period":"месяц"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.True(t, decodeError(t, rec) != "")
	})

	t.Run("bad_json", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/budgets", `{"limit":}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.True(t, strings.HasPrefix(decodeError(t, rec), "некорректный JSON"))
	})
}

func TestAPI_Transactions(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	_ = svc.SetBudget(context.Background(), ledger.Budget{Category: "еда", Limit: 5000, Period: "месяц"})
	router := NewRouter(svc)

	t.Run("ok", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/transactions",
			`{"amount":1500.50,"category":"еда","description":"Пятёрочка","date":"2026-09-19"}`)
		assert.Equal(t, http.StatusCreated, rec.Code)
		var tx TransactionResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tx))
		assert.Equal(t, 1, tx.ID)
		assert.Equal(t, "2026-09-19", tx.Date)
	})

	t.Run("list", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodGet, "/api/transactions", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		var list []TransactionResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Len(t, list, 1)
	})

	t.Run("exceeded", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/transactions",
			`{"amount":4000,"category":"еда","description":"Ашан","date":"2026-09-21"}`)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "бюджет превышен", decodeError(t, rec))
	})

	t.Run("validation_400", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/transactions",
			`{"amount":-5,"category":"еда","date":"2026-09-21"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.True(t, decodeError(t, rec) != "")
	})

	t.Run("bad_json", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodPost, "/api/transactions", `{"amount":}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.True(t, strings.HasPrefix(decodeError(t, rec), "некорректный JSON"))
	})

	t.Run("report_summary", func(t *testing.T) {
		rec := doRequest(t, router, http.MethodGet,
			"/api/reports/summary?from=2026-09-01&to=2026-09-30", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		var summary ledger.ReportSummary
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
		assert.Equal(t, 1500.50, summary.Total)
		assert.Len(t, summary.Items, 1)
	})
}
