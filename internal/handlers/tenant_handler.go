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

type TenantHandler struct {
	tenantRepo      repository.TenantRepository
	logger          *zerolog.Logger
	authHelper      *helpers.AuthHelper
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
}

func NewTenantHandler(
	tenantRepo repository.TenantRepository,
	logger *zerolog.Logger,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
) *TenantHandler {
	return &TenantHandler{
		tenantRepo:      tenantRepo,
		logger:          logger,
		authHelper:      authHelper,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
	}
}

// OnboardTenantHandler godoc
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
func (h *TenantHandler) OnboardTenantHandler(w http.ResponseWriter, r *http.Request) {
	var data models.OnboardTenantRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	tenant_secret := uuid.New().String()

	tenant := &models.TenantModel{
		ID:     data.DepartmentID,
		Secret: tenant_secret,
		Name:   data.DepartmentName,
	}

	err = h.tenantRepo.Create(tenant)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
	}

	hashed_secret, err := h.authHelper.HashPassword(tenant_secret)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
	}

	res := &models.OnboardTenantResponse{
		DepartmentID:   tenant.ID,
		DepartmentName: tenant.Name,
		TenantSecret:   hashed_secret,
	}

	token := fmt.Sprintf("Bearer %s", h.authHelper.GenerateBasicAuthToken(tenant.ID, tenant.Secret))

	w.Header().Set(constants.AuthorizationHeader, token)
	h.responseHelper.SendSuccessResponse(w, "Tenant onboarded successfully", res)
}

// DeleteTenantHandler godoc
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
func (h *TenantHandler) DeleteTenantHandler(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	tenant_id := vars.Get("id")

	if tenant_id == "" {
		message := fmt.Sprintf(constants.EntityNotFound, "Tenant", "id", tenant_id)
		h.responseHelper.SendErrorResponse(w, message, constants.NotFound, nil)
	}

	err := h.tenantRepo.Delete(tenant_id)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		h.responseHelper.SendErrorResponse(w, message, constants.InternalServerError, err)
	}

	// Note: if we introduced sessions, we would need to delete all sessions associated with the tenant here

	h.responseHelper.SendSuccessResponse(w, "Tenant deleted successfully", nil)
}
