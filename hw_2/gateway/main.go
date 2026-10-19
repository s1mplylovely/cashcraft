package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"gateway/internal/api"
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
	const budgetsFile = "ledger/budgets.json"
	loadBudgetsFromFile(budgetsFile)

	port := ":8080"
	log.Printf("Gateway запущен на %s", port)
	if err := http.ListenAndServe(port, api.NewRouter()); err != nil {
		log.Fatalf("Неудалось запустить сервер: %v", err)
	}
}
