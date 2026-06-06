package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uas/config"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupRequestSizeTest(t *testing.T) (*RequestSizeMiddleware, func()) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		MaxRequestSizeMB: 1,
	}
	log := zerolog.New(io.Discard)
	m := NewRequestSizeMiddleware(&log)
	return m, func() { config.AppConfig = orig }
}

func TestRequestSizeWithinLimit(t *testing.T) {
	m, cleanup := setupRequestSizeTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	body := strings.NewReader("small body")
	req := httptest.NewRequest("POST", "/api/test", body)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestSizeExceedsLimit(t *testing.T) {
	m, cleanup := setupRequestSizeTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	largeBody := strings.NewReader(string(make([]byte, 2*1024*1024)))
	req := httptest.NewRequest("POST", "/api/test", largeBody)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}
