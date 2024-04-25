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
	"github.com/rs/zerolog"
)

type DepartmentHandler struct {
	departmentRepo  repository.DepartmentRepository
	logger          *zerolog.Logger
	authHelper      *helpers.AuthHelper
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
}

func NewDepartmentHandler(
	departmentRepo repository.DepartmentRepository,
	logger *zerolog.Logger,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
) *DepartmentHandler {
	return &DepartmentHandler{
		departmentRepo:  departmentRepo,
		logger:          logger,
		authHelper:      authHelper,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
	}
}

// OnboardDepartmentHandler godoc
// @Summary Onboard Tenant
// @Description Onboard Tenant
// @Tags Tenant
// @Accept  json
// @Produce  json
// @Success 200 {object} OnboardTenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants [post]
func (h *DepartmentHandler) OnboardDepartmentHandler(w http.ResponseWriter, r *http.Request) {
	var data models.OnboardTenantRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	secret := uuid.New().String()

	hashed_secret, err := h.authHelper.HashPassword(secret)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Department")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
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
	}

	res := &models.OnboardDepartmentResponse{
		DepartmentID:   department.ID,
		DepartmentName: department.Name,
	}

	token := fmt.Sprintf("Bearer %s", h.authHelper.GenerateBasicAuthToken(department.ID, department.Secret))

	w.Header().Set(constants.AuthorizationHeader, token)
	h.responseHelper.SendSuccessResponse(w, "Department onboarded successfully", res)
}

// DeleteDepartmentHandler godoc
// @Summary Delete Tenant
// @Description Delete Tenant
// @Tags Tenant
// @Accept  json
// @Produce  json
// @Param id path string true "Tenant ID"
// @Success 200 {object} DeleteTenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{id} [delete]
func (h *DepartmentHandler) DeleteDepartmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	tenant_id := vars.Get("id")

	if tenant_id == "" {
		message := fmt.Sprintf(constants.EntityNotFound, "Tenant", "id", tenant_id)
		h.responseHelper.SendErrorResponse(w, message, constants.NotFound, nil)
	}

	err := h.departmentRepo.Delete(tenant_id)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
	}

	// Note: if we introduced sessions, we would need to delete all sessions associated with the tenant here
	h.responseHelper.SendSuccessResponse(w, "Tenant deleted successfully", nil)
}
