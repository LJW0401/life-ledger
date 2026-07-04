package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
