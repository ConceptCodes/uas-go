package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
)

func TraceRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestId := r.Header.Get(constants.TraceIdHeader)

		if requestId == "" {
			requestId = uuid.New().String()
		}

		model := &models.Request{
			Id: requestId,
		}

		w.Header().Add(constants.TraceIdHeader, requestId)

		authToken := r.Header.Get("Authorization")

		if authToken != "" {
			authToken = strings.Replace(authToken, "Bearer ", "", 1)

			tenantId, err := helpers.ValidateBasicAuthToken(authToken)

			if err != nil {
				log.Error().Str("request_id", requestId).Msgf("Error: %s", err)
			}

			model.TenantId = tenantId
		}

		ctx := context.WithValue(r.Context(), "ctx", model)

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
