package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
)

type userHandlerDeps struct {
	handler     *UserHandler
	userRepo    *MockUserRepository
	authRepo    *MockAuthRepository
	sessionRepo *MockSessionRepository
	mr          *miniredis.Miniredis
}

func setupUserHandler(t *testing.T) *userHandlerDeps {
	t.Helper()
	saveAppConfig(t)

	config.AppConfig.PasswordMinLength = 8
	config.AppConfig.PasswordRequireUppercase = true
	config.AppConfig.PasswordRequireLowercase = true
	config.AppConfig.PasswordRequireNumber = true
	config.AppConfig.PasswordRequireSpecial = true

	config.AppConfig.AccessJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.RefreshJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.CookieHashKey = "12345678901234567890123456789012"
	config.AppConfig.CookieBlockKey = "12345678901234567890123456789012"
	config.AppConfig.AccessJwtExpire = 1
	config.AppConfig.RefreshJwtExpire = 24
	config.AppConfig.JwtIssuer = "uas"
	config.AppConfig.JwtAudience = "uas"

	log := newTestLogger()
	userRepo := new(MockUserRepository)
	authRepo := new(MockAuthRepository)
	sessionRepo := new(MockSessionRepository)
	auditRepo := new(MockSecurityAuditRepository)

	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)
	passwordHelper := newTestPasswordHelper(&log)
	authHelper := newTestAuthHelper(&log)
	securityLogger := helpers.NewSecurityLoggerHelper(&log, auditRepo)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisHelper := helpers.NewRedisHelper(redisClient, &log, context.Background())
	tokenHelper := helpers.NewTokenHelper(&log, redisHelper)
	authHelper.WithTokenHelper(tokenHelper)

	auditRepo.On("Create", mock.Anything).Return(nil).Maybe()

	handler := NewUserHandler(
		userRepo,
		authRepo,
		nil,
		nil,
		sessionRepo,
		nil,
		&log,
		authHelper,
		responseHelper,
		validatorHelper,
		nil,
		nil,
		nil,
		passwordHelper,
		nil,
		tokenHelper,
		securityLogger,
		nil,
		nil,
		nil,
	)

	return &userHandlerDeps{
		handler:     handler,
		userRepo:    userRepo,
		authRepo:    authRepo,
		sessionRepo: sessionRepo,
		mr:          mr,
	}
}

func generateRefreshToken(authHelper *helpers.AuthHelper, userID, deptID string) string {
	user := &models.UserModel{ID: userID, Email: "user@test.com", Name: "Test"}
	token, err := authHelper.GenerateRefreshJwtToken(user, deptID)
	if err != nil {
		panic(err)
	}
	return token
}

func generateAccessToken(authHelper *helpers.AuthHelper, userID, deptID string) string {
	user := &models.UserModel{ID: userID, Email: "user@test.com", Name: "Test"}
	token, err := authHelper.GenerateAccessJwtToken(user, deptID)
	if err != nil {
		panic(err)
	}
	return token
}

func TestUserHandler_ResetPasswordHandler_HappyPath(t *testing.T) {
	d := setupUserHandler(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("oldPassword1!"), bcrypt.DefaultCost)
	require.NoError(t, err)

	d.authRepo.On("FindByTokenAndType", "reset-token-123", models.ResetPassword).
		Return(&models.AuthModel{
			UserID:       "user-1",
			DepartmentID: "dept-1",
			Token:        "reset-token-123",
			Type:         models.ResetPassword,
		}, nil)

	d.userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:       "user-1",
		Name:     "Alice",
		Email:    "alice@test.com",
		Password: string(hash),
	}, nil)

	d.userRepo.On("Save", mock.MatchedBy(func(u *models.UserModel) bool {
		return u.ID == "user-1" && u.Password != string(hash)
	}), "dept-1").Return(nil)

	d.authRepo.On("DeleteByTokenAndType", "reset-token-123", models.ResetPassword, "dept-1").Return(nil)

	body := map[string]string{
		"token":    "reset-token-123",
		"password": "NewStrongPass1!",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/credential/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.CredentialsResetPasswordHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Password reset successfully", resp.Message)
}

