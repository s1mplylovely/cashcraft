package domain

import (
	"encoding/json"
	"fmt"
	"io"
)

// читает JSON-массив бюджетов из r и сохраняет каждый через
// SetBudget (UPSERT в БД) (используется для первичного наполнения из файла)
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
	}
	return nil
}
