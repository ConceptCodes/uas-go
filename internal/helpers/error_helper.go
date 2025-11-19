package helpers

import (
	"context"
	"net/http"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/rs/zerolog"
)

type ErrorHandler struct {
	log *zerolog.Logger
}

func NewErrorHandler(log *zerolog.Logger) *ErrorHandler {
	return &ErrorHandler{
		log: log,
	}
}

// HandleError logs and handles application errors
func (eh *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err *models.AppError) {
	traceID := r.Context().Value(constants.RequestIdCtxKey).(string)

	// Add context to error
	err = err.WithContext(traceID, traceID, r.URL.Path, r.Method)

	// Log the error
	eh.log.Error().
		Str("trace_id", traceID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("error_code", err.Code).
		Str("error_message", err.Message).
		Err(err.Cause).
		Msg("Request error occurred")

	// Send error response
	eh.sendErrorResponse(w, err)
}

// HandlePanic recovers from panics and logs them
func (eh *ErrorHandler) HandlePanic(w http.ResponseWriter, r *http.Request) {
	if rec := recover(); rec != nil {
		traceID := r.Context().Value(constants.RequestIdCtxKey).(string)

		eh.log.Error().
			Str("trace_id", traceID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Interface("panic", rec).
			Msg("Panic occurred in request handler")

		appErr := models.NewInternalServerError("Internal server error", nil)
		appErr = appErr.WithContext(traceID, traceID, r.URL.Path, r.Method)

		eh.sendErrorResponse(w, appErr)
	}
}

// ValidateRequest validates common request fields
func (eh *ErrorHandler) ValidateRequest(w http.ResponseWriter, r *http.Request, data interface{}) bool {
	// Validate content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && r.Method != http.MethodGet && r.Method != http.MethodDelete {
		appErr := models.NewBadRequestError("Content-Type must be application/json")
		eh.HandleError(w, r, appErr)
		return false
	}

	// Validate request size
	if r.ContentLength > 0 {
		maxSize := int64(10 * 1024 * 1024) // 10MB
		if r.ContentLength > maxSize {
			appErr := models.NewBadRequestError("Request size exceeds maximum allowed size")
			eh.HandleError(w, r, appErr)
			return false
		}
	}

	return true
}

// LogRequest logs successful requests
func (eh *ErrorHandler) LogRequest(r *http.Request, statusCode int, duration int64) {
	traceID := r.Context().Value(constants.RequestIdCtxKey).(string)

	level := eh.log.Info()
	if statusCode >= 400 {
		level = eh.log.Warn()
	}
	if statusCode >= 500 {
		level = eh.log.Error()
	}

	level.
		Str("trace_id", traceID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Int("status_code", statusCode).
		Int64("duration_ms", duration).
		Msg("Request completed")
}

// LogDependency logs dependency health checks
func (eh *ErrorHandler) LogDependency(dependency, status string, err error) {
	if err != nil {
		eh.log.Error().
			Str("dependency", dependency).
			Str("status", status).
			Err(err).
			Msg("Dependency health check failed")
	} else {
		eh.log.Info().
			Str("dependency", dependency).
			Str("status", status).
			Msg("Dependency health check completed")
	}
}

// sendErrorResponse sends JSON error response
func (eh *ErrorHandler) sendErrorResponse(w http.ResponseWriter, appErr *models.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)

	response := appErr.ToResponse()

	// Use json.NewEncoder from encoding/json package
	if err := json.NewEncoder(w).Encode(response); err != nil {
		eh.log.Error().Err(err).Msg("Failed to encode error response")
	}
}

// WrapError wraps any error into AppError with context
func (eh *ErrorHandler) WrapError(err error, message string) *models.AppError {
	if err == nil {
		return nil
	}

	return models.NewInternalServerError(message, err)
}

// IsClientError checks if error is a client error (4xx)
func (eh *ErrorHandler) IsClientError(err *models.AppError) bool {
	return err.HTTPStatus >= 400 && err.HTTPStatus < 500
}

// IsServerError checks if error is a server error (5xx)
func (eh *ErrorHandler) IsServerError(err *models.AppError) bool {
	return err.HTTPStatus >= 500
}

// GetErrorCode extracts error code from error
func (eh *ErrorHandler) GetErrorCode(err error) string {
	if appErr, ok := err.(*models.AppError); ok {
		return appErr.Code
	}
	return constants.InternalServerError
}

// ContextWithTraceID adds trace ID to context
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, constants.RequestIdCtxKey, traceID)
}

// TraceIDFromContext extracts trace ID from context
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(constants.RequestIdCtxKey).(string); ok {
		return traceID
	}
	return ""
}
