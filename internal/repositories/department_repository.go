package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type DepartmentRepository interface {
	FindById(id string) (*models.DepartmentModel, error)
	Create(user *models.DepartmentModel) error
	Delete(id string) error
}

type GormDepartmentRepository struct {
	db *gorm.DB
}

func (r *GormDepartmentRepository) FindById(id string) (*models.DepartmentModel, error) {
	var tenant models.DepartmentModel
	if err := r.db.Where(constants.FindByIdQuery, id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *GormDepartmentRepository) Create(tenant *models.DepartmentModel) error {
	return r.db.Create(tenant).Error
}

func (r *GormDepartmentRepository) Delete(id string) error {
	return r.db.Delete(&models.DepartmentModel{}, id).Error
}

func NewGormDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return &GormDepartmentRepository{db}
}
