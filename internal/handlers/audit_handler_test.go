package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"uas/internal/models"
)

type auditHandlerDeps struct {
	handler     *AuditHandler
	auditRepo   *MockAuditLogRepository
}

func setupAuditHandler(t *testing.T) *auditHandlerDeps {
	t.Helper()
	saveAppConfig(t)

	log := zerolog.Nop()
	auditRepo := new(MockAuditLogRepository)
	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)

	handler := NewAuditHandler(auditRepo, &log, responseHelper, validatorHelper)

	return &auditHandlerDeps{
		handler:   handler,
		auditRepo: auditRepo,
	}
}

func TestAuditHandler_GetAuditLogs_HappyPath(t *testing.T) {
	d := setupAuditHandler(t)

	result := &models.AuditLogQuery{Total: 1}
	d.auditRepo.On("FindMany", mockFilterWith("dept-1")).Return(result, nil)

	req := httptest.NewRequest(http.MethodGet, "/audit/logs", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditLogs(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuditHandler_GetAuditLogs_Error(t *testing.T) {
	d := setupAuditHandler(t)

	d.auditRepo.On("FindMany", mockFilterWith("dept-1")).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/audit/logs", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditLogs(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAuditHandler_GetAuditLogByID_HappyPath(t *testing.T) {
	d := setupAuditHandler(t)

	uID := "user-1"
	deptID := "dept-1"
	auditLog := &models.AuditLog{ID: "log-1", UserID: &uID, DepartmentID: &deptID}
	d.auditRepo.On("FindByID", "log-1").Return(auditLog, nil)

	req := httptest.NewRequest(http.MethodGet, "/audit/logs/log-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "log-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditLogByID(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuditHandler_GetAuditLogByID_MissingID(t *testing.T) {
	d := setupAuditHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/audit/logs/", nil)
	rec := httptest.NewRecorder()

	d.handler.GetAuditLogByID(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_GetAuditLogByID_NotFound(t *testing.T) {
	d := setupAuditHandler(t)

	d.auditRepo.On("FindByID", "log-999").Return(nil, gormErrRecordNotFound)

	req := httptest.NewRequest(http.MethodGet, "/audit/logs/log-999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "log-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditLogByID(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAuditHandler_GetAuditLogByID_TenantMismatch(t *testing.T) {
	d := setupAuditHandler(t)

	uID2 := "user-1"
	otherDept := "dept-other"
	auditLog := &models.AuditLog{ID: "log-1", UserID: &uID2, DepartmentID: &otherDept}
	d.auditRepo.On("FindByID", "log-1").Return(auditLog, nil)

	req := httptest.NewRequest(http.MethodGet, "/audit/logs/log-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "log-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditLogByID(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAuditHandler_GetAuditStats_HappyPath(t *testing.T) {
	d := setupAuditHandler(t)

	d.auditRepo.On("Count", mockFilterWith("dept-1")).Return(int64(42), nil)

	req := httptest.NewRequest(http.MethodGet, "/audit/stats", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditStats(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := decodeResponse(rec, &resp)
	require.NoError(t, err)
	assert.Equal(t, "Audit statistics retrieved successfully", resp.Message)
}

func TestAuditHandler_GetAuditStats_Error(t *testing.T) {
	d := setupAuditHandler(t)

	d.auditRepo.On("Count", mockFilterWith("dept-1")).Return(int64(0), assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/audit/stats", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetAuditStats(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func mockFilterWith(deptID string) interface{} {
	return mock.MatchedBy(func(f *models.AuditLogFilter) bool {
		return f.DepartmentID != nil && *f.DepartmentID == deptID
	})
}

func decodeResponse(rec *httptest.ResponseRecorder, v interface{}) error {
	return json.Unmarshal(rec.Body.Bytes(), v)
}

var gormErrRecordNotFound = gorm.ErrRecordNotFound
