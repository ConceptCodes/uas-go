package repository

import (
	"uas/internal/models"

	"gorm.io/gorm"
)

type PasswordHistoryRepository interface {
	Create(history *models.PasswordHistory) error
	GetRecentPasswords(userID string, limit int) ([]models.PasswordHistory, error)
	DeleteOldPasswords(userID string, keepCount int) error
}

type GormPasswordHistoryRepository struct {
	db *gorm.DB
}

func NewGormPasswordHistoryRepository(db *gorm.DB) PasswordHistoryRepository {
	return &GormPasswordHistoryRepository{db: db}
}

func (r *GormPasswordHistoryRepository) Create(history *models.PasswordHistory) error {
	return r.db.Create(history).Error
}

func (r *GormPasswordHistoryRepository) GetRecentPasswords(userID string, limit int) ([]models.PasswordHistory, error) {
	var passwords []models.PasswordHistory
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&passwords).Error
	return passwords, err
}

func (r *GormPasswordHistoryRepository) DeleteOldPasswords(userID string, keepCount int) error {
	var passwordCount int64
	r.db.Model(&models.PasswordHistory{}).Where("user_id = ?", userID).Count(&passwordCount)

	if passwordCount <= int64(keepCount) {
		return nil
	}

	offset := passwordCount - int64(keepCount)
	return r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(int(offset)).
		Delete(&models.PasswordHistory{}).Error
}
