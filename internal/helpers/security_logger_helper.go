package helpers

import (
	"net"
	"net/http"
	"uas/internal/constants"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type SecurityLoggerHelper struct {
	log       *zerolog.Logger
	auditRepo repository.SecurityAuditRepository
}

func NewSecurityLoggerHelper(log *zerolog.Logger, auditRepo repository.SecurityAuditRepository) *SecurityLoggerHelper {
	return &SecurityLoggerHelper{
		log:       log,
		auditRepo: auditRepo,
	}
}

func (s *SecurityLoggerHelper) LogAuthEvent(r *http.Request, eventType string, userID string, departmentID string, status string, errorMsg string) {
	go func() {
		event := &models.SecurityEvent{
			ID:           uuid.New().String(),
			EventType:    eventType,
			UserID:       userID,
			DepartmentID: departmentID,
			IPAddress:    s.getClientIP(r),
			UserAgent:    r.UserAgent(),
			Status:       status,
			ErrorMessage: errorMsg,
			RequestID:    r.Header.Get(constants.TraceIdHeader),
		}

		if err := s.auditRepo.Create(event); err != nil {
			s.log.Error().Err(err).Str("event_type", eventType).Msg("Failed to log security event")
		}
	}()
}

func (s *SecurityLoggerHelper) getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
