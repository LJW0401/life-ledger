package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"life-ledger/internal/config"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordCommand(t *testing.T) {
	var out bytes.Buffer
	if err := hashPassword(strings.NewReader("secret\n"), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	hash := lines[len(lines)-1]
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret")); err != nil {
		t.Fatalf("hash does not verify: %v", err)
	}
}

func TestHashPasswordRejectsEmptyPassword(t *testing.T) {
	var out bytes.Buffer
	if err := hashPassword(strings.NewReader("\n"), &out); err == nil {
		t.Fatal("expected empty password error")
	}
}

func TestGenerateSecretCommand(t *testing.T) {
	var out bytes.Buffer
	if err := generateSecret(&out); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); len(got) < 32 {
		t.Fatalf("secret too short: %q", got)
	}
}

func TestInitConfigCreatesRunnableLocalConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	var out bytes.Buffer
	if err := initConfig(path, false, &out); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.Username != "admin" {
		t.Fatalf("unexpected username: %s", cfg.Auth.Username)
	}
	if cfg.Security.CookieSecure {
		t.Fatal("expected local init config to disable secure cookies")
	}
	password := outputValue(t, out.String(), "password: ")
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.Auth.PasswordHash), []byte(password)); err != nil {
		t.Fatalf("generated password does not verify: %v", err)
	}
	for _, dir := range []string{"data", "backups"} {
		info, err := os.Stat(filepath.Join(filepath.Dir(path), dir))
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("%s dir mode = %s", dir, info.Mode().Perm())
		}
	}
}

func TestInitConfigRejectsExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("already here"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := initConfig(path, false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "config already exists") {
		t.Fatalf("expected existing config error, got %v", err)
	}
}

func TestDefaultConfigPathUsesExecutableDirectory(t *testing.T) {
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	path, err := defaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(filepath.Dir(executable), "config.toml") {
		t.Fatalf("default config path = %s, want executable directory", path)
	}
}

func TestRunRejectsMissingConfig(t *testing.T) {
	err := run([]string{"--config", filepath.Join(t.TempDir(), "missing.toml")})
	if err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("expected missing config error, got %v", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := run([]string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func outputValue(t *testing.T, output string, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	t.Fatalf("missing %q in output:\n%s", prefix, output)
	return ""
}
