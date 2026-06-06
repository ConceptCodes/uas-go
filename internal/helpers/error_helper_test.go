package helpers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newErrorHandler(t *testing.T) *ErrorHandler {
	log := zerolog.New(io.Discard)
	return NewErrorHandler(&log)
}

func TestTraceIDFromContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	assert.Equal(t, "", TraceIDFromContext(req.Context()))

	req = httptest.NewRequest("GET", "/", nil)
	ctx := ContextWithTraceID(req.Context(), "trace-abc")
	req = req.WithContext(ctx)
	assert.Equal(t, "trace-abc", TraceIDFromContext(req.Context()))
}

func TestErrorHandlerIsClientError(t *testing.T) {
	h := newErrorHandler(t)
	assert.True(t, h.IsClientError(models.NewBadRequestError("bad")))
	assert.True(t, h.IsClientError(models.NewNotFoundError("user", "id")))
	assert.False(t, h.IsClientError(models.NewInternalServerError("err", nil)))
}

func TestErrorHandlerIsServerError(t *testing.T) {
	h := newErrorHandler(t)
	assert.True(t, h.IsServerError(models.NewInternalServerError("err", nil)))
	assert.False(t, h.IsServerError(models.NewBadRequestError("bad")))
}

func TestErrorHandlerGetErrorCode(t *testing.T) {
	h := newErrorHandler(t)

	code := h.GetErrorCode(models.NewBadRequestError("bad"))
	assert.Equal(t, "BAD_REQUEST", code)

	code = h.GetErrorCode(models.NewNotFoundError("user", "id"))
	assert.Equal(t, "NOT_FOUND", code)

	code = h.GetErrorCode(errors.New("plain error"))
	assert.Equal(t, constants.InternalServerError, code)
}

func TestErrorHandlerWrapError(t *testing.T) {
	h := newErrorHandler(t)

	appErr := h.WrapError(errors.New("db connection failed"), "could not query users")
	require.NotNil(t, appErr)
	assert.Equal(t, "INTERNAL_ERROR", appErr.Code)
	assert.Equal(t, http.StatusInternalServerError, appErr.HTTPStatus)
	assert.True(t, strings.Contains(appErr.Error(), "could not query users"))

	appErr = h.WrapError(nil, "no error")
	assert.Nil(t, appErr)
}

func TestErrorHandlerValidateRequestBadContentType(t *testing.T) {
	h := newErrorHandler(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/data", nil)
	req.Header.Set("Content-Type", "text/plain")

	valid := h.ValidateRequest(w, req, nil)
	assert.False(t, valid)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestErrorHandlerValidateRequestGET(t *testing.T) {
	h := newErrorHandler(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)

	valid := h.ValidateRequest(w, req, nil)
	assert.True(t, valid)
}

func TestErrorHandlerLogDependency(t *testing.T) {
	h := newErrorHandler(t)

	h.LogDependency("mysql", "up", nil)
	h.LogDependency("redis", "down", errors.New("connection refused"))
}

func TestAppErrorAppendContext(t *testing.T) {
	err := models.NewBadRequestError("bad input")
	err.WithContext("trace-1", "req-1", "/api/test", "POST")

	assert.Equal(t, "trace-1", err.TraceID)
	assert.Equal(t, "req-1", err.RequestID)
	assert.Equal(t, "/api/test", err.Path)
	assert.Equal(t, "POST", err.Method)
}

func TestAppErrorToResponse(t *testing.T) {
	err := models.NewNotFoundError("user", "email=test@test.com")
	err.WithContext("trace-1", "req-1", "/api/users", "GET")

	resp := err.ToResponse()
	assert.Equal(t, "NOT_FOUND", resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Code)
	assert.Equal(t, "trace-1", resp.TraceID)
	assert.Equal(t, "req-1", resp.RequestID)
	assert.Equal(t, "/api/users", resp.Path)
	assert.Equal(t, "GET", resp.Method)
	assert.False(t, resp.Timestamp.IsZero())
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := models.NewInternalServerError("wrapped", cause)
	assert.Equal(t, cause, err.Unwrap())
}

func TestNewValidationError(t *testing.T) {
	validationErrs := []models.ValidationError{
		{Field: "email", Message: "invalid email"},
	}
	err := models.NewValidationError(validationErrs)
	assert.Equal(t, "VALIDATION_ERROR", err.Code)
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
	assert.NotNil(t, err.Details)
}

func TestNewRateLimitError(t *testing.T) {
	err := models.NewRateLimitError("too fast")
	assert.Equal(t, "RATE_LIMITED", err.Code)
	assert.Equal(t, http.StatusTooManyRequests, err.HTTPStatus)
}

func TestNewServiceUnavailableError(t *testing.T) {
	err := models.NewServiceUnavailableError("down for maintenance")
	assert.Equal(t, "SERVICE_UNAVAILABLE", err.Code)
	assert.Equal(t, http.StatusServiceUnavailable, err.HTTPStatus)
}

func TestNewConflictError(t *testing.T) {
	err := models.NewConflictError("already exists")
	assert.Equal(t, "CONFLICT", err.Code)
	assert.Equal(t, http.StatusConflict, err.HTTPStatus)
}
