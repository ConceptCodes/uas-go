package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type AuthRepository interface {
	Create(user *models.AuthModel) error
	Delete(id string) error
	FindByTokenAndType(token string, authType models.AuthModelType) (*models.AuthModel, error)
}

type GormAuthRepository struct {
	db *gorm.DB
}

func (r *GormAuthRepository) FindByTokenAndType(id string, authType models.AuthModelType) (*models.AuthModel, error) {
	var model models.AuthModel
	if err := r.db.Where(constants.FindByTokenAndTypeQuery, id).First(&model).Error; err != nil {
		return nil, err
	}

	return &model, nil
}

func (r *GormAuthRepository) Create(model *models.AuthModel) error {
	return r.db.Create(model).Error
}

func (r *GormAuthRepository) Delete(id string) error {
	return r.db.Where(constants.FindByIdQuery, id).Delete(&models.AuthModel{}).Error
}

func NewGormAuthRepository(db *gorm.DB) AuthRepository {
	return &GormAuthRepository{db}
}
