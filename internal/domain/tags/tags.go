// Package tags implements the shared tag dictionary and polymorphic entity
// associations used by dates, transactions, and decisions.
package tags

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"life-ledger/internal/security"
)

var allowedEntities = map[string]bool{
	"important_date": true,
	"transaction":    true,
	"decision":       true,
}

type Store struct {
	DB *sql.DB
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s Store) SetForEntity(ctx context.Context, tx *sql.Tx, entityType, entityID string, names []string) error {
	if !allowedEntities[entityType] {
		return fmt.Errorf("invalid entity type")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_tags WHERE entity_type = ? AND entity_id = ?`, entityType, entityID); err != nil {
		return err
	}
	for _, name := range normalizeNames(names) {
		tag, err := s.ensure(ctx, tx, name)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO entity_tags(entity_type, entity_id, tag_id, created_at) VALUES (?, ?, ?, ?)`,
			entityType, entityID, tag.ID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ListForEntity(ctx context.Context, entityType, entityID string) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT t.name FROM tags t
		JOIN entity_tags et ON et.tag_id = t.id
		WHERE et.entity_type = ? AND et.entity_id = ?
		ORDER BY t.name`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s Store) ListForEntities(ctx context.Context, entityType string, entityIDs []string) (map[string][]string, error) {
	if !allowedEntities[entityType] {
		return nil, fmt.Errorf("invalid entity type")
	}
	result := make(map[string][]string, len(entityIDs))
	if len(entityIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, 0, len(entityIDs))
	args := []any{entityType}
	for _, id := range entityIDs {
		result[id] = []string{}
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT et.entity_id, t.name FROM tags t
		JOIN entity_tags et ON et.tag_id = t.id
		WHERE et.entity_type = ? AND et.entity_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY et.entity_id, t.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var entityID string
		var name string
		if err := rows.Scan(&entityID, &name); err != nil {
			return nil, err
		}
		result[entityID] = append(result[entityID], name)
	}
	return result, rows.Err()
}

func (s Store) Search(ctx context.Context, query string) ([]Tag, error) {
	query = strings.TrimSpace(query)
	sqlText := `SELECT id, name FROM tags ORDER BY name LIMIT 50`
	args := []any{}
	if query != "" {
		sqlText = `SELECT id, name FROM tags WHERE name LIKE ? ORDER BY name LIMIT 50`
		args = append(args, "%"+query+"%")
	}
	rows, err := s.DB.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Tag{}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func (s Store) ensure(ctx context.Context, tx *sql.Tx, name string) (Tag, error) {
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tags(id, name, created_at) VALUES (?, ?, ?)`, newID("tag"), name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return Tag{}, err
	}
	var tag Tag
	if err := tx.QueryRowContext(ctx, `SELECT id, name FROM tags WHERE name = ?`, name).Scan(&tag.ID, &tag.Name); err != nil {
		return Tag{}, err
	}
	return tag, nil
}

func normalizeNames(names []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		if len([]rune(trimmed)) > 32 {
			trimmed = string([]rune(trimmed)[:32])
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func newID(prefix string) string {
	token, err := security.RandomToken()
	if err != nil {
		panic(err)
	}
	return prefix + "_" + token[:16]
}
