package transactions

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Budget struct {
	ID         string  `json:"id"`
	Month      string  `json:"month"`
	Category   string  `json:"category"`
	Amount     string  `json:"amount"`
	Used       string  `json:"used"`
	Remaining  string  `json:"remaining"`
	UsageRatio float64 `json:"usage_ratio"`
	Overspent  bool    `json:"overspent"`
}

type BudgetInput struct {
	Month    string `json:"month"`
	Category string `json:"category"`
	Amount   string `json:"amount"`
}

func (s Service) ListBudgets(ctx context.Context, month string) ([]Budget, error) {
	sqlText := `SELECT id, month, category, amount_cents FROM budgets ORDER BY month DESC, category ASC`
	args := []any{}
	if month != "" {
		sqlText = `SELECT id, month, category, amount_cents FROM budgets WHERE month = ? ORDER BY category ASC`
		args = append(args, month)
	}
	rows, err := s.DB.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Budget{}
	for rows.Next() {
		var id, budgetMonth, category string
		var amount int64
		if err := rows.Scan(&id, &budgetMonth, &category, &amount); err != nil {
			return nil, err
		}
		used, err := s.budgetUsed(ctx, budgetMonth, category)
		if err != nil {
			return nil, err
		}
		result = append(result, budgetFromCents(id, budgetMonth, category, amount, used))
	}
	return result, rows.Err()
}

func (s Service) SaveBudget(ctx context.Context, input BudgetInput) (Budget, error) {
	if err := validateBudget(input); err != nil {
		return Budget{}, err
	}
	amount, _ := parseAmount(input.Amount)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var existingID string
	err := s.DB.QueryRowContext(ctx, `SELECT id FROM budgets WHERE month = ? AND category = ?`, input.Month, input.Category).Scan(&existingID)
	if err == nil {
		if _, err := s.DB.ExecContext(ctx, `UPDATE budgets SET amount_cents = ?, updated_at = ? WHERE id = ?`, amount, now, existingID); err != nil {
			return Budget{}, err
		}
		used, err := s.budgetUsed(ctx, input.Month, input.Category)
		return budgetFromCents(existingID, input.Month, input.Category, amount, used), err
	}
	if err != sql.ErrNoRows {
		return Budget{}, err
	}
	id := newID("budget")
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO budgets(id, month, category, amount_cents, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, id, input.Month, input.Category, amount, now, now); err != nil {
		return Budget{}, err
	}
	used, err := s.budgetUsed(ctx, input.Month, input.Category)
	return budgetFromCents(id, input.Month, input.Category, amount, used), err
}

func (s Service) DeleteBudget(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM budgets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func validateBudget(input BudgetInput) error {
	if _, err := time.Parse("2006-01", input.Month); err != nil {
		return fmt.Errorf("%w: month must be YYYY-MM", ErrValidation)
	}
	if input.Category == "" {
		return fmt.Errorf("%w: category is required", ErrValidation)
	}
	if _, err := parseAmount(input.Amount); err != nil {
		return err
	}
	return nil
}

func (s Service) budgetUsed(ctx context.Context, month, category string) (int64, error) {
	var used sql.NullInt64
	err := s.DB.QueryRowContext(ctx, `SELECT SUM(amount_cents) FROM transactions
		WHERE type = '支出' AND include_budget = 1 AND category = ? AND occurred_date >= ? AND occurred_date <= ?`,
		category, month+"-01", month+"-31").Scan(&used)
	if err != nil {
		return 0, err
	}
	if !used.Valid {
		return 0, nil
	}
	return used.Int64, nil
}

func budgetFromCents(id, month, category string, amount, used int64) Budget {
	remaining := amount - used
	ratio := 0.0
	if amount > 0 {
		ratio = float64(used) / float64(amount)
	}
	return Budget{
		ID:         id,
		Month:      month,
		Category:   category,
		Amount:     formatAmount(amount),
		Used:       formatAmount(used),
		Remaining:  formatAmount(remaining),
		UsageRatio: ratio,
		Overspent:  used > amount,
	}
}
