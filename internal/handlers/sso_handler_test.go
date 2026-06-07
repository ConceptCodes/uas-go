package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"uas/config"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"uas/internal/helpers"
	"uas/internal/models"
)

type ssoHandlerDeps struct {
	handler      *SsoHandler
	idPRepo      *MockIdentityProviderRepository
	identityRepo *MockUserIdentityRepository
	userRepo     *MockUserRepository
	deptRoleRepo *MockDepartmentRoleRepository
	sessionRepo  *MockSessionRepository
}

func setupSsoHandler(t *testing.T) *ssoHandlerDeps {
	t.Helper()
	saveAppConfig(t)

	config.AppConfig.AccessJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.RefreshJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.CookieHashKey = "12345678901234567890123456789012"
	config.AppConfig.CookieBlockKey = "12345678901234567890123456789012"
	config.AppConfig.AccessJwtExpire = 1
	config.AppConfig.RefreshJwtExpire = 24
	config.AppConfig.SsoStateExpireMin = 10

	log := zerolog.Nop()
	idPRepo := new(MockIdentityProviderRepository)
	identityRepo := new(MockUserIdentityRepository)
	userRepo := new(MockUserRepository)
	deptRoleRepo := new(MockDepartmentRoleRepository)
	sessionRepo := new(MockSessionRepository)

	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)
	authHelper := newTestAuthHelper(&log)
	ssoHelper := helpers.NewSSOHelper(&log)
	mfaHelper := newTestMfaHelper(&log)

	handler := NewSsoHandler(
		idPRepo, identityRepo, userRepo, deptRoleRepo, sessionRepo,
		authHelper, ssoHelper, mfaHelper, responseHelper, validatorHelper, &log,
	)

	return &ssoHandlerDeps{
		handler:      handler,
		idPRepo:      idPRepo,
		identityRepo: identityRepo,
		userRepo:     userRepo,
		deptRoleRepo: deptRoleRepo,
		sessionRepo:  sessionRepo,
	}
}

