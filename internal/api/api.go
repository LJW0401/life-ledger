// Package api maps HTTP requests to authentication, security, audit, and later
// business services. It owns JSON response shape and HTTP status codes.
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"life-ledger/internal/audit"
	"life-ledger/internal/auth"
	"life-ledger/internal/config"
	"life-ledger/internal/domain/decisions"
	"life-ledger/internal/domain/importantdates"
	"life-ledger/internal/domain/tags"
	"life-ledger/internal/domain/transactions"
	"life-ledger/internal/excel"
	"life-ledger/internal/security"
)

type API struct {
	auth           auth.Service
	audit          audit.Recorder
	importantDates importantdates.Service
	decisions      decisions.Service
	transactions   transactions.Service
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
		decisions:      decisions.Service{DB: conn, Tags: tagStore},
		transactions:   transactions.Service{DB: conn, Tags: tagStore},
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
	case r.URL.Path == "/api/decisions" && r.Method == http.MethodGet:
		a.listDecisions(w, r)
	case r.URL.Path == "/api/decisions" && r.Method == http.MethodPost:
		a.createDecision(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/decisions/"):
		a.decisionByID(w, r, session)
	case r.URL.Path == "/api/tags" && r.Method == http.MethodGet:
		a.listTags(w, r)
	case r.URL.Path == "/api/transactions" && r.Method == http.MethodGet:
		a.listTransactions(w, r)
	case r.URL.Path == "/api/transactions" && r.Method == http.MethodPost:
		a.createTransaction(w, r)
	case r.URL.Path == "/api/transactions/summary" && r.Method == http.MethodGet:
		a.transactionSummary(w, r)
	case r.URL.Path == "/api/transactions/template.xlsx" && r.Method == http.MethodGet:
		a.transactionTemplate(w, r)
	case r.URL.Path == "/api/transactions/export.xlsx" && r.Method == http.MethodGet:
		a.transactionExport(w, r)
	case r.URL.Path == "/api/transactions/import.xlsx" && r.Method == http.MethodPost:
		a.transactionImport(w, r, session)
	case strings.HasPrefix(r.URL.Path, "/api/transactions/"):
		a.transactionByID(w, r, session)
	case r.URL.Path == "/api/budgets" && r.Method == http.MethodGet:
		a.listBudgets(w, r)
	case r.URL.Path == "/api/budgets" && r.Method == http.MethodPost:
		a.saveBudget(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/budgets/") && r.Method == http.MethodDelete:
		a.deleteBudget(w, r)
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

func (a *API) listDecisions(w http.ResponseWriter, r *http.Request) {
	items, err := a.decisions.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取决策失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createDecision(w http.ResponseWriter, r *http.Request) {
	var input decisions.Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
		return
	}
	item, err := a.decisions.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) decisionByID(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := strings.TrimPrefix(r.URL.Path, "/api/decisions/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := a.decisions.Get(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var input decisions.Input
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
			return
		}
		item, err := a.decisions.Update(r.Context(), id, input)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := a.decisions.Delete(r.Context(), id); err != nil {
			writeDomainError(w, err)
			return
		}
		clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
		_ = a.audit.Record(r.Context(), audit.Event{
			EventType:    "delete_decision",
			ClientIP:     clientIP,
			DeviceID:     session.ID,
			UserAgent:    r.UserAgent(),
			ResourceType: "decision",
			ResourceID:   id,
			Result:       "success",
		})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	}
}

func (a *API) listTransactions(w http.ResponseWriter, r *http.Request) {
	result, err := a.transactions.List(r.Context(), transactionFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取账单失败")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeTransactionInput(w, r)
	if !ok {
		return
	}
	item, err := a.transactions.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

type transactionRequest struct {
	Date          string   `json:"date"`
	Time          string   `json:"time"`
	Type          string   `json:"type"`
	Amount        string   `json:"amount"`
	Category      string   `json:"category"`
	IncludeIncome *bool    `json:"include_income"`
	IncludeBudget *bool    `json:"include_budget"`
	Ledger        string   `json:"ledger"`
	Counterparty  string   `json:"counterparty"`
	Account       string   `json:"account"`
	Note          string   `json:"note"`
	Tags          []string `json:"tags"`
}

func decodeTransactionInput(w http.ResponseWriter, r *http.Request) (transactions.Input, bool) {
	var req transactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
		return transactions.Input{}, false
	}
	input, err := req.input()
	if err != nil {
		writeDomainError(w, err)
		return transactions.Input{}, false
	}
	return input, true
}

func (r transactionRequest) input() (transactions.Input, error) {
	if r.IncludeIncome == nil {
		return transactions.Input{}, fmt.Errorf("%w: include_income is required", transactions.ErrValidation)
	}
	if r.IncludeBudget == nil {
		return transactions.Input{}, fmt.Errorf("%w: include_budget is required", transactions.ErrValidation)
	}
	return transactions.Input{
		Date:          r.Date,
		Time:          r.Time,
		Type:          r.Type,
		Amount:        r.Amount,
		Category:      r.Category,
		IncludeIncome: *r.IncludeIncome,
		IncludeBudget: *r.IncludeBudget,
		Ledger:        r.Ledger,
		Counterparty:  r.Counterparty,
		Account:       r.Account,
		Note:          r.Note,
		Tags:          r.Tags,
	}, nil
}

func (a *API) transactionSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := a.transactions.Summary(r.Context(), transactionFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取账单统计失败")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) transactionTemplate(w http.ResponseWriter, r *http.Request) {
	content, err := excel.Template()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "生成模板失败")
		return
	}
	writeXLSX(w, "life-ledger-transactions-template.xlsx", content)
}

func (a *API) transactionExport(w http.ResponseWriter, r *http.Request) {
	items, err := a.transactions.ListAll(r.Context(), transactionFilter(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取账单失败")
		return
	}
	content, err := excel.Export(items)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "导出账单失败")
		return
	}
	writeXLSX(w, "life-ledger-transactions.xlsx", content)
}

