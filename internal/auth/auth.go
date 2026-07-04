// Package auth implements single-user authentication, signed session cookies,
// device records, CSRF tokens, and persistent login failure rate limiting.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"life-ledger/internal/audit"
	"life-ledger/internal/config"
	"life-ledger/internal/security"

	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "ll_session"

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrLocked       = errors.New("login locked")
	ErrCSRF         = errors.New("csrf failed")
)

type Service struct {
	DB     *sql.DB
	Config config.Config
	Audit  audit.Recorder
	Now    func() time.Time
}

type Session struct {
	ID         string     `json:"id"`
	DeviceName string     `json:"device_name"`
	UserAgent  string     `json:"user_agent"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	LastSeenIP string     `json:"last_seen_ip"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	Current    bool       `json:"current"`
	RawToken   string     `json:"-"`
	CSRFToken  string     `json:"-"`
}

func (s Service) Login(ctx context.Context, username, password, deviceName, userAgent, clientIP string) (Session, error) {
	if locked, err := s.locked(ctx, username, clientIP); err != nil {
		return Session{}, err
	} else if locked {
		_ = s.recordAudit(ctx, "login_locked", "", userAgent, clientIP, "failure", "login temporarily locked")
		return Session{}, ErrLocked
	}

	if username != s.Config.Auth.Username || bcrypt.CompareHashAndPassword([]byte(s.Config.Auth.PasswordHash), []byte(password)) != nil {
		_ = s.recordFailure(ctx, username, clientIP)
		_ = s.recordAudit(ctx, "login_failure", "", userAgent, clientIP, "failure", "invalid credentials")
		return Session{}, ErrUnauthorized
	}

	if err := s.clearFailures(ctx, username, clientIP); err != nil {
		return Session{}, err
	}

	session, err := s.createSession(ctx, deviceName, userAgent, clientIP)
	if err != nil {
		return Session{}, err
	}
	_ = s.recordAudit(ctx, "login_success", session.ID, userAgent, clientIP, "success", "")
	return session, nil
}

func (s Service) AuthenticateRequest(ctx context.Context, r *http.Request) (Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	rawToken, ok := security.VerifySignedValue(s.Config.Auth.SessionSecret, cookie.Value)
	if !ok {
		return Session{}, ErrUnauthorized
	}
	session, err := s.sessionByToken(ctx, rawToken)
	if err != nil {
		return Session{}, err
	}
	if session.ID == "" {
		return Session{}, ErrUnauthorized
	}
	now := s.now()
	if session.ExpiresAt.Before(now) || session.RevokedAt != nil {
		return Session{}, ErrUnauthorized
	}
	clientIP := security.ClientIP(r, s.Config.Security.TrustedProxies)
	_, _ = s.DB.ExecContext(ctx, `UPDATE device_sessions SET last_seen_at = ?, last_seen_ip = ? WHERE id = ?`, now.UTC().Format(time.RFC3339Nano), clientIP, session.ID)
	session.RawToken = rawToken
	session.LastSeenAt = now
	session.LastSeenIP = clientIP
	session.Current = true
	return session, nil
}

func (s Service) RotateCSRF(ctx context.Context, session Session) (string, error) {
	token, err := security.RandomToken()
	if err != nil {
		return "", err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE device_sessions SET csrf_token_hash = ? WHERE id = ?`, security.HashToken(token), session.ID)
	return token, err
}

func (s Service) VerifyCSRF(ctx context.Context, session Session, token string) error {
	if token == "" {
		return ErrCSRF
	}
	var count int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM device_sessions WHERE id = ? AND csrf_token_hash = ?`, session.ID, security.HashToken(token)).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return ErrCSRF
	}
	return nil
}

func (s Service) Logout(ctx context.Context, session Session, userAgent, clientIP string) error {
	if err := s.Revoke(ctx, session.ID, userAgent, clientIP); err != nil {
		return err
	}
	return nil
}

func (s Service) Revoke(ctx context.Context, id, userAgent, clientIP string) error {
	now := s.now().UTC().Format(time.RFC3339Nano)
	res, err := s.DB.ExecContext(ctx, `UPDATE device_sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return s.recordAudit(ctx, "device_revoked", id, userAgent, clientIP, "success", "")
}

func (s Service) ListSessions(ctx context.Context, currentID string) ([]Session, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, device_name, user_agent, last_seen_at, last_seen_ip, expires_at, revoked_at FROM device_sessions ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []Session
	for rows.Next() {
		var session Session
		var lastSeen, expires string
		var revoked sql.NullString
		if err := rows.Scan(&session.ID, &session.DeviceName, &session.UserAgent, &lastSeen, &session.LastSeenIP, &expires, &revoked); err != nil {
			return nil, err
		}
		session.LastSeenAt, _ = time.Parse(time.RFC3339Nano, lastSeen)
		session.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
		if revoked.Valid {
			t, _ := time.Parse(time.RFC3339Nano, revoked.String)
			session.RevokedAt = &t
		}
		session.Current = session.ID == currentID
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s Service) SetCookie(w http.ResponseWriter, session Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    security.SignValue(s.Config.Auth.SessionSecret, session.RawToken),
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   s.Config.Security.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.Config.Security.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s Service) createSession(ctx context.Context, deviceName, userAgent, clientIP string) (Session, error) {
	token, err := security.RandomToken()
	if err != nil {
		return Session{}, err
	}
	csrfToken, err := security.RandomToken()
	if err != nil {
		return Session{}, err
	}
	idToken, err := security.RandomToken()
	if err != nil {
		return Session{}, err
	}
	now := s.now()
	expires := now.Add(time.Duration(s.Config.Auth.SessionDays) * 24 * time.Hour)
	if deviceName == "" {
		deviceName = "Unknown device"
	}
	session := Session{
		ID:         "dev_" + idToken[:16],
		DeviceName: deviceName,
		UserAgent:  userAgent,
		LastSeenAt: now,
		LastSeenIP: clientIP,
		ExpiresAt:  expires,
		RawToken:   token,
		CSRFToken:  csrfToken,
		Current:    true,
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO device_sessions
		(id, device_name, token_hash, csrf_token_hash, user_agent, first_login_at, last_seen_at, last_seen_ip, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.DeviceName,
		security.HashToken(token),
		security.HashToken(csrfToken),
		userAgent,
		now.UTC().Format(time.RFC3339Nano),
		now.UTC().Format(time.RFC3339Nano),
		clientIP,
		expires.UTC().Format(time.RFC3339Nano),
	)
	return session, err
}

func (s Service) sessionByToken(ctx context.Context, rawToken string) (Session, error) {
	var session Session
	var lastSeen, expires string
	var revoked sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT id, device_name, user_agent, last_seen_at, last_seen_ip, expires_at, revoked_at
		FROM device_sessions WHERE token_hash = ?`, security.HashToken(rawToken)).
		Scan(&session.ID, &session.DeviceName, &session.UserAgent, &lastSeen, &session.LastSeenIP, &expires, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, nil
	}
	if err != nil {
		return Session{}, err
	}
	session.LastSeenAt, _ = time.Parse(time.RFC3339Nano, lastSeen)
	session.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	if revoked.Valid {
		t, _ := time.Parse(time.RFC3339Nano, revoked.String)
		session.RevokedAt = &t
	}
	return session, nil
}

