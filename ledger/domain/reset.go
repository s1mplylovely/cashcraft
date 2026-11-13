package domain

// очищает внутренние коллекции транзакций и бюджетов
func Reset() {
	transactions = nil
	budgets = map[string][]Budget{}
}
