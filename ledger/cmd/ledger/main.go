// Собирает сервис через фабрику, наполняет бюджеты из budgets.json
// и печатает текущее состояние. Демонстрирует, что внешний код работает
// только с интерфейсом service.Service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"ledger"
)

func main() {
	log.Println("Ledger service started")

	ctx := context.Background()
	svc, closeFn, err := ledger.New(ctx)
	if err != nil {
		log.Fatalf("Ledger: не удалось инициализировать сервис: %v", err)
	}
	defer closeFn()

	loadBudgetsFromFile(ctx, svc, "ledger/budgets.json")

	budgets, err := svc.ListBudgets(ctx)
	if err != nil {
		log.Fatalf("ListBudgets: %v", err)
	}
	fmt.Println("\nСписок бюджетов:")
	for _, b := range budgets {
		fmt.Printf("  Category: %s, Limit: %.2f, Period: %s\n", b.Category, b.Limit, b.Period)
	}

	txs, err := svc.ListTransactions(ctx)
	if err != nil {
		log.Fatalf("ListTransactions: %v", err)
	}
	fmt.Println("\nСписок транзакций:")
	for _, tx := range txs {
		fmt.Printf("  ID: %d, Amount: %.2f, Category: %s, Description: %s, Date: %s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date)
	}
}

// читает бюджеты из JSON-файла и сохраняет их через сервис
func loadBudgetsFromFile(ctx context.Context, svc ledger.Service, path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Файл %q не открыт: %v (пропускаем загрузку)", path, err)
		return
	}
	defer f.Close()

	var budgets []ledger.Budget
	if err := json.NewDecoder(f).Decode(&budgets); err != nil {
		log.Printf("Не удалось распарсить %q: %v", path, err)
		return
	}
	for _, b := range budgets {
		if err := svc.SetBudget(ctx, b); err != nil {
			log.Printf("Не удалось сохранить бюджет %+v: %v", b, err)
			continue
		}
	}
	log.Printf("Бюджеты загружены из %q", path)
}
