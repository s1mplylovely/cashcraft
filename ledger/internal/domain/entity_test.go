package domain

import (
	"testing"

	"ledger/internal/assert"
)

// table-driven проверка правил валидации расхода.
// Validate - чистая функция, поэтому тест и подтесты параллельные.
func TestTransaction_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		tx Transaction
		wantErr error // nil -> успех; иначе ожидается errors.Is(err, wantErr)
	}{
		{name: "валидная транзакция", tx: Transaction{Amount: 1500.50, Category: "еда", Date: "2026-09-19"}},
		{name: "нулевая сумма", tx: Transaction{Amount: 0, Category: "еда", Date: "2026-09-19"}, wantErr: ErrZeroAmount},
		{name: "отрицательная сумма", tx: Transaction{Amount: -100, Category: "еда", Date: "2026-09-19"}, wantErr: ErrNegativeAmount},
		{name: "пустая категория", tx: Transaction{Amount: 100, Category: "", Date: "2026-09-19"}, wantErr: ErrEmptyCategory},
		{name: "некорректная дата", tx: Transaction{Amount: 100, Category: "еда", Date: "27.09.2026"}, wantErr: ErrInvalidDate},
		{name: "пустая дата", tx: Transaction{Amount: 100, Category: "еда", Date: ""}, wantErr: ErrInvalidDate},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.tx.Validate()
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// table-driven проверка правил валидации бюджета
func TestBudget_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		b       Budget
		wantErr error
	}{
		{name: "валидный бюджет", b: Budget{Category: "еда", Limit: 5000, Period: "месяц"}},
		{name: "положительный лимит без периода", b: Budget{Category: "еда", Limit: 1}},
		{name: "пустая категория", b: Budget{Category: "", Limit: 5000, Period: "месяц"}, wantErr: ErrEmptyCategory},
		{name: "нулевой лимит", b: Budget{Category: "еда", Limit: 0, Period: "месяц"}, wantErr: ErrNotPositiveLimit},
		{name: "отрицательный лимит", b: Budget{Category: "еда", Limit: -100, Period: "месяц"}, wantErr: ErrNotPositiveLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.b.Validate()
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}
