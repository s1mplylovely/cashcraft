-- +goose Up
-- +goose StatementBegin
CREATE TABLE budgets (
    id           SERIAL PRIMARY KEY,
    category     TEXT NOT NULL UNIQUE,
    limit_amount NUMERIC(14, 2) NOT NULL CHECK (limit_amount > 0),
    period       VARCHAR(10) NOT NULL CHECK (period IN ('месяц', 'год'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE expenses (
    id          SERIAL PRIMARY KEY,
    amount      NUMERIC(14, 2) NOT NULL CHECK (amount <> 0),
    category    TEXT NOT NULL,
    description TEXT,
    date        DATE NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_expenses_category_date ON expenses (category, date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_expenses_category_date;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS expenses;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS budgets;
-- +goose StatementEnd