func (a *API) transactionImport(w http.ResponseWriter, r *http.Request, session auth.Session) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(a.auth.Config.Export.MaxUploadMB)*1024*1024)
	if err := r.ParseMultipartForm(int64(a.auth.Config.Export.MaxUploadMB) * 1024 * 1024); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "文件过大或表单非法")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "缺少上传文件")
		return
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		writeError(w, http.StatusBadRequest, "validation_failed", "仅支持 .xlsx")
		return
	}
	inputs, err := excel.ParseImport(file, a.auth.Config.Export.MaxImportRows)
	if err != nil {
		a.recordExcelAudit(r, session, "failure", "excel validation failed")
		var validation excel.ValidationError
		if errors.As(err, &validation) {
			writeExcelValidation(w, validation)
			return
		}
		writeError(w, http.StatusBadRequest, "validation_failed", "Excel 解析失败")
		return
	}
	if err := a.transactions.CreateMany(r.Context(), inputs); err != nil {
		a.recordExcelAudit(r, session, "failure", "transaction import failed")
		writeDomainError(w, err)
		return
	}
	a.recordExcelAudit(r, session, "success", "")
	writeJSON(w, http.StatusOK, map[string]any{"imported": len(inputs)})
}

func (a *API) transactionByID(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := strings.TrimPrefix(r.URL.Path, "/api/transactions/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := a.transactions.Get(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		input, ok := decodeTransactionInput(w, r)
		if !ok {
			return
		}
		item, err := a.transactions.Update(r.Context(), id, input)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := a.transactions.Delete(r.Context(), id); err != nil {
			writeDomainError(w, err)
			return
		}
		clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
		_ = a.audit.Record(r.Context(), audit.Event{
			EventType:    "delete_transaction",
			ClientIP:     clientIP,
			DeviceID:     session.ID,
			UserAgent:    r.UserAgent(),
			ResourceType: "transaction",
			ResourceID:   id,
			Result:       "success",
		})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	}
}

func (a *API) listBudgets(w http.ResponseWriter, r *http.Request) {
	items, err := a.transactions.ListBudgets(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "读取预算失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) saveBudget(w http.ResponseWriter, r *http.Request) {
	var input transactions.BudgetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "请求体不是合法 JSON")
		return
	}
	item, err := a.transactions.SaveBudget(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) deleteBudget(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/budgets/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
		return
	}
	if err := a.transactions.DeleteBudget(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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

func writeExcelValidation(w http.ResponseWriter, validation excel.ValidationError) {
	details := make([]any, 0, len(validation.Errors))
	for _, item := range validation.Errors {
		details = append(details, item)
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"error": map[string]any{
			"code":    "validation_failed",
			"message": "Excel 校验失败",
			"details": details,
		},
	})
}

func writeXLSX(w http.ResponseWriter, filename string, content []byte) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (a *API) recordExcelAudit(r *http.Request, session auth.Session, result string, reason string) {
	clientIP := security.ClientIP(r, a.auth.Config.Security.TrustedProxies)
	_ = a.audit.Record(r.Context(), audit.Event{
		EventType:    "excel_import",
		ClientIP:     clientIP,
		DeviceID:     session.ID,
		UserAgent:    r.UserAgent(),
		ResourceType: "transaction",
		Result:       result,
		Reason:       reason,
	})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, importantdates.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "请求参数不合法")
	case errors.Is(err, decisions.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "请求参数不合法")
	case errors.Is(err, transactions.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "请求参数不合法")
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "资源不存在")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "服务端错误")
	}
}

func transactionFilter(r *http.Request) transactions.Filter {
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	return transactions.Filter{
		From:     query.Get("from"),
		To:       query.Get("to"),
		Type:     query.Get("type"),
		Category: query.Get("category"),
		Account:  query.Get("account"),
		Tag:      query.Get("tag"),
		Page:     page,
		PageSize: pageSize,
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
