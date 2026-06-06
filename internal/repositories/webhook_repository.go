package repository

import (
	"time"
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"gorm.io/gorm"
)

type WebhookEndpointRepository interface {
	Create(endpoint *models.WebhookEndpoint) error
	FindByID(id, departmentID string) (*models.WebhookEndpoint, error)
	FindByDepartment(departmentID string) ([]models.WebhookEndpoint, error)
	FindActiveByDepartmentAndEvent(departmentID string, event models.WebhookEventType) ([]models.WebhookEndpoint, error)
	Update(endpoint *models.WebhookEndpoint) error
	UpdateSecret(id, departmentID, secret string) error
	Delete(id, departmentID string) error
}

type WebhookDeliveryRepository interface {
	Create(delivery *models.WebhookDelivery) error
	FindByID(id, endpointID string) (*models.WebhookDelivery, error)
	FindByEndpoint(endpointID, departmentID string, limit, offset int) ([]models.WebhookDelivery, error)
	FindByDepartment(departmentID string, limit, offset int) ([]models.WebhookDelivery, error)
	FindPendingRetries() ([]models.WebhookDelivery, error)
	UpdateDelivery(id, endpointID string, status models.WebhookDeliveryStatus, responseCode int, responseBody string) error
	UpdateDeliveryWithRetry(id, endpointID string, status models.WebhookDeliveryStatus, responseCode int, responseBody string, attempt int, nextRetryAt *time.Time) error
	DeleteOlderThan(duration time.Duration) (int64, error)
}

type GormWebhookEndpointRepository struct {
	db *gorm.DB
}

type GormWebhookDeliveryRepository struct {
	db *gorm.DB
}

func NewGormWebhookEndpointRepository(db *gorm.DB) WebhookEndpointRepository {
	return &GormWebhookEndpointRepository{db: db}
}

func NewGormWebhookDeliveryRepository(db *gorm.DB) WebhookDeliveryRepository {
	return &GormWebhookDeliveryRepository{db: db}
}

func (r *GormWebhookEndpointRepository) Create(endpoint *models.WebhookEndpoint) error {
	return r.db.Create(endpoint).Error
}

func (r *GormWebhookEndpointRepository) FindByID(id, departmentID string) (*models.WebhookEndpoint, error) {
	var endpoint models.WebhookEndpoint
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("id = ?", id).First(&endpoint).Error
	return &endpoint, err
}

func (r *GormWebhookEndpointRepository) FindByDepartment(departmentID string) ([]models.WebhookEndpoint, error) {
	var endpoints []models.WebhookEndpoint
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Find(&endpoints).Error
	return endpoints, err
}

func (r *GormWebhookEndpointRepository) FindActiveByDepartmentAndEvent(departmentID string, event models.WebhookEventType) ([]models.WebhookEndpoint, error) {
	var endpoints []models.WebhookEndpoint
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("is_active = ? AND JSON_CONTAINS(events, ?)", true, `"`+string(event)+`"`).
		Find(&endpoints).Error
	return endpoints, err
}

func (r *GormWebhookEndpointRepository) Update(endpoint *models.WebhookEndpoint) error {
	return r.db.Save(endpoint).Error
}

func (r *GormWebhookEndpointRepository) UpdateSecret(id, departmentID, secret string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.WebhookEndpoint{}).
		Where("id = ?", id).
		Update("secret", secret).Error
}

func (r *GormWebhookEndpointRepository) Delete(id, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).Delete(&models.WebhookEndpoint{}, id).Error
}

func (r *GormWebhookDeliveryRepository) Create(delivery *models.WebhookDelivery) error {
	return r.db.Create(delivery).Error
}

func (r *GormWebhookDeliveryRepository) FindByID(id, endpointID string) (*models.WebhookDelivery, error) {
	var delivery models.WebhookDelivery
	err := r.db.Where("id = ? AND endpoint_id = ?", id, endpointID).First(&delivery).Error
	return &delivery, err
}

func (r *GormWebhookDeliveryRepository) FindByEndpoint(endpointID, departmentID string, limit, offset int) ([]models.WebhookDelivery, error) {
	var deliveries []models.WebhookDelivery
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("endpoint_id = ?", endpointID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&deliveries).Error
	return deliveries, err
}

func (r *GormWebhookDeliveryRepository) FindByDepartment(departmentID string, limit, offset int) ([]models.WebhookDelivery, error) {
	var deliveries []models.WebhookDelivery
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&deliveries).Error
	return deliveries, err
}

func (r *GormWebhookDeliveryRepository) FindPendingRetries() ([]models.WebhookDelivery, error) {
	var deliveries []models.WebhookDelivery
	now := time.Now()
	err := r.db.Where("status = ? AND next_retry_at <= ? AND attempt < max_attempts", models.WebhookDeliveryPending, now).
		Find(&deliveries).Error
	return deliveries, err
}

func (r *GormWebhookDeliveryRepository) UpdateDelivery(id, endpointID string, status models.WebhookDeliveryStatus, responseCode int, responseBody string) error {
	return r.db.Model(&models.WebhookDelivery{}).
		Where("id = ? AND endpoint_id = ?", id, endpointID).
		Updates(map[string]interface{}{
			"status":        status,
			"response_code": responseCode,
			"response_body": responseBody,
		}).Error
}

func (r *GormWebhookDeliveryRepository) UpdateDeliveryWithRetry(id, endpointID string, status models.WebhookDeliveryStatus, responseCode int, responseBody string, attempt int, nextRetryAt *time.Time) error {
	return r.db.Model(&models.WebhookDelivery{}).
		Where("id = ? AND endpoint_id = ?", id, endpointID).
		Updates(map[string]interface{}{
			"status":        status,
			"response_code": responseCode,
			"response_body": responseBody,
			"attempt":       attempt,
			"next_retry_at": nextRetryAt,
		}).Error
}

func (r *GormWebhookDeliveryRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-duration)
	result := r.db.Where("created_at < ?", cutoff).Delete(&models.WebhookDelivery{})
	return result.RowsAffected, result.Error
}