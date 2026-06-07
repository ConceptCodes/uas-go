package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
)

func setupAdminHandler(t *testing.T) (*AdminHandler, *MockUserRepository, *MockSessionRepository) {
	t.Helper()
	saveAppConfig(t)

	config.AppConfig.PasswordMinLength = 8
	config.AppConfig.PasswordRequireUppercase = true
	config.AppConfig.PasswordRequireLowercase = true
	config.AppConfig.PasswordRequireNumber = true
	config.AppConfig.PasswordRequireSpecial = true

	log := newTestLogger()
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)

	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)
	passwordHelper := newTestPasswordHelper(&log)
	authHelper := newTestAuthHelper(&log)

	handler := NewAdminHandler(
		userRepo,
		nil,
		sessionRepo,
		nil,
		passwordHelper,
		authHelper,
		responseHelper,
		validatorHelper,
		&log,
	)

	return handler, userRepo, sessionRepo
}

func withDepartment(r *http.Request, deptID string) *http.Request {
	return helpers.SetDepartmentId(r, deptID)
}

func TestAdminHandler_ListUsersHandler(t *testing.T) {
	handler, _, _ := setupAdminHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.ListUsersHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Users retrieved", resp.Message)
}

func TestAdminHandler_GetUserHandler_Found(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:            "user-1",
		Name:          "Alice",
		Email:         "alice@test.com",
		PhoneNumber:   "+1234567890",
		EmailVerified: true,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/user-1", nil)
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.GetUserHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "User retrieved", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "data should be a map")
	assert.Equal(t, "user-1", data["id"])
	assert.Equal(t, "Alice", data["name"])
	assert.Equal(t, "alice@test.com", data["email"])
	assert.Equal(t, "+1234567890", data["phoneNumber"])
	assert.Equal(t, true, data["emailVerified"])
}

func TestAdminHandler_GetUserHandler_NotFound(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/user-1", nil)
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.GetUserHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp models.ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, constants.NotFound, resp.Error)
	assert.Equal(t, "User not found", resp.Message)
}

func TestAdminHandler_UpdateUserHandler_HappyPath(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	existing := &models.UserModel{
		ID:            "user-1",
		Name:          "Alice",
		Email:         "alice@test.com",
		EmailVerified: false,
	}

	userRepo.On("FindById", "user-1", "dept-1").Return(existing, nil)
	userRepo.On("Save", existing, "dept-1").Return(nil)

	body := map[string]interface{}{
		"name":          "Alice Updated",
		"emailVerified": true,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/users/user-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.UpdateUserHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "User updated", resp.Message)
	assert.Equal(t, "Alice Updated", existing.Name)
	assert.True(t, existing.EmailVerified)
}

func TestAdminHandler_UpdateUserHandler_DecodeError(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	req := httptest.NewRequest(http.MethodPut, "/admin/users/user-1", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.UpdateUserHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminHandler_UpdateUserHandler_UserNotFound(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(nil, assert.AnError)

	body := map[string]interface{}{"name": "Alice Updated"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/users/user-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.UpdateUserHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminHandler_UpdateUserHandler_SaveError(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	existing := &models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}

	userRepo.On("FindById", "user-1", "dept-1").Return(existing, nil)
	userRepo.On("Save", existing, "dept-1").Return(assert.AnError)

	body := map[string]interface{}{"email": "new@test.com"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/users/user-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.UpdateUserHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminHandler_ResetUserPasswordHandler_HappyPath(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	existing := &models.UserModel{
		ID:       "user-1",
		Name:     "Alice",
		Email:    "alice@test.com",
		Password: "$2a$10$abcdefghijklmnopqrstuvwxyz12345678901234567890",
	}

	userRepo.On("FindById", "user-1", "dept-1").Return(existing, nil)
	userRepo.On("Save", existing, "dept-1").Return(nil)

	body := map[string]string{"newPassword": "StrongPass1!"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-1/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.ResetUserPasswordHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Password reset successfully", resp.Message)
	assert.NotEqual(t, "$2a$10$abcdefghijklmnopqrstuvwxyz12345678901234567890", existing.Password)
}

func TestAdminHandler_ResetUserPasswordHandler_WeakPassword(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	body := map[string]string{"newPassword": "short"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-1/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.ResetUserPasswordHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminHandler_ResetUserPasswordHandler_CommonPassword(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	config.AppConfig.PasswordMinLength = 1
	config.AppConfig.PasswordRequireUppercase = false
	config.AppConfig.PasswordRequireLowercase = false
	config.AppConfig.PasswordRequireNumber = false
	config.AppConfig.PasswordRequireSpecial = false

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	body := map[string]string{"newPassword": "password"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-1/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.ResetUserPasswordHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminHandler_ResetUserPasswordHandler_UserNotFound(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(nil, assert.AnError)

	body := map[string]string{"newPassword": "StrongPass1!"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-1/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.ResetUserPasswordHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminHandler_ResetUserPasswordHandler_ValidationError(t *testing.T) {
	handler, userRepo, _ := setupAdminHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-1/reset-password", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.ResetUserPasswordHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminHandler_RevokeUserSessionsHandler_HappyPath(t *testing.T) {
	handler, _, sessionRepo := setupAdminHandler(t)

	sessionRepo.On("RevokeAllUserSessions", "user-1", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/user-1/sessions", nil)
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.RevokeUserSessionsHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "All user sessions revoked", resp.Message)
}

func TestAdminHandler_RevokeUserSessionsHandler_Error(t *testing.T) {
	handler, _, sessionRepo := setupAdminHandler(t)

	sessionRepo.On("RevokeAllUserSessions", "user-1", "dept-1").Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/user-1/sessions", nil)
	req = withDepartment(req, "dept-1")
	req = mux.SetURLVars(req, map[string]string{"id": "user-1"})
	rec := httptest.NewRecorder()

	handler.RevokeUserSessionsHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
