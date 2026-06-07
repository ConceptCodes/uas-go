package handlers

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	"uas/config"
	"uas/internal/helpers"
	"uas/internal/models"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindById(id, departmentID string) (*models.UserModel, error) {
	args := m.Called(id, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserModel), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email, departmentID string) (*models.UserModel, error) {
	args := m.Called(email, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserModel), args.Error(1)
}

func (m *MockUserRepository) FindByPhoneNumber(phoneNumber, departmentID string) (*models.UserModel, error) {
	args := m.Called(phoneNumber, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserModel), args.Error(1)
}

func (m *MockUserRepository) Create(user *models.UserModel) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

func (m *MockUserRepository) Save(user *models.UserModel, departmentID string) error {
	args := m.Called(user, departmentID)
	return args.Error(0)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(session *models.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByRefreshToken(token, departmentID string) (*models.Session, error) {
	args := m.Called(token, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) FindByUserID(userID, departmentID string) ([]models.Session, error) {
	args := m.Called(userID, departmentID)
	return args.Get(0).([]models.Session), args.Error(1)
}

func (m *MockSessionRepository) RevokeSession(sessionID, departmentID string) error {
	args := m.Called(sessionID, departmentID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllUserSessions(userID, departmentID string) error {
	args := m.Called(userID, departmentID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllByDepartment(departmentID string) error {
	args := m.Called(departmentID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpiredSessions() error {
	args := m.Called()
	return args.Error(0)
}

func saveAppConfig(t *testing.T) {
	t.Helper()
	orig := config.AppConfig
	t.Cleanup(func() { config.AppConfig = orig })
}

func newTestLogger() zerolog.Logger {
	return zerolog.Nop()
}

func newTestResponseHelper(log *zerolog.Logger) *helpers.ResponseHelper {
	return helpers.NewResponseHelper(log)
}

func newTestValidatorHelper(log *zerolog.Logger, rh *helpers.ResponseHelper) *helpers.ValidatorHelper {
	return helpers.NewValidatorHelper(log, rh)
}

func newTestAuthHelper(log *zerolog.Logger) *helpers.AuthHelper {
	return helpers.NewAuthHelper(log, nil, helpers.RedisHelper{})
}

func newTestPasswordHelper(log *zerolog.Logger) *helpers.PasswordHelper {
	return helpers.NewPasswordHelper(log)
}

func newTestMfaHelper(log *zerolog.Logger) *helpers.MfaHelper {
	return helpers.NewMfaHelper(log, nil)
}

type MockMfaFactorRepository struct {
	mock.Mock
}

func (m *MockMfaFactorRepository) Create(factor *models.MfaFactor) error {
	args := m.Called(factor)
	return args.Error(0)
}

func (m *MockMfaFactorRepository) FindByID(id, departmentID string) (*models.MfaFactor, error) {
	args := m.Called(id, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MfaFactor), args.Error(1)
}

func (m *MockMfaFactorRepository) FindByUserID(userID, departmentID string) ([]models.MfaFactor, error) {
	args := m.Called(userID, departmentID)
	return args.Get(0).([]models.MfaFactor), args.Error(1)
}

func (m *MockMfaFactorRepository) FindActiveByUserID(userID, departmentID string) ([]models.MfaFactor, error) {
	args := m.Called(userID, departmentID)
	return args.Get(0).([]models.MfaFactor), args.Error(1)
}

func (m *MockMfaFactorRepository) SetPrimary(id, userID, departmentID string) error {
	args := m.Called(id, userID, departmentID)
	return args.Error(0)
}

func (m *MockMfaFactorRepository) UpdateBackupCodes(id, departmentID, backupCodes string) error {
	args := m.Called(id, departmentID, backupCodes)
	return args.Error(0)
}

func (m *MockMfaFactorRepository) UpdateLastUsed(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

func (m *MockMfaFactorRepository) Delete(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

func (m *MockMfaFactorRepository) DeleteByUserID(userID, departmentID string) error {
	args := m.Called(userID, departmentID)
	return args.Error(0)
}

func (m *MockMfaFactorRepository) GetPrimaryFactor(userID, departmentID string) (*models.MfaFactor, error) {
	args := m.Called(userID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MfaFactor), args.Error(1)
}

func (m *MockMfaFactorRepository) CountActiveByUserID(userID, departmentID string) (int64, error) {
	args := m.Called(userID, departmentID)
	return args.Get(0).(int64), args.Error(1)
}

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) Create(auth *models.AuthModel) error {
	args := m.Called(auth)
	return args.Error(0)
}

func (m *MockAuthRepository) Delete(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

func (m *MockAuthRepository) DeleteByTokenAndType(token string, authType models.AuthModelType, departmentID string) error {
	args := m.Called(token, authType, departmentID)
	return args.Error(0)
}

func (m *MockAuthRepository) FindByTokenAndType(token string, authType models.AuthModelType) (*models.AuthModel, error) {
	args := m.Called(token, authType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthModel), args.Error(1)
}

func (m *MockAuthRepository) FindByTokenAndTypeScoped(token string, authType models.AuthModelType, departmentID string) (*models.AuthModel, error) {
	args := m.Called(token, authType, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthModel), args.Error(1)
}

type MockDepartmentRoleRepository struct {
	mock.Mock
}

func (m *MockDepartmentRoleRepository) Create(role *models.DepartmentRoles) error {
	args := m.Called(role)
	return args.Error(0)
}

func (m *MockDepartmentRoleRepository) Update(role *models.DepartmentRoles) error {
	args := m.Called(role)
	return args.Error(0)
}

func (m *MockDepartmentRoleRepository) FindById(departmentID, userID string) (*models.DepartmentRoles, error) {
	args := m.Called(departmentID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DepartmentRoles), args.Error(1)
}

func (m *MockDepartmentRoleRepository) FindByUserID(userID, departmentID string) (*models.DepartmentRoles, error) {
	args := m.Called(userID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DepartmentRoles), args.Error(1)
}

type MockSecurityAuditRepository struct {
	mock.Mock
}

func (m *MockSecurityAuditRepository) Create(event *models.SecurityEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockSecurityAuditRepository) GetByUserID(userID string, limit int) ([]models.SecurityEvent, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]models.SecurityEvent), args.Error(1)
}

func (m *MockSecurityAuditRepository) GetByDepartmentID(departmentID string, limit int) ([]models.SecurityEvent, error) {
	args := m.Called(departmentID, limit)
	return args.Get(0).([]models.SecurityEvent), args.Error(1)
}

func (m *MockSecurityAuditRepository) GetByEventType(eventType string, days int) ([]models.SecurityEvent, error) {
	args := m.Called(eventType, days)
	return args.Get(0).([]models.SecurityEvent), args.Error(1)
}

func (m *MockSecurityAuditRepository) DeleteOldEvents(days int) error {
	args := m.Called(days)
	return args.Error(0)
}

type MockWebhookEndpointRepository struct {
	mock.Mock
}

func (m *MockWebhookEndpointRepository) Create(endpoint *models.WebhookEndpoint) error {
	args := m.Called(endpoint)
	return args.Error(0)
}

func (m *MockWebhookEndpointRepository) FindByID(id, departmentID string) (*models.WebhookEndpoint, error) {
	args := m.Called(id, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookEndpointRepository) FindByDepartment(departmentID string) ([]models.WebhookEndpoint, error) {
	args := m.Called(departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookEndpointRepository) FindActiveByDepartmentAndEvent(departmentID string, event models.WebhookEventType) ([]models.WebhookEndpoint, error) {
	args := m.Called(departmentID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookEndpointRepository) Update(endpoint *models.WebhookEndpoint) error {
	args := m.Called(endpoint)
	return args.Error(0)
}

func (m *MockWebhookEndpointRepository) UpdateSecret(id, departmentID, secret string) error {
	args := m.Called(id, departmentID, secret)
	return args.Error(0)
}

func (m *MockWebhookEndpointRepository) Delete(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

type MockWebhookDeliveryRepository struct {
	mock.Mock
}

func (m *MockWebhookDeliveryRepository) Create(delivery *models.WebhookDelivery) error {
	args := m.Called(delivery)
	return args.Error(0)
}

func (m *MockWebhookDeliveryRepository) FindByID(id, endpointID string) (*models.WebhookDelivery, error) {
	args := m.Called(id, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookDeliveryRepository) FindByEndpoint(endpointID, departmentID string, limit, offset int) ([]models.WebhookDelivery, error) {
	args := m.Called(endpointID, departmentID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookDeliveryRepository) FindByDepartment(departmentID string, limit, offset int) ([]models.WebhookDelivery, error) {
	args := m.Called(departmentID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookDeliveryRepository) FindPendingRetries() ([]models.WebhookDelivery, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookDeliveryRepository) UpdateDelivery(id, endpointID string, status models.WebhookDeliveryStatus, responseCode int, responseBody string) error {
	args := m.Called(id, endpointID, status, responseCode, responseBody)
	return args.Error(0)
}

func (m *MockWebhookDeliveryRepository) UpdateDeliveryWithRetry(id, endpointID string, status models.WebhookDeliveryStatus, responseCode int, responseBody string, attempt int, nextRetryAt *time.Time) error {
	args := m.Called(id, endpointID, status, responseCode, responseBody, attempt, nextRetryAt)
	return args.Error(0)
}

func (m *MockWebhookDeliveryRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	args := m.Called(duration)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), args.Error(1)
}

type MockIdentityProviderRepository struct {
	mock.Mock
}

func (m *MockIdentityProviderRepository) Create(provider *models.IdentityProvider) error {
	args := m.Called(provider)
	return args.Error(0)
}

func (m *MockIdentityProviderRepository) FindByID(id, departmentID string) (*models.IdentityProvider, error) {
	args := m.Called(id, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IdentityProvider), args.Error(1)
}

func (m *MockIdentityProviderRepository) FindByDepartment(departmentID string) ([]models.IdentityProvider, error) {
	args := m.Called(departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.IdentityProvider), args.Error(1)
}

func (m *MockIdentityProviderRepository) FindByDepartmentAndType(departmentID string, providerType models.IdentityProviderType) ([]models.IdentityProvider, error) {
	args := m.Called(departmentID, providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.IdentityProvider), args.Error(1)
}

func (m *MockIdentityProviderRepository) Update(provider *models.IdentityProvider) error {
	args := m.Called(provider)
	return args.Error(0)
}

func (m *MockIdentityProviderRepository) Delete(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

type MockUserIdentityRepository struct {
	mock.Mock
}

func (m *MockUserIdentityRepository) Create(identity *models.UserIdentity) error {
	args := m.Called(identity)
	return args.Error(0)
}

func (m *MockUserIdentityRepository) FindByID(id, departmentID string) (*models.UserIdentity, error) {
	args := m.Called(id, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserIdentity), args.Error(1)
}

func (m *MockUserIdentityRepository) FindByUserID(userID, departmentID string) ([]models.UserIdentity, error) {
	args := m.Called(userID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.UserIdentity), args.Error(1)
}

func (m *MockUserIdentityRepository) FindByProvider(providerID, providerUserID, departmentID string) (*models.UserIdentity, error) {
	args := m.Called(providerID, providerUserID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserIdentity), args.Error(1)
}

func (m *MockUserIdentityRepository) FindByProviderEmail(providerEmail, departmentID string) (*models.UserIdentity, error) {
	args := m.Called(providerEmail, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserIdentity), args.Error(1)
}

func (m *MockUserIdentityRepository) Delete(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}

func (m *MockUserIdentityRepository) DeleteByUserID(userID, departmentID string) error {
	args := m.Called(userID, departmentID)
	return args.Error(0)
}

func (m *MockUserIdentityRepository) UpdateLastLogin(id, departmentID string) error {
	args := m.Called(id, departmentID)
	return args.Error(0)
}
