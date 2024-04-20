package models

import "gorm.io/gorm"

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
}

type PasswordResetModel struct {
	gorm.Model
	UserID string `gorm:"type:varchar(36);unique_index"`
	Token  string `gorm:"type:varchar(36);unique_index"`
}
