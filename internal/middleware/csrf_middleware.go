package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"uas/config"
	"uas/internal/constants"

	"github.com/rs/zerolog"
)

type CSRFMiddleware struct {
	log *zerolog.Logger
}

func NewCSRFMiddleware(log *zerolog.Logger) *CSRFMiddleware {
	return &CSRFMiddleware{log: log}
}

func (m *CSRFMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get(constants.OriginHeader)
		referer := r.Header.Get(constants.RefererHeader)

		if origin == "" && referer == "" {
			m.log.Warn().Str("path", r.URL.Path).Str("method", r.Method).Msg("CSRF: no Origin or Referer")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		allowedOrigins := strings.Split(config.AppConfig.CORSAllowedOrigins, ",")
		validOrigin := false

		if origin != "" {
			for _, allowed := range allowedOrigins {
				allowed = strings.TrimSpace(allowed)
				if origin == allowed {
					validOrigin = true
					break
				}
			}
		}

		if !validOrigin && referer != "" {
			parsedReferer, err := url.Parse(referer)
			if err == nil {
				refererOrigin := strings.ToLower(parsedReferer.Scheme) + "://" + strings.ToLower(parsedReferer.Host)
				for _, allowed := range allowedOrigins {
					allowed = strings.TrimSpace(strings.ToLower(allowed))
					if refererOrigin == allowed {
						validOrigin = true
						break
					}
				}
			}
		}

		if !validOrigin {
			m.log.Warn().Str("origin", origin).Str("referer", referer).Str("path", r.URL.Path).Msg("CSRF: invalid origin")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
