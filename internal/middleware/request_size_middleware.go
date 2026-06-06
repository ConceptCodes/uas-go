package middleware

import (
	"errors"
	"net/http"
	"uas/config"

	"github.com/rs/zerolog"
)

type RequestSizeMiddleware struct {
	log *zerolog.Logger
}

func NewRequestSizeMiddleware(log *zerolog.Logger) *RequestSizeMiddleware {
	return &RequestSizeMiddleware{log: log}
}

func (m *RequestSizeMiddleware) Start(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxSize := int64(config.AppConfig.MaxRequestSizeMB) * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)

		if err := r.ParseForm(); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				m.log.Warn().Int64("limit", maxBytesErr.Limit).Msg("Request body exceeded size limit")
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
