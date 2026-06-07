package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"uas/internal/helpers"
	"uas/internal/models"
)

type webhookHandlerDeps struct {
	handler      *WebhookHandler
	endpointRepo *MockWebhookEndpointRepository
	deliveryRepo *MockWebhookDeliveryRepository
}

func setupWebhookHandler(t *testing.T) *webhookHandlerDeps {
	t.Helper()
	saveAppConfig(t)

	log := zerolog.Nop()
	endpointRepo := new(MockWebhookEndpointRepository)
	deliveryRepo := new(MockWebhookDeliveryRepository)
	webhookHelper := helpers.NewWebhookHelper(&log, endpointRepo, deliveryRepo)
	responseHelper := newTestResponseHelper(&log)
	validatorHelper := newTestValidatorHelper(&log, responseHelper)

	handler := NewWebhookHandler(
		endpointRepo,
		deliveryRepo,
		webhookHelper,
		responseHelper,
		validatorHelper,
		&log,
	)

	return &webhookHandlerDeps{
		handler:      handler,
		endpointRepo: endpointRepo,
		deliveryRepo: deliveryRepo,
	}
}

func TestWebhookHandler_CreateEndpointHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("Create", mock.MatchedBy(func(e *models.WebhookEndpoint) bool {
		return e.Name == "test-endpoint" && e.URL == "https://example.com/hook" && e.IsActive
	})).Return(nil)

	body := map[string]interface{}{
		"name":   "test-endpoint",
		"url":    "https://example.com/hook",
		"events": []string{"user.created"},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/endpoints", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.CreateEndpointHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Webhook endpoint created", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	endpoint, ok := data["endpoint"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-endpoint", endpoint["name"])
	assert.Equal(t, "https://example.com/hook", endpoint["url"])
	assert.True(t, endpoint["isActive"].(bool))

	secret, ok := data["secret"].(string)
	require.True(t, ok)
	assert.Len(t, secret, 32)
}

func TestWebhookHandler_CreateEndpointHandler_DecodeError(t *testing.T) {
	d := setupWebhookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/endpoints", bytes.NewReader([]byte(`invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.CreateEndpointHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookHandler_CreateEndpointHandler_ValidationError(t *testing.T) {
	d := setupWebhookHandler(t)

	body := map[string]interface{}{
		"name": "ok",
		"url":  "not-a-url",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/endpoints", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.CreateEndpointHandler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookHandler_CreateEndpointHandler_CreateError(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("Create", mock.Anything).Return(assert.AnError)

	body := map[string]interface{}{
		"name":   "test-endpoint",
		"url":    "https://example.com/hook",
		"events": []string{"user.created"},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/endpoints", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.CreateEndpointHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWebhookHandler_GetEndpointHandler_Found(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByID", "ep-1", "dept-1").Return(&models.WebhookEndpoint{
		ID:   "ep-1",
		Name: "test-endpoint",
		URL:  "https://example.com/hook",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/endpoints/ep-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ep-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetEndpointHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_GetEndpointHandler_NotFound(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByID", "ep-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/endpoints/ep-999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ep-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetEndpointHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWebhookHandler_ListEndpointsHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	endpoints := []models.WebhookEndpoint{
		{ID: "ep-1", Name: "Endpoint 1", URL: "https://example.com/1"},
		{ID: "ep-2", Name: "Endpoint 2", URL: "https://example.com/2"},
	}
	d.endpointRepo.On("FindByDepartment", "dept-1").Return(endpoints, nil)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/endpoints", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListEndpointsHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_ListEndpointsHandler_Error(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByDepartment", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/endpoints", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListEndpointsHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWebhookHandler_UpdateEndpointHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByID", "ep-1", "dept-1").Return(&models.WebhookEndpoint{
		ID:   "ep-1",
		Name: "old-name",
		URL:  "https://old.com/hook",
	}, nil)

	d.endpointRepo.On("Update", mock.MatchedBy(func(e *models.WebhookEndpoint) bool {
		return e.ID == "ep-1" && e.Name == "new-name" && e.URL == "https://new.com/hook"
	})).Return(nil)

	body := map[string]interface{}{
		"name": "new-name",
		"url":  "https://new.com/hook",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/webhooks/endpoints/ep-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "ep-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.UpdateEndpointHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_UpdateEndpointHandler_NotFound(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByID", "ep-999", "dept-1").Return(nil, assert.AnError)

	body := map[string]interface{}{
		"name": "new-name",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/webhooks/endpoints/ep-999", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "ep-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.UpdateEndpointHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWebhookHandler_DeleteEndpointHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("Delete", "ep-1", "dept-1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/endpoints/ep-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ep-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.DeleteEndpointHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_DeleteEndpointHandler_Error(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("Delete", "ep-1", "dept-1").Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/webhooks/endpoints/ep-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ep-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.DeleteEndpointHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWebhookHandler_RotateSecretHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByID", "ep-1", "dept-1").Return(&models.WebhookEndpoint{
		ID:   "ep-1",
		Name: "test-endpoint",
		URL:  "https://example.com/hook",
	}, nil)

	d.endpointRepo.On("UpdateSecret", "ep-1", "dept-1", mock.AnythingOfType("string")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/endpoints/ep-1/rotate-secret", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ep-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RotateSecretHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.SuccessResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Secret rotated", resp.Message)
}

func TestWebhookHandler_RotateSecretHandler_NotFound(t *testing.T) {
	d := setupWebhookHandler(t)

	d.endpointRepo.On("FindByID", "ep-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/endpoints/ep-999/rotate-secret", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ep-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RotateSecretHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWebhookHandler_ListDeliveriesHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	deliveries := []models.WebhookDelivery{
		{ID: "d-1", EndpointID: "ep-1", Status: models.WebhookDeliverySuccess},
		{ID: "d-2", EndpointID: "ep-1", Status: models.WebhookDeliveryFailed},
	}
	d.deliveryRepo.On("FindByDepartment", "dept-1", 50, 0).Return(deliveries, nil)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/deliveries", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListDeliveriesHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_ListDeliveriesHandler_Error(t *testing.T) {
	d := setupWebhookHandler(t)

	d.deliveryRepo.On("FindByDepartment", "dept-1", 50, 0).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/deliveries", nil)
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.ListDeliveriesHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWebhookHandler_GetDeliveryHandler_Found(t *testing.T) {
	d := setupWebhookHandler(t)

	d.deliveryRepo.On("FindByID", "d-1", "dept-1").Return(&models.WebhookDelivery{
		ID:     "d-1",
		Status: models.WebhookDeliverySuccess,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/deliveries/d-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "d-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetDeliveryHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_GetDeliveryHandler_NotFound(t *testing.T) {
	d := setupWebhookHandler(t)

	d.deliveryRepo.On("FindByID", "d-999", "dept-1").Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/deliveries/d-999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "d-999"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.GetDeliveryHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWebhookHandler_RetryDeliveryHandler_HappyPath(t *testing.T) {
	d := setupWebhookHandler(t)

	d.deliveryRepo.On("FindByDepartment", "dept-1", 1, 0).Return([]models.WebhookDelivery{
		{ID: "d-1", EndpointID: "ep-1", Status: models.WebhookDeliveryFailed, DepartmentID: "dept-1"},
	}, nil)

	d.endpointRepo.On("FindByID", "ep-1", "dept-1").Return(&models.WebhookEndpoint{
		ID:       "ep-1",
		URL:      "https://example.com/hook",
		IsActive: true,
	}, nil)

	d.deliveryRepo.On("UpdateDelivery", "d-1", "ep-1", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/deliveries/d-1/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "d-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RetryDeliveryHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookHandler_RetryDeliveryHandler_NoEndpoints(t *testing.T) {
	d := setupWebhookHandler(t)

	d.deliveryRepo.On("FindByDepartment", "dept-1", 1, 0).Return([]models.WebhookDelivery{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/deliveries/d-1/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "d-1"})
	req = withDepartment(req, "dept-1")
	rec := httptest.NewRecorder()

	d.handler.RetryDeliveryHandler(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
