package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	path := writeConfig(t, validConfig())

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address() != "127.0.0.1:18080" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address())
	}
	if cfg.Auth.SessionDays != 7 {
		t.Fatalf("expected default session days, got %d", cfg.Auth.SessionDays)
	}
	baseDir := filepath.Dir(path)
	if cfg.Data.Dir != filepath.Join(baseDir, "data") {
		t.Fatalf("expected data dir relative to config, got %s", cfg.Data.Dir)
	}
	if cfg.Backup.Dir != filepath.Join(baseDir, "backups") {
		t.Fatalf("expected backup dir relative to config, got %s", cfg.Backup.Dir)
	}
}

func TestLoadRejectsUnsafeConfigPermission(t *testing.T) {
	path := writeConfig(t, validConfig())
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "permission must be 600") {
		t.Fatalf("expected permission error, got %v", err)
	}
}

func TestLoadRejectsPlainPassword(t *testing.T) {
	cfg := strings.Replace(validConfig(), "password_hash = \"hash\"", "password = \"secret\"", 1)
	path := writeConfig(t, cfg)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "auth.password is not allowed") {
		t.Fatalf("expected plain password error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked password: %v", err)
	}
}

func TestLoadRejectsShortSessionSecret(t *testing.T) {
	cfg := strings.Replace(validConfig(), strings.Repeat("s", 32), "too-short", 1)
	path := writeConfig(t, cfg)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "session_secret") {
		t.Fatalf("expected session secret error, got %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validConfig() string {
	return `
[server]
host = "127.0.0.1"
port = 18080

[data]
dir = "./data"
database = "life-ledger.db"

[auth]
username = "admin"
password_hash = "hash"
session_secret = "` + strings.Repeat("s", 32) + `"
`
}