func TestSsoHandler_CreateProviderHandler_HappyPath(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("Create", mock.MatchedBy(func(p *models.IdentityProvider) bool {
		return p.Name == "Test Provider" && p.ProviderType == models.IDPGoogle
	})).Return(nil)

	body := map[string]interface{}{
		"name":         "Test Provider",
		"providerType": "google",
		"clientId":     "client-123",
		"clientSecret": "secret-456",
		"redirectUrls": []string{"https://example.com/callback"},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sso/providers", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.Admin)
	rec := httptest.NewRecorder()

	d.handler.CreateProviderHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Identity provider created", resp.Message)
}

func TestSsoHandler_CreateProviderHandler_Forbidden(t *testing.T) {
	d := setupSsoHandler(t)

	body := map[string]interface{}{
		"name":         "Test Provider",
		"providerType": "google",
		"clientId":     "client-123",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sso/providers", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.User)
	rec := httptest.NewRecorder()

	d.handler.CreateProviderHandler(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSsoHandler_CreateProviderHandler_DecodeError(t *testing.T) {
	d := setupSsoHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/sso/providers", bytes.NewReader([]byte(`invalid`)))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.Admin)
	rec := httptest.NewRecorder()

	d.handler.CreateProviderHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSsoHandler_ListProvidersHandler_HappyPath(t *testing.T) {
	d := setupSsoHandler(t)

	providers := []models.IdentityProvider{
		{ID: "idp-1", Name: "Google", ProviderType: models.IDPGoogle},
		{ID: "idp-2", Name: "GitHub", ProviderType: models.IDPGithub},
	}
	d.idPRepo.On("FindByDepartment", "dept-1").Return(providers, nil)

	req := httptest.NewRequest(http.MethodGet, "/sso/providers", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListProvidersHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSsoHandler_ListProvidersHandler_Error(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("FindByDepartment", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/sso/providers", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListProvidersHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSsoHandler_GetProviderHandler_Found(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("FindByID", "idp-1", "dept-1").Return(&models.IdentityProvider{
		ID: "idp-1", Name: "Google", ProviderType: models.IDPGoogle,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/sso/providers/idp-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetProviderHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSsoHandler_GetProviderHandler_NotFound(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("FindByID", "idp-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/sso/providers/idp-999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetProviderHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSsoHandler_UpdateProviderHandler_HappyPath(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("FindByID", "idp-1", "dept-1").Return(&models.IdentityProvider{
		ID: "idp-1", Name: "Old Name", ProviderType: models.IDPGoogle,
	}, nil)

	d.idPRepo.On("Update", mock.MatchedBy(func(p *models.IdentityProvider) bool {
		return p.ID == "idp-1" && p.Name == "New Name"
	})).Return(nil)

	body := map[string]interface{}{
		"name": "New Name",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/sso/providers/idp-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.Admin)
	rec := httptest.NewRecorder()

	d.handler.UpdateProviderHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSsoHandler_UpdateProviderHandler_Forbidden(t *testing.T) {
	d := setupSsoHandler(t)

	body := map[string]interface{}{
		"name": "New Name",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/sso/providers/idp-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.User)
	rec := httptest.NewRecorder()

	d.handler.UpdateProviderHandler(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSsoHandler_UpdateProviderHandler_NotFound(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("FindByID", "idp-999", "dept-1").Return(nil, assert.AnError)

	body := map[string]interface{}{
		"name": "New Name",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/sso/providers/idp-999", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "idp-999"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.Admin)
	rec := httptest.NewRecorder()

	d.handler.UpdateProviderHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSsoHandler_DeleteProviderHandler_HappyPath(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("Delete", "idp-1", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/sso/providers/idp-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.Admin)
	rec := httptest.NewRecorder()

	d.handler.DeleteProviderHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSsoHandler_DeleteProviderHandler_Forbidden(t *testing.T) {
	d := setupSsoHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/sso/providers/idp-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.User)
	rec := httptest.NewRecorder()

	d.handler.DeleteProviderHandler(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSsoHandler_DeleteProviderHandler_Error(t *testing.T) {
	d := setupSsoHandler(t)

	d.idPRepo.On("Delete", "idp-1", "dept-1").Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/sso/providers/idp-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetRole(req, models.Admin)
	rec := httptest.NewRecorder()

	d.handler.DeleteProviderHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSsoHandler_ListIdentitiesHandler_HappyPath(t *testing.T) {
	d := setupSsoHandler(t)

	d.identityRepo.On("FindByUserID", "user-1", "dept-1").Return([]models.UserIdentity{
		{ID: "ident-1", ProviderID: "idp-1", ProviderUserID: "ext-1"},
	}, nil)

	d.idPRepo.On("FindByID", "idp-1", "dept-1").Return(&models.IdentityProvider{
		ID: "idp-1", Name: "Google", ProviderType: models.IDPGoogle,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/sso/identities", nil)
	req = withDepartment(req, "dept-1")
	req = helpers.SetUserId(req, "user-1")
	rec := httptest.NewRecorder()

	d.handler.ListIdentitiesHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSsoHandler_ListIdentitiesHandler_NoAuth(t *testing.T) {
	d := setupSsoHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/sso/identities", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListIdentitiesHandler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSsoHandler_DeleteIdentityHandler_HappyPath(t *testing.T) {
	d := setupSsoHandler(t)

	d.identityRepo.On("FindByID", "ident-1", "dept-1").Return(&models.UserIdentity{
		ID: "ident-1", UserID: "user-1", ProviderID: "idp-1",
	}, nil)

	d.identityRepo.On("Delete", "ident-1", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/sso/identities/ident-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ident-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetUserId(req, "user-1")
	rec := httptest.NewRecorder()

	d.handler.DeleteIdentityHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSsoHandler_DeleteIdentityHandler_NoAuth(t *testing.T) {
	d := setupSsoHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/sso/identities/ident-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ident-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.DeleteIdentityHandler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSsoHandler_DeleteIdentityHandler_NotFound(t *testing.T) {
	d := setupSsoHandler(t)

	d.identityRepo.On("FindByID", "ident-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/sso/identities/ident-999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ident-999"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetUserId(req, "user-1")
	rec := httptest.NewRecorder()

	d.handler.DeleteIdentityHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSsoHandler_DeleteIdentityHandler_WrongUser(t *testing.T) {
	d := setupSsoHandler(t)

	d.identityRepo.On("FindByID", "ident-1", "dept-1").Return(&models.UserIdentity{
		ID: "ident-1", UserID: "other-user", ProviderID: "idp-1",
	}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/sso/identities/ident-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ident-1"})
	req = withDepartment(req, "dept-1")
	req = helpers.SetUserId(req, "user-1")
	rec := httptest.NewRecorder()

	d.handler.DeleteIdentityHandler(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
