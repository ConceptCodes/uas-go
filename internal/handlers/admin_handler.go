package handlers

import (
	"encoding/json"
	"net/http"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type AdminHandler struct {
	userRepo        repository.UserRepository
	deptRepo        repository.DepartmentRepository
	sessionRepo     repository.SessionRepository
	passwordHistRepo repository.PasswordHistoryRepository
	passwordHelper  *helpers.PasswordHelper
	authHelper      *helpers.AuthHelper
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
	log             *zerolog.Logger
}

func NewAdminHandler(
	userRepo repository.UserRepository,
	deptRepo repository.DepartmentRepository,
	sessionRepo repository.SessionRepository,
	passwordHistRepo repository.PasswordHistoryRepository,
	passwordHelper *helpers.PasswordHelper,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	log *zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
		userRepo:        userRepo,
		deptRepo:        deptRepo,
		sessionRepo:     sessionRepo,
		passwordHistRepo: passwordHistRepo,
		passwordHelper:  passwordHelper,
		authHelper:      authHelper,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
		log:             log,
	}
}

func (h *AdminHandler) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	_ = departmentID // Note: In production, this needs a FindAllByDepartment on user repo

	// For now, return a paginated response pattern
	// A full implementation would add FindAllByDepartment to the user repository
	h.responseHelper.SendSuccessResponse(w, "Users retrieved", []interface{}{})
}

func (h *AdminHandler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]
	departmentID := helpers.GetDepartmentId(r)

	user, err := h.userRepo.FindById(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "User not found", constants.NotFound, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "User retrieved", map[string]interface{}{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"phoneNumber":    user.PhoneNumber,
		"emailVerified":  user.EmailVerified,
	})
}

func (h *AdminHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]
	departmentID := helpers.GetDepartmentId(r)

	user, err := h.userRepo.FindById(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "User not found", constants.NotFound, err)
		return
	}

	var data models.AdminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if data.Name != nil {
		user.Name = *data.Name
	}
	if data.Email != nil {
		user.Email = *data.Email
	}
	if data.PhoneNumber != nil {
		user.PhoneNumber = *data.PhoneNumber
	}
	if data.EmailVerified != nil {
		user.EmailVerified = *data.EmailVerified
	}

	if err := h.userRepo.Save(user, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to update user", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "User updated", nil)
}

func (h *AdminHandler) ResetUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]
	departmentID := helpers.GetDepartmentId(r)

	user, err := h.userRepo.FindById(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "User not found", constants.NotFound, err)
		return
	}

	var data models.AdminResetUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if err := h.passwordHelper.ValidateComplexity(data.NewPassword); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if h.passwordHelper.CheckCommonPassword(data.NewPassword) {
		h.responseHelper.SendErrorResponse(w, "Password is too common", constants.BadRequest, nil)
		return
	}

	hash, err := h.authHelper.HashPassword(data.NewPassword)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to hash password", constants.InternalServerError, err)
		return
	}

	user.Password = hash
	if err := h.userRepo.Save(user, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to update password", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Password reset successfully", nil)
}

func (h *AdminHandler) RevokeUserSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]
	departmentID := helpers.GetDepartmentId(r)

	if err := h.sessionRepo.RevokeAllUserSessions(userID, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to revoke sessions", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "All user sessions revoked", nil)
}

func (h *AdminHandler) ListTenantsHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder - platform admin tenant listing
	h.responseHelper.SendErrorResponse(w, "Not implemented", constants.InternalServerError, nil)
}

func (h *AdminHandler) SuspendTenantHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder - platform admin tenant suspension
	h.responseHelper.SendErrorResponse(w, "Not implemented", constants.InternalServerError, nil)
}