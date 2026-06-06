package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentTypeJSON(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, "application/json;charset=utf8", w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, w.Code)
}
