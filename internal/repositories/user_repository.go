package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type UserRepository interface {
	FindByEmail(email string) (*models.UserModel, error)
	Create(user *models.UserModel) error
	Delete(id string) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func (r *GormUserRepository) FindByEmail(id string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.Where(constants.FindByEmailQuery, id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Create(user *models.UserModel) error {
	return r.db.Create(user).Error
}

func (r *GormUserRepository) Delete(id string) error {
	return r.db.Delete(&models.UserModel{}, id).Error
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db}
}