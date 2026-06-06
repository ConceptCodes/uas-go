package mysql

import (
	"gorm.io/gorm"
)

func TenantScope(departmentID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("department_id = ?", departmentID)
	}
}

func UserTenantScope(departmentID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN department_roles ON department_roles.user_id = users.id").
			Where("department_roles.id = ?", departmentID)
	}
}
