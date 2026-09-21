package main

import (
	"errors"
	"fmt"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

var transactions []Transaction

var ErrZeroAmount = errors.New("сумма транзакции не может быть 0")

func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return ErrZeroAmount
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

func ListTransactions() []Transaction {
	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}

func main() {
	fmt.Println("Ledger service started")

	transactions = []Transaction{}

	txData := []Transaction{
		{Amount: 1500.50, Category: "еда", Description: "Пятерочка", Date: "2026-09-19"},
		{Amount: 1000.00, Category: "транспорт", Description: "Такси", Date: "2026-09-20"},
		{Amount: 0.00, Category: "развлечения", Description: "Боулинг", Date: "2026-09-21"},
	}

	for _, tx := range txData {
		if err := AddTransaction(tx); err != nil {
			fmt.Printf("Ошибка добавления транзакции %+v: %v\n", tx, err)
			continue
		}
		fmt.Printf("Добавлена транзакция: %+v\n", tx)
	}

	fmt.Println("\nСписок транзакций:")
	for _, tx := range ListTransactions() {
		fmt.Printf(
			"ID: %d, Amount: %.2f, Category: %s, Description: %s, Date: %s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date,
		)
	}
}
