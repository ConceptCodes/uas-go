package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type TenantRepository interface {
	FindById(id string) (*models.TenantModel, error)
	Save(user *models.TenantModel) error
	Delete(id string) error
}

type GormTenantRepository struct {
	db *gorm.DB
}

func (r *GormTenantRepository) FindById(id string) (*models.TenantModel, error) {
	var tenant models.TenantModel
	if err := r.db.Where(constants.FindByIdQuery, id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *GormTenantRepository) Save(tenant *models.TenantModel) error {
	return r.db.Save(tenant).Error
}

func (r *GormTenantRepository) Delete(id string) error {
	return r.db.Delete(&models.TenantModel{}, id).Error
}

func NewGormTenantRepository(db *gorm.DB) TenantRepository {
	return &GormTenantRepository{db}
}
