package repository

import (
	"time"
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"gorm.io/gorm"
)

type MfaFactorRepository interface {
	Create(factor *models.MfaFactor) error
	FindByID(id, departmentID string) (*models.MfaFactor, error)
	FindByUserID(userID, departmentID string) ([]models.MfaFactor, error)
	FindActiveByUserID(userID, departmentID string) ([]models.MfaFactor, error)
	SetPrimary(id, userID, departmentID string) error
	UpdateBackupCodes(id, departmentID, backupCodes string) error
	UpdateLastUsed(id, departmentID string) error
	Delete(id, departmentID string) error
	DeleteByUserID(userID, departmentID string) error
	GetPrimaryFactor(userID, departmentID string) (*models.MfaFactor, error)
	CountActiveByUserID(userID, departmentID string) (int64, error)
}

type MfaChallengeRepository interface {
	Create(challenge *models.MfaChallenge) error
	FindByID(id string) (*models.MfaChallenge, error)
	FindByTempToken(token string) (*models.MfaChallenge, error)
	VerifyChallenge(id string) error
	ExpireChallenge(id string) error
	ExpireOldChallenges(userID, departmentID string) error
}

type GormMfaFactorRepository struct {
	db *gorm.DB
}

type GormMfaChallengeRepository struct {
	db *gorm.DB
}

func NewGormMfaFactorRepository(db *gorm.DB) MfaFactorRepository {
	return &GormMfaFactorRepository{db: db}
}

func NewGormMfaChallengeRepository(db *gorm.DB) MfaChallengeRepository {
	return &GormMfaChallengeRepository{db: db}
}

func (r *GormMfaFactorRepository) Create(factor *models.MfaFactor) error {
	return r.db.Create(factor).Error
}

func (r *GormMfaFactorRepository) FindByID(id, departmentID string) (*models.MfaFactor, error) {
	var factor models.MfaFactor
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("id = ?", id).First(&factor).Error
	return &factor, err
}

func (r *GormMfaFactorRepository) FindByUserID(userID, departmentID string) ([]models.MfaFactor, error) {
	var factors []models.MfaFactor
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("user_id = ?", userID).Order("created_at DESC").Find(&factors).Error
	return factors, err
}

func (r *GormMfaFactorRepository) FindActiveByUserID(userID, departmentID string) ([]models.MfaFactor, error) {
	var factors []models.MfaFactor
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Find(&factors).Error
	return factors, err
}

func (r *GormMfaFactorRepository) SetPrimary(id, userID, departmentID string) error {
	tx := r.db.Begin()
	if err := tx.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.MfaFactor{}).
		Where("user_id = ?", userID).
		Update("is_primary", false).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.MfaFactor{}).
		Where("id = ?", id).
		Update("is_primary", true).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *GormMfaFactorRepository) UpdateBackupCodes(id, departmentID, backupCodes string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.MfaFactor{}).
		Where("id = ?", id).
		Update("backup_codes", backupCodes).Error
}

func (r *GormMfaFactorRepository) UpdateLastUsed(id, departmentID string) error {
	now := time.Now()
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.MfaFactor{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}

func (r *GormMfaFactorRepository) Delete(id, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).Delete(&models.MfaFactor{}, id).Error
}

func (r *GormMfaFactorRepository) DeleteByUserID(userID, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("user_id = ?", userID).
		Delete(&models.MfaFactor{}).Error
}

func (r *GormMfaFactorRepository) GetPrimaryFactor(userID, departmentID string) (*models.MfaFactor, error) {
	var factor models.MfaFactor
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("user_id = ? AND is_primary = ?", userID, true).
		First(&factor).Error
	return &factor, err
}

func (r *GormMfaFactorRepository) CountActiveByUserID(userID, departmentID string) (int64, error) {
	var count int64
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.MfaFactor{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *GormMfaChallengeRepository) Create(challenge *models.MfaChallenge) error {
	return r.db.Create(challenge).Error
}

func (r *GormMfaChallengeRepository) FindByID(id string) (*models.MfaChallenge, error) {
	var challenge models.MfaChallenge
	err := r.db.Where("id = ?", id).First(&challenge).Error
	return &challenge, err
}

func (r *GormMfaChallengeRepository) FindByTempToken(token string) (*models.MfaChallenge, error) {
	var challenge models.MfaChallenge
	err := r.db.Where("temp_token = ?", token).First(&challenge).Error
	return &challenge, err
}

func (r *GormMfaChallengeRepository) VerifyChallenge(id string) error {
	return r.db.Model(&models.MfaChallenge{}).Where("id = ?", id).Update("state", models.MfaChallengeVerified).Error
}

func (r *GormMfaChallengeRepository) ExpireChallenge(id string) error {
	return r.db.Model(&models.MfaChallenge{}).Where("id = ?", id).Update("state", models.MfaChallengeExpired).Error
}

func (r *GormMfaChallengeRepository) ExpireOldChallenges(userID, departmentID string) error {
	return r.db.Model(&models.MfaChallenge{}).
		Where("user_id = ? AND department_id = ? AND state = ?", userID, departmentID, models.MfaChallengePending).
		Update("state", models.MfaChallengeExpired).Error
}