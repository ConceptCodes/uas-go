package constants

const (
	// Error codes
	NotFound            = "RP-404"
	BadRequest          = "RP-400"
	Unauthorized        = "RP-401"
	Forbidden           = "RP-403"
	InternalServerError = "RP-500"

	// Endpoints
	ApiPrefix           = "/api/v1"
	HealthCheckEndpoint = ApiPrefix + "/health/alive"
	ReadinessEndpoint   = ApiPrefix + "/health/status"
	OnboardUserEndpoint = ApiPrefix + "/onboard"

	// Messages
	EntityNotFound       = "%s with id %d does not exist."
	GetEntityByIdMessage = "Found %s with id %d."
	SaveEntityError      = "Error while saving %s."
	SuccessMessage       = "You have successfully %s!"
	CreateEntityError    = "Error while creating %s."
	CreateEntityMessage  = "Created %s successfully."

	// Queries
	FindByIdQuery          = "id = ?"
	FindByApiKeyQuery      = "api_key = ?"
	FindByUserIdAndIdQuery = "user_id = ? AND id = ?"

	// Misc
	TimeFormat          = "2006-01-02 15:04:05"
	TraceIdHeader       = "x-trace-id"
	AuthorizationHeader = "Authorization"
	HealthCheckMessage  = "Performing healthcheck for service: %s"
	DBTablePrefix       = "uas_%s"
	LocalEnv            = "local"
	DevelopmentEnv      = "development"
	ProductionEnv       = "prod"
	StartMessage        = "Starting API Service on PORT=%s | ENV=%s"
	RequestIdCtxKey     = "request_id"
	ApiKeyCtxKey        = "api_key"
	UserIdCtxKey        = "user_id"

	// Errors
	HealthCheckError = "Error while performing health-check for service: %s"
)
