package api

import (
	"encoding/json"
	"errors"
	"ledger/domain"
	"net/http"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON: "+err.Error())
		return
	}

	tx := req.ToDomain()

	if err := tx.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// проверяет лимит бюджета и записывает id в tx.ID
	if err := domain.AddTransaction(&tx); err != nil {
		if errors.Is(err, domain.ErrBudgetExceeded) {
			writeError(w, http.StatusConflict, "бюджет превышен")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, NewTransactionResponse(tx))
}

func listTransactions(w http.ResponseWriter, r *http.Request) {
	list, err := domain.ListTransactions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	resp := make([]TransactionResponse, 0, len(list))
	for _, tx := range list {
		resp = append(resp, NewTransactionResponse(tx))
	}
	writeJSON(w, http.StatusOK, resp)
}

func createBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON: "+err.Error())
		return
	}

	b := req.ToDomain()

	if err := b.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := domain.SetBudget(b); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, NewBudgetResponse(b))
}

func listBudgets(w http.ResponseWriter, r *http.Request) {
	list, err := domain.ListBudgets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	resp := make([]BudgetResponse, 0, len(list))
	for _, b := range list {
		resp = append(resp, NewBudgetResponse(b))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Отдает агрегированный отчет за интервал [from, to].
// Параметры from/to — в query, формат YYYY-MM-DD, по умолчанию берется
// текущий месяц
func reportSummary(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := r.URL.Query().Get("from")
	if from == "" {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	to := r.URL.Query().Get("to")
	if to == "" {
		to = now.Format("2006-01-02")
	}

	summary, err := domain.GetReportSummary(r.Context(), from, to)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidDate) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