func (s Service) locked(ctx context.Context, username, clientIP string) (bool, error) {
	now := s.now().UTC().Format(time.RFC3339Nano)
	var count int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM login_failures
		WHERE ((username = ? AND client_ip = '*') OR (username = '*' AND client_ip = ?))
		AND locked_until IS NOT NULL AND locked_until > ?`, username, clientIP, now).Scan(&count)
	return count > 0, err
}

func (s Service) recordFailure(ctx context.Context, username, clientIP string) error {
	if err := s.recordFailureScope(ctx, username, "*"); err != nil {
		return err
	}
	return s.recordFailureScope(ctx, "*", clientIP)
}

func (s Service) recordFailureScope(ctx context.Context, username, clientIP string) error {
	now := s.now()
	windowStart := now.Add(-time.Duration(s.Config.Security.LoginFailureWindowMinutes) * time.Minute)
	var id string
	var count int
	var started string
	err := s.DB.QueryRowContext(ctx, `SELECT id, failure_count, window_started_at FROM login_failures WHERE username = ? AND client_ip = ?`, username, clientIP).Scan(&id, &count, &started)
	if errors.Is(err, sql.ErrNoRows) {
		token, tokenErr := security.RandomToken()
		if tokenErr != nil {
			return tokenErr
		}
		_, err = s.DB.ExecContext(ctx, `INSERT INTO login_failures(id, username, client_ip, failure_count, window_started_at, last_failed_at)
			VALUES (?, ?, ?, 1, ?, ?)`, "lf_"+token[:16], username, clientIP, now.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
		return err
	}
	if err != nil {
		return err
	}
	parsedStarted, _ := time.Parse(time.RFC3339Nano, started)
	if parsedStarted.Before(windowStart) {
		count = 0
		parsedStarted = now
	}
	count++
	var lockedUntil any
	if count >= s.Config.Security.LoginFailureLimit {
		lockedUntil = now.Add(time.Duration(s.Config.Security.LoginLockMinutes) * time.Minute).UTC().Format(time.RFC3339Nano)
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE login_failures SET failure_count = ?, window_started_at = ?, last_failed_at = ?, locked_until = ? WHERE id = ?`,
		count, parsedStarted.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano), lockedUntil, id)
	return err
}

func (s Service) clearFailures(ctx context.Context, username, clientIP string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM login_failures WHERE (username = ? AND client_ip = '*') OR (username = '*' AND client_ip = ?)`, username, clientIP)
	return err
}

func (s Service) recordAudit(ctx context.Context, eventType, deviceID, userAgent, clientIP, result, reason string) error {
	return s.Audit.Record(ctx, audit.Event{
		EventType: eventType,
		DeviceID:  deviceID,
		UserAgent: userAgent,
		ClientIP:  clientIP,
		Result:    result,
		Reason:    reason,
	})
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s Session) Public() map[string]any {
	return map[string]any{
		"id":           s.ID,
		"device_name":  s.DeviceName,
		"user_agent":   s.UserAgent,
		"last_seen_at": s.LastSeenAt.UTC().Format(time.RFC3339Nano),
		"last_seen_ip": s.LastSeenIP,
		"expires_at":   s.ExpiresAt.UTC().Format(time.RFC3339Nano),
		"revoked_at":   formatOptionalTime(s.RevokedAt),
		"current":      s.Current,
	}
}

func formatOptionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func IsWriteMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func Status(err error) int {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrLocked):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrCSRF):
		return http.StatusForbidden
	case errors.Is(err, sql.ErrNoRows):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func Message(err error) string {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return "未登录或凭据错误"
	case errors.Is(err, ErrLocked):
		return "登录尝试过多，请稍后重试"
	case errors.Is(err, ErrCSRF):
		return "CSRF token 无效"
	case errors.Is(err, sql.ErrNoRows):
		return "资源不存在"
	default:
		return fmt.Sprintf("%v", err)
	}
}
