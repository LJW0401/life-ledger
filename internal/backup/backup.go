// Package backup creates local backup directories containing config.toml,
// SQLite data, and metadata. It never starts the HTTP service.
package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"life-ledger/internal/config"

	_ "modernc.org/sqlite"
)

type Metadata struct {
	AppVersion    string   `json:"app_version"`
	BackupTime    string   `json:"backup_time"`
	SchemaVersion int      `json:"schema_version"`
	Files         []string `json:"files"`
}

func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.Backup.Dir, 0o700); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	dbPath := filepath.Join(cfg.Data.Dir, cfg.Data.Database)
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("read database: %w", err)
	}
	targetDir := filepath.Join(cfg.Backup.Dir, "life-ledger-backup-"+time.Now().UTC().Format("20060102-150405"))
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		return fmt.Errorf("create backup target: %w", err)
	}
	targetDB := filepath.Join(targetDir, cfg.Data.Database)
	if err := snapshotDatabase(context.Background(), dbPath, targetDB); err != nil {
		return err
	}
	if err := copyFile(configPath, filepath.Join(targetDir, "config.toml"), 0o600); err != nil {
		return err
	}
	version, err := schemaVersion(context.Background(), targetDB)
	if err != nil {
		return err
	}
	meta := Metadata{
		AppVersion:    "0.1.0",
		BackupTime:    time.Now().UTC().Format(time.RFC3339Nano),
		SchemaVersion: version,
		Files:         []string{cfg.Data.Database, "config.toml"},
	}
	content, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(targetDir, "backup-meta.json"), content, 0o600); err != nil {
		return fmt.Errorf("write backup metadata: %w", err)
	}
	fmt.Printf("backup created: %s\n", targetDir)
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open backup source: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy backup file: %w", err)
	}
	return nil
}

func snapshotDatabase(ctx context.Context, src string, dst string) error {
	conn, err := sql.Open("sqlite", src+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("open database for snapshot: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "VACUUM INTO "+sqliteString(dst)); err != nil {
		return fmt.Errorf("snapshot database: %w", err)
	}
	if err := os.Chmod(dst, 0o600); err != nil {
		return fmt.Errorf("set snapshot permission: %w", err)
	}
	return nil
}

func sqliteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func schemaVersion(ctx context.Context, dbPath string) (int, error) {
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=query_only(1)")
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	var version sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}
