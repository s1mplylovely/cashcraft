package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"
)

var budgetsFile = "budgets.json"

type Validatable interface {
	Validate() error
}

type Transaction struct {
	ID int
	Amount float64
	Category string
	Description string
	Date string
}

type Budget struct {
	Category string
	Limit float64
	Period string // "месяц"/"год"
}

var transactions []Transaction

var budgets = map[string][]Budget{
	"еда": {
		{Category: "еда", Limit: 5000, Period: "месяц"},
		{Category: "еда", Limit: 60000, Period: "год"},
	},
	"транспорт": {
		{Category: "транспорт", Limit: 3000, Period: "месяц"},
	},
}

var (
	ErrZeroAmount = errors.New("сумма транзакции не может быть 0")
	ErrNegativeAmount = errors.New("сумма транзакции не может быть отрицательной")
	ErrEmptyCategory = errors.New("категория не может быть пустой")
	ErrInvalidDate = errors.New("некорректный формат даты, ожидается YYYY-MM-DD")
	ErrNotPositiveLimit = errors.New("лимит бюджета должен быть положительным числом")
	ErrBudgetExceeded = errors.New("бюджет превышен")
)

func (tx Transaction) Validate() error {
	if tx.Amount == 0 {
		return ErrZeroAmount
	}
	if tx.Amount < 0 {
		return fmt.Errorf("%w: %.2f", ErrNegativeAmount, tx.Amount)
	}
	if tx.Category == "" {
		return ErrEmptyCategory
	}
	if _, err := time.Parse("2006-01-02", tx.Date); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidDate, tx.Date)
	}
	return nil
}

func (b Budget) Validate() error {
	if b.Category == "" {
		return ErrEmptyCategory
	}
	if b.Limit <= 0 {
		return fmt.Errorf("%w: %.2f", ErrNotPositiveLimit, b.Limit)
	}
	return nil
}

func CheckValid(v Validatable) error {
	return v.Validate()
}

// сумма добавленных транзакций по категории
func currentSumByCategory(category string) float64 {
	var sum float64
	for _, tx := range transactions {
		if tx.Category == category {
			sum += tx.Amount
		}
	}
	return sum
}

// не превысит ли новая транзакция лимит бюджета
func checkBudget(tx Transaction) error {
	categoryBudgets, ok := budgets[tx.Category]
	if !ok || len(categoryBudgets) == 0 {
		return nil
	}

	currentSum := currentSumByCategory(tx.Category)
	newSum := currentSum + tx.Amount

	for _, b := range categoryBudgets {
		if newSum > b.Limit {
			return fmt.Errorf("%w: категория %q, лимит %.2f (%s), сумма с новой транзакцией %.2f",
				ErrBudgetExceeded, tx.Category, b.Limit, b.Period, newSum)
		}
	}
	return nil
}

func AddTransaction(tx Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	if err := checkBudget(tx); err != nil {
		return err
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

func removeByPeriod(categoryBudgets []Budget, period string) []Budget {
	return slices.DeleteFunc(categoryBudgets, func(budget Budget) bool {
		return budget.Period == period
	})
}

// добавляет бюджет
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}

	categoryBudgets := budgets[b.Category]
	categoryBudgets = removeByPeriod(categoryBudgets, b.Period)
	categoryBudgets = append(categoryBudgets, b)
	budgets[b.Category] = categoryBudgets
	return nil
}

// читает JSON-массив бюджетов и добавляет через SetBudget
func LoadBudgets(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("не удалось прочитать данные бюджетов: %w", err)
	}

	var loaded []Budget
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("не удалось распарсить JSON бюджетов: %w", err)
	}

	for _, b := range loaded {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("не удалось установить бюджет %+v: %w", b, err)
		}
		fmt.Printf("Добавлен бюджет: %+v\n", b)
	}
	return nil
}

func loadBudgetsFromFile(path string) {
	fmt.Printf("Добавление бюджетов из %q...\n", path)
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Файл %q не открыт: %v (пропускаем загрузку)\n", path, err)
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	if err := LoadBudgets(reader); err != nil {
		fmt.Printf("Ошибка загрузки бюджетов из %q: %v\n", path, err)
		return
	}
	fmt.Printf("Бюджеты успешно загружены из %q\n", path)
}

func main() {
	fmt.Println("Ledger service started")

	loadBudgetsFromFile(budgetsFile)

	bData := []Budget{
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
		if err := SetBudget(b); err != nil {
			fmt.Printf("Ошибка добавления бюджета %+v: %v\n", b, err)
			continue
		}
		fmt.Printf("Добавлен бюджет: %+v\n", b)
	}

	fmt.Println("\nСписок бюджетов:")
	for key, list := range budgets {
		fmt.Printf("%s:\n", key)
		for _, b := range list {
			fmt.Printf("  Category: %s, Limit: %.0f, Period: %s\n",
				b.Category, b.Limit, b.Period)
		}
	}

	transactions = []Transaction{}

	txData := []Transaction{
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
