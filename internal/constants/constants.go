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

	// Misc
	TimeFormat          = "2006-01-02 15:04:05"
	TraceIdHeader       = "x-trace-id"
	AuthorizationHeader = "Authorization"
	JwtHeader           = "x-jwt-token"
	HealthCheckMessage  = "Performing healthcheck for service: %s"
	DBTablePrefix       = "uas_%s"
	LocalEnv            = "local"
	DevelopmentEnv      = "development"
	ProductionEnv       = "prod"
	StartMessage        = "Starting API Service on PORT=%s | ENV=%s"

	RequestIdCtxKey = "request_id"
	UserIdCtxKey    = "user_id"
	TenantIdCtxKey  = "tenant_id"

	// Errors
	HealthCheckError = "Error while performing health-check for service: %s"
)
