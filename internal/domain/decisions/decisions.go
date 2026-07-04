// Package decisions implements decision records, structured options, status
// grouping, review metadata, and tag associations.
package decisions

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
	DB       *sql.DB
	Tags     tags.Store
	Location *time.Location
	Now      func() time.Time
}

type Decision struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Background  string   `json:"background"`
	FinalChoice string   `json:"final_choice"`
	Status      string   `json:"status"`
	ReviewDate  string   `json:"review_date"`
	ReviewNote  string   `json:"review_note"`
	Options     []Option `json:"options"`
	Tags        []string `json:"tags"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type Option struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Pros  string `json:"pros"`
	Cons  string `json:"cons"`
	Note  string `json:"note"`
	Order int    `json:"sort_order"`
}

type Input struct {
	Title       string   `json:"title"`
	Background  string   `json:"background"`
	FinalChoice string   `json:"final_choice"`
	Status      string   `json:"status"`
	ReviewDate  string   `json:"review_date"`
	ReviewNote  string   `json:"review_note"`
	Options     []Option `json:"options"`
	Tags        []string `json:"tags"`
}

func (s Service) List(ctx context.Context, status string) ([]Decision, error) {
	location, err := s.location()
	if err != nil {
		return nil, err
	}
	now := s.now()
	rows, err := s.DB.QueryContext(ctx, `SELECT id, title, background, final_choice, status, COALESCE(review_date, ''), review_note, created_at, updated_at FROM decisions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Decision{}
	for rows.Next() {
		item, err := s.scan(ctx, rows)
		if err != nil {
			return nil, err
		}
		item.Status = effectiveStatusAt(item, now, location)
		if status == "" || item.Status == status {
			items = append(items, item)
		}
	}
	return items, rows.Err()
}

func (s Service) Get(ctx context.Context, id string) (Decision, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, title, background, final_choice, status, COALESCE(review_date, ''), review_note, created_at, updated_at FROM decisions WHERE id = ?`, id)
	return s.scan(ctx, row)
}

func (s Service) Create(ctx context.Context, input Input) (Decision, error) {
	if err := validate(input); err != nil {
		return Decision{}, err
	}
	id := newID("decision")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	err := db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO decisions(id, title, background, final_choice, status, review_date, review_note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, input.Title, input.Background, input.FinalChoice, statusOrDefault(input.Status), nullString(input.ReviewDate), input.ReviewNote, now, now); err != nil {
			return err
		}
		if err := s.replaceOptions(ctx, tx, id, input.Options); err != nil {
			return err
		}
		return s.Tags.SetForEntity(ctx, tx, "decision", id, input.Tags)
	})
	if err != nil {
		return Decision{}, err
	}
	return s.Get(ctx, id)
}

func (s Service) Update(ctx context.Context, id string, input Input) (Decision, error) {
	if err := validate(input); err != nil {
		return Decision{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	err := db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE decisions SET title = ?, background = ?, final_choice = ?, status = ?, review_date = ?, review_note = ?, updated_at = ? WHERE id = ?`,
			input.Title, input.Background, input.FinalChoice, statusOrDefault(input.Status), nullString(input.ReviewDate), input.ReviewNote, now, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		if err := s.replaceOptions(ctx, tx, id, input.Options); err != nil {
			return err
		}
		return s.Tags.SetForEntity(ctx, tx, "decision", id, input.Tags)
	})
	if err != nil {
		return Decision{}, err
	}
	return s.Get(ctx, id)
}

func (s Service) Delete(ctx context.Context, id string) error {
	return db.WithinTx(ctx, s.DB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_tags WHERE entity_type = 'decision' AND entity_id = ?`, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM decisions WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

func (s Service) replaceOptions(ctx context.Context, tx *sql.Tx, decisionID string, options []Option) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM decision_options WHERE decision_id = ?`, decisionID); err != nil {
		return err
	}
	for i, option := range options {
		if option.Name == "" {
			return fmt.Errorf("%w: option name is required", ErrValidation)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO decision_options(id, decision_id, name, pros, cons, note, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			newID("option"), decisionID, option.Name, option.Pros, option.Cons, option.Note, i); err != nil {
			return err
		}
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func (s Service) scan(ctx context.Context, row scanner) (Decision, error) {
	var item Decision
	if err := row.Scan(&item.ID, &item.Title, &item.Background, &item.FinalChoice, &item.Status, &item.ReviewDate, &item.ReviewNote, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Decision{}, err
	}
	options, err := s.options(ctx, item.ID)
	if err != nil {
		return Decision{}, err
	}
	item.Options = options
	item.Tags, err = s.Tags.ListForEntity(ctx, "decision", item.ID)
	return item, err
}

func (s Service) options(ctx context.Context, id string) ([]Option, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, pros, cons, note, sort_order FROM decision_options WHERE decision_id = ? ORDER BY sort_order ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	options := []Option{}
	for rows.Next() {
		var option Option
		if err := rows.Scan(&option.ID, &option.Name, &option.Pros, &option.Cons, &option.Note, &option.Order); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}

func validate(input Input) error {
	if input.Title == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if !validStatus(statusOrDefault(input.Status)) {
		return fmt.Errorf("%w: status is invalid", ErrValidation)
	}
	if input.ReviewDate != "" {
		if _, err := time.Parse("2006-01-02", input.ReviewDate); err != nil {
			return fmt.Errorf("%w: review_date must be YYYY-MM-DD", ErrValidation)
		}
	}
	for _, option := range input.Options {
		if option.Name == "" {
			return fmt.Errorf("%w: option name is required", ErrValidation)
		}
	}
	return nil
}

func (s Service) location() (*time.Location, error) {
	if s.Location == nil {
		return nil, fmt.Errorf("decision service location is required")
	}
	return s.Location, nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func effectiveStatusAt(item Decision, now time.Time, location *time.Location) string {
	if item.Status == "已归档" {
		return item.Status
	}
	if item.ReviewDate != "" {
		if _, err := time.Parse("2006-01-02", item.ReviewDate); err == nil && item.ReviewDate <= now.In(location).Format("2006-01-02") {
			return "待复盘"
		}
	}
	return item.Status
}

func statusOrDefault(status string) string {
	if status == "" {
		return "进行中"
	}
	return status
}

func validStatus(status string) bool {
	switch status {
	case "进行中", "待复盘", "已归档":
		return true
	default:
		return false
	}
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func newID(prefix string) string {
	token, err := security.RandomToken()
	if err != nil {
		panic(err)
	}
	return prefix + "_" + token[:16]
}
