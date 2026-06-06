package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"uas/config"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupCSRFTest(t *testing.T) (*CSRFMiddleware, func()) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		CORSAllowedOrigins: "http://localhost:3000,http://example.com",
	}
	log := zerolog.New(io.Discard)
	m := NewCSRFMiddleware(&log)
	return m, func() { config.AppConfig = orig }
}

func TestCSRFGetRequestsBypassCheck(t *testing.T) {
	m, cleanup := setupCSRFTest(t)
	defer cleanup()

	handler := m.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	for _, method := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/test", nil)
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "expected %s to bypass CSRF", method)
	}
}

func TestCSRFMissingOriginAndReferer(t *testing.T) {
	m, cleanup := setupCSRFTest(t)
	defer cleanup()

	handler := m.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFValidOrigin(t *testing.T) {
	m, cleanup := setupCSRFTest(t)
	defer cleanup()

	handler := m.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFInvalidOrigin(t *testing.T) {
	m, cleanup := setupCSRFTest(t)
	defer cleanup()

	handler := m.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFValidReferer(t *testing.T) {
	m, cleanup := setupCSRFTest(t)
	defer cleanup()

	handler := m.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Referer", "http://example.com/login")
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFInvalidReferer(t *testing.T) {
	m, cleanup := setupCSRFTest(t)
	defer cleanup()

	handler := m.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Referer", "http://evil.com/phishing")
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
