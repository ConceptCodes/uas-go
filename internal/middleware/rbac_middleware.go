package middleware

import (
	"net/http"
	"time"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"

	"github.com/rs/zerolog"
)

type RBACMiddleware struct {
	log       *zerolog.Logger
	jwtHelper *helpers.AuthHelper
}

func NewRBACMiddleware(log *zerolog.Logger) *RBACMiddleware {
	return &RBACMiddleware{log: log}
}

func (m *RBACMiddleware) Authorize(roles []models.Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		access_token, err := r.Cookie(constants.AccessTokenCookie)

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtHelper.ParseAccessJwtToken(access_token.Value)

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		exp_time, err := claims.GetExpirationTime()

		if err != nil {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if exp_time.Before(time.Now()) {
			m.log.Error().Msgf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		m.log.Info().Msgf("Access token is valid")
		user_role := claims["role"].(models.Role)

		for _, role := range roles {
			if role == user_role {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}
