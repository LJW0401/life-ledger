// Package api maps HTTP requests to authentication, security, audit, and later
// business services. It owns JSON response shape and HTTP status codes.
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"life-ledger/internal/audit"
	"life-ledger/internal/auth"
	"life-ledger/internal/config"
	"life-ledger/internal/domain/importantdates"
	"life-ledger/internal/domain/tags"
	"life-ledger/internal/security"
)

type API struct {
	auth           auth.Service
	audit          audit.Recorder
	importantDates importantdates.Service
	tags           tags.Store
}

func New(cfg config.Config, conn *sql.DB) http.Handler {
	recorder := audit.Recorder{DB: conn}
	tagStore := tags.Store{DB: conn}
	return &API{
		auth: auth.Service{
			DB:     conn,
			Config: cfg,
			Audit:  recorder,
		},
		audit:          recorder,
		importantDates: importantdates.Service{DB: conn, Tags: tagStore},
		tags:           tagStore,
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.URL.Path == "/api/auth/login" && r.Method == http.MethodPost {
		a.login(w, r)
		return
	}

	session, err := a.auth.AuthenticateRequest(r.Context(), r)
	if err != nil {
		a.auth.ClearCookie(w)
		writeError(w, auth.Status(err), "unauthorized", auth.Message(err))
		return
	}

	if auth.IsWriteMethod(r.Method) {
		if err := a.auth.VerifyCSRF(r.Context(), session, r.Header.Get("X-CSRF-Token")); err != nil {
			writeError(w, auth.Status(err), "csrf_failed", auth.Message(err))
			return
		}
	}

	switch {
	case r.URL.Path == "/api/session" && r.Method == http.MethodGet:
		a.session(w, r, session)
	case r.URL.Path == "/api/auth/logout" && r.Method == http.MethodPost:
		a.logout(w, r, session)
	case r.URL.Path == "/api/important-dates" && r.Method == http.MethodGet:
		a.listImportantDates(w, r)
	case r.URL.Path == "/api/important-dates" && r.Method == http.MethodPost:
		a.createImportantDate(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/important-dates/"):
		a.importantDateByID(w, r, session)
	case r.URL.Path == "/api/tags" && r.Method == http.MethodGet:
		a.listTags(w, r)
	case r.URL.Path == "/api/devices" && r.Method == http.MethodGet:
		a.devices(w, r, session)
	case strings.HasPrefix(r.URL.Path, "/api/devices/") && r.Method == http.MethodDelete:
		a.revokeDevice(w, r, session)
	case r.URL.Path == "/api/audit-events" && r.Method == http.MethodGet:
		a.auditEvents(w, r)
	default:
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	}
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		DeviceName string `json:"device_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
		return
	}
	clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
	session, err := a.auth.Login(r.Context(), req.Username, req.Password, req.DeviceName, r.UserAgent(), clientIP)
	if err != nil {
		writeError(w, auth.Status(err), errorCode(err), auth.Message(err))
		return
	}
	a.auth.SetCookie(w, session)
	writeJSON(w, http.StatusOK, map[string]any{
		"session": map[string]any{
			"expires_at": session.ExpiresAt,
		},
		"device":     session.Public(),
		"csrf_token": session.CSRFToken,
	})
}

func (a *API) session(w http.ResponseWriter, r *http.Request, session auth.Session) {
	token, err := a.auth.RotateCSRF(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "刷新 CSRF token 失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"device":        session.Public(),
		"csrf_token":    token,
	})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request, session auth.Session) {
	clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
	if err := a.auth.Logout(r.Context(), session, r.UserAgent(), clientIP); err != nil {
		writeError(w, auth.Status(err), errorCode(err), auth.Message(err))
		return
	}
	a.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) devices(w http.ResponseWriter, r *http.Request, session auth.Session) {
	devices, err := a.auth.ListSessions(r.Context(), session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取设备列表失败")
		return
	}
	items := make([]map[string]any, 0, len(devices))
	for _, device := range devices {
		items = append(items, device.Public())
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) listImportantDates(w http.ResponseWriter, r *http.Request) {
	items, err := a.importantDates.List(r.Context(), r.URL.Query().Get("tag"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取重要日期失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createImportantDate(w http.ResponseWriter, r *http.Request) {
	var input importantdates.Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
		return
	}
	item, err := a.importantDates.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) importantDateByID(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := strings.TrimPrefix(r.URL.Path, "/api/important-dates/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := a.importantDates.Get(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var input importantdates.Input
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
			return
		}
		item, err := a.importantDates.Update(r.Context(), id, input)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := a.importantDates.Delete(r.Context(), id); err != nil {
			writeDomainError(w, err)
			return
		}
		clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
		_ = a.audit.Record(r.Context(), audit.Event{
			EventType:    "delete_important_date",
			ClientIP:     clientIP,
			DeviceID:     session.ID,
			UserAgent:    r.UserAgent(),
			ResourceType: "important_date",
			ResourceID:   id,
			Result:       "success",
		})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	}
}

func (a *API) listTags(w http.ResponseWriter, r *http.Request) {
	items, err := a.tags.Search(r.Context(), r.URL.Query().Get("query"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取标签失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) revokeDevice(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
		return
	}
	clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
	if err := a.auth.Revoke(r.Context(), id, r.UserAgent(), clientIP); err != nil {
		writeError(w, auth.Status(err), errorCode(err), auth.Message(err))
		return
	}
	if id == session.ID {
		a.auth.ClearCookie(w)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "current_device_revoked": id == session.ID})
}

func (a *API) auditEvents(w http.ResponseWriter, r *http.Request) {
	events, err := a.audit.List(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取审计事件失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": []any{},
		},
	})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, importantdates.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "请求参数不合法")
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "服务端错误")
	}
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, auth.ErrLocked):
		return "rate_limited"
	case errors.Is(err, auth.ErrCSRF):
		return "csrf_failed"
	case errors.Is(err, auth.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, sql.ErrNoRows):
		return "not_found"
	default:
		return "internal_error"
	}
}
