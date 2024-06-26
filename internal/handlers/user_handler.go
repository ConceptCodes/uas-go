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
	"github.com/gorilla/securecookie"
	"github.com/rs/zerolog"
)

type UserHandler struct {
	userRepo           repository.UserRepository
	authRepo           repository.AuthRepository
	departmentRoleRepo repository.DepartmentRoleRepository
	departmentRepo     repository.DepartmentRepository
	log                *zerolog.Logger
	authHelper         *helpers.AuthHelper
	responseHelper     *helpers.ResponseHelper
	validatorHelper    *helpers.ValidatorHelper
	emailHelper        *helpers.EmailHelper
	twilioHelper       *helpers.TwilioHelper
}

func NewUserHandler(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	departmentRoleRepo repository.DepartmentRoleRepository,
	departmentRepo repository.DepartmentRepository,
	log *zerolog.Logger,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	emailHelper *helpers.EmailHelper,
	twilioHelper *helpers.TwilioHelper,
) *UserHandler {
	return &UserHandler{
		userRepo:           userRepo,
		authRepo:           authRepo,
		departmentRoleRepo: departmentRoleRepo,
		departmentRepo:     departmentRepo,
		log:                log,
		authHelper:         authHelper,
		responseHelper:     responseHelper,
		validatorHelper:    validatorHelper,
		emailHelper:        emailHelper,
		twilioHelper:       twilioHelper,
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
	var data models.RegisterRequest

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

	userId := uuid.New().String()

	user := models.UserModel{
		ID:            userId,
		Name:          data.Name,
		Email:         data.Email,
		Password:      password_hash,
		PhoneNumber:   data.PhoneNumber,
		EmailVerified: false,
	}

	err = h.userRepo.Create(&user)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	departmentId := helpers.GetDepartmentId(r)

	user_role := models.DepartmentRoles{
		ID:     departmentId,
		Role:   models.User,
		UserID: userId,
	}

	err = h.departmentRoleRepo.Create(&user_role)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error creating user role", constants.InternalServerError, err)
	}

	code, err := h.authHelper.GenerateOtpCode(data.PhoneNumber)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error w/ otp generation", constants.InternalServerError, err)
	}

	tmpl_data := models.VerifyEmailData{
		Name: user.Name,
		Otp:  code,
	}

	err = h.emailHelper.SendEmail(data.Email, "verify-email", tmpl_data)

	if err != nil {
		h.log.Error().Err(err).Msg("Error sending verification email")
		h.responseHelper.SendErrorResponse(w, "Error w/ sending verification email", constants.InternalServerError, err)
	}

	res := &models.RegisterUserResponse{
		UserID: userId,
		Name:   data.Name,
		Email:  data.Email,
	}

	h.responseHelper.SendSuccessResponse(w, "User registered successfully", res)

}

