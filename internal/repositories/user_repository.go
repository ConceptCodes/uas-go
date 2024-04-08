package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type UserRepository interface {
	FindById(id string) (*models.UserModel, error)
	Save(user *models.UserModel) error
	Delete(id string) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func (r *GormUserRepository) FindById(id string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.Where(constants.FindByIdQuery, id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Save(user *models.UserModel) error {
	return r.db.Save(user).Error
}

func (r *GormUserRepository) Delete(id string) error {
	return r.db.Delete(&models.UserModel{}, id).Error
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db}
}
