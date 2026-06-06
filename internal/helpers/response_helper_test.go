package helpers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"uas/internal/models"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newResponseHelper(t *testing.T) *ResponseHelper {
	log := zerolog.New(io.Discard)
	return NewResponseHelper(&log)
}

func TestSendSuccessResponse(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendSuccessResponse(w, "ok", map[string]string{"key": "val"})
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp models.SuccessResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message)
}

func TestSendSuccessResponseNilData(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendSuccessResponse(w, "no data", nil)
	assert.Equal(t, 200, w.Code)

	var resp models.SuccessResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "no data", resp.Message)
	assert.Nil(t, resp.Data)
}

func TestSendErrorResponseBadRequest(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendErrorResponse(w, "invalid input", "UAS-400", nil)
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp models.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "UAS-400", resp.Code)
	assert.Equal(t, "invalid input", resp.Message)
}

func TestSendErrorResponseNotFound(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendErrorResponse(w, "not found", "UAS-404", nil)
	assert.Equal(t, 404, w.Code)
}

func TestSendErrorResponseUnauthorized(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendErrorResponse(w, "unauthorized", "UAS-401", nil)
	assert.Equal(t, 401, w.Code)
}

func TestSendErrorResponseForbidden(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendErrorResponse(w, "forbidden", "UAS-403", nil)
	assert.Equal(t, 403, w.Code)
}

func TestSendErrorResponseInternalServerError(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendErrorResponse(w, "server error", "UAS-500", nil)
	assert.Equal(t, 500, w.Code)
}

func TestSendErrorResponseDefaultError(t *testing.T) {
	h := newResponseHelper(t)
	w := httptest.NewRecorder()

	h.SendErrorResponse(w, "unknown error", "UNKNOWN", nil)
	assert.Equal(t, 500, w.Code)
}
