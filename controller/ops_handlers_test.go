package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func setupOpsControllerDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true
	require.NoError(t, db.AutoMigrate(
		&model.SeatBinding{},
		&model.BillingEvent{},
		&model.AuditEvent{},
		&model.Tenant{},
		&model.Channel{},
	))
}

func performJSON(handler gin.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, path, handler)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAdminBindSeatAndList(t *testing.T) {
	setupOpsControllerDB(t)
	common.CryptoSecret = "ops-test-secret"

	w := performJSON(AdminBindSeat, http.MethodPost, "/bind", `{"tenant_id":"t1","user_id":"u1","seat_id":"s1","github_token":"ghp_xxx","account_type":"enterprise"}`)
	require.Equal(t, http.StatusOK, w.Code)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/list", AdminListSeatBindings)
	req := httptest.NewRequest(http.MethodGet, "/list?tenant_id=t1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"seat_id":"s1"`)

	w2 := performJSON(AdminBindSeat, http.MethodPost, "/bind", `{"tenant_id":"t1"}`)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), `"success":false`)

	w3 := performJSON(AdminBindSeat, http.MethodPost, "/bind", `{"tenant_id":"t1","user_id":"u2","seat_id":"s2","github_token":"ghp_yyy"}`)
	require.Equal(t, http.StatusOK, w3.Code)
	require.Contains(t, w3.Body.String(), `"account_type":"individual"`)

	recAll := httptest.NewRecorder()
	reqAll := httptest.NewRequest(http.MethodGet, "/list", nil)
	r.ServeHTTP(recAll, reqAll)
	require.Equal(t, http.StatusOK, recAll.Code)
	require.NotContains(t, recAll.Body.String(), "ghp_")
}

func TestBillingStatementHandlers(t *testing.T) {
	setupOpsControllerDB(t)
	require.NoError(t, model.DB.Create(&model.BillingEvent{
		TenantID:     "t1",
		UserID:       "u1",
		SeatID:       "s1",
		RequestID:    "r1",
		Model:        "gpt-4o",
		Protocol:     "consume",
		InputTokens:  10,
		OutputTokens: 5,
		BillingCycle: "2026-04",
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/statements", AdminGetBillingStatements)
	r.GET("/export", AdminExportBillingStatementsCSV)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/statements?tenant_id=t1", nil)
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"tenant_id":"t1"`)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/export?tenant_id=t1", nil)
	r.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
	require.Contains(t, rec2.Body.String(), "tenant_id")
	require.Contains(t, rec2.Body.String(), "t1")

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/statements", nil)
	r.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusOK, rec3.Code)
}

func TestAuditAndOverviewHandlers(t *testing.T) {
	setupOpsControllerDB(t)
	require.NoError(t, model.DB.Create(&model.AuditEvent{TenantID: "t1", UserID: "u1", Model: "gpt-4o"}).Error)
	require.NoError(t, model.DB.Create(&model.Tenant{TenantID: "t1", Name: "tenant"}).Error)
	require.NoError(t, model.DB.Create(&model.SeatBinding{TenantID: "t1", UserID: "u1", SeatID: "s1", EncryptedToken: "enc", Status: "active"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Type: constant.ChannelTypeGitHubCopilot, Name: "copilot", Key: "ghp_x", Status: common.ChannelStatusEnabled}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit", AdminListAuditEvents)
	r.DELETE("/audit", AdminCleanupAuditEvents)
	r.GET("/health", AdminCopilotTokenHealth)
	r.GET("/overview", AdminOperationsOverview)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/audit?tenant_id=t1", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"tenant_id":"t1"`)

	recLimit := httptest.NewRecorder()
	r.ServeHTTP(recLimit, httptest.NewRequest(http.MethodGet, "/audit?limit=2000", nil))
	require.Equal(t, http.StatusOK, recLimit.Code)

	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec2.Code)
	require.Contains(t, rec2.Body.String(), `"status":"ok"`)

	require.NoError(t, model.DB.Create(&model.Channel{Type: constant.ChannelTypeGitHubCopilot, Name: "copilot2", Key: "", Status: common.ChannelStatusEnabled}).Error)
	recHealth2 := httptest.NewRecorder()
	r.ServeHTTP(recHealth2, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, recHealth2.Code)
	require.Contains(t, recHealth2.Body.String(), `"missing_token"`)

	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/overview", nil))
	require.Equal(t, http.StatusOK, rec3.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec3.Body.Bytes(), &payload))
	require.Equal(t, true, payload["success"])

	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, httptest.NewRequest(http.MethodDelete, "/audit?tenant_id=t1&before=9999999999", nil))
	require.Equal(t, http.StatusOK, rec4.Code)
	require.Contains(t, rec4.Body.String(), `"success":true`)
}

func TestParseIntQuery(t *testing.T) {
	require.Equal(t, 7, parseIntQuery("", 7))
	require.Equal(t, 7, parseIntQuery("abc", 7))
	require.Equal(t, 9, parseIntQuery("9", 7))
}
