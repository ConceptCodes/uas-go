package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

func (h *AuditHandler) enforceTenantScope(r *http.Request, filter *models.AuditLogFilter) {
	contextTenant := helpers.GetDepartmentId(r)
	if contextTenant != "" && (filter.DepartmentID == nil || *filter.DepartmentID == "") {
		filter.DepartmentID = &contextTenant
	}
}

type AuditHandler struct {
	auditLogRepo    repository.AuditLogRepository
	log             *zerolog.Logger
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
}

func NewAuditHandler(
	auditLogRepo repository.AuditLogRepository,
	log *zerolog.Logger,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
) *AuditHandler {
	return &AuditHandler{
		auditLogRepo:    auditLogRepo,
		log:             log,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
	}
}

// GetAuditLogs godoc
// @Summary Get audit logs
// @Description Retrieve paginated audit logs with optional filtering
// @Tags audit
// @Accept json
// @Produce json
// @Param userId query string false "Filter by user ID"
// @Param departmentId query string false "Filter by department ID"
// @Param action query string false "Filter by action type"
// @Param resource query string false "Filter by resource type"
// @Param severity query string false "Filter by severity level"
// @Param status query string false "Filter by status"
// @Param ipAddress query string false "Filter by IP address"
// @Param deviceId query string false "Filter by device ID"
// @Param sessionId query string false "Filter by session ID"
// @Param startDate query string false "Filter by start date (RFC3339)"
// @Param endDate query string false "Filter by end date (RFC3339)"
// @Param limit query int false "Maximum number of results to return"
// @Param offset query int false "Number of results to skip"
// @Success 200 {object} models.AuditLogQuery
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/audit/logs [get]
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := &models.AuditLogFilter{}

	if userID := r.URL.Query().Get("userId"); userID != "" {
		filter.UserID = &userID
	}

	if departmentID := r.URL.Query().Get("departmentId"); departmentID != "" {
		filter.DepartmentID = &departmentID
	}

	if action := r.URL.Query().Get("action"); action != "" {
		act := models.AuditAction(action)
		filter.Action = &act
	}

	if resource := r.URL.Query().Get("resource"); resource != "" {
		res := models.AuditResource(resource)
		filter.Resource = &res
	}

	if severity := r.URL.Query().Get("severity"); severity != "" {
		sev := models.AuditSeverity(severity)
		filter.Severity = &sev
	}

	if status := r.URL.Query().Get("status"); status != "" {
		st := models.AuditStatus(status)
		filter.Status = &st
	}

	if ipAddress := r.URL.Query().Get("ipAddress"); ipAddress != "" {
		filter.IPAddress = &ipAddress
	}

	if deviceID := r.URL.Query().Get("deviceId"); deviceID != "" {
		filter.DeviceID = &deviceID
	}

	if sessionID := r.URL.Query().Get("sessionId"); sessionID != "" {
		filter.SessionID = &sessionID
	}

	// Parse pagination parameters
	limit := 100 // default limit
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o > 0 {
			offset = o
		}
	}

	filter.Limit = &limit
	filter.Offset = &offset

	h.enforceTenantScope(r, filter)

	// Query audit logs
	result, err := h.auditLogRepo.FindMany(filter)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to query audit logs")
		h.responseHelper.SendErrorResponse(w, "Failed to query audit logs", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Audit logs retrieved successfully", result)
}

// GetAuditLogByID godoc
// @Summary Get audit log by ID
// @Description Retrieve a specific audit log entry by its ID
// @Tags audit
// @Accept json
// @Produce json
// @Param id path string true "Audit log ID"
// @Success 200 {object} models.AuditLog
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/audit/logs/{id} [get]
func (h *AuditHandler) GetAuditLogByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok || id == "" {
		h.responseHelper.SendErrorResponse(w, "Audit log ID is required", constants.BadRequest, nil)
		return
	}

	filter := &models.AuditLogFilter{}
	h.enforceTenantScope(r, filter)

	auditLog, err := h.auditLogRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.responseHelper.SendErrorResponse(w, "Audit log not found", constants.NotFound, err)
			return
		}
		h.log.Error().Err(err).Str("auditId", id).Msg("Failed to get audit log")
		h.responseHelper.SendErrorResponse(w, "Failed to get audit log", constants.InternalServerError, err)
		return
	}

	if filter.DepartmentID != nil && *filter.DepartmentID != "" && auditLog.DepartmentID != nil && *auditLog.DepartmentID != *filter.DepartmentID {
		h.responseHelper.SendErrorResponse(w, "Audit log not found", constants.NotFound, nil)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Audit log retrieved successfully", auditLog)
}

// GetAuditStats godoc
// @Summary Get audit statistics
// @Description Get audit log statistics for a given time range
// @Tags audit
// @Accept json
// @Produce json
// @Param startDate query string false "Start date for statistics (RFC3339)"
// @Param endDate query string false "End date for statistics (RFC3339)"
// @Param userId query string false "Filter by user ID"
// @Param departmentId query string false "Filter by department ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/audit/stats [get]
func (h *AuditHandler) GetAuditStats(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := &models.AuditLogFilter{}

	if userID := r.URL.Query().Get("userId"); userID != "" {
		filter.UserID = &userID
	}

	if departmentID := r.URL.Query().Get("departmentId"); departmentID != "" {
		filter.DepartmentID = &departmentID
	}

	// Parse date range
	var startDate, endDate *time.Time
	if startDateStr := r.URL.Query().Get("startDate"); startDateStr != "" {
		if sd, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = &sd
		}
	}

	if endDateStr := r.URL.Query().Get("endDate"); endDateStr != "" {
		if ed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = &ed
		}
	}

	filter.StartDate = startDate
	filter.EndDate = endDate

	h.enforceTenantScope(r, filter)

	// Get total count
	total, err := h.auditLogRepo.Count(filter)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get audit statistics")
		h.responseHelper.SendErrorResponse(w, "Failed to get audit statistics", constants.InternalServerError, err)
		return
	}

	// Get detailed statistics
	stats := map[string]interface{}{
		"totalLogs": total,
		"dateRange": map[string]interface{}{
			"startDate": startDate,
			"endDate":   endDate,
		},
		"filters": map[string]interface{}{
			"userId":       filter.UserID,
			"departmentId": filter.DepartmentID,
		},
	}

	h.responseHelper.SendSuccessResponse(w, "Audit statistics retrieved successfully", stats)
}

// RegisterRoutes registers audit log routes
func (h *AuditHandler) RegisterRoutes(router *mux.Router) {
	// Apply authentication middleware for audit endpoints
	// These endpoints require appropriate permissions
	router.HandleFunc("/api/v1/audit/logs", h.GetAuditLogs).Methods("GET")
	router.HandleFunc("/api/v1/audit/logs/{id}", h.GetAuditLogByID).Methods("GET")
	router.HandleFunc("/api/v1/audit/stats", h.GetAuditStats).Methods("GET")
}
