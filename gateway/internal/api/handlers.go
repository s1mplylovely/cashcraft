package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ledger"
)

// зависимость от прикладного сервиса (интерфейс)
type Handlers struct {
	svc ledger.Service
}

func NewHandlers(svc ledger.Service) *Handlers {
	return &Handlers{svc: svc}
}

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

func (h *Handlers) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON: "+err.Error())
		return
	}

	tx := req.ToDomain()

	// валидация и бизнес-правило — внутри сервиса; здесь только маппинг в HTTP
	if err := h.svc.AddTransaction(r.Context(), &tx); err != nil {
		switch {
		case errors.Is(err, ledger.ErrBudgetExceeded):
			writeError(w, http.StatusConflict, "бюджет превышен")
		case ledger.IsValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, NewTransactionResponse(tx))
}

func (h *Handlers) listTransactions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListTransactions(r.Context())
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

func (h *Handlers) createBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON: "+err.Error())
		return
	}

	b := req.ToDomain()

	if err := h.svc.SetBudget(r.Context(), b); err != nil {
		if ledger.IsValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, NewBudgetResponse(b))
}

func (h *Handlers) listBudgets(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListBudgets(r.Context())
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

// отдает агрегированный отчёт за интервал [from, to];
// параметры from/to — в query, формат YYYY-MM-DD; по умолчанию — текущий месяц
func (h *Handlers) reportSummary(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := r.URL.Query().Get("from")
	if from == "" {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	to := r.URL.Query().Get("to")
	if to == "" {
		to = now.Format("2006-01-02")
	}

	summary, err := h.svc.GetReportSummary(r.Context(), from, to)
	if err != nil {
		if errors.Is(err, ledger.ErrInvalidDate) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
