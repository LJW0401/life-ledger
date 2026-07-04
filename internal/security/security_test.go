package security

import (
	"net/http/httptest"
	"testing"
)

func TestSignedValueRoundTrip(t *testing.T) {
	signed := SignValue("01234567890123456789012345678901", "token")
	value, ok := VerifySignedValue("01234567890123456789012345678901", signed)
	if !ok || value != "token" {
		t.Fatalf("expected signed value to verify, got %q %v", value, ok)
	}
	if _, ok := VerifySignedValue("wrong-secret-wrong-secret-wrong-secret", signed); ok {
		t.Fatal("expected wrong secret to fail")
	}
}

func TestClientIPTrustsProxyOnlyWhenRemoteAddrTrusted(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if got := ClientIP(req, []string{"127.0.0.1"}); got != "203.0.113.9" {
		t.Fatalf("expected forwarded ip, got %s", got)
	}

	req.RemoteAddr = "198.51.100.4:1234"
	if got := ClientIP(req, []string{"127.0.0.1"}); got != "198.51.100.4" {
		t.Fatalf("expected remote ip, got %s", got)
	}
}

func TestRedactSensitiveKeys(t *testing.T) {
	got := Redact("password=secret")
	if got != "password=<redacted>" {
		t.Fatalf("unexpected redaction: %s", got)
	}
}
