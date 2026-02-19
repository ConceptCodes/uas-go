package constants

import (
	"time"
)

const (
	// Error codes
	NotFound            = "UAS-404"
	BadRequest          = "UAS-400"
	Unauthorized        = "UAS-401"
	Forbidden           = "UAS-403"
	InternalServerError = "UAS-500"
	ErrInternalServer   = "UAS-500"
	ErrForbidden        = "UAS-403"

	// Enhanced error codes
	ValidationError    = "UAS-VALIDATION"
	Conflict           = "UAS-CONFLICT"
	RateLimited        = "UAS-RATE_LIMIT"
	ServiceUnavailable = "UAS-SERVICE_UNAVAILABLE"
	AccountLocked      = "UAS-ACCOUNT_LOCKED"
	EmailNotVerified   = "UAS-EMAIL_NOT_VERIFIED"
	InvalidCredentials = "UAS-INVALID_CREDENTIALS"
	TokenExpired       = "UAS-TOKEN_EXPIRED"
	TokenInvalid       = "UAS-TOKEN_INVALID"
	DatabaseError      = "UAS-DATABASE_ERROR"
	NetworkError       = "UAS-NETWORK_ERROR"
	EncryptionError    = "UAS-ENCRYPTION_ERROR"

	// Error messages
	MsgNotFound            = "Resource not found"
	MsgBadRequest          = "Bad request"
	MsgUnauthorized        = "Unauthorized access"
	MsgForbidden           = "Access forbidden"
	MsgInternalServerError = "Internal server error"
	MsgValidationError     = "Request validation failed"
	MsgConflict            = "Resource conflict"
	MsgRateLimited         = "Rate limit exceeded"
	MsgServiceUnavailable  = "Service temporarily unavailable"
	MsgAccountLocked       = "Account temporarily locked"
	MsgEmailNotVerified    = "Email address not verified"
	MsgInvalidCredentials  = "Invalid credentials provided"
	MsgTokenExpired        = "Authentication token has expired"
	MsgTokenInvalid        = "Invalid authentication token"
	MsgDatabaseError       = "Database operation failed"
	MsgNetworkError        = "Network operation failed"
	MsgEncryptionError     = "Data encryption/decryption failed"

	// Endpoints
	ApiPrefix                     = "/api/v1"
	HealthCheckEndpoint           = ApiPrefix + "/health/alive"
	ReadinessEndpoint             = ApiPrefix + "/health/status"
	OnboardTenantEndpoint         = ApiPrefix + "/tenants"
	DeleteTenantEndpoint          = ApiPrefix + "/tenants/{id}"
	CredentialsLoginEndpoint      = ApiPrefix + "/users/credential/login"
	CredentialsRegisterEndpoint   = ApiPrefix + "/users/credential/register"
	CredentialsForgotEndpoint     = ApiPrefix + "/users/credential/forgot-password"
	CredentialsResetEndpoint      = ApiPrefix + "/users/credential/reset-password"
	CredentialsVerifyEndpoint     = ApiPrefix + "/users/credential/verify-email"
	CredentialsLoginEndpointV2    = ApiPrefix + "/users/credentials/login"
	CredentialsRegisterEndpointV2 = ApiPrefix + "/users/credentials/register"
	CredentialsForgotEndpointV2   = ApiPrefix + "/users/credentials/forgot-password"
	CredentialsResetEndpointV2    = ApiPrefix + "/users/credentials/reset-password"
	CredentialsVerifyEndpointV2   = ApiPrefix + "/users/credentials/verify-email"
	OtpSendEndpoint               = ApiPrefix + "/users/otp/send"
	OtpVerifyEndpoint             = ApiPrefix + "/users/otp/verify"
	MagicLinkSendEndpoint         = ApiPrefix + "/users/magic-link/send"
	MagicLinkVerifyEndpoint       = ApiPrefix + "/users/magic-link/verify"
	RefreshTokenEndpoint          = ApiPrefix + "/users/refresh-token"

	// Context keys
	RequestIdCtxKey    = "request_id"
	UserIdCtxKey       = "userId"
	DepartmentIdCtxKey = "department_id"
	RoleCtxKey         = "department_role"

	// Headers
	AuthorizationHeader = "Authorization"
	JwtHeader           = "x-jwt-token"
	AccessTokenCookie   = "access-token"

	// Queries
	FindByIdQuery           = "id = ?"
	FindByEmailQuery        = "email = ?"
	FindByTokenAndTypeQuery = "token = ? AND type = ?"
	FindByUserIdQuery       = "userId = ?"
	FindByPhoneNumberQuery  = "phone_number = ?"
	FindByIdAndUserIdQuery  = "id = ? AND userId = ?"

	// Messages
	EntityNotFound             = "%s with %s %s does not exist."
	GetEntityByIdMessage       = "Found %s with id %d."
	SaveEntityError            = "Error while saving %s."
	CreateEntityError          = "Error while creating %s."
	CreateEntityMessage        = "Created %s successfully."
	OtpCodeMessage             = "Your OTP code is %s."
	InternalServerErrorMessage = "Internal server error."
	HealthCheckMessage         = "Performing health-check for service: %s"
	HealthCheckError           = "Error while performing health-check for service: %s"
	TimeFormat                 = "2006-01-02 15:04:05"
	TraceIdHeader              = "x-trace-id"
	StartMessage               = "Starting API Service on PORT=%s | ENV=%s"
	DefaultRedisTtl            = 1 * time.Hour

	// Email
	EmailTemplatePath        = "%s/web/emails/%s.html"
	EmailFrom                = "Example <team@%s>"
	WelcomeEmailSubject      = "Welcome to Example!"
	InvalidTemplatePathError = "invalid template path: %s"

	// Misc
	LocalEnv       = "local"
	DevelopmentEnv = "development"
	ProductionEnv  = "prod"
)
