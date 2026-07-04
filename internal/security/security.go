// Package security contains small, auditable primitives for token generation,
// signed cookies, CSRF comparison, trusted proxy IP parsing, and redaction.
package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
)

// RandomToken returns a URL-safe 32-byte random token.
func RandomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken returns a stable SHA-256 hex digest for server-side token storage.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// SignValue signs a cookie value with HMAC-SHA256.
func SignValue(secret string, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

// VerifySignedValue verifies and extracts a signed cookie value.
func VerifySignedValue(secret string, signed string) (string, bool) {
	value, sig, ok := strings.Cut(signed, ".")
	if !ok || value == "" || sig == "" {
		return "", false
	}
	expected := SignValue(secret, value)
	_, expectedSig, _ := strings.Cut(expected, ".")
	return value, hmac.Equal([]byte(sig), []byte(expectedSig))
}

// ClientIP returns the effective client IP. Proxy headers are trusted only when
// the TCP peer is configured as a trusted proxy.
func ClientIP(r *http.Request, trustedProxies []string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if isTrusted(host, trustedProxies) {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			first, _, _ := strings.Cut(forwarded, ",")
			if ip := strings.TrimSpace(first); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	return host
}

func isTrusted(ip string, trusted []string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, candidate := range trusted {
		if parsed.Equal(net.ParseIP(candidate)) {
			return true
		}
	}
	return false
}

// Redact replaces sensitive values in log and audit-adjacent messages.
func Redact(input string) string {
	sensitive := []string{"password", "password_hash", "session_secret", "cookie", "csrf", "token"}
	out := input
	for _, key := range sensitive {
		out = redactKey(out, key)
	}
	return out
}

func redactKey(input, key string) string {
	lower := strings.ToLower(input)
	idx := strings.Index(lower, key)
	if idx < 0 {
		return input
	}
	return input[:idx+len(key)] + "=<redacted>"
}
