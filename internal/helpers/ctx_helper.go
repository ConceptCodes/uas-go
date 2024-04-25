package helpers

import (
	"context"
	"net/http"
	"uas/internal/constants"
	"uas/internal/models"
)

type ContextKey string

const (
	RequestIDKey ContextKey = constants.RequestIdCtxKey
	UserId       ContextKey = constants.UserIdCtxKey
	DepartmentId ContextKey = constants.DepartmentIdCtxKey
	Role         ContextKey = constants.RoleCtxKey
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

func SetDepartmentId(r *http.Request, departmentId string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, DepartmentId, departmentId)
	return r.WithContext(ctx)
}

func GetDepartmentId(r *http.Request) string {
	tenantId := r.Context().Value(DepartmentId)
	if tenantId == nil {
		return ""
	}
	return tenantId.(string)
}

func SetRole(r *http.Request, role models.Role) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, Role, role)
	return r.WithContext(ctx)
}

func GetRole(r *http.Request) models.Role {
	role := r.Context().Value(Role)
	if role == nil {
		return ""
	}
	return role.(models.Role)
}
