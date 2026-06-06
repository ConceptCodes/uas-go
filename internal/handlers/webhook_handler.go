package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type WebhookHandler struct {
	endpointRepo   repository.WebhookEndpointRepository
	deliveryRepo   repository.WebhookDeliveryRepository
	webhookHelper  *helpers.WebhookHelper
	responseHelper *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
	log            *zerolog.Logger
}

func NewWebhookHandler(
	endpointRepo repository.WebhookEndpointRepository,
	deliveryRepo repository.WebhookDeliveryRepository,
	webhookHelper *helpers.WebhookHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	log *zerolog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		endpointRepo:   endpointRepo,
		deliveryRepo:   deliveryRepo,
		webhookHelper:  webhookHelper,
		responseHelper: responseHelper,
		validatorHelper: validatorHelper,
		log:            log,
	}
}

func (h *WebhookHandler) CreateEndpointHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)

	var data models.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	secret, err := h.webhookHelper.GenerateSecret()
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate webhook secret", constants.InternalServerError, err)
		return
	}

	events := models.SerializeWebhookEvents(data.Events)

	endpoint := &models.WebhookEndpoint{
		DepartmentID: departmentID,
		Name:         data.Name,
		URL:          data.URL,
		Secret:       secret,
		Events:       events,
		IsActive:     true,
	}

	if err := h.endpointRepo.Create(endpoint); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to create webhook endpoint", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Webhook endpoint created", map[string]interface{}{
		"endpoint": h.toEndpointResponse(endpoint),
		"secret":   secret,
	})
}

func (h *WebhookHandler) ListEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)

	endpoints, err := h.endpointRepo.FindByDepartment(departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to list endpoints", constants.InternalServerError, err)
		return
	}

	var response []models.WebhookEndpointResponse
	for _, e := range endpoints {
		response = append(response, h.toEndpointResponse(&e))
	}

	h.responseHelper.SendSuccessResponse(w, "Webhook endpoints retrieved", response)
}

func (h *WebhookHandler) GetEndpointHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	endpoint, err := h.endpointRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Endpoint not found", constants.NotFound, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Endpoint retrieved", h.toEndpointResponse(endpoint))
}

func (h *WebhookHandler) UpdateEndpointHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	endpoint, err := h.endpointRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Endpoint not found", constants.NotFound, err)
		return
	}

	var data models.UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if data.Name != nil {
		endpoint.Name = *data.Name
	}
	if data.URL != nil {
		endpoint.URL = *data.URL
	}
	if data.Events != nil {
		endpoint.Events = models.SerializeWebhookEvents(data.Events)
	}
	if data.IsActive != nil {
		endpoint.IsActive = *data.IsActive
	}

	if err := h.endpointRepo.Update(endpoint); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to update endpoint", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Endpoint updated", h.toEndpointResponse(endpoint))
}

func (h *WebhookHandler) DeleteEndpointHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	if err := h.endpointRepo.Delete(id, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to delete endpoint", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Endpoint deleted", nil)
}

func (h *WebhookHandler) RotateSecretHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	_, err := h.endpointRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Endpoint not found", constants.NotFound, err)
		return
	}

	newSecret, err := h.webhookHelper.GenerateSecret()
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate new secret", constants.InternalServerError, err)
		return
	}

	if err := h.webhookHelper.RotateSecret(id, departmentID, newSecret); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to rotate secret", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Secret rotated", models.WebhookSecretResponse{Secret: newSecret})
}

func (h *WebhookHandler) ListDeliveriesHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	deliveries, err := h.deliveryRepo.FindByDepartment(departmentID, limit, offset)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to list deliveries", constants.InternalServerError, err)
		return
	}

	var response []models.WebhookDeliveryResponse
	for _, d := range deliveries {
		response = append(response, h.toDeliveryResponse(&d))
	}

	h.responseHelper.SendSuccessResponse(w, "Deliveries retrieved", response)
}

func (h *WebhookHandler) GetDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	delivery, err := h.deliveryRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Delivery not found", constants.NotFound, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Delivery retrieved", h.toDeliveryResponse(delivery))
}

func (h *WebhookHandler) RetryDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	endpoints, err := h.endpointRepo.FindByDepartment(departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "No endpoints found", constants.NotFound, err)
		return
	}

	var endpointID string
	for _, e := range endpoints {
		endpointID = e.ID
		break
	}

	if err := h.webhookHelper.RetryDelivery(id, endpointID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to retry delivery", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Delivery retry initiated", nil)
}

func (h *WebhookHandler) toEndpointResponse(e *models.WebhookEndpoint) models.WebhookEndpointResponse {
	return models.WebhookEndpointResponse{
		ID:        e.ID,
		Name:      e.Name,
		URL:       e.URL,
		Events:    models.DeserializeWebhookEvents(e.Events),
		IsActive:  e.IsActive,
		CreatedAt: e.CreatedAt,
	}
}

func (h *WebhookHandler) toDeliveryResponse(d *models.WebhookDelivery) models.WebhookDeliveryResponse {
	return models.WebhookDeliveryResponse{
		ID:           d.ID,
		EndpointID:   d.EndpointID,
		Event:        d.Event,
		ResponseCode: d.ResponseCode,
		Status:       d.Status,
		Attempt:      d.Attempt,
		MaxAttempts:  d.MaxAttempts,
		NextRetryAt:  d.NextRetryAt,
		CreatedAt:    d.CreatedAt,
	}
}