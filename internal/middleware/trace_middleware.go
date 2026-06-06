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
		r = helpers.SetRequestId(r, requestId)

		authHeader := r.Header.Get(constants.AuthorizationHeader)
		authToken := strings.TrimPrefix(authHeader, "Bearer ")

		if authHeader != "" {
			tenantID, err := m.authHelper.ValidateBasicAuthToken(authToken)
			if err != nil {
				m.log.Warn().Str(constants.RequestIdCtxKey, requestId).Err(err).Msg("Invalid tenant authorization token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			r = helpers.SetDepartmentId(r, tenantID)
		}

		next.ServeHTTP(w, r)
	})
}
