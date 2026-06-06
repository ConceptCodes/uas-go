package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type DepartmentHandler struct {
	departmentRepo      repository.DepartmentRepository
	sessionRepo         repository.SessionRepository
	passwordHistoryRepo repository.PasswordHistoryRepository
	authRepo            repository.AuthRepository
	userRepo            repository.UserRepository
	departmentRoleRepo  repository.DepartmentRoleRepository
	logger              *zerolog.Logger
	authHelper          *helpers.AuthHelper
	responseHelper      *helpers.ResponseHelper
	validatorHelper     *helpers.ValidatorHelper
}

func NewDepartmentHandler(
	departmentRepo repository.DepartmentRepository,
	sessionRepo repository.SessionRepository,
	passwordHistoryRepo repository.PasswordHistoryRepository,
	authRepo repository.AuthRepository,
	userRepo repository.UserRepository,
	departmentRoleRepo repository.DepartmentRoleRepository,
	logger *zerolog.Logger,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
) *DepartmentHandler {
	return &DepartmentHandler{
		departmentRepo:      departmentRepo,
		sessionRepo:         sessionRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		authRepo:            authRepo,
		userRepo:            userRepo,
		departmentRoleRepo:  departmentRoleRepo,
		logger:              logger,
		authHelper:          authHelper,
		responseHelper:      responseHelper,
		validatorHelper:     validatorHelper,
	}
}

func (h *DepartmentHandler) OnboardDepartmentHandler(w http.ResponseWriter, r *http.Request) {
	var data models.OnboardTenantRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	secret := uuid.New().String()

	hashed_secret, err := h.authHelper.HashPassword(secret)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Department")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
		return
	}

	department := &models.DepartmentModel{
		ID:     data.DepartmentID,
		Secret: hashed_secret,
		Name:   data.DepartmentName,
	}

	err = h.departmentRepo.Create(department)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Department")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
		return
	}

	res := &models.OnboardDepartmentResponse{
		DepartmentID:   department.ID,
		DepartmentName: department.Name,
	}

	token := fmt.Sprintf("Bearer %s", h.authHelper.GenerateBasicAuthToken(department.ID, secret))

	w.Header().Set(constants.AuthorizationHeader, token)
	h.responseHelper.SendSuccessResponse(w, "Department onboarded successfully", res)
}

func (h *DepartmentHandler) DeleteDepartmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID := vars["id"]

	if tenantID == "" {
		message := fmt.Sprintf(constants.EntityNotFound, "Tenant", "id", tenantID)
		h.responseHelper.SendErrorResponse(w, message, constants.NotFound, nil)
		return
	}

	if err := h.sessionRepo.RevokeAllByDepartment(tenantID); err != nil {
		h.logger.Error().Err(err).Str("tenantID", tenantID).Msg("Failed to revoke sessions for tenant")
		h.responseHelper.SendErrorResponse(w, "Failed to delete tenant", constants.InternalServerError, err)
		return
	}

	if err := h.passwordHistoryRepo.DeleteByDepartment(tenantID); err != nil {
		h.logger.Error().Err(err).Str("tenantID", tenantID).Msg("Failed to delete password history for tenant")
		h.responseHelper.SendErrorResponse(w, "Failed to delete tenant", constants.InternalServerError, err)
		return
	}

	err := h.departmentRepo.Delete(tenantID)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Tenant deleted successfully", nil)
}
