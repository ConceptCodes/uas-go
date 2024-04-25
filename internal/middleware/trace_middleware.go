package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"uas/internal/constants"
	"uas/internal/helpers"
)

type TraceRequestMiddleware struct {
	log        *zerolog.Logger
	authHelper *helpers.AuthHelper
}

func NewTraceRequestMiddleware(log *zerolog.Logger, authHelper *helpers.AuthHelper) *TraceRequestMiddleware {
	return &TraceRequestMiddleware{log: log, authHelper: authHelper}
}

func (m *TraceRequestMiddleware) Start(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestId := r.Header.Get(constants.TraceIdHeader)

		if requestId == "" {
			requestId = uuid.New().String()
		}

		w.Header().Add(constants.TraceIdHeader, requestId)

		authToken := r.Header.Get(constants.AuthorizationHeader)

		if authToken != "" {
			authToken = strings.Replace(authToken, "Bearer ", "", 1)

			tenantId, err := m.authHelper.ValidateBasicAuthToken(authToken)

			if err != nil {
				m.log.Error().Str(constants.RequestIdCtxKey, requestId).Msgf("Error: %s", err)
			}

			r = helpers.SetDepartmentId(r, tenantId)
		}

		r = helpers.SetRequestId(r, requestId)

		next.ServeHTTP(w, r)
	})
}
