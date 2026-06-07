package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uas/internal/helpers"
	"uas/internal/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type AuditMiddleware struct {
	log          *zerolog.Logger
	auditHelper  AuditHelperInterface
	excludePaths []string
}

// AuditHelperInterface defines the interface for audit logging to avoid import cycles
type AuditHelperInterface interface {
	LogAuditEntry(auditLog *models.AuditLog) error
}

type AuditLogEntry struct {
	Action      models.AuditAction     `json:"action"`
	Resource    models.AuditResource   `json:"resource"`
	ResourceID  *string                `json:"resourceId,omitempty"`
	Severity    models.AuditSeverity   `json:"severity"`
	Status      models.AuditStatus     `json:"status"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func NewAuditMiddleware(log *zerolog.Logger, auditHelper AuditHelperInterface) *AuditMiddleware {
	return &AuditMiddleware{
		log:         log,
		auditHelper: auditHelper,
		excludePaths: []string{
			"/health",
			"/metrics",
			"/debug",
			"/api/v1/health",
			"/api/v1/health/alive",
			"/api/v1/health/status",
		},
	}
}

func (m *AuditMiddleware) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapper := &auditResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		if m.shouldSkipAudit(r) {
			return
		}

		action, resource, severity := m.determineAuditAction(r, wrapper.statusCode)

		auditLog := &models.AuditLog{
			ID:          uuid.New().String(),
			Action:      action,
			Resource:    resource,
			Severity:    severity,
			Status:      m.determineStatus(wrapper.statusCode),
			Description: m.generateDescription(r, wrapper.statusCode),
			IPAddress:   m.getClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
			Timestamp:   start,
		}

		m.addContextInfo(auditLog, r)

		metadata := make(map[string]interface{})
		metadata["method"] = r.Method
		metadata["path"] = sanitizePath(r.URL.Path)
		metadata["query"] = sanitizeQueryString(r.URL.RawQuery)
		metadata["userAgent"] = r.Header.Get("User-Agent")
		metadata["responseTime"] = time.Since(start).Milliseconds()

		if wrapper.statusCode >= 400 && wrapper.body != nil && wrapper.body.Len() > 0 {
			bodyBytes := wrapper.body.Bytes()
			if len(bodyBytes) > 2048 {
				bodyBytes = bodyBytes[:2048]
			}
			var errorResponse map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil {
				metadata["error"] = errorResponse
			}
		}

		if metadataBytes, err := json.Marshal(metadata); err == nil {
			auditLog.Metadata = string(metadataBytes)
		}

		if err := m.auditHelper.LogAuditEntry(auditLog); err != nil {
			m.log.Error().Err(err).Str("auditId", auditLog.ID).Msg("Failed to log audit entry")
		}
	})
}

func (m *AuditMiddleware) shouldSkipAudit(r *http.Request) bool {
	path := r.URL.Path
	for _, excludePath := range m.excludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}
	return false
}

func (m *AuditMiddleware) determineAuditAction(r *http.Request, statusCode int) (models.AuditAction, models.AuditResource, models.AuditSeverity) {
	path := r.URL.Path
	method := r.Method

	// Authentication endpoints
	if strings.Contains(path, "/login") {
		if statusCode == http.StatusOK {
			return models.AuditActionLogin, models.AuditResourceAuth, models.AuditSeverityInfo
		}
		return models.AuditActionFailedLogin, models.AuditResourceAuth, models.AuditSeverityWarning
	}

	if strings.Contains(path, "/register") {
		return models.AuditActionRegister, models.AuditResourceUser, models.AuditSeverityInfo
	}

	if strings.Contains(path, "/logout") {
		return models.AuditActionLogout, models.AuditResourceSession, models.AuditSeverityInfo
	}

	if strings.Contains(path, "/refresh-token") {
		return models.AuditActionTokenRefresh, models.AuditResourceAuth, models.AuditSeverityInfo
	}

	// Password management endpoints
	if strings.Contains(path, "/forgot-password") {
		return models.AuditActionPasswordReset, models.AuditResourceUser, models.AuditSeverityInfo
	}

	if strings.Contains(path, "/reset-password") {
		if statusCode == http.StatusOK {
			return models.AuditActionPasswordChange, models.AuditResourceUser, models.AuditSeverityInfo
		}
		return models.AuditActionPasswordReset, models.AuditResourceUser, models.AuditSeverityWarning
	}

	// Email verification
	if strings.Contains(path, "/verify-email") {
		if statusCode == http.StatusOK {
			return models.AuditActionEmailVerify, models.AuditResourceUser, models.AuditSeverityInfo
		}
		return models.AuditActionEmailVerify, models.AuditResourceUser, models.AuditSeverityWarning
	}

	// Magic link authentication
	if strings.Contains(path, "/magic-link") {
		if strings.Contains(path, "/send") {
			return models.AuditActionLogin, models.AuditResourceAuth, models.AuditSeverityInfo
		}
		if strings.Contains(path, "/verify") && statusCode == http.StatusOK {
			return models.AuditActionMagicLinkLogin, models.AuditResourceAuth, models.AuditSeverityInfo
		}
		return models.AuditActionMagicLinkLogin, models.AuditResourceAuth, models.AuditSeverityWarning
	}

	// OTP authentication
	if strings.Contains(path, "/otp") {
		if strings.Contains(path, "/send") {
			return models.AuditActionOTPSend, models.AuditResourceAuth, models.AuditSeverityInfo
		}
		if strings.Contains(path, "/verify") {
			if statusCode == http.StatusOK {
				return models.AuditActionOTPVerify, models.AuditResourceAuth, models.AuditSeverityInfo
			}
			return models.AuditActionOTPVerify, models.AuditResourceAuth, models.AuditSeverityWarning
		}
	}

	// Data access/modification - determine by HTTP method and resource
	if strings.Contains(path, "/users") {
		switch method {
		case http.MethodGet:
			return models.AuditActionDataAccess, models.AuditResourceUser, models.AuditSeverityInfo
		case http.MethodPost, http.MethodPut:
			return models.AuditActionDataModify, models.AuditResourceUser, models.AuditSeverityInfo
		case http.MethodDelete:
			return models.AuditActionDataDelete, models.AuditResourceUser, models.AuditSeverityWarning
		}
	}

	if strings.Contains(path, "/tenants") || strings.Contains(path, "/departments") {
		switch method {
		case http.MethodGet:
			return models.AuditActionDataAccess, models.AuditResourceDepartment, models.AuditSeverityInfo
		case http.MethodPost, http.MethodPut:
			return models.AuditActionDataModify, models.AuditResourceDepartment, models.AuditSeverityInfo
		case http.MethodDelete:
			return models.AuditActionDataDelete, models.AuditResourceDepartment, models.AuditSeverityWarning
		}
	}

	// Default to data access for unknown endpoints
	return models.AuditActionDataAccess, models.AuditResourceUser, models.AuditSeverityInfo
}

func (m *AuditMiddleware) determineStatus(statusCode int) models.AuditStatus {
	if statusCode >= 200 && statusCode < 300 {
		return models.AuditStatusSuccess
	}
	return models.AuditStatusFailure
}

func (m *AuditMiddleware) generateDescription(r *http.Request, statusCode int) string {
	path := r.URL.Path
	method := r.Method

	if statusCode >= 200 && statusCode < 300 {
		return fmt.Sprintf("%s %s successful", method, path)
	}

	return fmt.Sprintf("%s %s failed with status %d", method, path, statusCode)
}

func (m *AuditMiddleware) getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (m *AuditMiddleware) addContextInfo(auditLog *models.AuditLog, r *http.Request) {
	// Extract user information from context
	if uid := helpers.GetUserId(r); uid != "" {
		auditLog.UserID = &uid
	}

	if did := helpers.GetDepartmentId(r); did != "" {
		auditLog.DepartmentID = &did
	}

	if sessionID := r.Context().Value("sessionId"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			auditLog.SessionID = &sid
		}
	}

	if deviceID := r.Context().Value("deviceId"); deviceID != nil {
		if did, ok := deviceID.(string); ok {
			auditLog.DeviceID = &did
		}
	}
}

func sanitizeQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}

	sensitiveKeys := map[string]struct{}{
		"token":         {},
		"access_token":  {},
		"refresh_token": {},
		"password":      {},
		"otp":           {},
	}

	for key := range values {
		if _, ok := sensitiveKeys[strings.ToLower(key)]; ok {
			values.Set(key, "REDACTED")
		}
	}

	return values.Encode()
}

var sensitivePathSegments = map[string]struct{}{
	"reset-password":  {},
	"verify-email":    {},
	"verify-magic":    {},
	"magic-link":      {},
}

func sanitizePath(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if _, sensitive := sensitivePathSegments[strings.ToLower(p)]; sensitive && i+1 < len(parts) {
			for j := i + 1; j < len(parts); j++ {
				parts[j] = "REDACTED"
			}
			break
		}
		if len(p) >= 32 && isLikelyToken(p) {
			parts[i] = "REDACTED"
		}
	}
	return strings.Join(parts, "/")
}

func isLikelyToken(s string) bool {
	if len(s) < 16 {
		return false
	}
	hex, alnum := 0, 0
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hex++
		}
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			alnum++
		} else {
			return false
		}
	}
	return hex == alnum
}

// auditResponseWriter wraps http.ResponseWriter to capture status code and body.
// The body buffer is allocated lazily when first written to.
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (w *auditResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode >= 400 && w.body == nil {
		w.body = bytes.NewBuffer(nil)
	}
	if w.body != nil {
		_, _ = w.body.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *auditResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
