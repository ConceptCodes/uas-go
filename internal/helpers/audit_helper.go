package helpers

import (
	"fmt"
	"time"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog"
)

type AuditHelper struct {
	log      *zerolog.Logger
	auditLog repository.AuditLogRepository
}

func NewAuditHelper(log *zerolog.Logger, auditLog repository.AuditLogRepository) *AuditHelper {
	return &AuditHelper{
		log:      log,
		auditLog: auditLog,
	}
}

func (h *AuditHelper) LogAuditEntry(auditLog *models.AuditLog) error {
	if h.auditLog != nil {
		if err := h.auditLog.Create(auditLog); err != nil {
			return err
		}
	}

	h.log.Info().
		Str("auditId", auditLog.ID).
		Str("action", string(auditLog.Action)).
		Str("resource", string(auditLog.Resource)).
		Str("severity", string(auditLog.Severity)).
		Str("status", string(auditLog.Status)).
		Str("ipAddress", auditLog.IPAddress).
		Str("userId", safeString(auditLog.UserID)).
		Str("departmentId", safeString(auditLog.DepartmentID)).
		Str("deviceId", safeString(auditLog.DeviceID)).
		Str("sessionId", safeString(auditLog.SessionID)).
		Str("description", auditLog.Description).
		Time("timestamp", auditLog.Timestamp).
		Msg("Audit log entry")

	return nil
}

func (h *AuditHelper) LogSecurityEvent(userID, departmentID, deviceID, ipAddress, userAgent, description string) error {
	auditLog := &models.AuditLog{
		ID:           generateUUID(),
		UserID:       &userID,
		DepartmentID: &departmentID,
		Action:       models.AuditActionSuspiciousActivity,
		Resource:     models.AuditResourceUser,
		Severity:     models.AuditSeverityWarning,
		Status:       models.AuditStatusSuccess,
		Description:  description,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		DeviceID:     &deviceID,
		Timestamp:    time.Now(),
	}

	return h.LogAuditEntry(auditLog)
}

func (h *AuditHelper) LogAccountLock(userID, departmentID, ipAddress, userAgent, reason string) error {
	auditLog := &models.AuditLog{
		ID:           generateUUID(),
		UserID:       &userID,
		DepartmentID: &departmentID,
		Action:       models.AuditActionAccountLock,
		Resource:     models.AuditResourceUser,
		Severity:     models.AuditSeverityWarning,
		Status:       models.AuditStatusSuccess,
		Description:  fmt.Sprintf("Account locked: %s", reason),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Timestamp:    time.Now(),
	}

	return h.LogAuditEntry(auditLog)
}

func (h *AuditHelper) LogAccountUnlock(userID, departmentID, ipAddress, userAgent, reason string) error {
	auditLog := &models.AuditLog{
		ID:           generateUUID(),
		UserID:       &userID,
		DepartmentID: &departmentID,
		Action:       models.AuditActionAccountUnlock,
		Resource:     models.AuditResourceUser,
		Severity:     models.AuditSeverityInfo,
		Status:       models.AuditStatusSuccess,
		Description:  fmt.Sprintf("Account unlocked: %s", reason),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Timestamp:    time.Now(),
	}

	return h.LogAuditEntry(auditLog)
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func generateUUID() string {
	// Simple UUID generation - in production, use a proper UUID library
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
