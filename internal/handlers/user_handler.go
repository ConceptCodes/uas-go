package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/rs/zerolog"
)

type UserHandler struct {
	userRepo        repository.UserRepository
	passwordResetRepo repository.GormPasswordResetRepository
	log             *zerolog.Logger
	authHelper      *helpers.AuthHelper
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
	emailHelper     *helpers.EmailHelper
}

func NewUserHandler(
	userRepo repository.UserRepository,
	passwordResetRepo repository.GormPasswordResetRepository,
	log *zerolog.Logger,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	emailHelper *helpers.EmailHelper,
) *UserHandler {
	return &UserHandler{
		userRepo:        userRepo,
		passwordResetRepo: passwordResetRepo,
		log:             log,
		authHelper:      authHelper,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
		emailHelper:     emailHelper,
	}
}

// RegisterUserHandler godoc
// @Summary Register User
// @Description Register User
// @Tags User
// @Accept  json
// @Produce  json
// @Success 200 {object} RegisterUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/credentials [post]
func (h *UserHandler) CredentialsRegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var data models.CredentialsRegisterRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	password_hash, err := h.authHelper.HashPassword(data.Password)
	err_message := fmt.Sprintf(constants.CreateEntityError, "User")

	if err != nil {
		h.log.Error().Err(err).Msg("Error hashing password")
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	user_id := uuid.New().String()

	user := models.UserModel{
		ID:       user_id,
		Name:     data.Name,
		Email:    data.Email,
		Password: password_hash,
	}

	err = h.userRepo.Create(&user)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	res := &models.RegisterUserResponse{
		UserID: user_id,
		Name:   data.Name,
		Email:  data.Email,
	}

	h.responseHelper.SendSuccessResponse(w, "User registered successfully", res)
}

// LoginUserHandler godoc
// @Summary Login User
// @Description Login User
// @Tags User
// @Accept  json
// @Produce  json
// @Success 200 {object} JwtTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/credentials/login [post]
func (h *UserHandler) CredentialsLoginUserHandler(w http.ResponseWriter, r *http.Request) {
	var data models.CredentialsLoginRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	user, err := h.userRepo.FindByEmail(data.Email)

	if err != nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "email:", data.Email)
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	if user == nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User", "email: ", data.Email)
		h.responseHelper.SendErrorResponse(w, err_message, constants.NotFound, nil)
	}

	valid := h.authHelper.CheckPasswordHash(data.Password, user.Password)

	if !valid {
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.BadRequest, err)
	}

	tenantId := helpers.GetTenantId(r)
	access_token, err := h.authHelper.GenerateAccessJwtToken(user, tenantId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating access token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	refresh_token, err := h.authHelper.GenerateRefreshJwtToken(user, tenantId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating refresh token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	cookieHashKey := []byte(config.AppConfig.CookieHashKey)
	cookieBlockKey := []byte(config.AppConfig.CookieBlockKey)

	var s = securecookie.New(cookieHashKey, cookieBlockKey)

	if encoded, err := s.Encode("access-token", access_token); err == nil {
		cookie := &http.Cookie{
			Name:     "access-token",
			Value:    encoded,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		w.Header().Set(constants.JwtHeader, refresh_token)
	}

	h.responseHelper.SendSuccessResponse(w, "Successful login", nil)
}

// ForgotPasswordHandler godoc
// @Summary Forgot Password
// @Description Forgot Password
// @Tags User
// @Accept  json
// @Produce  json
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/credentials/forgot-password [post]
func (h *UserHandler) CredentialsForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var data models.ForgotPasswordRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	user, err := h.userRepo.FindByEmail(data.Email)

	if err != nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "email:", data.Email)
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	if user == nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User", "email: ", data.Email)
		h.responseHelper.SendErrorResponse(w, err_message, constants.NotFound, nil)
	}

	reset_token := h.authHelper.GenerateResetPasswordToken()

	tmpl_data := models.ForgotPasswordData{
		Token: reset_token,
		Name:  user.Name,
	}

	err = h.emailHelper.SendEmail(data.Email, "reset-password", tmpl_data)

	if err != nil {
		h.log.Error().Err(err).Msg("Error sending reset password email")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

}

func (h *UserHandler) CredentialsResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	if token == "" {
		h.log.Error().Msg("Token is empty")
		h.responseHelper.SendErrorResponse(w, "Token is empty", constants.InternalServerError, nil)
	}

	record, err := h.passwordResetRepo.FindByToken(token)

	if err == nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "email:", "")
		h.responseHelper.SendErrorResponse(w, err_message, constants.BadRequest, err)
	}

	if token == record.Token {
		// TODO: where is their new password
		// NOTE: should we  soft delete the record?
	}

}