func TestUserHandler_ResetPasswordHandler_InvalidToken(t *testing.T) {
	d := setupUserHandler(t)

	d.authRepo.On("FindByTokenAndType", "invalid-token", models.ResetPassword).
		Return(nil, assert.AnError)

	body := map[string]string{
		"token":    "invalid-token",
		"password": "NewStrongPass1!",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/credential/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.CredentialsResetPasswordHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_ResetPasswordHandler_WeakPassword(t *testing.T) {
	d := setupUserHandler(t)

	d.authRepo.On("FindByTokenAndType", "reset-token-123", models.ResetPassword).
		Return(&models.AuthModel{
			UserID:       "user-1",
			DepartmentID: "dept-1",
			Token:        "reset-token-123",
			Type:         models.ResetPassword,
		}, nil)

	d.userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	body := map[string]string{
		"token":    "reset-token-123",
		"password": "short",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/credential/reset-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handler.CredentialsResetPasswordHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_RefreshAccessTokenHandler_HappyPath(t *testing.T) {
	d := setupUserHandler(t)

	refreshToken := generateRefreshToken(d.handler.authHelper, "user-1", "dept-1")

	d.sessionRepo.On("FindByRefreshToken", refreshToken, "dept-1").Return(&models.Session{
		ID:           "session-1",
		UserID:       "user-1",
		DepartmentID: "dept-1",
		RefreshToken: refreshToken,
	}, nil)

	d.userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	d.sessionRepo.On("RevokeSession", "session-1", "dept-1").Return(nil)

	d.sessionRepo.On("Create", mock.MatchedBy(func(s *models.Session) bool {
		return s.UserID == "user-1" && s.DepartmentID == "dept-1"
	})).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/refresh-token", nil)
	req.Header.Set(constants.JwtHeader, refreshToken)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RefreshAccessTokenHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Access token refreshed successfully", resp.Message)

	newRefreshToken := rec.Header().Get(constants.JwtHeader)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, refreshToken, newRefreshToken)
}

func TestUserHandler_RefreshAccessTokenHandler_MissingToken(t *testing.T) {
	d := setupUserHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/users/refresh-token", nil)
	rec := httptest.NewRecorder()

	d.handler.RefreshAccessTokenHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_RefreshAccessTokenHandler_InvalidToken(t *testing.T) {
	d := setupUserHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/users/refresh-token", nil)
	req.Header.Set(constants.JwtHeader, "invalid.jwt.token")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RefreshAccessTokenHandler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserHandler_RefreshAccessTokenHandler_SessionNotFound(t *testing.T) {
	d := setupUserHandler(t)

	refreshToken := generateRefreshToken(d.handler.authHelper, "user-1", "dept-1")

	d.sessionRepo.On("FindByRefreshToken", refreshToken, "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodPost, "/users/refresh-token", nil)
	req.Header.Set(constants.JwtHeader, refreshToken)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RefreshAccessTokenHandler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserHandler_LogoutHandler_WithRefreshToken(t *testing.T) {
	d := setupUserHandler(t)

	refreshToken := generateRefreshToken(d.handler.authHelper, "user-1", "dept-1")

	d.sessionRepo.On("FindByRefreshToken", refreshToken, "dept-1").Return(&models.Session{
		ID:           "session-1",
		UserID:       "user-1",
		DepartmentID: "dept-1",
		RefreshToken: refreshToken,
	}, nil)

	d.sessionRepo.On("RevokeAllUserSessions", "user-1", "dept-1").Return(nil)
	d.sessionRepo.On("RevokeSession", "session-1", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/logout", nil)
	req.Header.Set(constants.JwtHeader, refreshToken)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.LogoutHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully", resp.Message)

	cookies := rec.Result().Cookies()
	var accessCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == constants.AccessTokenCookie {
			accessCookie = c
			break
		}
	}
	require.NotNil(t, accessCookie)
	assert.Equal(t, "", accessCookie.Value)
	assert.True(t, accessCookie.MaxAge < 0)
}

func TestUserHandler_LogoutHandler_WithAccessCookie(t *testing.T) {
	d := setupUserHandler(t)

	accessToken := generateAccessToken(d.handler.authHelper, "user-1", "dept-1")

	d.sessionRepo.On("RevokeAllUserSessions", "user-1", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  constants.AccessTokenCookie,
		Value: accessToken,
	})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.LogoutHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_LogoutHandler_NoAuth(t *testing.T) {
	d := setupUserHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/users/logout", nil)
	rec := httptest.NewRecorder()

	d.handler.LogoutHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
