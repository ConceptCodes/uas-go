package models

import (
	"database/sql/driver"

	"gorm.io/gorm"
)

type Role string

const (
	User  Role = "user"
	Admin Role = "admin"
)

type TenantModel struct {
	gorm.Model
	ID     string `gorm:"primaryKey;type:varchar(36);unique_index"`
	Name   string `gorm:"type:varchar(100);unique_index"`
	Secret string `gorm:"type:varchar(36);unique_index"`
}

type UserModel struct {
	gorm.Model
	ID            string `gorm:"primaryKey;type:varchar(36);unique_index"`
	Name          string `gorm:"type:varchar(100)"`
	Email         string `gorm:"type:varchar(100);unique_index"`
	Password      string `gorm:"type:varchar(100);unique_index"`
	PhoneNumber   string `gorm:"type:varchar(14);unique_index"`
	EmailVerified bool   `gorm:"type:boolean"`
	Role          Role  `json:"ecosystem" sql:"type:ENUM('production', 'testsystem')"`
}

type PasswordResetModel struct {
	gorm.Model
	UserID string `gorm:"type:varchar(36);unique_index"`
	Token  string `gorm:"type:varchar(36);unique_index"`
}

func (e *Role) Scan(value interface{}) error {
	*e = Role(value.([]byte))
	return nil
}

func (e Role) Value() (driver.Value, error) {
	return string(e), nil
}
