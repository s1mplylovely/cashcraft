package api

import "ledger"

type CreateTransactionRequest struct {
	Amount float64
	Category string
	Description string
	Date string
}

func (r CreateTransactionRequest) ToDomain() ledger.Transaction {
	return ledger.Transaction{
		Amount: r.Amount,
		Category: r.Category,
		Description: r.Description,
		Date: r.Date,
	}
}

type TransactionResponse struct {
	ID int
	Amount float64
	Category string
	Description string
	Date string
}

func NewTransactionResponse(tx ledger.Transaction) TransactionResponse {
	return TransactionResponse{
		ID: tx.ID,
		Amount: tx.Amount,
		Category: tx.Category,
		Description: tx.Description,
		Date: tx.Date,
	}
}

type CreateBudgetRequest struct {
	Category string
	Limit float64
	Period string
}

func (r CreateBudgetRequest) ToDomain() ledger.Budget {
	return ledger.Budget{
		Category: r.Category,
		Limit: r.Limit,
		Period: r.Period,
	}
}

type BudgetResponse struct {
	Category string
	Limit float64
	Period string
}

func NewBudgetResponse(b ledger.Budget) BudgetResponse {
	return BudgetResponse{
		Category: b.Category,
		Limit: b.Limit,
		Period: b.Period,
	}
}
