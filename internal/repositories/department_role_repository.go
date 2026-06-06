package repository

import (
	"uas/internal/constants"
	"uas/internal/models"

	"gorm.io/gorm"
)

type DepartmentRoleRepository interface {
	Create(user *models.DepartmentRoles) error
	Update(user *models.DepartmentRoles) error
	FindById(departmentId string, userId string) (*models.DepartmentRoles, error)
	FindByUserID(userID string, departmentID string) (*models.DepartmentRoles, error)
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

func (r *GormDepartmentRoleRepository) FindById(departmentId string, userId string) (*models.DepartmentRoles, error) {
	var model models.DepartmentRoles
	if err := r.db.Where(constants.FindByIdAndUserIdQuery, departmentId, userId).First(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func (r *GormDepartmentRoleRepository) FindByUserID(userID string, departmentID string) (*models.DepartmentRoles, error) {
	var model models.DepartmentRoles
	if err := r.db.Where("user_id = ? AND id = ?", userID, departmentID).Order("updated_at DESC").First(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func NewGormDepartmentRoleRepository(db *gorm.DB) DepartmentRoleRepository {
	return &GormDepartmentRoleRepository{db}
}
