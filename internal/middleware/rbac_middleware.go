package middleware

import (
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
	jwtHelper          *helpers.AuthHelper
	departmentRoleRepo repository.DepartmentRoleRepository
}

func NewRBACMiddleware(log *zerolog.Logger, departmentRoleRepo repository.DepartmentRoleRepository) *RBACMiddleware {
	return &RBACMiddleware{log: log, departmentRoleRepo: departmentRoleRepo}
}

func (m *RBACMiddleware) Authorize(roles []models.Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken, err := r.Cookie(constants.AccessTokenCookie)

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtHelper.ParseAccessJwtToken(accessToken.Value)

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expTime, err := claims.GetExpirationTime()

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if expTime.Before(time.Now()) {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		m.log.Info().Msgf("Access token is valid")

		userId := claims["userId"].(string)
		departmentId := claims["departmentId"].(string)

		if userId == "" || departmentId == "" {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		departmentRole, err := m.departmentRoleRepo.FindById(departmentId, userId)

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
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
