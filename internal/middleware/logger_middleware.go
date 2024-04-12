package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type LoggerMiddleware struct {
	log zerolog.Logger
}

type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func NewLoggerMiddleware(log zerolog.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{log: log}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (lrw *responseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (m *LoggerMiddleware) Start(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rw := &responseWriter{ResponseWriter: w, body: &bytes.Buffer{}}

		start := time.Now()

		m.log.
			Info().
			Str("method", r.Method).
			Str("url", r.URL.RequestURI()).
			Str("user_agent", r.UserAgent()).
			Dur("elapsed_ms", time.Since(start)).
			Int("status_code", rw.statusCode).
			Msgf("Incoming %s request", r.Method)

		next.ServeHTTP(w, r)
	})
}
