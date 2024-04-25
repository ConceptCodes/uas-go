package repository

import (
	"gorm.io/gorm"

	"uas/internal/constants"
	"uas/internal/models"
)

type DepartmentRoleRepository interface {
	Create(user *models.DepartmentRoles) error
	Update(user *models.DepartmentRoles) error
	FindById(id string) (*models.DepartmentRoles, error)
}

type GormDepartmentRoleRepository struct {
	db *gorm.DB
}

func (r *GormDepartmentRoleRepository) Create(model *models.DepartmentRoles) error {
	return r.db.Create(model).Error
}

func (r *GormDepartmentRoleRepository) Update(model *models.DepartmentRoles) error {
	return r.db.Save(model).Error
}

func (r *GormDepartmentRoleRepository) FindById(departmentId string) (*models.DepartmentRoles, error) {
	var model models.DepartmentRoles
	if err := r.db.Where(constants.FindByIdQuery, departmentId).First(&model).Error; err != nil {
		return nil, err
	}

	return &model, nil
}

func NewGormDepartmentRoleRepository(db *gorm.DB) DepartmentRoleRepository {
	return &GormDepartmentRoleRepository{db}
}
