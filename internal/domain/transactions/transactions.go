// Package transactions implements bill CRUD, filtering, lightweight summaries,
// and month/category budgets.
package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"life-ledger/internal/db"
	"life-ledger/internal/domain/tags"
	"life-ledger/internal/security"
)

var ErrValidation = errors.New("validation failed")

type Service struct {
	DB   *sql.DB
	Tags tags.Store
}

type Transaction struct {
	ID            string   `json:"id"`
	Date          string   `json:"date"`
	Time          string   `json:"time"`
	Type          string   `json:"type"`
	Amount        string   `json:"amount"`
	Category      string   `json:"category"`
	IncludeIncome bool     `json:"include_income"`
	IncludeBudget bool     `json:"include_budget"`
	Ledger        string   `json:"ledger"`
	Counterparty  string   `json:"counterparty"`
	Account       string   `json:"account"`
	Note          string   `json:"note"`
	Tags          []string `json:"tags"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type Input struct {
	Date          string   `json:"date"`
	Time          string   `json:"time"`
	Type          string   `json:"type"`
	Amount        string   `json:"amount"`
	Category      string   `json:"category"`
	IncludeIncome bool     `json:"include_income"`
	IncludeBudget bool     `json:"include_budget"`
	Ledger        string   `json:"ledger"`
	Counterparty  string   `json:"counterparty"`
	Account       string   `json:"account"`
	Note          string   `json:"note"`
	Tags          []string `json:"tags"`
}

type Filter struct {
	From     string
	To       string
	Type     string
	Category string
	Account  string
	Tag      string
	Page     int
	PageSize int
}

type ListResult struct {
	Items    []Transaction `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
}

type Summary struct {
	Income     string            `json:"income"`
	Expense    string            `json:"expense"`
	Balance    string            `json:"balance"`
	ByCategory []CategorySummary `json:"by_category"`
}

type CategorySummary struct {
	Category string  `json:"category"`
	Expense  string  `json:"expense"`
	Ratio    float64 `json:"ratio"`
}

func (s Service) List(ctx context.Context, filter Filter) (ListResult, error) {
	filter = normalizeFilter(filter)
	where, args := filterWhere(filter)
	totalSQL := `SELECT COUNT(1) FROM transactions ` + where
	var total int
	if err := s.DB.QueryRowContext(ctx, totalSQL, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.DB.QueryContext(ctx, `SELECT id, occurred_date, occurred_time, type, amount_cents, category, include_income, include_budget, ledger, counterparty, account, note, created_at, updated_at
		FROM transactions `+where+` ORDER BY occurred_date DESC, occurred_time DESC, created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	items := []Transaction{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return ListResult{}, err
		}
		item.Tags, err = s.Tags.ListForEntity(ctx, "transaction", item.ID)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, item)
	}
	return ListResult{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, rows.Err()
}

func (s Service) Summary(ctx context.Context, filter Filter) (Summary, error) {
	filter.Page = 1
	filter.PageSize = 5000
	list, err := s.List(ctx, filter)
	if err != nil {
		return Summary{}, err
	}
	var income, expense int64
	byCategory := map[string]int64{}
	for _, item := range list.Items {
		cents, _ := parseAmount(item.Amount)
		if !item.IncludeIncome {
			continue
		}
		if item.Type == "收入" {
			income += cents
		}
		if item.Type == "支出" {
			expense += cents
			byCategory[item.Category] += cents
		}
	}
	categories := make([]CategorySummary, 0, len(byCategory))
	for category, value := range byCategory {
		ratio := 0.0
		if expense > 0 {
			ratio = float64(value) / float64(expense)
		}
		categories = append(categories, CategorySummary{Category: category, Expense: formatAmount(value), Ratio: ratio})
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].Category < categories[j].Category })
	return Summary{Income: formatAmount(income), Expense: formatAmount(expense), Balance: formatAmount(income - expense), ByCategory: categories}, nil
}

func (s Service) Get(ctx context.Context, id string) (Transaction, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, occurred_date, occurred_time, type, amount_cents, category, include_income, include_budget, ledger, counterparty, account, note, created_at, updated_at FROM transactions WHERE id = ?`, id)
	item, err := scan(row)
	if err != nil {
		return Transaction{}, err
	}
	item.Tags, err = s.Tags.ListForEntity(ctx, "transaction", item.ID)
	return item, err
}

func (s Service) Create(ctx context.Context, input Input) (Transaction, error) {
	if err := validate(input); err != nil {
		return Transaction{}, err
	}
	cents, _ := parseAmount(input.Amount)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := newID("txn")
	err := db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO transactions
			(id, occurred_date, occurred_time, type, amount_cents, category, include_income, include_budget, ledger, counterparty, account, note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, input.Date, input.Time, input.Type, cents, input.Category, boolInt(input.IncludeIncome), boolInt(input.IncludeBudget), input.Ledger, input.Counterparty, input.Account, input.Note, now, now); err != nil {
			return err
		}
		return s.Tags.SetForEntity(ctx, tx, "transaction", id, input.Tags)
	})
	if err != nil {
		return Transaction{}, err
	}
	return s.Get(ctx, id)
}