// VerifyEmailHandler godoc
// @Summary Verify Email
// @Description Verify Email
// @Tags User
// @Accept  json
// @Produce  json
// @Param token path string true "Token"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/credentials/verify-email/{token} [post]
func (h *UserHandler) CredentialsVerifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	var data models.VerifyEmailRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	err = h.authHelper.ValidateOtpCode(data.Email, data.Otp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error verifying OTP code", constants.InternalServerError, err)
	}

	user, err := h.userRepo.FindByEmail(data.Email)

	if err != nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "email:", data.Email)
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	if user == nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User", "email: ", data.Email)
		h.responseHelper.SendErrorResponse(w, err_message, constants.NotFound, nil)
	} else {
		user.EmailVerified = true
		err = h.userRepo.Save(user)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error verifying email", constants.InternalServerError, err)
		}
	}

	h.responseHelper.SendSuccessResponse(w, "Email verified successfully", nil)

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
	} else {
		if !user.EmailVerified {
			h.responseHelper.SendErrorResponse(w, "Email not verified", constants.BadRequest, err)
		}
	}

	valid := h.authHelper.CheckPasswordHash(data.Password, user.Password)

	if !valid {
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.BadRequest, err)
	}

	departmentId := helpers.GetDepartmentId(r)
	access_token, err := h.authHelper.GenerateAccessJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating access token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	refresh_token, err := h.authHelper.GenerateRefreshJwtToken(user, departmentId)

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
	} else {
		if !user.EmailVerified {
			h.responseHelper.SendErrorResponse(w, "Email not verified", constants.BadRequest, err)
		}
	}

	reset_token := h.authHelper.GenerateAuthToken()

	tmp := models.AuthModel{
		UserID: user.ID,
		Token:  reset_token,
		Type: models.ResetPassword,
	}

	err = h.authRepo.Create(&tmp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error sending reset password email", constants.InternalServerError, err)
	}

	url := fmt.Sprintf("%s?token=%s", r.URL, reset_token)

	tmpl_data := models.ForgotPasswordData{
		Name: user.Name,
		Url:  url,
	}

	err = h.emailHelper.SendEmail(data.Email, "reset-password", tmpl_data)

	if err != nil {
		h.log.Error().Err(err).Msg("Error sending reset password email")
		h.responseHelper.SendErrorResponse(w, "Error sending reset password email", constants.InternalServerError, err)
	}

	h.responseHelper.SendSuccessResponse(w, "Reset password email sent successfully", nil)

}

// ResetPasswordHandler godoc
// @Summary Reset Password
// @Description Reset Password
// @Tags User
// @Accept  json
// @Produce  json
// @Param token path string true "Token"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/credentials/reset-password/{token} [post]
func (h *UserHandler) CredentialsResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	token := params.Get("token")

	if token == "" {
		h.log.Error().Msg("Token is empty")
		h.responseHelper.SendErrorResponse(w, "Token is empty", constants.InternalServerError, nil)
	}

	var data models.ResetPasswordRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	record, err := h.authRepo.FindByTokenAndType(token, models.ResetPassword)

	if err == nil {
		err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "email:", "")
		h.responseHelper.SendErrorResponse(w, err_message, constants.BadRequest, err)
	}

	if token == record.Token {
		user, err := h.userRepo.FindById(record.UserID)

		if err != nil {
			err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "id:", record.UserID)
			h.responseHelper.SendErrorResponse(w, err_message, constants.BadRequest, err)
		}

		password_hash, err := h.authHelper.HashPassword(data.Password)

		if err != nil {
			h.log.Error().Err(err).Msg("Error hashing password")
			h.responseHelper.SendErrorResponse(w, "Error resetting password", constants.InternalServerError, err)
		}

		user.Password = password_hash

		err = h.userRepo.Save(user)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error resetting password", constants.InternalServerError, err)
		}

	} else {
		h.responseHelper.SendErrorResponse(w, "Invalid token", constants.BadRequest, nil)
	}

	h.responseHelper.SendSuccessResponse(w, "Password reset successfully", nil)

}

func (h *UserHandler) SendOtpCode(w http.ResponseWriter, r *http.Request) {
	var data models.SendOtpRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	// NOTE: should we retry this operation if it fails?
	code, err := h.authHelper.GenerateOtpCode(data.PhoneNumber)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error generating OTP code", constants.InternalServerError, err)
	}

	msg := fmt.Sprintf(constants.OtpCodeMessage, code)

	// NOTE: same here, ^^^
	err = h.twilioHelper.SendSMS(data.PhoneNumber, msg)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error sending OTP code", constants.InternalServerError, err)
	}

	h.responseHelper.SendSuccessResponse(w, "OTP code sent successfully", nil)

}

