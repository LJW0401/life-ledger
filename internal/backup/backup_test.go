package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"life-ledger/internal/config"
	"life-ledger/internal/db"
)

func TestRunCreatesBackupPackage(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	backupDir := filepath.Join(root, "backups")
	if err := os.Mkdir(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Data:   config.DataConfig{Dir: dataDir, Database: "life-ledger.db"},
		Auth:   config.AuthConfig{Username: "admin", PasswordHash: "hash", SessionSecret: "01234567890123456789012345678901", SessionDays: 7},
		Backup: config.BackupConfig{Dir: backupDir},
	}
	conn, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	configPath := filepath.Join(root, "config.toml")
	content := `[data]
dir = "` + dataDir + `"
database = "life-ledger.db"

[auth]
username = "admin"
password_hash = "hash"
session_secret = "01234567890123456789012345678901"

[backup]
dir = "` + backupDir + `"
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Run(configPath); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "life-ledger-backup-") {
		t.Fatalf("unexpected backup entries: %#v", entries)
	}
	target := filepath.Join(backupDir, entries[0].Name())
	for _, name := range []string{"life-ledger.db", "config.toml", "backup-meta.json"} {
		if _, err := os.Stat(filepath.Join(target, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}
