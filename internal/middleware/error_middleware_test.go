package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"uas/internal/models"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newErrorMiddleware(t *testing.T) *ErrorMiddleware {
	log := zerolog.New(io.Discard)
	return NewErrorMiddleware(&log)
}

func TestErrorMiddlewareAddsTraceID(t *testing.T) {
	em := newErrorMiddleware(t)

	handler := em.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("x-trace-id"))
}

func TestErrorMiddlewarePreservesExistingTraceID(t *testing.T) {
	em := newErrorMiddleware(t)

	handler := em.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("x-trace-id", "existing-trace")
	handler.ServeHTTP(w, req)

	assert.Equal(t, "existing-trace", w.Header().Get("x-trace-id"))
}

func TestErrorMiddlewarePanicRecovery(t *testing.T) {
	em := newErrorMiddleware(t)

	handler := em.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp models.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
}

func TestErrorMiddlewareHandleError(t *testing.T) {
	em := newErrorMiddleware(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/users", nil)

	appErr := models.NewBadRequestError("invalid email")
	em.HandleError(w, req, appErr)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp models.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "BAD_REQUEST", resp.Code)
	assert.Equal(t, "invalid email", resp.Message)
}
