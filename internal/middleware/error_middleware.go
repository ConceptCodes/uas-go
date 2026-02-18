package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ErrorMiddleware struct {
	log *zerolog.Logger
}

func NewErrorMiddleware(log *zerolog.Logger) *ErrorMiddleware {
	return &ErrorMiddleware{
		log: log,
	}
}

func (em *ErrorMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate correlation ID if not present
		traceID := r.Header.Get(constants.TraceIdHeader)
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Add trace ID to response header
		w.Header().Set(constants.TraceIdHeader, traceID)

		// Add trace ID to request context
		ctx := context.WithValue(r.Context(), constants.RequestIdCtxKey, traceID)
		r = r.WithContext(ctx)

		// Create response writer to capture status code
		rw := &errorResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Serve the request
		defer func() {
			if rec := recover(); rec != nil {
				em.logError(r, rw, models.NewInternalServerError("Internal server error", nil))
				em.sendErrorResponse(w, models.NewInternalServerError("Internal server error", nil).WithContext(traceID, traceID, r.URL.Path, r.Method))
			}
		}()

		next.ServeHTTP(rw, r)

		// Log request completion
		em.logRequest(r, rw, traceID)
	})
}

func (em *ErrorMiddleware) HandleError(w http.ResponseWriter, r *http.Request, appErr *models.AppError) {
	traceID := helpers.TraceIDFromContext(r.Context())

	// Add context to error
	appErr = appErr.WithContext(traceID, traceID, r.URL.Path, r.Method)

	// Log the error
	em.logError(r, nil, appErr)

	// Send error response
	em.sendErrorResponse(w, appErr)
}

func (em *ErrorMiddleware) logError(r *http.Request, rw *errorResponseWriter, appErr *models.AppError) {
	traceID := helpers.TraceIDFromContext(r.Context())

	statusCode := http.StatusInternalServerError
	if appErr != nil {
		statusCode = appErr.HTTPStatus
	} else if rw != nil {
		statusCode = rw.statusCode
	}

	logEvent := em.log.Error().
		Str("trace_id", traceID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Int("status_code", statusCode)
	if appErr != nil {
		logEvent = logEvent.
			Str("error_code", appErr.Code).
			Str("error_message", appErr.Message)
	}
	logEvent.Msg("Request error occurred")
}

func (em *ErrorMiddleware) logRequest(r *http.Request, rw *errorResponseWriter, traceID string) {
	em.log.Info().
		Str("trace_id", traceID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Int("status_code", rw.statusCode).
		Msg("Request completed")
}

func (em *ErrorMiddleware) sendErrorResponse(w http.ResponseWriter, appErr *models.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)

	response := appErr.ToResponse()

	// Write JSON response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		em.log.Error().Err(err).Msg("Failed to encode error response")
	}
}

// errorResponseWriter wraps http.ResponseWriter to capture status code
type errorResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *errorResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
