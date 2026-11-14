package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"ledger/domain"
)

// читает бюджеты из JSON-файла и сохраняет их в БД
func loadBudgetsFromFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Файл %q не открыт: %v (пропускаем загрузку)", path, err)
		return
	}
	defer f.Close()

	if err := domain.LoadBudgets(bufio.NewReader(f)); err != nil {
		log.Printf("Ошибка загрузки бюджетов из %q: %v", path, err)
		return
	}
	log.Printf("Бюджеты загружены из %q", path)
}

func main() {
	log.Println("Ledger service started")

	// PostgreSQL (обязательно) + Redis (опционально)
	domain.MustConnectInfra()

	// первичное наполнение бюджетов из файла
	loadBudgetsFromFile("ledger/budgets.json")

	budgets, err := domain.ListBudgets()
	if err != nil {
		log.Fatalf("ListBudgets: %v", err)
	}
	fmt.Println("\nСписок бюджетов:")
	for _, b := range budgets {
		fmt.Printf("  Category: %s, Limit: %.2f, Period: %s\n", b.Category, b.Limit, b.Period)
	}

	txs, err := domain.ListTransactions()
	if err != nil {
		log.Fatalf("ListTransactions: %v", err)
	}
	fmt.Println("\nСписок транзакций:")
	for _, tx := range txs {
		fmt.Printf("  ID: %d, Amount: %.2f, Category: %s, Description: %s, Date: %s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date)
	}
}
