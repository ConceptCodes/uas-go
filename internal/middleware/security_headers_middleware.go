package middleware

import (
	"net/http"
	"strings"
	"uas/config"
	"uas/internal/constants"

	"github.com/rs/zerolog"
)

type SecurityHeadersMiddleware struct {
	log *zerolog.Logger
}

func NewSecurityHeadersMiddleware(log *zerolog.Logger) *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{log: log}
}

func (m *SecurityHeadersMiddleware) Start(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")

		if isDeprecatedPath(r.URL.Path) {
			w.Header().Set("Deprecation", "true")
			w.Header().Set("Sunset", "Sat, 31 Dec 2026 23:59:59 GMT")
			w.Header().Set("Link", "</api/v1/users/credentials/login>; rel=\"successor-version\"")
		}
		if config.AppConfig.Env == constants.ProductionEnv && (r.TLS != nil || r.Header.Get(constants.ForwardedProtoHeader) == "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=()")

		if strings.HasPrefix(r.URL.Path, "/api") {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		origin := r.Header.Get(constants.OriginHeader)
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-trace-id, x-jwt-token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			if origin != "" && !isAllowedOrigin(origin) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isDeprecatedPath(path string) bool {
	return path == "/api/v1/users/credential/register" ||
		path == "/api/v1/users/credential/login" ||
		path == "/api/v1/users/credential/forgot-password" ||
		path == "/api/v1/users/credential/reset-password" ||
		path == "/api/v1/users/credential/verify-email"
}

func isAllowedOrigin(origin string) bool {
	allowedOrigins := strings.Split(config.AppConfig.CORSAllowedOrigins, ",")
	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
