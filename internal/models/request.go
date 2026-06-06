package models

import "encoding/json"

type Request struct {
	ID           string
	DepartmentID string
	User         UserModel
}

type OnboardTenantRequest struct {
	DepartmentName string `json:"departmentName" validate:"required,noSQLKeywords"`
	DepartmentID   string `json:"departmentId" validate:"required,noSQLKeywords"`
}

type CredentialsLoginRequest struct {
	Email    string `json:"email" validate:"email,required,noSQLKeywords"`
	Password string `json:"password" validate:"required,noSQLKeywords"`
}

type RegisterRequest struct {
	Name        string `json:"name" validate:"required,noSQLKeywords"`
	Email       string `json:"email" validate:"email,required,noSQLKeywords"`
	Password    string `json:"password" validate:"required,noSQLKeywords"`
	PhoneNumber string `json:"phoneNumber" validate:"required,e164,noSQLKeywords"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"email,required,noSQLKeywords"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" validate:"required,noSQLKeywords"`
}

type SendOtpRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"required,noSQLKeywords"`
}

type VerifyOtpRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"required,noSQLKeywords"`
	Otp         string `json:"otp" validate:"required,noSQLKeywords,numeric"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"email,required,noSQLKeywords"`
	Otp   string `json:"otp" validate:"required,noSQLKeywords,numeric"`
}

type MagicLinkEmailRequest = ForgotPasswordRequest

// MFA request types
type MfaEnrollRequest struct {
	FactorType MfaFactorType `json:"factorType" validate:"required,oneof=totp sms email"`
	Name       string        `json:"name" validate:"noSQLKeywords"`
	PhoneNumber string       `json:"phoneNumber,omitempty"`
}

type MfaVerifyEnrollRequest struct {
	FactorID string `json:"factorId" validate:"required"`
	Code     string `json:"code" validate:"required,numeric"`
}

type MfaChallengeVerifyRequest struct {
	TempToken string `json:"tempToken" validate:"required"`
	Code      string `json:"code" validate:"required,numeric"`
	FactorID  string `json:"factorId,omitempty"`
}

type MfaDisableRequest struct {
	FactorID string `json:"factorId" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type MfaRecoverRequest struct {
	TempToken  string `json:"tempToken" validate:"required"`
	BackupCode string `json:"backupCode" validate:"required"`
}

// SSO request types
type CreateIdPRequest struct {
	Name             string               `json:"name" validate:"required,noSQLKeywords"`
	ProviderType     IdentityProviderType `json:"providerType" validate:"required,oneof=google github oidc saml"`
	ClientID         string               `json:"clientId" validate:"required"`
	ClientSecret     string               `json:"clientSecret"`
	IssuerURL        string               `json:"issuerUrl"`
	AuthorizationURL string               `json:"authorizationUrl"`
	TokenURL         string               `json:"tokenUrl"`
	UserInfoURL      string               `json:"userInfoUrl"`
	JWKSURI          string               `json:"jwksUri"`
	MetadataURL      string               `json:"metadataUrl"`
	RedirectURLs     []string             `json:"redirectUrls"`
	Scopes           []string             `json:"scopes"`
}

type UpdateIdPRequest struct {
	Name             *string             `json:"name,omitempty"`
	ClientID         *string             `json:"clientId,omitempty"`
	ClientSecret     *string             `json:"clientSecret,omitempty"`
	IssuerURL        *string             `json:"issuerUrl,omitempty"`
	AuthorizationURL *string             `json:"authorizationUrl,omitempty"`
	TokenURL         *string             `json:"tokenUrl,omitempty"`
	UserInfoURL      *string             `json:"userInfoUrl,omitempty"`
	JWKSURI          *string             `json:"jwksUri,omitempty"`
	MetadataURL      *string             `json:"metadataUrl,omitempty"`
	RedirectURLs     []string            `json:"redirectUrls,omitempty"`
	Scopes           []string            `json:"scopes,omitempty"`
	Enabled          *bool               `json:"enabled,omitempty"`
}

type SsoLoginRequest struct {
	ProviderID string `json:"providerId" validate:"required"`
	RedirectURI string `json:"redirectUri" validate:"required"`
}

type SsoCallbackRequest struct {
	Code         string `json:"code" validate:"required"`
	State        string `json:"state" validate:"required"`
	ProviderID   string `json:"providerId" validate:"required"`
	RedirectURI  string `json:"redirectUri" validate:"required"`
	CodeVerifier string `json:"codeVerifier"`
}

type LinkIdentityRequest struct {
	ProviderID     string `json:"providerId" validate:"required"`
	ProviderUserID string `json:"providerUserId" validate:"required"`
	ProviderEmail  string `json:"providerEmail" validate:"email"`
	AccessToken    string `json:"accessToken"`
}

// Webhook request types
type CreateWebhookRequest struct {
	Name     string            `json:"name" validate:"required,noSQLKeywords"`
	URL      string            `json:"url" validate:"required"`
	Events   []WebhookEventType `json:"events" validate:"required,min=1"`
}

type UpdateWebhookRequest struct {
	Name     *string            `json:"name,omitempty"`
	URL      *string            `json:"url,omitempty"`
	Events   []WebhookEventType `json:"events,omitempty"`
	IsActive *bool              `json:"isActive,omitempty"`
}

type RotateWebhookSecretRequest struct {
	WebhookID string `json:"webhookId" validate:"required"`
}

type RetryWebhookDeliveryRequest struct {
	DeliveryID string `json:"deliveryId" validate:"required"`
}

// Admin API request types
type AdminUpdateUserRequest struct {
	Name         *string `json:"name,omitempty"`
	Email        *string `json:"email,omitempty"`
	PhoneNumber  *string `json:"phoneNumber,omitempty"`
	EmailVerified *bool  `json:"emailVerified,omitempty"`
}

type AdminLockUserRequest struct {
	Locked bool `json:"locked"`
	Reason string `json:"reason,omitempty"`
}

type AdminResetUserPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required"`
}

// Helper to serialize slices to JSON for GORM JSON columns
func SerializeStringSlice(v []string) string {
	if v == nil {
		return "[]"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func DeserializeStringSlice(v string) []string {
	var result []string
	json.Unmarshal([]byte(v), &result)
	return result
}

func SerializeWebhookEvents(v []WebhookEventType) string {
	if v == nil {
		return "[]"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func DeserializeWebhookEvents(v string) []WebhookEventType {
	var result []WebhookEventType
	json.Unmarshal([]byte(v), &result)
	return result
}
