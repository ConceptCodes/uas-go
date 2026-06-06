package models

import (
	"encoding/json"
	"net/http"
	"time"
)

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResponse struct {
	Error     string      `json:"error"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	TraceID   string      `json:"trace_id,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Path      string      `json:"path,omitempty"`
	Method    string      `json:"method,omitempty"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Details    interface{}
	Cause      error
	Timestamp  time.Time
	TraceID    string
	RequestID  string
	Path       string
	Method     string
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// Error constructors
func NewNotFoundError(entity, identifier string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    entity + " with " + identifier + " not found",
		HTTPStatus: http.StatusNotFound,
		Timestamp:  time.Now(),
	}
}

func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:       "BAD_REQUEST",
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Timestamp:  time.Now(),
	}
}

func NewValidationError(errors []ValidationError) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    "Request validation failed",
		HTTPStatus: http.StatusBadRequest,
		Details:    errors,
		Timestamp:  time.Now(),
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
		Timestamp:  time.Now(),
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		HTTPStatus: http.StatusForbidden,
		Timestamp:  time.Now(),
	}
}

func NewInternalServerError(message string, cause error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
		Timestamp:  time.Now(),
	}
}

func NewRateLimitError(message string) *AppError {
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
		Timestamp:  time.Now(),
	}
}

func NewServiceUnavailableError(message string) *AppError {
	return &AppError{
		Code:       "SERVICE_UNAVAILABLE",
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
		Timestamp:  time.Now(),
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    message,
		HTTPStatus: http.StatusConflict,
		Timestamp:  time.Now(),
	}
}

// WithContext adds request context to an error
func (e *AppError) WithContext(traceID, requestID, path, method string) *AppError {
	e.TraceID = traceID
	e.RequestID = requestID
	e.Path = path
	e.Method = method
	return e
}

// ToResponse converts AppError to ErrorResponse
func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error:     e.Code,
		Code:      e.Code,
		Message:   e.Message,
		Details:   e.Details,
		Timestamp: e.Timestamp,
		TraceID:   e.TraceID,
		RequestID: e.RequestID,
		Path:      e.Path,
		Method:    e.Method,
	}
}

type OnboardDepartmentResponse struct {
	DepartmentID   string `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
}

type HealthCheckResponse struct {
	Service string `json:"service"`
	Status  bool   `json:"status"`
}

type RegisterUserResponse struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// Profiling response types
type SystemMetricsResponse struct {
	Timestamp time.Time      `json:"timestamp"`
	Memory    MemoryMetrics  `json:"memory"`
	Runtime   RuntimeMetrics `json:"runtime"`
	GCStats   GCStatsInfo    `json:"gc_stats"`
}

type MemoryMetrics struct {
	Alloc      uint64       `json:"alloc"`
	TotalAlloc uint64       `json:"total_alloc"`
	Sys        uint64       `json:"sys"`
	Lookups    uint64       `json:"lookups"`
	Mallocs    uint64       `json:"mallocs"`
	Frees      uint64       `json:"frees"`
	Heap       HeapMetrics  `json:"heap"`
	Stack      StackMetrics `json:"stack"`
	GC         GCMetrics    `json:"gc"`
}

type HeapMetrics struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	Idle       uint64 `json:"idle"`
	Inuse      uint64 `json:"inuse"`
	Released   uint64 `json:"released"`
	Objects    uint64 `json:"objects"`
}

type StackMetrics struct {
	Inuse uint64 `json:"inuse"`
	Sys   uint64 `json:"sys"`
}

type GCMetrics struct {
	NumGC         uint32  `json:"num_gc"`
	NumForcedGC   uint32  `json:"num_forced_gc"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
	EnableGC      bool    `json:"enable_gc"`
	DebugGC       bool    `json:"debug_gc"`
}

type RuntimeMetrics struct {
	Goroutines int    `json:"goroutines"`
	NumCPU     int    `json:"num_cpu"`
	CgoCalls   uint64 `json:"cgo_calls"`
	GoVersion  string `json:"go_version"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
}

type GCStatsInfo struct {
	NumGC          uint32          `json:"num_gc"`
	TotalPause     time.Duration   `json:"total_pause"`
	LastGC         time.Time       `json:"last_gc"`
	Pause          []time.Duration `json:"pause"`
	PauseEnd       []time.Time     `json:"pause_end"`
	PauseQuantiles []time.Duration `json:"pause_quantiles"`
}

type DatabaseMetricsResponse struct {
	Timestamp time.Time     `json:"timestamp"`
	MySQL     MySQLMetrics  `json:"mysql"`
	Redis     *RedisMetrics `json:"redis,omitempty"`
}

