package middleware

import (
	"fmt"
	"net/http"
	"time"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog"
)

type RBACMiddleware struct {
	log                *zerolog.Logger
	authHelper         *helpers.AuthHelper
	departmentRoleRepo repository.DepartmentRoleRepository
}

func NewRBACMiddleware(log *zerolog.Logger, departmentRoleRepo repository.DepartmentRoleRepository, authHelper *helpers.AuthHelper) *RBACMiddleware {
	return &RBACMiddleware{log: log, departmentRoleRepo: departmentRoleRepo, authHelper: authHelper}
}

func (m *RBACMiddleware) Authorize(roles []models.Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken, err := r.Cookie(constants.AccessTokenCookie)

		if err != nil {
			m.log.Warn().Err(err).Msg("Missing access token cookie")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rawToken, err := m.authHelper.DecodeAccessCookie(accessToken.Value)
		if err != nil {
			m.log.Warn().Err(err).Msg("Failed to decode access token cookie")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.authHelper.ParseAccessJwtToken(rawToken)

		if err != nil {
			m.log.Warn().Err(err).Msg("Failed to parse access token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expTime, err := claims.GetExpirationTime()

		if err != nil {
			m.log.Warn().Err(err).Msg("Missing token expiration")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if expTime == nil || expTime.Before(time.Now()) {
			m.log.Warn().Msg("Access token expired")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userId, err := getStringClaim(claims, constants.JwtSubKey)
		if err != nil {
			m.log.Warn().Err(err).Msg("Invalid sub claim")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		departmentId, err := getStringClaim(claims, constants.JwtTidKey)
		if err != nil {
			m.log.Warn().Err(err).Msg("Invalid tid claim")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Assert JWT tid matches context tenant
		contextTenant := helpers.GetDepartmentId(r)
		if contextTenant != "" && departmentId != contextTenant {
			m.log.Warn().
				Str("jwt_tid", departmentId).
				Str("context_tid", contextTenant).
				Msg("JWT tid does not match context tenant — possible cross-tenant access")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		departmentRole, err := m.departmentRoleRepo.FindById(departmentId, userId)

		if err != nil {
			m.log.Warn().Err(err).Msg("Failed to resolve role binding")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r = helpers.SetUserId(r, userId)
		r = helpers.SetRole(r, departmentRole.Role)

		for _, role := range roles {
			if role == departmentRole.Role {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func getStringClaim(claims map[string]interface{}, key string) (string, error) {
	value, ok := claims[key]
	if !ok {
		return "", fmt.Errorf("missing claim %s", key)
	}
	s, ok := value.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("invalid claim %s", key)
	}
	return s, nil
}
