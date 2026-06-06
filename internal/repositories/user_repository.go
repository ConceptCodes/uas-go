package repository

import (
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"gorm.io/gorm"
)

type UserRepository interface {
	FindById(id, departmentID string) (*models.UserModel, error)
	FindByEmail(email, departmentID string) (*models.UserModel, error)
	FindByPhoneNumber(phoneNumber, departmentID string) (*models.UserModel, error)
	Create(user *models.UserModel) error
	Delete(id, departmentID string) error
	Save(user *models.UserModel, departmentID string) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func (r *GormUserRepository) FindByEmail(email, departmentID string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.Scopes(mysql.UserTenantScope(departmentID)).
		Where("users.email = ?", email).
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Create(user *models.UserModel) error {
	return r.db.Create(user).Error
}

func (r *GormUserRepository) Delete(id, departmentID string) error {
	return r.db.Scopes(mysql.UserTenantScope(departmentID)).
		Delete(&models.UserModel{}, id).Error
}

func (r *GormUserRepository) FindById(id, departmentID string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.Scopes(mysql.UserTenantScope(departmentID)).
		Where("users.id = ?", id).
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Save(user *models.UserModel, departmentID string) error {
	return r.db.Scopes(mysql.UserTenantScope(departmentID)).
		Save(user).Error
}

func (r *GormUserRepository) FindByPhoneNumber(phoneNumber, departmentID string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.Scopes(mysql.UserTenantScope(departmentID)).
		Where("users.phone_number = ?", phoneNumber).
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db}
}
