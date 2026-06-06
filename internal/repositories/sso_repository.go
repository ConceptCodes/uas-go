package repository

import (
	"time"
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"gorm.io/gorm"
)

type IdentityProviderRepository interface {
	Create(provider *models.IdentityProvider) error
	FindByID(id, departmentID string) (*models.IdentityProvider, error)
	FindByDepartment(departmentID string) ([]models.IdentityProvider, error)
	FindByDepartmentAndType(departmentID string, providerType models.IdentityProviderType) ([]models.IdentityProvider, error)
	Update(provider *models.IdentityProvider) error
	Delete(id, departmentID string) error
}

type UserIdentityRepository interface {
	Create(identity *models.UserIdentity) error
	FindByID(id, departmentID string) (*models.UserIdentity, error)
	FindByUserID(userID, departmentID string) ([]models.UserIdentity, error)
	FindByProvider(providerID, providerUserID, departmentID string) (*models.UserIdentity, error)
	FindByProviderEmail(providerEmail, departmentID string) (*models.UserIdentity, error)
	Delete(id, departmentID string) error
	DeleteByUserID(userID, departmentID string) error
	UpdateLastLogin(id, departmentID string) error
}

type GormIdentityProviderRepository struct {
	db *gorm.DB
}

type GormUserIdentityRepository struct {
	db *gorm.DB
}

func NewGormIdentityProviderRepository(db *gorm.DB) IdentityProviderRepository {
	return &GormIdentityProviderRepository{db: db}
}

func NewGormUserIdentityRepository(db *gorm.DB) UserIdentityRepository {
	return &GormUserIdentityRepository{db: db}
}

func (r *GormIdentityProviderRepository) Create(provider *models.IdentityProvider) error {
	return r.db.Create(provider).Error
}

func (r *GormIdentityProviderRepository) FindByID(id, departmentID string) (*models.IdentityProvider, error) {
	var provider models.IdentityProvider
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("id = ?", id).First(&provider).Error
	return &provider, err
}

func (r *GormIdentityProviderRepository) FindByDepartment(departmentID string) ([]models.IdentityProvider, error) {
	var providers []models.IdentityProvider
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Find(&providers).Error
	return providers, err
}

func (r *GormIdentityProviderRepository) FindByDepartmentAndType(departmentID string, providerType models.IdentityProviderType) ([]models.IdentityProvider, error) {
	var providers []models.IdentityProvider
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("provider_type = ?", providerType).Find(&providers).Error
	return providers, err
}

func (r *GormIdentityProviderRepository) Update(provider *models.IdentityProvider) error {
	return r.db.Save(provider).Error
}

func (r *GormIdentityProviderRepository) Delete(id, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).Delete(&models.IdentityProvider{}, id).Error
}

func (r *GormUserIdentityRepository) Create(identity *models.UserIdentity) error {
	return r.db.Create(identity).Error
}

func (r *GormUserIdentityRepository) FindByID(id, departmentID string) (*models.UserIdentity, error) {
	var identity models.UserIdentity
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("id = ?", id).First(&identity).Error
	return &identity, err
}

func (r *GormUserIdentityRepository) FindByUserID(userID, departmentID string) ([]models.UserIdentity, error) {
	var identities []models.UserIdentity
	err := r.db.Scopes(mysql.TenantScope(departmentID)).Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *GormUserIdentityRepository) FindByProvider(providerID, providerUserID, departmentID string) (*models.UserIdentity, error) {
	var identity models.UserIdentity
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("provider_id = ? AND provider_user_id = ?", providerID, providerUserID).
		First(&identity).Error
	return &identity, err
}

func (r *GormUserIdentityRepository) FindByProviderEmail(providerEmail, departmentID string) (*models.UserIdentity, error) {
	var identity models.UserIdentity
	err := r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("provider_email = ?", providerEmail).
		First(&identity).Error
	return &identity, err
}

func (r *GormUserIdentityRepository) Delete(id, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).Delete(&models.UserIdentity{}, id).Error
}

func (r *GormUserIdentityRepository) DeleteByUserID(userID, departmentID string) error {
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Where("user_id = ?", userID).
		Delete(&models.UserIdentity{}).Error
}

func (r *GormUserIdentityRepository) UpdateLastLogin(id, departmentID string) error {
	now := time.Now()
	return r.db.Scopes(mysql.TenantScope(departmentID)).
		Model(&models.UserIdentity{}).
		Where("id = ?", id).
		Update("last_login_at", now).Error
}