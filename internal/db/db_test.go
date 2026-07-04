package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"life-ledger/internal/config"

	_ "modernc.org/sqlite"
)

func TestOpenCreatesDatabaseAndRunsMigrations(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(dir)

	conn, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	version, err := SchemaVersion(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	if version != 3 {
		t.Fatalf("expected schema version 3, got %d", version)
	}

	info, err := os.Stat(filepath.Join(dir, "life-ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected database permission 600, got %03o", info.Mode().Perm())
	}
}

func TestOpenRejectsWideDataDirPermission(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Open(testConfig(dir))
	if err == nil {
		t.Fatal("expected data dir permission error")
	}
}

func TestMigrateDoesNotReapplyExistingMigrations(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	if err := Migrate(context.Background(), conn, fstest.MapFS{
		"migrations/001_first.sql": {Data: []byte(`CREATE TABLE things(id TEXT PRIMARY KEY);`)},
	}); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), conn, fstest.MapFS{
		"migrations/001_first.sql": {Data: []byte(`CREATE TABLE broken(`)},
	}); err != nil {
		t.Fatalf("already applied migration should be skipped, got %v", err)
	}
}

func TestMigrateReportsFailingVersion(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	err := Migrate(context.Background(), conn, fstest.MapFS{
		"migrations/001_bad.sql": {Data: []byte(`CREATE TABLE broken(`)},
	})
	if err == nil {
		t.Fatal("expected migration error")
	}
	if !errors.Is(err, context.Canceled) && err.Error() == "" {
		t.Fatal("expected descriptive error")
	}
}

func TestWithinTxRollsBackOnError(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()
	if _, err := conn.Exec(`CREATE TABLE things(id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}

	err := WithinTx(context.Background(), conn, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO things(id) VALUES ('one')`); err != nil {
			return err
		}
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected transaction error")
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM things`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected rollback, got %d rows", count)
	}
}

func openMemory(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func testConfig(dir string) config.Config {
	return config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 18080},
		Data:   config.DataConfig{Dir: dir, Database: "life-ledger.db"},
		Auth:   config.AuthConfig{Username: "admin", PasswordHash: "hash", SessionSecret: "01234567890123456789012345678901"},
	}
}
