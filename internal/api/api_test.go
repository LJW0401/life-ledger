package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"life-ledger/internal/config"
	"life-ledger/internal/db"

	"golang.org/x/crypto/bcrypt"
)

func TestLoginSessionDevicesAndLogout(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()

	login := request(t, handler, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"password","device_name":"Test device"}`, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", login.Code, login.Body.String())
	}
	cookie := login.Result().Cookies()[0]
	csrf := jsonString(t, login.Body.Bytes(), "csrf_token")

	var storedToken string
	if err := conn.QueryRow(`SELECT token_hash FROM device_sessions LIMIT 1`).Scan(&storedToken); err != nil {
		t.Fatal(err)
	}
	if storedToken == cookie.Value {
		t.Fatal("database stored raw cookie value")
	}

	session := request(t, handler, http.MethodGet, "/api/session", "", []*http.Cookie{cookie})
	if session.Code != http.StatusOK {
		t.Fatalf("session status = %d body = %s", session.Code, session.Body.String())
	}
	csrf = jsonString(t, session.Body.Bytes(), "csrf_token")

	devices := request(t, handler, http.MethodGet, "/api/devices", "", []*http.Cookie{cookie})
	if devices.Code != http.StatusOK {
		t.Fatalf("devices status = %d body = %s", devices.Code, devices.Body.String())
	}

	missingCSRF := request(t, handler, http.MethodPost, "/api/auth/logout", "", []*http.Cookie{cookie})
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d", missingCSRF.Code)
	}

	logout := requestWithHeaders(t, handler, http.MethodPost, "/api/auth/logout", "", []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d body = %s", logout.Code, logout.Body.String())
	}
	afterLogout := request(t, handler, http.MethodGet, "/api/session", "", []*http.Cookie{cookie})
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("session after logout status = %d", afterLogout.Code)
	}
}

func TestLoginFailuresLockUsernameAndWriteAudit(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()

	for i := 0; i < 5; i++ {
		response := request(t, handler, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d status = %d", i, response.Code)
		}
	}
	locked := request(t, handler, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"password"}`, nil)
	if locked.Code != http.StatusTooManyRequests {
		t.Fatalf("locked status = %d body = %s", locked.Code, locked.Body.String())
	}

	var auditCount int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM audit_events WHERE event_type IN ('login_failure', 'login_locked')`).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount < 6 {
		t.Fatalf("expected audit events, got %d", auditCount)
	}
}

func TestThirdPartyOriginDoesNotGetWildcardCORS(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()

	response := requestWithHeaders(t, handler, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil, map[string]string{"Origin": "https://evil.example.com"})
	if response.Header().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("unexpected wildcard CORS")
	}
}

func testAPI(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 12)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 18080},
		Data:   config.DataConfig{Dir: dir, Database: "life-ledger.db"},
		Auth:   config.AuthConfig{Username: "admin", PasswordHash: string(hash), SessionSecret: "01234567890123456789012345678901", SessionDays: 7},
		Security: config.SecurityConfig{
			TrustedProxies:            []string{"127.0.0.1"},
			LoginFailureWindowMinutes: 10,
			LoginFailureLimit:         5,
			LoginLockMinutes:          15,
		},
	}
	conn, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return New(cfg, conn), conn
}

func request(t *testing.T, handler http.Handler, method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return requestWithHeaders(t, handler, method, path, body, cookies, nil)
}

func requestWithHeaders(t *testing.T, handler http.Handler, method, path, body string, cookies []*http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.RemoteAddr = "127.0.0.1:1234"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func jsonString(t *testing.T, content []byte, key string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatal(err)
	}
	value, ok := payload[key].(string)
	if !ok || value == "" {
		t.Fatalf("missing string key %s in %s", key, content)
	}
	return value
}
