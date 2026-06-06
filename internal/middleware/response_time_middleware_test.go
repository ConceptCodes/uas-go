package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"uas/internal/helpers"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func newResponseTimeMiddleware(t *testing.T) *ResponseTimeMiddleware {
	log := zerolog.New(io.Discard)
	mh := helpers.NewMetricsHelper(&log)
	return NewResponseTimeMiddleware(&log, mh)
}

func TestResponseTimeMiddleware(t *testing.T) {
	rtm := newResponseTimeMiddleware(t)

	handler := rtm.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseTimeWriter(t *testing.T) {
	rtm := newResponseTimeMiddleware(t)

	handler := rtm.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/data", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "hello", w.Body.String())
}

