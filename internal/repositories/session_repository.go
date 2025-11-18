package repository

import (
	"time"
	"uas/internal/models"

	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(session *models.Session) error
	FindByRefreshToken(token string) (*models.Session, error)
	FindByUserID(userID string) ([]models.Session, error)
	RevokeSession(sessionID string) error
	RevokeAllUserSessions(userID string) error
	DeleteExpiredSessions() error
}

type GormSessionRepository struct {
	db *gorm.DB
}

func NewGormSessionRepository(db *gorm.DB) SessionRepository {
	return &GormSessionRepository{db: db}
}

func (r *GormSessionRepository) Create(session *models.Session) error {
	return r.db.Create(session).Error
}

func (r *GormSessionRepository) FindByRefreshToken(token string) (*models.Session, error) {
	var session models.Session
	err := r.db.
		Where("refresh_token = ? AND revoked_at IS NULL AND expires_at > ?", token, time.Now()).
		First(&session).Error
	return &session, err
}

func (r *GormSessionRepository) FindByUserID(userID string) ([]models.Session, error) {
	var sessions []models.Session
	err := r.db.
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *GormSessionRepository) RevokeSession(sessionID string) error {
	now := time.Now()
	return r.db.Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("revoked_at", now).Error
}

func (r *GormSessionRepository) RevokeAllUserSessions(userID string) error {
	now := time.Now()
	return r.db.Model(&models.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (r *GormSessionRepository) DeleteExpiredSessions() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{}).Error
}
