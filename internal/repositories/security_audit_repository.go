package repository

import (
	"time"
	"uas/internal/models"

	"gorm.io/gorm"
)

type SecurityAuditRepository interface {
	Create(event *models.SecurityEvent) error
	GetByUserID(userID string, limit int) ([]models.SecurityEvent, error)
	GetByDepartmentID(departmentID string, limit int) ([]models.SecurityEvent, error)
	GetByEventType(eventType string, days int) ([]models.SecurityEvent, error)
	DeleteOldEvents(days int) error
}

type GormSecurityAuditRepository struct {
	db *gorm.DB
}

func NewGormSecurityAuditRepository(db *gorm.DB) SecurityAuditRepository {
	return &GormSecurityAuditRepository{db: db}
}

func (r *GormSecurityAuditRepository) Create(event *models.SecurityEvent) error {
	return r.db.Create(event).Error
}

func (r *GormSecurityAuditRepository) GetByUserID(userID string, limit int) ([]models.SecurityEvent, error) {
	var events []models.SecurityEvent
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *GormSecurityAuditRepository) GetByDepartmentID(departmentID string, limit int) ([]models.SecurityEvent, error) {
	var events []models.SecurityEvent
	err := r.db.
		Where("department_id = ?", departmentID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *GormSecurityAuditRepository) GetByEventType(eventType string, days int) ([]models.SecurityEvent, error) {
	var events []models.SecurityEvent
	since := time.Now().AddDate(0, 0, -days)
	err := r.db.
		Where("event_type = ? AND created_at >= ?", eventType, since).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

func (r *GormSecurityAuditRepository) DeleteOldEvents(days int) error {
	before := time.Now().AddDate(0, 0, -days)
	return r.db.Where("created_at < ?", before).Delete(&models.SecurityEvent{}).Error
}
