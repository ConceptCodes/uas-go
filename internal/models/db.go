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
	Password      string `gorm:"type:varchar(255)"`
	PhoneNumber   string `gorm:"type:varchar(14);unique_index"`
	EmailVerified bool   `gorm:"type:boolean"`

	// Encrypted versions of sensitive fields
	EncryptedName        EncryptedField `gorm:"type:text"`
	EncryptedEmail       EncryptedField `gorm:"type:text"`
	EncryptedPhoneNumber EncryptedField `gorm:"type:text"`
}

// BeforeSave GORM hook to encrypt sensitive fields before saving
func (u *UserModel) BeforeSave(tx *gorm.DB) error {
	if u.Name != "" {
		u.EncryptedName.Set(u.Name)
	}
	if u.Email != "" {
		u.EncryptedEmail.Set(u.Email)
	}
	if u.PhoneNumber != "" {
		u.EncryptedPhoneNumber.Set(u.PhoneNumber)
	}
	return nil
}

// AfterFind GORM hook to decrypt sensitive fields after finding
func (u *UserModel) AfterFind(tx *gorm.DB) error {
	if u.EncryptedName.Data != "" {
		u.Name = u.EncryptedName.String()
	}
	if u.EncryptedEmail.Data != "" {
		u.Email = u.EncryptedEmail.String()
	}
	if u.EncryptedPhoneNumber.Data != "" {
		u.PhoneNumber = u.EncryptedPhoneNumber.String()
	}
	return nil
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
	Type      AuthModelType `gorm:"primaryKey;type:varchar(36)"`
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

type PasswordHistory struct {
	gorm.Model
	ID           string `gorm:"primaryKey;type:varchar(36)"`
	UserID       string `gorm:"type:varchar(36);index"`
	PasswordHash string `gorm:"type:varchar(255)"`
	CreatedAt    time.Time
}

type SecurityEvent struct {
	gorm.Model
	ID           string    `gorm:"primaryKey;type:varchar(36)"`
	EventType    string    `gorm:"type:varchar(50);index"`
	UserID       string    `gorm:"type:varchar(36);index"`
	DepartmentID string    `gorm:"type:varchar(36);index"`
	IPAddress    string    `gorm:"type:varchar(45)"`
	UserAgent    string    `gorm:"type:text"`
	Status       string    `gorm:"type:varchar(20)"`
	ErrorMessage string    `gorm:"type:text"`
	RequestID    string    `gorm:"type:varchar(36)"`
	CreatedAt    time.Time `gorm:"index"`
}

type Session struct {
	gorm.Model
	ID           string `gorm:"primaryKey;type:varchar(36)"`
	UserID       string `gorm:"type:varchar(36);index"`
	DepartmentID string `gorm:"type:varchar(36)"`
	RefreshToken string `gorm:"type:varchar(500);unique"`
	IPAddress    string `gorm:"type:varchar(45)"`
	UserAgent    string `gorm:"type:text"`
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