func (h *UserHandler) VerifyOtpCode(w http.ResponseWriter, r *http.Request) {
	var data models.VerifyOtpRequest
	var user *models.UserModel

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	err = h.authHelper.ValidateOtpCode(data.PhoneNumber, data.Otp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error verifying OTP code", constants.InternalServerError, err)
	}

	user, err = h.userRepo.FindByPhoneNumber(data.PhoneNumber)
	departmentId := helpers.GetDepartmentId(r)

	if err != nil {
		h.log.Info().Str("phoneNumber", data.PhoneNumber).Msg("User does not exist")

		userId := uuid.New().String()

		user = &models.UserModel{
			ID:          userId,
			PhoneNumber: data.PhoneNumber,
		}

		err = h.userRepo.Create(user)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user", constants.InternalServerError, err)
		}

		user_role := models.DepartmentRoles{
			ID:     departmentId,
			Role:   models.User,
			UserID: userId,
		}

		err = h.departmentRoleRepo.Create(&user_role)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user role", constants.InternalServerError, err)
		}

	}

	access_token, err := h.authHelper.GenerateAccessJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating access token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	refresh_token, err := h.authHelper.GenerateRefreshJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating refresh token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	h.authHelper.GenerateAccessCookie(access_token, w)

	w.Header().Set(constants.JwtHeader, refresh_token)

	h.responseHelper.SendSuccessResponse(w, "OTP code verified successfully", nil)

}

func (h *UserHandler) RefreshAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	var accessToken string
	cookies := r.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "access-token" {
			accessToken = cookie.Value
		}

		if accessToken == "" {
			h.responseHelper.SendErrorResponse(w, "Access token is empty", constants.BadRequest, nil)
		}

		claims, err := h.authHelper.ParseAccessJwtToken(accessToken)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error parsing access token", constants.InternalServerError, err)
		}

		userId := claims["userId"].(string)
		departmentId := claims["departmentId"].(string)

		user, err := h.userRepo.FindById(userId)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error finding user", constants.InternalServerError, err)
		}

		access_token, err := h.authHelper.GenerateAccessJwtToken(user, departmentId)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error generating access token", constants.InternalServerError, err)
		}

		h.authHelper.GenerateAccessCookie(access_token, w)

		h.responseHelper.SendSuccessResponse(w, "Access token refreshed successfully", nil)
	}

}

func (h *UserHandler) SendMagicLinkEmail(w http.ResponseWriter, r *http.Request) {
	var data models.MagicLinkEmailRequest
	var user *models.UserModel

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	h.validatorHelper.ValidateStruct(w, &data)

	user, err = h.userRepo.FindByEmail(data.Email)
	departmentId := helpers.GetDepartmentId(r)

	if err != nil {
		h.log.Info().Str("email", data.Email).Msg("User does not exist")

		userId := uuid.New().String()

		user = &models.UserModel{
			ID:    userId,
			Email: data.Email,
		}

		err = h.userRepo.Create(user)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user", constants.InternalServerError, err)
		}

		user_role := models.DepartmentRoles{
			ID:     departmentId,
			Role:   models.User,
			UserID: userId,
		}

		err = h.departmentRoleRepo.Create(&user_role)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user role", constants.InternalServerError, err)
		}

	}

	token := h.authHelper.GenerateAuthToken()

	tmp := models.AuthModel{
		UserID: user.ID,
		Token:  token,
		Type: models.MagicLink,
	}

	err = h.authRepo.Create(&tmp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	tmpl_data := models.MagicEmailData{
		Name: user.Name,
		Url:  fmt.Sprintf("%s?token=%s", r.URL, token),
	}

	err = h.emailHelper.SendEmail(data.Email, "magic-link", tmpl_data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
	}

	h.responseHelper.SendSuccessResponse(w, "magic link sent to", nil)
}

func (h *UserHandler) VerifyMagicLinkEmail(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	token := params.Get("token")

	if token == "" {
		h.log.Error().Msg("Token is empty")
		h.responseHelper.SendErrorResponse(w, "Token is empty", constants.InternalServerError, nil)
	}

	record, err := h.authRepo.FindByTokenAndType(token, models.MagicLink)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, nil)
	}

	if token == record.Token {
		user, err := h.userRepo.FindById(record.UserID)

		if err != nil {
			err_message := fmt.Sprintf(constants.EntityNotFound, "User ", "id:", record.UserID)
			h.responseHelper.SendErrorResponse(w, err_message, constants.BadRequest, err)
		}

	}

}
