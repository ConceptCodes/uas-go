package models

import (
	"time"

	"gorm.io/gorm"
)

type Role string
type AuthModelType string

const (
	User  Role = "user"
	Admin Role = "admin"
)

const (
	ResetPassword AuthModelType = "reset-password"
	MagicLink     AuthModelType = "magic-link"
)

type DepartmentModel struct {
	gorm.Model
	ID               string           `gorm:"primaryKey;type:varchar(36);unique_index"`
	Name             string           `gorm:"type:varchar(100);unique_index"`
	Secret           string           `gorm:"type:varchar(36);unique_index"`
	DepartmentConfig DepartmentConfig `gorm:"foreignKey:DepartmentID"`
}

type UserModel struct {
	gorm.Model
	ID            string `gorm:"primaryKey;type:varchar(36);unique_index"`
	Name          string `gorm:"type:varchar(100)"`
	Email         string `gorm:"type:varchar(100);unique_index"`
	Password      string `gorm:"type:varchar(100);unique_index"`
	PhoneNumber   string `gorm:"type:varchar(14);unique_index"`
	EmailVerified bool   `gorm:"type:boolean"`
}

type DepartmentRoles struct {
	ID        string `gorm:"primaryKey;type:varchar(36);unique_index"`
	Role      Role   `gorm:"type:varchar(10);unique_index"`
	UserID    string `gorm:"primaryKey;type:varchar(36);unique_index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type AuthModel struct {
	UserID    string        `gorm:"type:varchar(36);unique_index"`
	Token     string        `gorm:"primaryKey;type:varchar(36)"`
	Type      AuthModelType `gorm:"primaryKey:type:varchar(36);`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type DepartmentConfig struct {
	gorm.Model
	ID               string `gorm:"primaryKey;type:varchar(36);unique_index"`
	Name             string `gorm:"type:varchar(100);unique_index"`
	DepartmentID     string `gorm:"type:varchar(36);unique_index"`
	MagicLinkBaseUrl string `gorm:"type:varchar(100);unique_index"`
}
