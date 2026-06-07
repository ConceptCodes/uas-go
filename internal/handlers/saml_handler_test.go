package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"uas/config"
	"uas/internal/models"
)

type samlHandlerDeps struct {
	handler      *SamlHandler
	idPRepo      *MockIdentityProviderRepository
	identityRepo *MockUserIdentityRepository
	userRepo     *MockUserRepository
	deptRoleRepo *MockDepartmentRoleRepository
	sessionRepo  *MockSessionRepository
}

func setupSamlHandler(t *testing.T) *samlHandlerDeps {
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
	authHelper := newTestAuthHelper(&log)
	mfaHelper := newTestMfaHelper(&log)

	handler := NewSamlHandler(
		idPRepo, identityRepo, userRepo, deptRoleRepo, sessionRepo,
		authHelper, mfaHelper, responseHelper, &log,
	)

	return &samlHandlerDeps{
		handler:      handler,
		idPRepo:      idPRepo,
		identityRepo: identityRepo,
		userRepo:     userRepo,
		deptRoleRepo: deptRoleRepo,
		sessionRepo:  sessionRepo,
	}
}

func TestSamlHandler_GetMetadataHandler_ProviderNotFound(t *testing.T) {
	d := setupSamlHandler(t)

	d.idPRepo.On("FindByID", "idp-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/saml/idp-999/metadata", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetMetadataHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSamlHandler_LoginHandler_ProviderNotFound(t *testing.T) {
	d := setupSamlHandler(t)

	d.idPRepo.On("FindByID", "idp-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/saml/idp-999/login", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.LoginHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSamlHandler_LoginHandler_ProviderDisabled(t *testing.T) {
	d := setupSamlHandler(t)

	d.idPRepo.On("FindByID", "idp-1", "dept-1").Return(&models.IdentityProvider{
		ID: "idp-1", Name: "SAML Provider", ProviderType: models.IDPSAML,
		Enabled: false,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/saml/idp-1/login", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.LoginHandler(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSamlHandler_AssertionConsumerServiceHandler_ProviderNotFound(t *testing.T) {
	d := setupSamlHandler(t)

	d.idPRepo.On("FindByDepartment", "").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodPost, "/saml/idp-999/acs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-999"})
	rec := httptest.NewRecorder()

	d.handler.AssertionConsumerServiceHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSamlHandler_AssertionConsumerServiceHandler_ProviderNotInResults(t *testing.T) {
	d := setupSamlHandler(t)

	d.idPRepo.On("FindByDepartment", "").Return([]models.IdentityProvider{
		{ID: "idp-other", Name: "Other Provider", ProviderType: models.IDPSAML},
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/saml/idp-999/acs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "idp-999"})
	rec := httptest.NewRecorder()

	d.handler.AssertionConsumerServiceHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
