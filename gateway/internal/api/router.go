package api

import (
	"net/http"

	"ledger"
)

// собирает маршрутизатор поверх прикладного сервиса; обработчики
// получают сервис через конструктор NewHandlers
func NewRouter(svc ledger.Service) http.Handler {
	h := NewHandlers(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/api/transactions", h.transactionsRoute)
	mux.HandleFunc("/api/budgets", h.budgetsRoute)
	mux.HandleFunc("/api/reports/summary", h.reportsSummaryRoute)

	return LoggingMiddleware(mux)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "pong"})
}

func (h *Handlers) transactionsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTransactions(w, r)
	case http.MethodPost:
		h.createTransaction(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
	}
}

func (h *Handlers) budgetsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBudgets(w, r)
	case http.MethodPost:
		h.createBudget(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
	}
}

func (h *Handlers) reportsSummaryRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.reportSummary(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
	}
}
