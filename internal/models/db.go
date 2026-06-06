package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string
type AuthModelType string
type MfaFactorType string
type MfaChallengeState string
type IdentityProviderType string
type WebhookEventType string
type WebhookDeliveryStatus string

const (
	User  Role = "user"
	Admin Role = "admin"

	PlatformAdmin Role = "platform_admin"
	TenantAdmin   Role = "tenant_admin"
	TenantMember  Role = "tenant_member"
	TenantViewer  Role = "tenant_viewer"
)

const (
	ResetPassword AuthModelType = "reset-password"
	MagicLink     AuthModelType = "magic-link"
)

const (
	MfaFactorTOTP MfaFactorType = "totp"
	MfaFactorSMS  MfaFactorType = "sms"
	MfaFactorEmail MfaFactorType = "email"
)

const (
	MfaChallengePending  MfaChallengeState = "pending"
	MfaChallengeVerified MfaChallengeState = "verified"
	MfaChallengeExpired  MfaChallengeState = "expired"
)

const (
	IDPGoogle IdentityProviderType = "google"
	IDPGithub IdentityProviderType = "github"
	IDPOIDC   IdentityProviderType = "oidc"
	IDPSAML   IdentityProviderType = "saml"
)

const (
	WebhookEventUserCreated     WebhookEventType = "user.created"
	WebhookEventUserUpdated     WebhookEventType = "user.updated"
	WebhookEventUserDeleted     WebhookEventType = "user.deleted"
	WebhookEventUserLogin       WebhookEventType = "user.login"
	WebhookEventUserLogout      WebhookEventType = "user.logout"
	WebhookEventSessionRevoked  WebhookEventType = "session.revoked"
	WebhookEventRoleAssigned    WebhookEventType = "role.assigned"
	WebhookEventRoleRevoked     WebhookEventType = "role.revoked"
	WebhookEventMFAEnabled      WebhookEventType = "mfa.enabled"
	WebhookEventMFADisabled     WebhookEventType = "mfa.disabled"
	WebhookEventSSOLinked       WebhookEventType = "sso.linked"
	WebhookEventSSOUnlinked     WebhookEventType = "sso.unlinked"
	WebhookEventPasswordChanged WebhookEventType = "password.changed"
	WebhookEventTenantCreated   WebhookEventType = "tenant.created"
	WebhookEventTenantDeleted   WebhookEventType = "tenant.deleted"
)

const (
	WebhookDeliveryPending  WebhookDeliveryStatus = "pending"
	WebhookDeliverySuccess  WebhookDeliveryStatus = "success"
	WebhookDeliveryFailed   WebhookDeliveryStatus = "failed"
)

type DepartmentModel struct {
	gorm.Model
	ID               string           `gorm:"primaryKey;type:varchar(36);unique_index"`
	Name             string           `gorm:"type:varchar(100);unique_index"`
	Secret           string           `gorm:"type:varchar(255)"`
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
	UserID       string        `gorm:"type:varchar(36);unique_index"`
	Token        string        `gorm:"primaryKey;type:varchar(128)"`
	Type         AuthModelType `gorm:"primaryKey;type:varchar(36)"`
	DepartmentID string        `gorm:"type:varchar(36);index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
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
	DepartmentID string `gorm:"type:varchar(36);index"`
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

type MfaFactor struct {
	ID           string        `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string        `gorm:"type:varchar(36);index;not null" json:"userId"`
	DepartmentID string        `gorm:"type:varchar(36);index;not null" json:"departmentId"`
	FactorType   MfaFactorType `gorm:"type:varchar(20);not null" json:"factorType"`
	Secret       string        `gorm:"type:varchar(255);not null" json:"-"`
	Name         string        `gorm:"type:varchar(100)" json:"name"`
	IsPrimary    bool          `gorm:"default:false" json:"isPrimary"`
	BackupCodes  string        `gorm:"type:json" json:"-"`
	LastUsedAt   *time.Time    `json:"lastUsedAt"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

func (m *MfaFactor) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type MfaChallenge struct {
	ID           string            `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string            `gorm:"type:varchar(36);index;not null" json:"userId"`
	DepartmentID string            `gorm:"type:varchar(36);index;not null" json:"departmentId"`
	FactorID     *string           `gorm:"type:varchar(36)" json:"factorId"`
	ChallengeType MfaFactorType    `gorm:"type:varchar(20);not null" json:"challengeType"`
	State        MfaChallengeState `gorm:"type:varchar(20);default:pending" json:"state"`
	TempToken    string            `gorm:"type:varchar(255);unique" json:"-"`
	Code         string            `gorm:"type:varchar(10)" json:"-"`
	ExpiresAt    time.Time         `json:"expiresAt"`
	CreatedAt    time.Time         `json:"createdAt"`
}

