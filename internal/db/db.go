// Package db owns SQLite opening, permission checks, embedded migrations, and
// transaction boundaries. Repositories build on this package.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"life-ledger/internal/config"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Open prepares the SQLite database and applies embedded migrations.
func Open(cfg config.Config) (*sql.DB, error) {
	if err := ensureDir(cfg.Data.Dir); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(cfg.Data.Dir, cfg.Data.Database)
	existed := true
	if info, err := os.Stat(dbPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect database: %w", err)
		}
		existed = false
	} else if perm := info.Mode().Perm(); perm != 0o600 {
		return nil, fmt.Errorf("database permission must be 600, got %03o", perm)
	}

	if !existed {
		file, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create database: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close database file: %w", err)
		}
	}

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := Migrate(context.Background(), conn, migrationFiles); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect data dir: %w", err)
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("data.dir must be a directory")
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		return fmt.Errorf("data dir permission must be 700, got %03o", perm)
	}
	return nil
}

// Migrate applies migration files from the provided FS in lexical order.
func Migrate(ctx context.Context, conn *sql.DB, migrations fs.FS) error {
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`); err != nil {
		return fmt.Errorf("initialize schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := migrationVersion(entry.Name())
		if err != nil {
			return err
		}
		applied, err := migrationApplied(ctx, conn, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		content, err := fs.ReadFile(migrations, filepath.ToSlash(filepath.Join("migrations", entry.Name())))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if err := applyMigration(ctx, conn, version, entry.Name(), string(content)); err != nil {
			return err
		}
	}
	return nil
}

func migrationVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %s must start with a numeric version", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %s has invalid version: %w", name, err)
	}
	return version, nil
}

func migrationApplied(ctx context.Context, conn *sql.DB, version int) (bool, error) {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %d: %w", version, err)
	}
	return count > 0, nil
}

func applyMigration(ctx context.Context, conn *sql.DB, version int, name string, sqlText string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		tx.Rollback()
		return fmt.Errorf("apply migration %d (%s): %w", version, name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, name) VALUES (?, ?)`, version, name); err != nil {
		tx.Rollback()
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	return nil
}

// WithinTx executes fn inside a transaction and rolls back on error.
func WithinTx(ctx context.Context, conn *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// SchemaVersion returns the highest applied migration version.
func SchemaVersion(ctx context.Context, conn *sql.DB) (int, error) {
	var version sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}
