package constants

import (
	"time"
)

const (
	NotFound            = "UAS-404"
	BadRequest          = "UAS-400"
	Unauthorized        = "UAS-401"
	Forbidden           = "UAS-403"
	InternalServerError = "UAS-500"

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
	LogoutEndpoint                = ApiPrefix + "/users/logout"

	JwtSubKey = "sub"
	JwtJtiKey = "jti"
	JwtTidKey = "tid"

	RequestIdCtxKey    = "request_id"
	UserIdCtxKey       = "userId"
	DepartmentIdCtxKey = "department_id"
	RoleCtxKey         = "department_role"

	AuthorizationHeader  = "Authorization"
	JwtHeader            = "x-jwt-token"
	AccessTokenCookie    = "access-token"
	TraceIdHeader        = "x-trace-id"
	OriginHeader         = "Origin"
	RefererHeader        = "Referer"
	ForwardedProtoHeader = "X-Forwarded-Proto"

	FindByIdQuery            = "id = ?"
	FindByEmailQuery         = "email = ?"
	FindByTokenAndTypeQuery  = "token = ? AND type = ?"
	FindByUserIdQuery        = "user_id = ?"
	FindByPhoneNumberQuery   = "phone_number = ?"
	FindByIdAndUserIdQuery   = "id = ? AND userId = ?"

	EntityNotFound             = "%s with %s %s does not exist."
	CreateEntityError          = "Error while creating %s."
	OtpCodeMessage             = "Your OTP code is %s."
	HealthCheckMessage         = "Performing health-check for service: %s"
	HealthCheckError           = "Error while performing health-check for service: %s"
	TimeFormat                 = "2006-01-02 15:04:05"
	StartMessage               = "Starting API Service on PORT=%s | ENV=%s"
	DefaultRedisTtl            = 1 * time.Hour

	EmailTemplatePath        = "%s/web/emails/%s.html"
	EmailFrom                = "Example <team@%s>"
	InvalidTemplatePathError = "invalid template path: %s"

	LocalEnv       = "local"
	DevelopmentEnv = "development"
	ProductionEnv  = "production"
)
