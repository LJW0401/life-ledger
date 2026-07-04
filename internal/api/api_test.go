package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"life-ledger/internal/config"
	"life-ledger/internal/db"
	excelpkg "life-ledger/internal/excel"

	"github.com/xuri/excelize/v2"
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

func TestProtectedBusinessEndpointsRequireAuth(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()

	for _, path := range []string{"/api/important-dates", "/api/transactions", "/api/budgets", "/api/decisions", "/api/devices", "/api/audit-events"} {
		response := request(t, handler, http.MethodGet, path, "", nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, response.Code)
		}
	}
}

func TestWriteEndpointsRequireCSRF(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()
	cookie, _ := loginForTest(t, handler)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/important-dates", `{"title":"x","date":"2026-07-04","date_type":"生日"}`},
		{http.MethodDelete, "/api/important-dates/missing", ""},
		{http.MethodPost, "/api/transactions", `{"date":"2026-07-04","time":"08:30","type":"支出","amount":"1.00","category":"餐饮","include_income":true,"include_budget":true,"ledger":"默认账本"}`},
		{http.MethodDelete, "/api/transactions/missing", ""},
		{http.MethodPost, "/api/budgets", `{"month":"2026-07","category":"餐饮","amount":"100.00"}`},
		{http.MethodDelete, "/api/budgets/missing", ""},
		{http.MethodPost, "/api/decisions", `{"title":"x"}`},
		{http.MethodPut, "/api/decisions/missing", `{"title":"x","status":"已归档"}`},
		{http.MethodDelete, "/api/decisions/missing", ""},
		{http.MethodDelete, "/api/devices/missing", ""},
		{http.MethodPost, "/api/auth/logout", ""},
	}
	for _, tc := range cases {
		response := request(t, handler, tc.method, tc.path, tc.body, []*http.Cookie{cookie})
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s %s status = %d, want 403", tc.method, tc.path, response.Code)
		}
	}
}

