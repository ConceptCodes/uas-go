package repository

import (
	"fmt"
	"time"

	"uas/internal/models"

	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(auditLog *models.AuditLog) error
	FindByID(id string) (*models.AuditLog, error)
	FindMany(filter *models.AuditLogFilter) (*models.AuditLogQuery, error)
	FindByUserID(userID string, limit, offset int) ([]models.AuditLog, error)
	FindByDepartmentID(departmentID string, limit, offset int) ([]models.AuditLog, error)
	FindByAction(action models.AuditAction, limit, offset int) ([]models.AuditLog, error)
	FindByDateRange(startDate, endDate time.Time, limit, offset int) ([]models.AuditLog, error)
	Count(filter *models.AuditLogFilter) (int64, error)
	DeleteOlderThan(duration time.Duration) (int64, error)
	DeleteOlderThanByAction(action models.AuditAction, duration time.Duration) (int64, error)
}

type GormAuditLogRepository struct {
	db *gorm.DB
}

func (r *GormAuditLogRepository) Create(auditLog *models.AuditLog) error {
	return r.db.Create(auditLog).Error
}

func (r *GormAuditLogRepository) FindByID(id string) (*models.AuditLog, error) {
	var auditLog models.AuditLog
	if err := r.db.Where("id = ?", id).First(&auditLog).Error; err != nil {
		return nil, err
	}
	return &auditLog, nil
}

func (r *GormAuditLogRepository) FindMany(filter *models.AuditLogFilter) (*models.AuditLogQuery, error) {
	var auditLogs []models.AuditLog
	query := r.db.Model(&models.AuditLog{})

	// Apply filters
	if filter != nil {
		if filter.UserID != nil && *filter.UserID != "" {
			query = query.Where("user_id = ?", *filter.UserID)
		}
		if filter.DepartmentID != nil && *filter.DepartmentID != "" {
			query = query.Where("department_id = ?", *filter.DepartmentID)
		}
		if filter.Action != nil && *filter.Action != "" {
			query = query.Where("action = ?", *filter.Action)
		}
		if filter.Resource != nil && *filter.Resource != "" {
			query = query.Where("resource = ?", *filter.Resource)
		}
		if filter.Severity != nil && *filter.Severity != "" {
			query = query.Where("severity = ?", *filter.Severity)
		}
		if filter.Status != nil && *filter.Status != "" {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.IPAddress != nil && *filter.IPAddress != "" {
			query = query.Where("ip_address = ?", *filter.IPAddress)
		}
		if filter.DeviceID != nil && *filter.DeviceID != "" {
			query = query.Where("device_id = ?", *filter.DeviceID)
		}
		if filter.SessionID != nil && *filter.SessionID != "" {
			query = query.Where("session_id = ?", *filter.SessionID)
		}
		if filter.StartDate != nil {
			query = query.Where("timestamp >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("timestamp <= ?", *filter.EndDate)
		}
	}

	// Apply ordering (newest first)
	query = query.Order("timestamp DESC")

	// Apply pagination
	limit := 100 // default limit
	offset := 0
	if filter != nil {
		if filter.Limit != nil && *filter.Limit > 0 {
			limit = *filter.Limit
		}
		if filter.Offset != nil && *filter.Offset > 0 {
			offset = *filter.Offset
		}
	}

	if err := query.Limit(limit).Offset(offset).Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}

	// Get total count
	var total int64
	countQuery := r.db.Model(&models.AuditLog{})
	if filter != nil {
		if filter.UserID != nil && *filter.UserID != "" {
			countQuery = countQuery.Where("user_id = ?", *filter.UserID)
		}
		if filter.DepartmentID != nil && *filter.DepartmentID != "" {
			countQuery = countQuery.Where("department_id = ?", *filter.DepartmentID)
		}
		if filter.Action != nil && *filter.Action != "" {
			countQuery = countQuery.Where("action = ?", *filter.Action)
		}
		if filter.Resource != nil && *filter.Resource != "" {
			countQuery = countQuery.Where("resource = ?", *filter.Resource)
		}
		if filter.Severity != nil && *filter.Severity != "" {
			countQuery = countQuery.Where("severity = ?", *filter.Severity)
		}
		if filter.Status != nil && *filter.Status != "" {
			countQuery = countQuery.Where("status = ?", *filter.Status)
		}
		if filter.IPAddress != nil && *filter.IPAddress != "" {
			countQuery = countQuery.Where("ip_address = ?", *filter.IPAddress)
		}
		if filter.DeviceID != nil && *filter.DeviceID != "" {
			countQuery = countQuery.Where("device_id = ?", *filter.DeviceID)
		}
		if filter.SessionID != nil && *filter.SessionID != "" {
			countQuery = countQuery.Where("session_id = ?", *filter.SessionID)
		}
		if filter.StartDate != nil {
			countQuery = countQuery.Where("timestamp >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			countQuery = countQuery.Where("timestamp <= ?", *filter.EndDate)
		}
	}
	countQuery.Count(&total)

	result := &models.AuditLogQuery{
		Total:   total,
		Results: auditLogs,
	}

	if filter != nil {
		result.Filter = *filter
	}

	return result, nil
}

func (r *GormAuditLogRepository) FindByUserID(userID string, limit, offset int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.db.Where("user_id = ?", userID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&auditLogs).Error
	return auditLogs, err
}

func (r *GormAuditLogRepository) FindByDepartmentID(departmentID string, limit, offset int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.db.Where("department_id = ?", departmentID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&auditLogs).Error
	return auditLogs, err
}

func (r *GormAuditLogRepository) FindByAction(action models.AuditAction, limit, offset int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.db.Where("action = ?", action).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&auditLogs).Error
	return auditLogs, err
}

func (r *GormAuditLogRepository) FindByDateRange(startDate, endDate time.Time, limit, offset int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	err := r.db.Where("timestamp BETWEEN ? AND ?", startDate, endDate).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&auditLogs).Error
	return auditLogs, err
}

