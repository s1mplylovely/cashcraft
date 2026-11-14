package api

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/api/transactions", transactionsRoute)
	mux.HandleFunc("/api/budgets", budgetsRoute)
	mux.HandleFunc("/api/reports/summary", reportsSummaryRoute)

	return LoggingMiddleware(mux)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{"status": "pong"})
}

func transactionsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listTransactions(w, r)
	case http.MethodPost:
		createTransaction(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
	}
}

func budgetsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listBudgets(w, r)
	case http.MethodPost:
		createBudget(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
	}
}

func reportsSummaryRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		reportSummary(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "недопустимый метод")
	}
}
