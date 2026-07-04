package main

import (
	"bytes"
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
