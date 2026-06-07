package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"uas/config"
	"uas/internal/models"
)

type departmentHandlerDeps struct {
	handler      *DepartmentHandler
	deptRepo     *MockDepartmentRepository
	sessionRepo  *MockSessionRepository
	pwHistRepo   *MockPasswordHistoryRepository
	authRepo     *MockAuthRepository
	userRepo     *MockUserRepository
	deptRoleRepo *MockDepartmentRoleRepository
}

type MockDepartmentRepository struct {
	mock.Mock
}

func (m *MockDepartmentRepository) Create(dept *models.DepartmentModel) error {
	args := m.Called(dept)
	return args.Error(0)
}

func (m *MockDepartmentRepository) FindById(id string) (*models.DepartmentModel, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DepartmentModel), args.Error(1)
}

func (m *MockDepartmentRepository) FindAll() ([]models.DepartmentModel, error) {
	args := m.Called()
	return args.Get(0).([]models.DepartmentModel), args.Error(1)
}

func (m *MockDepartmentRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

type MockPasswordHistoryRepository struct {
	mock.Mock
}

func (m *MockPasswordHistoryRepository) Create(history *models.PasswordHistory) error {
	args := m.Called(history)
	return args.Error(0)
}

func (m *MockPasswordHistoryRepository) GetRecentPasswords(userID, departmentID string, limit int) ([]models.PasswordHistory, error) {
	args := m.Called(userID, departmentID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.PasswordHistory), args.Error(1)
}

func (m *MockPasswordHistoryRepository) DeleteOldPasswords(userID, departmentID string, keepCount int) error {
	args := m.Called(userID, departmentID, keepCount)
	return args.Error(0)
}

func (m *MockPasswordHistoryRepository) DeleteByDepartment(departmentID string) error {
	args := m.Called(departmentID)
	return args.Error(0)
}

func setupDepartmentHandler(t *testing.T) *departmentHandlerDeps {
	t.Helper()
	saveAppConfig(t)

	config.AppConfig.AccessJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.RefreshJwtSecret = "12345678901234567890123456789012"

	log := zerolog.Nop()
	deptRepo := new(MockDepartmentRepository)
	sessionRepo := new(MockSessionRepository)
	pwHistRepo := new(MockPasswordHistoryRepository)
	authRepo := new(MockAuthRepository)
	userRepo := new(MockUserRepository)
	deptRoleRepo := new(MockDepartmentRoleRepository)

	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)
	authHelper := newTestAuthHelper(&log)

	handler := NewDepartmentHandler(
		deptRepo, sessionRepo, pwHistRepo, authRepo, userRepo, deptRoleRepo,
		&log, authHelper, responseHelper, validatorHelper,
	)

	return &departmentHandlerDeps{
		handler:      handler,
		deptRepo:     deptRepo,
		sessionRepo:  sessionRepo,
		pwHistRepo:   pwHistRepo,
		authRepo:     authRepo,
		userRepo:     userRepo,
		deptRoleRepo: deptRoleRepo,
	}
}

func TestDepartmentHandler_OnboardDepartmentHandler_HappyPath(t *testing.T) {
	d := setupDepartmentHandler(t)

	d.deptRepo.On("Create", mock.MatchedBy(func(dept *models.DepartmentModel) bool {
		return dept.ID == "dept-1" && dept.Name == "Test Department" && dept.Secret != ""
	})).Return(nil)

	body := map[string]string{
		"departmentId":   "dept-1",
		"departmentName": "Test Department",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/departments/onboard", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.OnboardDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Department onboarded successfully", resp.Message)

	authHeader := rec.Header().Get("Authorization")
	assert.Contains(t, authHeader, "Bearer ")
}

func TestDepartmentHandler_OnboardDepartmentHandler_DecodeError(t *testing.T) {
	d := setupDepartmentHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/departments/onboard", bytes.NewReader([]byte(`invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.OnboardDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDepartmentHandler_OnboardDepartmentHandler_ValidationError(t *testing.T) {
	d := setupDepartmentHandler(t)

	body := map[string]string{
		"departmentId":   "",
		"departmentName": "",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/departments/onboard", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.OnboardDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDepartmentHandler_OnboardDepartmentHandler_CreateError(t *testing.T) {
	d := setupDepartmentHandler(t)

	d.deptRepo.On("Create", mock.Anything).Return(assert.AnError)

	body := map[string]string{
		"departmentId":   "dept-1",
		"departmentName": "Test Department",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/departments/onboard", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.OnboardDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDepartmentHandler_DeleteDepartmentHandler_HappyPath(t *testing.T) {
	d := setupDepartmentHandler(t)

	d.sessionRepo.On("RevokeAllByDepartment", "dept-1").Return(nil)
	d.pwHistRepo.On("DeleteByDepartment", "dept-1").Return(nil)
	d.deptRepo.On("Delete", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/departments/dept-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "dept-1"})
	rec := httptest.NewRecorder()

	d.handler.DeleteDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Tenant deleted successfully", resp.Message)
}

func TestDepartmentHandler_DeleteDepartmentHandler_EmptyID(t *testing.T) {
	d := setupDepartmentHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/departments/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	rec := httptest.NewRecorder()

	d.handler.DeleteDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDepartmentHandler_DeleteDepartmentHandler_RevokeError(t *testing.T) {
	d := setupDepartmentHandler(t)

	d.sessionRepo.On("RevokeAllByDepartment", "dept-1").Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/departments/dept-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "dept-1"})
	rec := httptest.NewRecorder()

	d.handler.DeleteDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDepartmentHandler_DeleteDepartmentHandler_DeleteError(t *testing.T) {
	d := setupDepartmentHandler(t)

	d.sessionRepo.On("RevokeAllByDepartment", "dept-1").Return(nil)
	d.pwHistRepo.On("DeleteByDepartment", "dept-1").Return(nil)
	d.deptRepo.On("Delete", "dept-1").Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/departments/dept-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "dept-1"})
	rec := httptest.NewRecorder()

	d.handler.DeleteDepartmentHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
