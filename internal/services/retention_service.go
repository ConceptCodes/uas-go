package services

import (
	"time"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog"
)

type RetentionPolicy struct {
	Action        models.AuditAction `json:"action"`
	RetentionDays int                `json:"retentionDays"`
	Description   string             `json:"description"`
}

type RetentionService struct {
	auditLogRepo repository.AuditLogRepository
	log          *zerolog.Logger
}

func NewRetentionService(auditLogRepo repository.AuditLogRepository, log *zerolog.Logger) *RetentionService {
	return &RetentionService{
		auditLogRepo: auditLogRepo,
		log:          log,
	}
}

// GetDefaultRetentionPolicies returns the default retention policies
func (s *RetentionService) GetDefaultRetentionPolicies() []RetentionPolicy {
	return []RetentionPolicy{
		{
			Action:        models.AuditActionLogin,
			RetentionDays: 90,
			Description:   "User login attempts are retained for 90 days",
		},
		{
			Action:        models.AuditActionLogout,
			RetentionDays: 90,
			Description:   "User logout events are retained for 90 days",
		},
		{
			Action:        models.AuditActionRegister,
			RetentionDays: 365,
			Description:   "User registration events are retained for 1 year",
		},
		{
			Action:        models.AuditActionPasswordReset,
			RetentionDays: 180,
			Description:   "Password reset events are retained for 6 months",
		},
		{
			Action:        models.AuditActionPasswordChange,
			RetentionDays: 365,
			Description:   "Password change events are retained for 1 year",
		},
		{
			Action:        models.AuditActionEmailVerify,
			RetentionDays: 180,
			Description:   "Email verification events are retained for 6 months",
		},
		{
			Action:        models.AuditActionMagicLinkLogin,
			RetentionDays: 7,
			Description:   "Magic link login events are retained for 7 days",
		},
		{
			Action:        models.AuditActionOTPSend,
			RetentionDays: 30,
			Description:   "OTP send events are retained for 30 days",
		},
		{
			Action:        models.AuditActionOTPVerify,
			RetentionDays: 30,
			Description:   "OTP verification events are retained for 30 days",
		},
		{
			Action:        models.AuditActionTokenRefresh,
			RetentionDays: 90,
			Description:   "Token refresh events are retained for 90 days",
		},
		{
			Action:        models.AuditActionAccountLock,
			RetentionDays: 365,
			Description:   "Account lock events are retained for 1 year",
		},
		{
			Action:        models.AuditActionAccountUnlock,
			RetentionDays: 365,
			Description:   "Account unlock events are retained for 1 year",
		},
		{
			Action:        models.AuditActionFailedLogin,
			RetentionDays: 90,
			Description:   "Failed login attempts are retained for 90 days",
		},
		{
			Action:        models.AuditActionSuspiciousActivity,
			RetentionDays: 365,
			Description:   "Suspicious activity events are retained for 1 year",
		},
		{
			Action:        models.AuditActionDataAccess,
			RetentionDays: 90,
			Description:   "Data access events are retained for 90 days",
		},
		{
			Action:        models.AuditActionDataModify,
			RetentionDays: 365,
			Description:   "Data modification events are retained for 1 year",
		},
		{
			Action:        models.AuditActionDataDelete,
			RetentionDays: 365,
			Description:   "Data deletion events are retained for 1 year",
		},
	}
}

// CleanupOldAuditLogs removes audit logs older than the retention period
func (s *RetentionService) CleanupOldAuditLogs() error {
	s.log.Info().Msg("Starting audit log cleanup based on retention policies")

	policies := s.GetDefaultRetentionPolicies()
	totalDeleted := int64(0)

	for _, policy := range policies {
		cutoffTime := time.Now().AddDate(0, 0, -policy.RetentionDays)

		// Delete old audit logs for this action type
		deleted, err := s.auditLogRepo.DeleteOlderThanByAction(policy.Action, time.Duration(policy.RetentionDays)*24*time.Hour)
		if err != nil {
			s.log.Error().Err(err).
				Str("action", string(policy.Action)).
				Int("retentionDays", policy.RetentionDays).
				Msg("Failed to cleanup old audit logs")
			return err
		}

		totalDeleted += deleted
		s.log.Info().
			Str("action", string(policy.Action)).
			Int("retentionDays", policy.RetentionDays).
			Int64("deletedCount", deleted).
			Time("cutoffTime", cutoffTime).
			Msg("Audit log cleanup completed for action")
	}

	s.log.Info().
		Int64("totalDeleted", totalDeleted).
		Msg("Audit log retention cleanup completed")

	return nil
}

// CleanupOldAuditLogsByAction removes audit logs older than the retention period for a specific action
func (s *RetentionService) CleanupOldAuditLogsByAction(action models.AuditAction, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	deleted, err := s.auditLogRepo.DeleteOlderThanByAction(action, time.Duration(retentionDays)*24*time.Hour)
	if err != nil {
		s.log.Error().Err(err).
			Str("action", string(action)).
			Int("retentionDays", retentionDays).
			Msg("Failed to cleanup old audit logs for specific action")
		return err
	}

	s.log.Info().
		Str("action", string(action)).
		Int("retentionDays", retentionDays).
		Int64("deletedCount", deleted).
		Time("cutoffTime", cutoffTime).
		Msg("Audit log cleanup completed for specific action")

	return nil
}

// GetRetentionStats returns statistics about audit log retention
func (s *RetentionService) GetRetentionStats() (map[string]interface{}, error) {
	policies := s.GetDefaultRetentionPolicies()
	stats := make(map[string]interface{})

	for _, policy := range policies {
		cutoffTime := time.Now().AddDate(0, 0, -policy.RetentionDays)

		// Count logs older than retention period
		filter := &models.AuditLogFilter{
			Action:  &policy.Action,
			EndDate: &cutoffTime,
		}

		count, err := s.auditLogRepo.Count(filter)
		if err != nil {
			s.log.Error().Err(err).
				Str("action", string(policy.Action)).
				Msg("Failed to count old audit logs")
			return nil, err
		}

		stats[string(policy.Action)] = map[string]interface{}{
			"retentionDays":  policy.RetentionDays,
			"description":    policy.Description,
			"olderThanCount": count,
			"cutoffTime":     cutoffTime,
		}
	}

	stats["generatedAt"] = time.Now()
	return stats, nil
}
