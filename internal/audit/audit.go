// Package audit writes and reads structured security and high-risk operation
// events. SQLite audit_events is the source of truth.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"life-ledger/internal/security"
)

// Event is a sanitized audit event.
type Event struct {
	ID           string            `json:"id"`
	EventType    string            `json:"event_type"`
	OccurredAt   time.Time         `json:"occurred_at"`
	ClientIP     string            `json:"client_ip"`
	DeviceID     string            `json:"device_id,omitempty"`
	UserAgent    string            `json:"user_agent"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Result       string            `json:"result"`
	Reason       string            `json:"reason"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Recorder persists audit events.
type Recorder struct {
	DB  *sql.DB
	Now func() time.Time
}

func (r Recorder) Record(ctx context.Context, event Event) error {
	now := r.now()
	if event.ID == "" {
		id, err := security.RandomToken()
		if err != nil {
			return err
		}
		event.ID = "audit_" + id[:16]
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	if event.Metadata == nil {
		event.Metadata = map[string]string{}
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	_, err = r.DB.ExecContext(ctx, `INSERT INTO audit_events
		(id, event_type, occurred_at, client_ip, device_id, user_agent, resource_type, resource_id, result, reason, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.EventType,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		event.ClientIP,
		nullString(event.DeviceID),
		event.UserAgent,
		event.ResourceType,
		event.ResourceID,
		event.Result,
		security.Redact(event.Reason),
		string(metadata),
	)
	if err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}
	return nil
}

func (r Recorder) List(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT id, event_type, occurred_at, client_ip, COALESCE(device_id, ''), user_agent, resource_type, resource_id, result, reason, metadata_json
		FROM audit_events ORDER BY occurred_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var event Event
		var occurred string
		var metadata string
		if err := rows.Scan(&event.ID, &event.EventType, &occurred, &event.ClientIP, &event.DeviceID, &event.UserAgent, &event.ResourceType, &event.ResourceID, &event.Result, &event.Reason, &metadata); err != nil {
			return nil, err
		}
		event.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurred)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r Recorder) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
