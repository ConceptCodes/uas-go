package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(session *models.Session) error
	FindByRefreshToken(token string, departmentID string) (*models.Session, error)
	FindByUserID(userID string, departmentID string) ([]models.Session, error)
	RevokeSession(sessionID string, departmentID string) error
	RevokeAllUserSessions(userID string, departmentID string) error
	RevokeAllByDepartment(departmentID string) error
	DeleteExpiredSessions() error
}

type GormSessionRepository struct {
	db *gorm.DB
}

func NewGormSessionRepository(db *gorm.DB) SessionRepository {
	return &GormSessionRepository{db: db}
}

func (r *GormSessionRepository) Create(session *models.Session) error {
	hashedSession := *session
	hashedSession.RefreshToken = hashRefreshToken(session.RefreshToken)
	return r.db.Create(&hashedSession).Error
}

func (r *GormSessionRepository) FindByRefreshToken(token string, departmentID string) (*models.Session, error) {
	var session models.Session
	hashedToken := hashRefreshToken(token)
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("refresh_token = ? AND revoked_at IS NULL AND expires_at > ?", hashedToken, time.Now()).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		legacyErr := r.db.Scopes(mysql.TenantScope(departmentID)).
			Where("refresh_token = ? AND revoked_at IS NULL AND expires_at > ?", token, time.Now()).
			First(&session).Error
		if legacyErr == nil {
			_ = r.db.Model(&session).Update("refresh_token", hashedToken).Error
			return &session, nil
		}
		return &session, legacyErr
	}
	return &session, err
}

func (r *GormSessionRepository) FindByUserID(userID string, departmentID string) ([]models.Session, error) {
	var sessions []models.Session
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *GormSessionRepository) RevokeSession(sessionID string, departmentID string) error {
	now := time.Now()
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("revoked_at", now).Error
}

func (r *GormSessionRepository) RevokeAllUserSessions(userID string, departmentID string) error {
	now := time.Now()
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (r *GormSessionRepository) RevokeAllByDepartment(departmentID string) error {
	now := time.Now()
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.Session{}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now).Error
}

func (r *GormSessionRepository) DeleteExpiredSessions() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{}).Error
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