type MySQLMetrics struct {
	OpenConnections   int           `json:"open_connections"`
	InUse             int           `json:"in_use"`
	Idle              int           `json:"idle"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

type RedisMetrics struct {
	Info string `json:"info"`
}

type ApplicationMetricsResponse struct {
	Timestamp     time.Time            `json:"timestamp"`
	Application   ApplicationInfo      `json:"application"`
	Configuration ConfigurationMetrics `json:"configuration"`
}

type ApplicationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Env     string `json:"env"`
	Debug   bool   `json:"debug"`
}

type ConfigurationMetrics struct {
	AccessJwtExpire     int  `json:"access_jwt_expire"`
	RefreshJwtExpire    int  `json:"refresh_jwt_expire"`
	MaxFailedAttempts   int  `json:"max_failed_attempts"`
	AccountLockMinutes  int  `json:"account_lock_minutes"`
	MaxRequestSizeMB    int  `json:"max_request_size_mb"`
	RateLimitCapacity   int  `json:"rate_limit_capacity"`
	TimeUnitInSeconds   int  `json:"time_unit_seconds"`
	EnableMetrics       bool `json:"enable_metrics"`
	EnableProfiling     bool `json:"enable_profiling"`
}

type ProfilingIndexResponse struct {
	Timestamp time.Time           `json:"timestamp"`
	Endpoints []ProfilingEndpoint `json:"endpoints"`
}

type ProfilingEndpoint struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Description string `json:"description"`
}

// MFA Response types
type MfaEnrollResponse struct {
	FactorID    string   `json:"factorId"`
	FactorType  string   `json:"factorType"`
	Secret      string   `json:"secret,omitempty"`
	QRCodeURI   string   `json:"qrCodeUri,omitempty"`
	BackupCodes []string `json:"backupCodes,omitempty"`
}

type MfaFactorResponse struct {
	ID         string     `json:"id"`
	FactorType string     `json:"factorType"`
	Name       string     `json:"name"`
	IsPrimary  bool       `json:"isPrimary"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type MfaChallengeResponse struct {
	MFARequired bool   `json:"mfaRequired"`
	MFAToken    string `json:"mfaToken,omitempty"`
	Factors     []MfaFactorResponse `json:"factors,omitempty"`
}

type LoginMFARequiredResponse struct {
	Message    string             `json:"message"`
	MFARequired bool              `json:"mfaRequired"`
	MFAToken   string             `json:"mfaToken"`
	Factors    []MfaFactorResponse `json:"factors"`
}

// SSO Response types
type IdentityProviderResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	ProviderType    string   `json:"providerType"`
	ClientID        string   `json:"clientId"`
	IssuerURL       string   `json:"issuerUrl,omitempty"`
	AuthorizationURL string  `json:"authorizationUrl,omitempty"`
	RedirectURLs    []string `json:"redirectUrls"`
	Scopes          []string `json:"scopes"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
}

type SsoLoginResponse struct {
	AuthURL string `json:"authUrl"`
	State   string `json:"state"`
}

type UserIdentityResponse struct {
	ID             string     `json:"id"`
	ProviderID     string     `json:"providerId"`
	ProviderName   string     `json:"providerName"`
	ProviderType   string     `json:"providerType"`
	ProviderUserID string     `json:"providerUserId"`
	ProviderEmail  string     `json:"providerEmail,omitempty"`
	LastLoginAt    *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// Webhook Response types
type WebhookEndpointResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Events    []WebhookEventType `json:"events"`
	IsActive  bool              `json:"isActive"`
	CreatedAt time.Time         `json:"createdAt"`
}

type WebhookDeliveryResponse struct {
	ID           string               `json:"id"`
	EndpointID   string               `json:"endpointId"`
	Event        WebhookEventType     `json:"event"`
	ResponseCode int                  `json:"responseCode"`
	Status       WebhookDeliveryStatus `json:"status"`
	Attempt      int                  `json:"attempt"`
	MaxAttempts  int                  `json:"maxAttempts"`
	NextRetryAt  *time.Time           `json:"nextRetryAt,omitempty"`
	CreatedAt    time.Time            `json:"createdAt"`
}

type WebhookSecretResponse struct {
	Secret string `json:"secret"`
}

// Helper functions for error handling
func NewAppError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Timestamp:  time.Now(),
	}
}

func WriteErrorResponse(w http.ResponseWriter, appErr *AppError) {
	response := appErr.ToResponse()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)
	json.NewEncoder(w).Encode(response)
}
