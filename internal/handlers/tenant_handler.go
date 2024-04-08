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
)

type TenantHandler struct {
	tenantRepo repository.TenantRepository
}

func NewTenantHandler(tenantRepo repository.TenantRepository) *TenantHandler {
	return &TenantHandler{tenantRepo: tenantRepo}
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
		helpers.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	helpers.ValidateStruct(w, &data)

	tenant_secret := uuid.New().String()

	tenant := &models.TenantModel{
		ID:     data.DepartmentID,
		Secret: tenant_secret,
		Name:   data.Name,
	}

	err = h.tenantRepo.Create(tenant)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		helpers.SendErrorResponse(w, message, constants.InternalServerError, err)
	}

	hashed_secret, err := helpers.HashPassword(tenant_secret)

	if err != nil {
		message := fmt.Sprintf(constants.CreateEntityError, "Tenant")
		helpers.SendErrorResponse(w, message, constants.InternalServerError, err)
	}

	res := &models.OnboardTenantResponse{
		DepartmentID:   tenant.ID,
		DepartmentName: tenant.Name,
		TenantSecret:   hashed_secret,
	}

	token := fmt.Sprintf("Bearer %s", helpers.GenerateBasicAuthToken(tenant.ID, tenant.Secret))

	w.Header().Set("Authorization", token)
	helpers.SendSuccessResponse(w, "Tenant onboarded successfully", res)
}
