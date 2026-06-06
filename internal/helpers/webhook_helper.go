package helpers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"uas/config"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog"
)

type WebhookHelper struct {
	log             *zerolog.Logger
	webhookRepo     repository.WebhookEndpointRepository
	deliveryRepo    repository.WebhookDeliveryRepository
	client          *http.Client
}

func NewWebhookHelper(
	log *zerolog.Logger,
	webhookRepo repository.WebhookEndpointRepository,
	deliveryRepo repository.WebhookDeliveryRepository,
) *WebhookHelper {
	return &WebhookHelper{
		log:          log,
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type WebhookPayload struct {
	Event       models.WebhookEventType `json:"event"`
	Timestamp   time.Time               `json:"timestamp"`
	DepartmentID string                 `json:"departmentId"`
	Data        interface{}             `json:"data"`
}

func (h *WebhookHelper) SignPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (h *WebhookHelper) FireEvent(departmentID string, event models.WebhookEventType, data interface{}) {
	go func() {
		endpoints, err := h.webhookRepo.FindActiveByDepartmentAndEvent(departmentID, event)
		if err != nil {
			h.log.Error().Err(err).Str("event", string(event)).Msg("Failed to fetch webhook endpoints for event")
			return
		}

		payload := WebhookPayload{
			Event:       event,
			Timestamp:   time.Now().UTC(),
			DepartmentID: departmentID,
			Data:        data,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			h.log.Error().Err(err).Msg("Failed to marshal webhook payload")
			return
		}

		for _, endpoint := range endpoints {
			h.deliverToEndpoint(departmentID, &endpoint, string(event), payloadBytes)
		}
	}()
}

func (h *WebhookHelper) deliverToEndpoint(departmentID string, endpoint *models.WebhookEndpoint, event string, payloadBytes []byte) {
	delivery := &models.WebhookDelivery{
		EndpointID:   endpoint.ID,
		DepartmentID: departmentID,
		Event:        models.WebhookEventType(event),
		Payload:      string(payloadBytes),
		Status:       models.WebhookDeliveryPending,
		Attempt:      0,
		MaxAttempts:  config.AppConfig.WebhookMaxRetries,
	}

	if err := h.deliveryRepo.Create(delivery); err != nil {
		h.log.Error().Err(err).Str("endpoint_id", endpoint.ID).Msg("Failed to create webhook delivery record")
		return
	}

	h.executeDelivery(delivery, endpoint)
}

func (h *WebhookHelper) executeDelivery(delivery *models.WebhookDelivery, endpoint *models.WebhookEndpoint) {
	signature := h.SignPayload([]byte(delivery.Payload), endpoint.Secret)

	req, err := http.NewRequest("POST", endpoint.URL, bytes.NewBuffer([]byte(delivery.Payload)))
	if err != nil {
		h.log.Error().Err(err).Str("delivery_id", delivery.ID).Msg("Failed to create webhook request")
		h.markDeliveryFailed(delivery, 0, "failed to create request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", string(delivery.Event))
	req.Header.Set("X-Webhook-Delivery", delivery.ID)
	req.Header.Set("User-Agent", "UAS-Webhook/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		h.log.Warn().Err(err).Str("delivery_id", delivery.ID).Str("url", endpoint.URL).Msg("Webhook delivery failed")
		h.markDeliveryFailed(delivery, 0, err.Error())
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.log.Debug().Str("delivery_id", delivery.ID).Int("status", resp.StatusCode).Msg("Webhook delivered successfully")
		h.deliveryRepo.UpdateDelivery(delivery.ID, endpoint.ID, models.WebhookDeliverySuccess, resp.StatusCode, string(body))
		return
	}

	bodyStr := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	h.markDeliveryFailed(delivery, resp.StatusCode, bodyStr)
}

func (h *WebhookHelper) markDeliveryFailed(delivery *models.WebhookDelivery, statusCode int, responseBody string) {
	delivery.Attempt++
	delivery.ResponseCode = statusCode
	delivery.ResponseBody = responseBody

	if delivery.Attempt >= delivery.MaxAttempts {
		h.deliveryRepo.UpdateDelivery(delivery.ID, delivery.EndpointID, models.WebhookDeliveryFailed, statusCode, responseBody)
		h.log.Warn().Str("delivery_id", delivery.ID).Int("attempts", delivery.Attempt).Msg("Webhook delivery failed after max retries")
		return
	}

	backoff := time.Duration(config.AppConfig.WebhookRetryBaseDelayMs) * time.Millisecond
	for i := 1; i < delivery.Attempt; i++ {
		backoff *= 2
	}
	nextRetry := time.Now().Add(backoff)

	h.deliveryRepo.UpdateDeliveryWithRetry(delivery.ID, delivery.EndpointID, models.WebhookDeliveryPending, statusCode, responseBody, delivery.Attempt, &nextRetry)
	h.log.Info().Str("delivery_id", delivery.ID).Int("attempt", delivery.Attempt).Time("next_retry", nextRetry).Msg("Webhook delivery scheduled for retry")
}

func (h *WebhookHelper) RetryDelivery(deliveryID, endpointID string) error {
	delivery, err := h.deliveryRepo.FindByID(deliveryID, endpointID)
	if err != nil {
		return fmt.Errorf("delivery not found: %w", err)
	}

	endpoint, err := h.webhookRepo.FindByID(endpointID, delivery.DepartmentID)
	if err != nil {
		return fmt.Errorf("endpoint not found: %w", err)
	}

	if !endpoint.IsActive {
		return fmt.Errorf("webhook endpoint is not active")
	}

	h.executeDelivery(delivery, endpoint)
	return nil
}

func (h *WebhookHelper) GenerateSecret() (string, error) {
	return GenerateRandomString(32)
}

func (h *WebhookHelper) RotateSecret(endpointID, departmentID, newSecret string) error {
	return h.webhookRepo.UpdateSecret(endpointID, departmentID, newSecret)
}