func TestImportantDatesCRUDTagsAndAudit(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()

	unauthorized := request(t, handler, http.MethodGet, "/api/important-dates", "", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}

	cookie, csrf := loginForTest(t, handler)
	invalid := requestWithHeaders(t, handler, http.MethodPost, "/api/important-dates", `{"title":"","date":"2026-12-01","date_type":"证件"}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d", invalid.Code)
	}

	create := requestWithHeaders(t, handler, http.MethodPost, "/api/important-dates", `{"title":"护照到期","date":"2026-12-01","date_type":"证件","repeat_rule":"不重复","tags":["证件","家庭"]}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", create.Code, create.Body.String())
	}
	id := jsonString(t, create.Body.Bytes(), "id")

	list := request(t, handler, http.MethodGet, "/api/important-dates?tag=证件", "", []*http.Cookie{cookie})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte("护照到期")) {
		t.Fatalf("list status = %d body = %s", list.Code, list.Body.String())
	}

	tagList := request(t, handler, http.MethodGet, "/api/tags?query=证", "", []*http.Cookie{cookie})
	if tagList.Code != http.StatusOK || !bytes.Contains(tagList.Body.Bytes(), []byte("证件")) {
		t.Fatalf("tag list status = %d body = %s", tagList.Code, tagList.Body.String())
	}

	missingCSRF := request(t, handler, http.MethodDelete, "/api/important-dates/"+id, "", []*http.Cookie{cookie})
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("delete without csrf status = %d", missingCSRF.Code)
	}

	deleted := requestWithHeaders(t, handler, http.MethodDelete, "/api/important-dates/"+id, "", []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d body = %s", deleted.Code, deleted.Body.String())
	}
	var auditCount int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM audit_events WHERE event_type = 'delete_important_date' AND resource_id = ?`, id).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected delete audit event, got %d", auditCount)
	}
}

func TestTransactionsBudgetsSummaryAndAudit(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()
	cookie, csrf := loginForTest(t, handler)

	invalid := requestWithHeaders(t, handler, http.MethodPost, "/api/transactions", `{"date":"2026-07-04","time":"08:30","type":"支出","amount":"0","category":"餐饮","include_income":true,"include_budget":true,"ledger":"默认账本"}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid transaction status = %d", invalid.Code)
	}

	create := requestWithHeaders(t, handler, http.MethodPost, "/api/transactions", `{"date":"2026-07-04","time":"08:30","type":"支出","amount":"25.50","category":"餐饮","include_income":true,"include_budget":true,"ledger":"默认账本","account":"现金","tags":["日常"]}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if create.Code != http.StatusCreated {
		t.Fatalf("create transaction status = %d body = %s", create.Code, create.Body.String())
	}
	id := jsonString(t, create.Body.Bytes(), "id")

	budget := requestWithHeaders(t, handler, http.MethodPost, "/api/budgets", `{"month":"2026-07","category":"餐饮","amount":"100.00"}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if budget.Code != http.StatusOK || !bytes.Contains(budget.Body.Bytes(), []byte(`"used":"25.50"`)) {
		t.Fatalf("budget status = %d body = %s", budget.Code, budget.Body.String())
	}

	summary := request(t, handler, http.MethodGet, "/api/transactions/summary?from=2026-07-01&to=2026-07-31", "", []*http.Cookie{cookie})
	if summary.Code != http.StatusOK || !bytes.Contains(summary.Body.Bytes(), []byte(`"expense":"25.50"`)) {
		t.Fatalf("summary status = %d body = %s", summary.Code, summary.Body.String())
	}

	list := request(t, handler, http.MethodGet, "/api/transactions?tag=日常", "", []*http.Cookie{cookie})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte("餐饮")) {
		t.Fatalf("list status = %d body = %s", list.Code, list.Body.String())
	}

	deleted := requestWithHeaders(t, handler, http.MethodDelete, "/api/transactions/"+id, "", []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete transaction status = %d body = %s", deleted.Code, deleted.Body.String())
	}
	var auditCount int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM audit_events WHERE event_type = 'delete_transaction' AND resource_id = ?`, id).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected transaction delete audit, got %d", auditCount)
	}
}

func TestTransactionExcelTemplateImportExport(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()
	cookie, csrf := loginForTest(t, handler)

	template := request(t, handler, http.MethodGet, "/api/transactions/template.xlsx", "", []*http.Cookie{cookie})
	if template.Code != http.StatusOK || template.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("template status = %d content-type = %s", template.Code, template.Header().Get("Content-Type"))
	}

	validFile := xlsxFile(t, true)
	imported := multipartRequest(t, handler, "/api/transactions/import.xlsx", validFile, []*http.Cookie{cookie}, csrf)
	if imported.Code != http.StatusOK || !bytes.Contains(imported.Body.Bytes(), []byte(`"imported":1`)) {
		t.Fatalf("import status = %d body = %s", imported.Code, imported.Body.String())
	}

	invalidFile := xlsxFile(t, false)
	failed := multipartRequest(t, handler, "/api/transactions/import.xlsx", invalidFile, []*http.Cookie{cookie}, csrf)
	if failed.Code != http.StatusBadRequest || !bytes.Contains(failed.Body.Bytes(), []byte("表头")) {
		t.Fatalf("invalid import status = %d body = %s", failed.Code, failed.Body.String())
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM transactions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("invalid import should not add rows, got %d", count)
	}

	exported := request(t, handler, http.MethodGet, "/api/transactions/export.xlsx", "", []*http.Cookie{cookie})
	if exported.Code != http.StatusOK || exported.Body.Len() == 0 {
		t.Fatalf("export status = %d len = %d", exported.Code, exported.Body.Len())
	}
}

func TestDecisionCRUDStatusAndAudit(t *testing.T) {
	handler, conn := testAPI(t)
	defer conn.Close()
	cookie, csrf := loginForTest(t, handler)

	create := requestWithHeaders(t, handler, http.MethodPost, "/api/decisions", `{"title":"是否搬家","background":"通勤太久","status":"进行中","review_date":"2020-01-01","options":[{"name":"搬近公司","pros":"省时间","cons":"租金高"}],"tags":["生活"]}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if create.Code != http.StatusCreated {
		t.Fatalf("create decision status = %d body = %s", create.Code, create.Body.String())
	}
	id := jsonString(t, create.Body.Bytes(), "id")

	list := request(t, handler, http.MethodGet, "/api/decisions?status=待复盘", "", []*http.Cookie{cookie})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte("是否搬家")) {
		t.Fatalf("decision list status = %d body = %s", list.Code, list.Body.String())
	}

	update := requestWithHeaders(t, handler, http.MethodPut, "/api/decisions/"+id, `{"title":"是否搬家","status":"已归档","review_note":"决定暂缓","options":[{"name":"暂缓","pros":"省钱","cons":"通勤久"}]}`, []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if update.Code != http.StatusOK || !bytes.Contains(update.Body.Bytes(), []byte("已归档")) {
		t.Fatalf("update decision status = %d body = %s", update.Code, update.Body.String())
	}

	deleted := requestWithHeaders(t, handler, http.MethodDelete, "/api/decisions/"+id, "", []*http.Cookie{cookie}, map[string]string{"X-CSRF-Token": csrf})
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete decision status = %d body = %s", deleted.Code, deleted.Body.String())
	}
	var auditCount int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM audit_events WHERE event_type = 'delete_decision' AND resource_id = ?`, id).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected decision delete audit, got %d", auditCount)
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
		Export: config.ExportConfig{MaxUploadMB: 5, MaxImportRows: 5000},
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

func loginForTest(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	t.Helper()
	login := request(t, handler, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"password","device_name":"Test device"}`, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", login.Code, login.Body.String())
	}
	return login.Result().Cookies()[0], jsonString(t, login.Body.Bytes(), "csrf_token")
}

func multipartRequest(t *testing.T, handler http.Handler, path string, content []byte, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "transactions.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-CSRF-Token", csrf)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func xlsxFile(t *testing.T, validHeader bool) []byte {
	t.Helper()
	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)
	headers := append([]string{}, excelpkg.Headers...)
	if !validHeader {
		headers[0] = "错误日期"
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}
	values := []string{"2026-07-04", "08:30", "支出", "25.50", "餐饮", "是", "是", "默认账本", "早餐店", "现金", "日常", ""}
	for i, value := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		file.SetCellValue(sheet, cell, value)
	}
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
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
