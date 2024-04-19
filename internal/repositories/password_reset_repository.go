package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type PasswordResetRepository interface {
	Create(user *models.PasswordResetModel) error
	Delete(id string) error
	FindByToken(token string) (*models.PasswordResetModel, error)
}

type GormPasswordResetRepository struct {
	db *gorm.DB
}

func (r *GormPasswordResetRepository) FindByToken(id string) (*models.PasswordResetModel, error) {
	var model models.PasswordResetModel
	if err := r.db.Where(constants.FindByToken, id).First(&model).Error; err != nil {
		return nil, err
	}

	return &model, nil
}

func (r *GormPasswordResetRepository) Create(model *models.PasswordResetModel) error {
	return r.db.Create(model).Error
}

func (r *GormPasswordResetRepository) Delete(id string) error {
	return r.db.Where(constants.FindByToken, id).Delete(&models.PasswordResetModel{}).Error
}
