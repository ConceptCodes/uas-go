package constants

const (
	// Error codes
	NotFound            = "UAS-404"
	BadRequest          = "UAS-400"
	Unauthorized        = "UAS-401"
	Forbidden           = "UAS-403"
	InternalServerError = "UAS-500"

	// Endpoints
	ApiPrefix                   = "/api/v1"
	HealthCheckEndpoint         = ApiPrefix + "/health/alive"
	ReadinessEndpoint           = ApiPrefix + "/health/status"
	OnboardTenantEndpoint       = ApiPrefix + "/tenants"
	CredentialsLoginEndpoint    = ApiPrefix + "/users/credential/login"
	CredentialsRegisterEndpoint = ApiPrefix + "/users/credential/register"
	CredentialsForgotEndpoint   = ApiPrefix + "/users/credential/forgot-password"
	CredentialsResetEndpoint    = ApiPrefix + "/users/credential/reset-password/:token"

	// Messages
	EntityNotFound       = "%s with %s %s does not exist."
	GetEntityByIdMessage = "Found %s with id %d."
	SaveEntityError      = "Error while saving %s."
	SuccessMessage       = "You have successfully %s!"
	CreateEntityError    = "Error while creating %s."
	CreateEntityMessage  = "Created %s successfully."

	// Queries
	FindByIdQuery    = "id = ?"
	FindByEmailQuery = "email = ?"
	FindByToken      = "token = ?"
	FindByUserId     = "user_id = ?"

	// Misc
	TimeFormat          = "2006-01-02 15:04:05"
	TraceIdHeader       = "x-trace-id"
	AuthorizationHeader = "Authorization"
	JwtHeader           = "x-jwt-token"
	HealthCheckMessage  = "Performing health-check for service: %s"
	DBTablePrefix       = "uas_%s"
	LocalEnv            = "local"
	DevelopmentEnv      = "development"
	ProductionEnv       = "prod"
	StartMessage        = "Starting API Service on PORT=%s | ENV=%s"

	// Email
	EmailTemplatePath   = "%s/web/emails/%s.html"
	EmailFrom           = "Example <team@%s>"
	WelcomeEmailSubject = "Welcome to Example!"

	RequestIdCtxKey = "request_id"
	UserIdCtxKey    = "user_id"
	TenantIdCtxKey  = "tenant_id"

	// Errors
	HealthCheckError         = "Error while performing health-check for service: %s"
	InvalidTemplatePathError = "invalid template path: %s"
	TokenExpiredError        = "Token expired"
	TokenInvalidError        = "Token invalid"
)
