package middleware

import (
	"net/http"
	"time"
	"uas/internal/helpers"

	"github.com/rs/zerolog"
)

type ResponseTimeMiddleware struct {
	log           *zerolog.Logger
	metricsHelper *helpers.MetricsHelper
}

func NewResponseTimeMiddleware(log *zerolog.Logger, metricsHelper *helpers.MetricsHelper) *ResponseTimeMiddleware {
	return &ResponseTimeMiddleware{
		log:           log,
		metricsHelper: metricsHelper,
	}
}

func (rtm *ResponseTimeMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer to capture status code
		rw := &responseTimeWriter{
			ResponseWriter: w,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics
		rtm.metricsHelper.RecordHTTPRequest(
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration,
			rw.responseSize,
		)

		// Log slow requests
		if duration > 5*time.Second {
			traceID := helpers.TraceIDFromContext(r.Context())
			rtm.log.Warn().
				Str("trace_id", traceID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Dur("duration", duration).
				Msg("Slow request detected")
		}
	})
}

// responseTimeWriter wraps http.ResponseWriter to capture status code and response size
type responseTimeWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func (rw *responseTimeWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseTimeWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.responseSize += int64(n)
	return n, err
}