func (m *MfaChallenge) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type IdentityProvider struct {
	ID               string               `gorm:"primaryKey;type:varchar(36)" json:"id"`
	DepartmentID     string               `gorm:"type:varchar(36);index;not null" json:"departmentId"`
	Name             string               `gorm:"type:varchar(100);not null" json:"name"`
	ProviderType     IdentityProviderType `gorm:"type:varchar(20);not null" json:"providerType"`
	ClientID         string               `gorm:"type:varchar(255);not null" json:"clientId"`
	ClientSecret     EncryptedField       `gorm:"type:text" json:"-"`
	IssuerURL        string               `gorm:"type:varchar(255)" json:"issuerUrl"`
	AuthorizationURL string               `gorm:"type:varchar(255)" json:"authorizationUrl"`
	TokenURL         string               `gorm:"type:varchar(255)" json:"tokenUrl"`
	UserInfoURL      string               `gorm:"type:varchar(255)" json:"userInfoUrl"`
	JWKSURI          string               `gorm:"type:varchar(255)" json:"jwksUri"`
	MetadataURL      string               `gorm:"type:varchar(255)" json:"metadataUrl"`
	RedirectURLs     string               `gorm:"type:json" json:"redirectUrls"`
	Scopes           string               `gorm:"type:json" json:"scopes"`
	Enabled          bool                 `gorm:"default:true" json:"enabled"`
	CreatedAt        time.Time            `json:"createdAt"`
	UpdatedAt        time.Time            `json:"updatedAt"`
}

func (i *IdentityProvider) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

type UserIdentity struct {
	ID              string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID          string    `gorm:"type:varchar(36);index;not null" json:"userId"`
	DepartmentID    string    `gorm:"type:varchar(36);index;not null" json:"departmentId"`
	ProviderID      string    `gorm:"type:varchar(36);index;not null" json:"providerId"`
	ProviderUserID  string    `gorm:"type:varchar(255);not null" json:"providerUserId"`
	ProviderEmail   string    `gorm:"type:varchar(255)" json:"providerEmail"`
	AccessToken     string    `gorm:"type:text" json:"-"`
	RefreshToken    string    `gorm:"type:text" json:"-"`
	LastLoginAt     *time.Time `json:"lastLoginAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (u *UserIdentity) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

type WebhookEndpoint struct {
	ID           string   `gorm:"primaryKey;type:varchar(36)" json:"id"`
	DepartmentID string   `gorm:"type:varchar(36);index;not null" json:"departmentId"`
	Name         string   `gorm:"type:varchar(100);not null" json:"name"`
	URL          string   `gorm:"type:varchar(500);not null" json:"url"`
	Secret       string   `gorm:"type:varchar(255);not null" json:"-"`
	Events       string   `gorm:"type:json" json:"events"`
	IsActive     bool     `gorm:"default:true" json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (w *WebhookEndpoint) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return nil
}

type WebhookDelivery struct {
	ID           string               `gorm:"primaryKey;type:varchar(36)" json:"id"`
	EndpointID   string               `gorm:"type:varchar(36);index;not null" json:"endpointId"`
	DepartmentID string               `gorm:"type:varchar(36);index;not null" json:"departmentId"`
	Event        WebhookEventType     `gorm:"type:varchar(50);not null" json:"event"`
	Payload      string               `gorm:"type:json" json:"payload"`
	ResponseCode int                  `json:"responseCode"`
	ResponseBody string               `gorm:"type:text" json:"responseBody"`
	Status       WebhookDeliveryStatus `gorm:"type:varchar(20);default:pending" json:"status"`
	Attempt      int                  `gorm:"default:0" json:"attempt"`
	MaxAttempts  int                  `gorm:"default:5" json:"maxAttempts"`
	NextRetryAt  *time.Time           `json:"nextRetryAt"`
	CreatedAt    time.Time            `json:"createdAt"`
	UpdatedAt    time.Time            `json:"updatedAt"`
}

func (w *WebhookDelivery) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return nil
}


