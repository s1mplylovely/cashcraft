package main

import (
	"bufio"
	"fmt"
	"os"
	"ledger/domain"
)

func loadBudgetsFromFile(path string) {
	fmt.Printf("Добавление бюджетов из %q...\n", path)
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Файл %q не открыт: %v (пропускаем загрузку)\n", path, err)
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	if err := domain.LoadBudgets(reader); err != nil {
		fmt.Printf("Ошибка загрузки бюджетов из %q: %v\n", path, err)
		return
	}
	fmt.Printf("Бюджеты успешно загружены из %q\n", path)
}

func main() {
	fmt.Println("Ledger service started")

	const budgetsFile = "ledger/budgets.json"
	loadBudgetsFromFile(budgetsFile)

	bData := []domain.Budget{
		// перезапись за период "месяц"
		{Category: "еда", Limit: 6000, Period: "месяц"},
		// новый бюджет
		{Category: "транспорт", Limit: 50000, Period: "год"},
		// ошибка - отрицательный бюджет
		{Category: "транспорт", Limit: -20, Period: "месяц"},
		// новый бюджет
		{Category: "развлечения", Limit: 5000, Period: "месяц"},
		// ошибка - нулевой бюджет
		{Category: "еда", Limit: 0, Period: "месяц"}, 
		// ошибка - пустая категория
		{Category: "", Limit: 1000, Period: "месяц"},
	}

	fmt.Printf("Добавление бюджетов из main...\n")
	for _, b := range bData {
		if err := domain.SetBudget(b); err != nil {
			fmt.Printf("Ошибка добавления бюджета %+v: %v\n", b, err)
			continue
		}
		fmt.Printf("Добавлен бюджет: %+v\n", b)
	}

	fmt.Println("\nСписок бюджетов:")
		for _, b := range domain.ListBudgets() {
			fmt.Printf("  Category: %s, Limit: %.0f, Period: %s\n",
				b.Category, b.Limit, b.Period)
		}

	txData := []domain.Transaction{
		// успешное добавление в пределах лимита
		{Amount: 1500.50, Category: "еда", Description: "Пятерочка", Date: "2026-09-19"},
		// второе добавление в пределах лимита (1500.50 + 1000 = 2500.50 < 6000)
		{Amount: 1000.00, Category: "еда", Description: "Магнит", Date: "2026-09-20"},
		// превышение лимита по категории "еда" (2500.50 + 4000 > 6000)
		{Amount: 4000.00, Category: "еда", Description: "Ашан", Date: "2026-09-21"},
		// превышение годового лимита по категории "транспорт" (50000)
		{Amount: 60000.00, Category: "транспорт", Description: "Авиабилет", Date: "2026-09-22"},
		// категория без бюджета без ограничений
		{Amount: 100000.00, Category: "другое", Description: "Телефон", Date: "2026-09-23"},
		// ошибка - нулевая сумма
		{Amount: 0.00, Category: "развлечения", Description: "Боулинг", Date: "2026-09-24"},
		// ошибка - отрицательная сумма
		{Amount: -5, Category: "еда", Description: "Тест", Date: "2026-09-25"},
		// ошибка - пустая категория
		{Amount: 100, Category: "", Description: "Тест", Date: "2026-09-26"},
		// ошибка - некорректная дата
		{Amount: 100, Category: "еда", Description: "Тест", Date: "27.09.2026"},	
	}

	for _, tx := range txData {
		if err := domain.AddTransaction(tx); err != nil {
			fmt.Printf("Ошибка добавления транзакции %+v: %v\n", tx, err)
			continue
		}
		fmt.Printf("Добавлена транзакция: %+v\n", tx)
	}

	fmt.Println("\nСписок транзакций:")
	for _, tx := range domain.ListTransactions() {
		fmt.Printf(
			"ID: %d, Amount: %.2f, Category: %s, Description: %s, Date: %s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date,
		)
	}
}
