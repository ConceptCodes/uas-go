package helpers

import (
	"context"
	"net/http"
	"uas/internal/constants"
)

type ContextKey string

const (
	RequestIDKey ContextKey = constants.RequestIdCtxKey
	UserId       ContextKey = constants.UserIdCtxKey
	TenantId     ContextKey = constants.TenantIdCtxKey
)

func SetRequestId(r *http.Request, requestID string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	return r.WithContext(ctx)
}

func GetRequestId(r *http.Request) string {
	return r.Context().Value(RequestIDKey).(string)
}

func SetUserId(r *http.Request, userId string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, UserId, userId)
	return r.WithContext(ctx)
}

func GetUserId(r *http.Request) string {
	userId := r.Context().Value(UserId)
	if userId == nil {
		return ""
	}
	return userId.(string)
}

func SetTenantId(r *http.Request, tenantId string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, TenantId, tenantId)
	return r.WithContext(ctx)
}

func GetTenantId(r *http.Request) string {
	tenantId := r.Context().Value(TenantId)
	if tenantId == nil {
		return ""
	}
	return tenantId.(string)
}
