// Package importantdates implements important date validation, persistence,
// tag association, and list filtering.
package importantdates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

type ImportantDate struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Date       string   `json:"date"`
	DateType   string   `json:"date_type"`
	RepeatRule string   `json:"repeat_rule"`
	Note       string   `json:"note"`
	Tags       []string `json:"tags"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type Input struct {
	Title      string   `json:"title"`
	Date       string   `json:"date"`
	DateType   string   `json:"date_type"`
	RepeatRule string   `json:"repeat_rule"`
	Note       string   `json:"note"`
	Tags       []string `json:"tags"`
}

func (s Service) List(ctx context.Context, tag string) ([]ImportantDate, error) {
	sqlText := `SELECT id, title, date, date_type, repeat_rule, note, created_at, updated_at FROM important_dates ORDER BY date ASC, created_at DESC`
	args := []any{}
	if tag != "" {
		sqlText = `SELECT d.id, d.title, d.date, d.date_type, d.repeat_rule, d.note, d.created_at, d.updated_at
			FROM important_dates d
			WHERE EXISTS (
				SELECT 1 FROM entity_tags et JOIN tags t ON t.id = et.tag_id
				WHERE et.entity_type = 'important_date' AND et.entity_id = d.id AND t.name = ?
			)
			ORDER BY d.date ASC, d.created_at DESC`
		args = append(args, tag)
	}
	rows, err := s.DB.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ImportantDate{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		item.Tags, err = s.Tags.ListForEntity(ctx, "important_date", item.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Service) Get(ctx context.Context, id string) (ImportantDate, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, title, date, date_type, repeat_rule, note, created_at, updated_at FROM important_dates WHERE id = ?`, id)
	item, err := scan(row)
	if err != nil {
		return ImportantDate{}, err
	}
	item.Tags, err = s.Tags.ListForEntity(ctx, "important_date", item.ID)
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, err
}

func (s Service) Create(ctx context.Context, input Input) (ImportantDate, error) {
	if err := validate(input); err != nil {
		return ImportantDate{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := ImportantDate{
		ID:         newID("date"),
		Title:      input.Title,
		Date:       input.Date,
		DateType:   input.DateType,
		RepeatRule: repeatRule(input.RepeatRule),
		Note:       input.Note,
		Tags:       input.Tags,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err := db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO important_dates(id, title, date, date_type, repeat_rule, note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.Title, item.Date, item.DateType, item.RepeatRule, item.Note, item.CreatedAt, item.UpdatedAt); err != nil {
			return err
		}
		return s.Tags.SetForEntity(ctx, tx, "important_date", item.ID, input.Tags)
	})
	if err != nil {
		return ImportantDate{}, err
	}
	return s.Get(ctx, item.ID)
}

func (s Service) Update(ctx context.Context, id string, input Input) (ImportantDate, error) {
	if err := validate(input); err != nil {
		return ImportantDate{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	err := db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE important_dates SET title = ?, date = ?, date_type = ?, repeat_rule = ?, note = ?, updated_at = ? WHERE id = ?`,
			input.Title, input.Date, input.DateType, repeatRule(input.RepeatRule), input.Note, now, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		return s.Tags.SetForEntity(ctx, tx, "important_date", id, input.Tags)
	})
	if err != nil {
		return ImportantDate{}, err
	}
	return s.Get(ctx, id)
}

func (s Service) Delete(ctx context.Context, id string) error {
	return db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_tags WHERE entity_type = 'important_date' AND entity_id = ?`, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM important_dates WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner) (ImportantDate, error) {
	var item ImportantDate
	err := row.Scan(&item.ID, &item.Title, &item.Date, &item.DateType, &item.RepeatRule, &item.Note, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func validate(input Input) error {
	if input.Title == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if input.DateType == "" {
		return fmt.Errorf("%w: date_type is required", ErrValidation)
	}
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return fmt.Errorf("%w: date must be YYYY-MM-DD", ErrValidation)
	}
	if !validRepeat(repeatRule(input.RepeatRule)) {
		return fmt.Errorf("%w: repeat_rule is invalid", ErrValidation)
	}
	return nil
}

func repeatRule(value string) string {
	if value == "" {
		return "不重复"
	}
	return value
}

func validRepeat(value string) bool {
	switch value {
	case "不重复", "每年", "每月", "每周":
		return true
	default:
		return false
	}
}

func newID(prefix string) string {
	token, err := security.RandomToken()
	if err != nil {
		panic(err)
	}
	return prefix + "_" + token[:16]
}
