package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"uas/internal/helpers"

	"github.com/rs/zerolog"
)

var tenantSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9\-]{3,32}$`)

var reservedSlugs = map[string]bool{
	"api": true, "admin": true, "auth": true, "www": true, "app": true, "mail": true,
}

type TenantResolverMiddleware struct {
	log *zerolog.Logger
}

func NewTenantResolverMiddleware(log *zerolog.Logger) *TenantResolverMiddleware {
	return &TenantResolverMiddleware{log: log}
}

func (m *TenantResolverMiddleware) Resolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")

		if tenantID == "" {
			host := r.Host
			if idx := strings.Index(host, "."); idx > 0 {
				slug := host[:idx]
				if tenantSlugPattern.MatchString(slug) && !reservedSlugs[slug] {
					tenantID = slug
				}
			}
		}

		if tenantID != "" {
			r = helpers.SetDepartmentId(r, tenantID)
		}

		next.ServeHTTP(w, r)
	})
}