func (s Service) Update(ctx context.Context, id string, input Input) (Transaction, error) {
	if err := validate(input); err != nil {
		return Transaction{}, err
	}
	cents, _ := parseAmount(input.Amount)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	err := db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE transactions SET occurred_date = ?, occurred_time = ?, type = ?, amount_cents = ?, category = ?, include_income = ?, include_budget = ?, ledger = ?, counterparty = ?, account = ?, note = ?, updated_at = ? WHERE id = ?`,
			input.Date, input.Time, input.Type, cents, input.Category, boolInt(input.IncludeIncome), boolInt(input.IncludeBudget), input.Ledger, input.Counterparty, input.Account, input.Note, now, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		return s.Tags.SetForEntity(ctx, tx, "transaction", id, input.Tags)
	})
	if err != nil {
		return Transaction{}, err
	}
	return s.Get(ctx, id)
}

func (s Service) Delete(ctx context.Context, id string) error {
	return db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_tags WHERE entity_type = 'transaction' AND entity_id = ?`, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

func normalizeFilter(filter Filter) Filter {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 50
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}
	return filter
}

func filterWhere(filter Filter) (string, []any) {
	clauses := []string{}
	args := []any{}
	if filter.From != "" {
		clauses = append(clauses, "occurred_date >= ?")
		args = append(args, filter.From)
	}
	if filter.To != "" {
		clauses = append(clauses, "occurred_date <= ?")
		args = append(args, filter.To)
	}
	if filter.Type != "" {
		clauses = append(clauses, "type = ?")
		args = append(args, filter.Type)
	}
	if filter.Category != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, filter.Category)
	}
	if filter.Account != "" {
		clauses = append(clauses, "account = ?")
		args = append(args, filter.Account)
	}
	if filter.Tag != "" {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM entity_tags et JOIN tags t ON t.id = et.tag_id
			WHERE et.entity_type = 'transaction' AND et.entity_id = transactions.id AND t.name = ?
		)`)
		args = append(args, filter.Tag)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner) (Transaction, error) {
	var item Transaction
	var cents int64
	var includeIncome, includeBudget int
	err := row.Scan(&item.ID, &item.Date, &item.Time, &item.Type, &cents, &item.Category, &includeIncome, &includeBudget, &item.Ledger, &item.Counterparty, &item.Account, &item.Note, &item.CreatedAt, &item.UpdatedAt)
	item.Amount = formatAmount(cents)
	item.IncludeIncome = includeIncome == 1
	item.IncludeBudget = includeBudget == 1
	return item, err
}

func validate(input Input) error {
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return fmt.Errorf("%w: date must be YYYY-MM-DD", ErrValidation)
	}
	if _, err := time.Parse("15:04", input.Time); err != nil {
		return fmt.Errorf("%w: time must be HH:MM", ErrValidation)
	}
	if input.Type != "收入" && input.Type != "支出" {
		return fmt.Errorf("%w: type is invalid", ErrValidation)
	}
	if _, err := parseAmount(input.Amount); err != nil {
		return err
	}
	if input.Category == "" || input.Ledger == "" {
		return fmt.Errorf("%w: category and ledger are required", ErrValidation)
	}
	return nil
}

func parseAmount(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%w: amount is required", ErrValidation)
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("%w: amount is invalid", ErrValidation)
	}
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || yuan < 0 {
		return 0, fmt.Errorf("%w: amount is invalid", ErrValidation)
	}
	fen := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, fmt.Errorf("%w: amount is invalid", ErrValidation)
		}
		padded := parts[1] + strings.Repeat("0", 2-len(parts[1]))
		fen, err = strconv.ParseInt(padded, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: amount is invalid", ErrValidation)
		}
	}
	cents := yuan*100 + fen
	if cents <= 0 {
		return 0, fmt.Errorf("%w: amount must be positive", ErrValidation)
	}
	return cents, nil
}

func formatAmount(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func newID(prefix string) string {
	token, err := security.RandomToken()
	if err != nil {
		panic(err)
	}
	return prefix + "_" + token[:16]
}