func (r *GormAuditLogRepository) Count(filter *models.AuditLogFilter) (int64, error) {
	var count int64
	query := r.db.Model(&models.AuditLog{})

	// Apply filters
	if filter != nil {
		if filter.UserID != nil && *filter.UserID != "" {
			query = query.Where("user_id = ?", *filter.UserID)
		}
		if filter.DepartmentID != nil && *filter.DepartmentID != "" {
			query = query.Where("department_id = ?", *filter.DepartmentID)
		}
		if filter.Action != nil && *filter.Action != "" {
			query = query.Where("action = ?", *filter.Action)
		}
		if filter.Resource != nil && *filter.Resource != "" {
			query = query.Where("resource = ?", *filter.Resource)
		}
		if filter.Severity != nil && *filter.Severity != "" {
			query = query.Where("severity = ?", *filter.Severity)
		}
		if filter.Status != nil && *filter.Status != "" {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.IPAddress != nil && *filter.IPAddress != "" {
			query = query.Where("ip_address = ?", *filter.IPAddress)
		}
		if filter.DeviceID != nil && *filter.DeviceID != "" {
			query = query.Where("device_id = ?", *filter.DeviceID)
		}
		if filter.SessionID != nil && *filter.SessionID != "" {
			query = query.Where("session_id = ?", *filter.SessionID)
		}
		if filter.StartDate != nil {
			query = query.Where("timestamp >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("timestamp <= ?", *filter.EndDate)
		}
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *GormAuditLogRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-duration)
	result := r.db.Where("timestamp < ?", cutoffTime).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}

func (r *GormAuditLogRepository) DeleteOlderThanByAction(action models.AuditAction, duration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-duration)
	result := r.db.Where("action = ? AND timestamp < ?", action, cutoffTime).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}

func NewGormAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &GormAuditLogRepository{db: db}
}
