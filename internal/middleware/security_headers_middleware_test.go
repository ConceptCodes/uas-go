package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"uas/config"
	"uas/internal/constants"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupSecurityHeadersTest(t *testing.T) (*SecurityHeadersMiddleware, func()) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		CORSAllowedOrigins: "http://localhost:3000",
		Env:                "development",
	}
	log := zerolog.New(io.Discard)
	m := NewSecurityHeadersMiddleware(&log)
	return m, func() { config.AppConfig = orig }
}

func TestSecurityHeaders(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "no-store, no-cache, must-revalidate, proxy-revalidate", w.Header().Get("Cache-Control"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"), "no origin header was sent")
}

func TestSecurityHeadersWithValidOrigin(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(w, req)

	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

func TestSecurityHeadersInvalidOrigin(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestSecurityHeadersCORSPreflight(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityHeadersCORSPreflightInvalidOrigin(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSecurityHeadersDeprecatedPath(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/users/credential/login", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, "true", w.Header().Get("Deprecation"))
	assert.Contains(t, w.Header().Get("Sunset"), "2026")
}

func TestSecurityHeadersHSTS(t *testing.T) {
	m, cleanup := setupSecurityHeadersTest(t)
	defer cleanup()

	handler := m.Start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(constants.ForwardedProtoHeader, "https")
	handler.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Strict-Transport-Security"),
		"HSTS should not be set in non-production")
}

func TestIsAllowedOriginWildcard(t *testing.T) {
	orig := config.AppConfig
	config.AppConfig = config.Config{CORSAllowedOrigins: "*"}
	defer func() { config.AppConfig = orig }()

	assert.True(t, isAllowedOrigin("http://anything.com"))
}

func TestIsAllowedOriginEmpty(t *testing.T) {
	orig := config.AppConfig
	config.AppConfig = config.Config{CORSAllowedOrigins: ""}
	defer func() { config.AppConfig = orig }()

	assert.False(t, isAllowedOrigin("http://something.com"))
}
