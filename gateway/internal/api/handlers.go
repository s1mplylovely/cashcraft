package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"ledger/domain"
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

	if err := domain.AddTransaction(tx); err != nil {
		if errors.Is(err, domain.ErrBudgetExceeded) {
			writeError(w, http.StatusConflict, "бюджет превышен")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	list := domain.ListTransactions()
	created := list[len(list)-1]

	writeJSON(w, http.StatusCreated, NewTransactionResponse(created))
}

func listTransactions(w http.ResponseWriter, r *http.Request) {
	list := domain.ListTransactions()
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
	list := domain.ListBudgets()
	resp := make([]BudgetResponse, 0, len(list))
	for _, b := range list {
		resp = append(resp, NewBudgetResponse(b))
	}
	writeJSON(w, http.StatusOK, resp)
}
