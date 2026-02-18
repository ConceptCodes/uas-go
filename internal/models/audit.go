package models

import (
	"database/sql/driver"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionLogin              AuditAction = "login"
	AuditActionLogout             AuditAction = "logout"
	AuditActionRegister           AuditAction = "register"
	AuditActionPasswordReset      AuditAction = "password_reset"
	AuditActionPasswordChange     AuditAction = "password_change"
	AuditActionEmailVerify        AuditAction = "email_verify"
	AuditActionMagicLinkLogin     AuditAction = "magic_link_login"
	AuditActionOTPSend            AuditAction = "otp_send"
	AuditActionOTPVerify          AuditAction = "otp_verify"
	AuditActionTokenRefresh       AuditAction = "token_refresh"
	AuditActionAccountLock        AuditAction = "account_lock"
	AuditActionAccountUnlock      AuditAction = "account_unlock"
	AuditActionFailedLogin        AuditAction = "failed_login"
	AuditActionSuspiciousActivity AuditAction = "suspicious_activity"
	AuditActionDataAccess         AuditAction = "data_access"
	AuditActionDataModify         AuditAction = "data_modify"
	AuditActionDataDelete         AuditAction = "data_delete"
)

// AuditResource represents the type of resource being accessed
type AuditResource string

const (
	AuditResourceUser       AuditResource = "user"
	AuditResourceAuth       AuditResource = "auth"
	AuditResourceSession    AuditResource = "session"
	AuditResourceDepartment AuditResource = "department"
	AuditResourceRole       AuditResource = "role"
	AuditResourcePermission AuditResource = "permission"
)

// AuditSeverity represents the severity level of the audit event
type AuditSeverity string

const (
	AuditSeverityInfo     AuditSeverity = "info"
	AuditSeverityWarning  AuditSeverity = "warning"
	AuditSeverityError    AuditSeverity = "error"
	AuditSeverityCritical AuditSeverity = "critical"
)

// AuditStatus represents the status of the audited action
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
	AuditStatusPending AuditStatus = "pending"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string        `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       *string       `gorm:"type:varchar(36);index" json:"userId,omitempty"`
	DepartmentID *string       `gorm:"type:varchar(36);index" json:"departmentId,omitempty"`
	Action       AuditAction   `gorm:"type:varchar(50);index" json:"action"`
	Resource     AuditResource `gorm:"type:varchar(50);index" json:"resource"`
	ResourceID   *string       `gorm:"type:varchar(36);index" json:"resourceId,omitempty"`
	Severity     AuditSeverity `gorm:"type:varchar(20);index" json:"severity"`
	Status       AuditStatus   `gorm:"type:varchar(20);index" json:"status"`
	Description  string        `gorm:"type:text" json:"description"`
	IPAddress    string        `gorm:"type:varchar(45);index" json:"ipAddress"`
	UserAgent    string        `gorm:"type:text" json:"userAgent"`
	DeviceID     *string       `gorm:"type:varchar(36);index" json:"deviceId,omitempty"`
	SessionID    *string       `gorm:"type:varchar(36);index" json:"sessionId,omitempty"`
	Metadata     string        `gorm:"type:json" json:"metadata"`
	Timestamp    time.Time     `gorm:"index" json:"timestamp"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

// AuditLogFilter represents filters for querying audit logs
type AuditLogFilter struct {
	UserID       *string        `json:"userId,omitempty"`
	DepartmentID *string        `json:"departmentId,omitempty"`
	Action       *AuditAction   `json:"action,omitempty"`
	Resource     *AuditResource `json:"resource,omitempty"`
	Severity     *AuditSeverity `json:"severity,omitempty"`
	Status       *AuditStatus   `json:"status,omitempty"`
	IPAddress    *string        `json:"ipAddress,omitempty"`
	DeviceID     *string        `json:"deviceId,omitempty"`
	SessionID    *string        `json:"sessionId,omitempty"`
	StartDate    *time.Time     `json:"startDate,omitempty"`
	EndDate      *time.Time     `json:"endDate,omitempty"`
	Limit        *int           `json:"limit,omitempty"`
	Offset       *int           `json:"offset,omitempty"`
}

// AuditLogQuery represents the result of an audit log query
type AuditLogQuery struct {
	Total   int64          `json:"total"`
	Results []AuditLog     `json:"results"`
	Filter  AuditLogFilter `json:"filter,omitempty"`
}

// TableName returns the table name for AuditLog
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate sets the ID and timestamps before creating an audit log
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	now := time.Now()
	a.Timestamp = now
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// BeforeUpdate updates the timestamp before updating an audit log
func (a *AuditLog) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = time.Now()
	return nil
}

// Scan implements the sql.Scanner interface for AuditAction
func (a *AuditAction) Scan(value interface{}) error {
	if value == nil {
		*a = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*a = AuditAction(v)
	case []byte:
		*a = AuditAction(v)
	default:
		return nil
	}
	return nil
}

// Value implements the driver.Valuer interface for AuditAction
func (a AuditAction) Value() (driver.Value, error) {
	return string(a), nil
}

// Scan implements the sql.Scanner interface for AuditResource
func (r *AuditResource) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*r = AuditResource(v)
	case []byte:
		*r = AuditResource(v)
	default:
		return nil
	}
	return nil
}

// Value implements the driver.Valuer interface for AuditResource
func (r AuditResource) Value() (driver.Value, error) {
	return string(r), nil
}

// Scan implements the sql.Scanner interface for AuditSeverity
func (s *AuditSeverity) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = AuditSeverity(v)
	case []byte:
		*s = AuditSeverity(v)
	default:
		return nil
	}
	return nil
}

// Value implements the driver.Valuer interface for AuditSeverity
func (s AuditSeverity) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements the sql.Scanner interface for AuditStatus
func (s *AuditStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = AuditStatus(v)
	case []byte:
		*s = AuditStatus(v)
	default:
		return nil
	}
	return nil
}

// Value implements the driver.Valuer interface for AuditStatus
func (s AuditStatus) Value() (driver.Value, error) {
	return string(s), nil
}
