package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"uas/internal/constants"
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"gorm.io/gorm"
)

type AuthRepository interface {
	Create(user *models.AuthModel) error
	Delete(id, departmentID string) error
	DeleteByTokenAndType(token string, authType models.AuthModelType, departmentID string) error
	FindByTokenAndType(token string, authType models.AuthModelType) (*models.AuthModel, error)
	FindByTokenAndTypeScoped(token string, authType models.AuthModelType, departmentID string) (*models.AuthModel, error)
}

type GormAuthRepository struct {
	db *gorm.DB
}

func (r *GormAuthRepository) FindByTokenAndType(token string, authType models.AuthModelType) (*models.AuthModel, error) {
	var model models.AuthModel
	hashedToken := hashAuthToken(token)
	if err := r.db.Where(constants.FindByTokenAndTypeQuery, hashedToken, authType).First(&model).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		legacyErr := r.db.Where(constants.FindByTokenAndTypeQuery, token, authType).First(&model).Error
		if legacyErr != nil {
			return nil, legacyErr
		}

		_ = r.db.Model(&model).Update("token", hashedToken).Error
		model.Token = hashedToken
	}

	return &model, nil
}

func (r *GormAuthRepository) FindByTokenAndTypeScoped(token string, authType models.AuthModelType, departmentID string) (*models.AuthModel, error) {
	var model models.AuthModel
	hashedToken := hashAuthToken(token)
	if err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where(constants.FindByTokenAndTypeQuery, hashedToken, authType).First(&model).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		legacyErr := r.db.Scopes(mysql.TenantScope(departmentID)).
			Where(constants.FindByTokenAndTypeQuery, token, authType).First(&model).Error
		if legacyErr != nil {
			return nil, legacyErr
		}

		_ = r.db.Model(&model).Update("token", hashedToken).Error
		model.Token = hashedToken
	}

	return &model, nil
}

func (r *GormAuthRepository) Create(model *models.AuthModel) error {
	hashedModel := *model
	hashedModel.Token = hashAuthToken(model.Token)
	return r.db.Create(&hashedModel).Error
}

func (r *GormAuthRepository) Delete(id, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Where(constants.FindByIdQuery, id).Delete(&models.AuthModel{}).Error
}

func (r *GormAuthRepository) DeleteByTokenAndType(token string, authType models.AuthModelType, departmentID string) error {
	hashedToken := hashAuthToken(token)
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("(token = ? OR token = ?) AND type = ?", hashedToken, token, authType).Delete(&models.AuthModel{}).Error
}

func NewGormAuthRepository(db *gorm.DB) AuthRepository {
	return &GormAuthRepository{db}
}

func hashAuthToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
