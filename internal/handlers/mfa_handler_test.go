package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"uas/config"
	"uas/internal/helpers"
	"uas/internal/models"
)

func setupMfaHandler(t *testing.T) (*MfaHandler, *MockUserRepository, *MockMfaFactorRepository) {
	t.Helper()
	saveAppConfig(t)

	config.AppConfig.MfaBackupCodeCount = 8
	config.AppConfig.MfaIssuer = "UAS"
	config.AppConfig.EncryptionKey = "0123456789abcdef0123456789abcdef"

	log := newTestLogger()
	userRepo := new(MockUserRepository)
	mfaFactorRepo := new(MockMfaFactorRepository)

	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)
	authHelper := newTestAuthHelper(&log)
	mfaHelper := newTestMfaHelper(&log)
	encryptionHelper, err := helpers.NewEncryptionHelper(&log)
	require.NoError(t, err)

	handler := NewMfaHandler(
		mfaFactorRepo,
		nil,
		userRepo,
		authHelper,
		mfaHelper,
		encryptionHelper,
		responseHelper,
		validatorHelper,
		&log,
	)

	return handler, userRepo, mfaFactorRepo
}

func TestMfaHandler_EnrollHandler_MissingAuth(t *testing.T) {
	handler, _, _ := setupMfaHandler(t)

	body := map[string]string{"factorType": "totp", "name": "My App"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/mfa/enroll", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.EnrollHandler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMfaHandler_EnrollHandler_TOTP(t *testing.T) {
	handler, userRepo, mfaFactorRepo := setupMfaHandler(t)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@test.com",
	}, nil)

	mfaFactorRepo.On("Create", mock.MatchedBy(func(f *models.MfaFactor) bool {
		return f.UserID == "user-1" && f.DepartmentID == "dept-1" &&
			f.FactorType == models.MfaFactorTOTP && f.Name == "My App" &&
			f.Secret != "" && f.BackupCodes != ""
	})).Return(nil)

	body := map[string]string{"factorType": "totp", "name": "My App"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/mfa/enroll", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.EnrollHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "MFA factor enrolled. Verify by providing a valid code.", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "data should be a map")
	assert.Equal(t, "totp", data["factorType"])
	assert.NotEmpty(t, data["secret"])
	assert.NotEmpty(t, data["qrCodeUri"])
	assert.NotEmpty(t, data["backupCodes"])
}

func TestMfaHandler_ListFactorsHandler(t *testing.T) {
	handler, _, mfaFactorRepo := setupMfaHandler(t)

	mfaFactorRepo.On("FindByUserID", "user-1", "dept-1").Return([]models.MfaFactor{
		{
			ID:         "factor-1",
			FactorType: models.MfaFactorTOTP,
			Name:       "My App",
			IsPrimary:  true,
			CreatedAt:  time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/mfa/factors", nil)
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.ListFactorsHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "MFA factors retrieved", resp.Message)

	factors, ok := resp.Data.([]interface{})
	require.True(t, ok)
	require.Len(t, factors, 1)

	factor := factors[0].(map[string]interface{})
	assert.Equal(t, "factor-1", factor["id"])
	assert.Equal(t, "totp", factor["factorType"])
	assert.Equal(t, "My App", factor["name"])
	assert.Equal(t, true, factor["isPrimary"])
}

func TestMfaHandler_ListFactorsHandler_Empty(t *testing.T) {
	handler, _, mfaFactorRepo := setupMfaHandler(t)

	mfaFactorRepo.On("FindByUserID", "user-1", "dept-1").Return([]models.MfaFactor{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/mfa/factors", nil)
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.ListFactorsHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Nil(t, resp["data"])
}

func TestMfaHandler_StatusHandler_Enabled(t *testing.T) {
	handler, _, mfaFactorRepo := setupMfaHandler(t)

	mfaFactorRepo.On("CountActiveByUserID", "user-1", "dept-1").Return(int64(2), nil)
	mfaFactorRepo.On("FindByUserID", "user-1", "dept-1").Return([]models.MfaFactor{
		{
			ID:         "factor-1",
			FactorType: models.MfaFactorTOTP,
			Name:       "My App",
			IsPrimary:  true,
			CreatedAt:  time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/mfa/status", nil)
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.StatusHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, data["mfaEnabled"])

	factors, ok := data["factors"].([]interface{})
	require.True(t, ok)
	require.Len(t, factors, 1)
}

func TestMfaHandler_StatusHandler_Disabled(t *testing.T) {
	handler, _, mfaFactorRepo := setupMfaHandler(t)

	mfaFactorRepo.On("CountActiveByUserID", "user-1", "dept-1").Return(int64(0), nil)
	mfaFactorRepo.On("FindByUserID", "user-1", "dept-1").Return([]models.MfaFactor{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/mfa/status", nil)
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.StatusHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, data["mfaEnabled"])

	factors := data["factors"]
	assert.Nil(t, factors)
}

func TestMfaHandler_DisableHandler_HappyPath(t *testing.T) {
	handler, userRepo, mfaFactorRepo := setupMfaHandler(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("StrongPass1!"), bcrypt.DefaultCost)
	require.NoError(t, err)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:       "user-1",
		Name:     "Alice",
		Email:    "alice@test.com",
		Password: string(hash),
	}, nil)

	mfaFactorRepo.On("Delete", "factor-1", "dept-1").Return(nil)

	body := map[string]string{
		"factorId": "factor-1",
		"password": "StrongPass1!",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/mfa/disable", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.DisableHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "MFA factor disabled successfully", resp.Message)
}

func TestMfaHandler_DisableHandler_WrongPassword(t *testing.T) {
	handler, userRepo, _ := setupMfaHandler(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("StrongPass1!"), bcrypt.DefaultCost)
	require.NoError(t, err)

	userRepo.On("FindById", "user-1", "dept-1").Return(&models.UserModel{
		ID:       "user-1",
		Name:     "Alice",
		Email:    "alice@test.com",
		Password: string(hash),
	}, nil)

	body := map[string]string{
		"factorId": "factor-1",
		"password": "WrongPassword1!",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/mfa/disable", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = helpers.SetUserId(req, "user-1")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	handler.DisableHandler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
