package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type UserHandler struct {
	userRepo            repository.UserRepository
	authRepo            repository.AuthRepository
	departmentRoleRepo  repository.DepartmentRoleRepository
	departmentRepo      repository.DepartmentRepository
	sessionRepo         repository.SessionRepository
	passwordHistoryRepo repository.PasswordHistoryRepository
	log                 *zerolog.Logger
	authHelper          *helpers.AuthHelper
	responseHelper      *helpers.ResponseHelper
	validatorHelper     *helpers.ValidatorHelper
	emailHelper         *helpers.EmailHelper
	twilioHelper        *helpers.TwilioHelper
	loginAttemptHelper  *helpers.LoginAttemptHelper
	passwordHelper      *helpers.PasswordHelper
	encryptionHelper    *helpers.EncryptionHelper
	tokenHelper         *helpers.TokenHelper
	securityLogger      *helpers.SecurityLoggerHelper
}

func NewUserHandler(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	departmentRoleRepo repository.DepartmentRoleRepository,
	departmentRepo repository.DepartmentRepository,
	sessionRepo repository.SessionRepository,
	passwordHistoryRepo repository.PasswordHistoryRepository,
	log *zerolog.Logger,
	authHelper *helpers.AuthHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	emailHelper *helpers.EmailHelper,
	twilioHelper *helpers.TwilioHelper,
	loginAttemptHelper *helpers.LoginAttemptHelper,
	passwordHelper *helpers.PasswordHelper,
	encryptionHelper *helpers.EncryptionHelper,
	tokenHelper *helpers.TokenHelper,
	securityLogger *helpers.SecurityLoggerHelper,
) *UserHandler {
	return &UserHandler{
		userRepo:            userRepo,
		authRepo:            authRepo,
		departmentRoleRepo:  departmentRoleRepo,
		departmentRepo:      departmentRepo,
		sessionRepo:         sessionRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		log:                 log,
		authHelper:          authHelper,
		responseHelper:      responseHelper,
		validatorHelper:     validatorHelper,
		emailHelper:         emailHelper,
		twilioHelper:        twilioHelper,
		loginAttemptHelper:  loginAttemptHelper,
		passwordHelper:      passwordHelper,
		encryptionHelper:    encryptionHelper,
		tokenHelper:         tokenHelper,
		securityLogger:      securityLogger,
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
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	if err := h.passwordHelper.ValidateComplexity(data.Password); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if h.passwordHelper.CheckCommonPassword(data.Password) {
		h.responseHelper.SendErrorResponse(w, "Password is too common", constants.BadRequest, nil)
		return
	}

	password_hash, err := h.authHelper.HashPassword(data.Password)
	err_message := fmt.Sprintf(constants.CreateEntityError, "User")

	if err != nil {
		h.log.Error().Err(err).Msg("Error hashing password")
		h.responseHelper.SendErrorResponse(w, err_message, constants.InternalServerError, err)
		return
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
		return
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
		return
	}

	code, err := h.authHelper.GenerateOtpCode(data.Email)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error w/ otp generation", constants.InternalServerError, err)
		return
	}

	tmpl_data := models.VerifyEmailData{
		Name: user.Name,
		Otp:  code,
	}

	err = h.emailHelper.SendEmail(data.Email, "verify-email", tmpl_data)

	if err != nil {
		h.log.Error().Err(err).Msg("Error sending verification email")
		h.responseHelper.SendErrorResponse(w, "Error w/ sending verification email", constants.InternalServerError, err)
		return
	}

	h.securityLogger.LogAuthEvent(r, "register", userId, departmentId, "success", "")

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
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	err = h.authHelper.ValidateOtpCode(data.Email, data.Otp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error verifying OTP code", constants.InternalServerError, err)
		return
	}

	departmentId := helpers.GetDepartmentId(r)
	user, err := h.userRepo.FindByEmailAndDepartment(data.Email, departmentId)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid verification request", constants.Unauthorized, nil)
		return
	}

	if user == nil {
		h.responseHelper.SendErrorResponse(w, "Invalid verification request", constants.Unauthorized, nil)
		return
	}

	user.EmailVerified = true
	err = h.userRepo.Save(user)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error verifying email", constants.InternalServerError, err)
		return
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
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	departmentId := helpers.GetDepartmentId(r)
	lockoutKey := departmentId + ":" + data.Email

	locked, err := h.loginAttemptHelper.IsAccountLocked(lockoutKey)
	if err != nil {
		h.log.Error().Err(err).Msg("Error checking account lock status")
		h.responseHelper.SendErrorResponse(w, "Internal server error", constants.InternalServerError, err)
		return
	}

	if locked {
		h.log.Warn().Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("Login attempt for locked account")
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.Unauthorized, nil)
		return
	}

	user, err := h.userRepo.FindByEmailAndDepartment(data.Email, departmentId)

	if err != nil {
		h.log.Warn().Err(err).Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("User not found in tenant")
		h.loginAttemptHelper.RecordFailedAttempt(lockoutKey)
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.Unauthorized, nil)
		return
	}

	if user == nil {
		h.log.Warn().Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("User not found")
		h.loginAttemptHelper.RecordFailedAttempt(lockoutKey)
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.Unauthorized, nil)
		return
	}

	if !user.EmailVerified {
		h.log.Warn().Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("Email not verified")
		h.loginAttemptHelper.RecordFailedAttempt(lockoutKey)
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.Unauthorized, nil)
		return
	}

	valid := h.authHelper.CheckPasswordHash(data.Password, user.Password)

	if !valid {
		h.log.Warn().Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("Invalid password")
		h.loginAttemptHelper.RecordFailedAttempt(lockoutKey)
		attemptCount, _ := h.loginAttemptHelper.GetFailedAttemptCount(lockoutKey)
		delay := h.loginAttemptHelper.GetProgressiveDelay(attemptCount)
		h.loginAttemptHelper.ApplyProgressiveDelay(r.Context(), delay)
		h.responseHelper.SendErrorResponse(w, "Invalid credentials", constants.Unauthorized, nil)
		return
	}

	access_token, err := h.authHelper.GenerateAccessJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating access token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
		return
	}

	refresh_token, err := h.authHelper.GenerateRefreshJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating refresh token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
		return
	}

	h.loginAttemptHelper.ClearFailedAttempts(lockoutKey)
	if err := h.createSession(r, user, departmentId, refresh_token); err != nil {
		h.responseHelper.SendErrorResponse(w, "Error creating session", constants.InternalServerError, err)
		return
	}
	h.securityLogger.LogAuthEvent(r, "login", user.ID, departmentId, "success", "")
	h.authHelper.GenerateAccessCookie(access_token, w)
	w.Header().Set(constants.JwtHeader, refresh_token)
	h.responseHelper.SendSuccessResponse(w, "Successful login", nil)
	return
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
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	departmentId := helpers.GetDepartmentId(r)
	user, err := h.userRepo.FindByEmailAndDepartment(data.Email, departmentId)
	if err != nil {
		h.log.Warn().Err(err).Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("Forgot password lookup failed")
	}

	if user == nil || !user.EmailVerified {
		h.responseHelper.SendSuccessResponse(w, "If the account exists, a password reset email has been sent", nil)
		return
	}

	reset_token := h.authHelper.GenerateAuthToken()

	tmp := models.AuthModel{
		UserID: user.ID,
		Token:  reset_token,
		Type:   models.ResetPassword,
	}

	err = h.authRepo.Create(&tmp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error sending reset password email", constants.InternalServerError, err)
		return
	}

	baseURL := strings.TrimRight(config.AppConfig.PasswordResetBaseUrl, "/")

	tmpl_data := models.ForgotPasswordData{
		Name:  user.Name,
		Url:   baseURL,
		Token: reset_token,
	}

	err = h.emailHelper.SendEmail(data.Email, "reset-password", tmpl_data)

	if err != nil {
		h.log.Error().Err(err).Msg("Error sending reset password email")
		h.responseHelper.SendErrorResponse(w, "Error sending reset password email", constants.InternalServerError, err)
		return
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
	var data struct {
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if data.Token == "" {
		h.responseHelper.SendErrorResponse(w, "Token is required", constants.BadRequest, nil)
		return
	}

	if err := h.passwordHelper.ValidateComplexity(data.Password); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if h.passwordHelper.CheckCommonPassword(data.Password) {
		h.responseHelper.SendErrorResponse(w, "Password is too common", constants.BadRequest, nil)
		return
	}

	record, err := h.authRepo.FindByTokenAndType(data.Token, models.ResetPassword)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid or expired reset token", constants.BadRequest, err)
		return
	}

	user, err := h.userRepo.FindById(record.UserID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid reset token", constants.BadRequest, nil)
		return
	}

	password_hash, err := h.authHelper.HashPassword(data.Password)
	if err != nil {
		h.log.Error().Err(err).Msg("Error hashing password")
		h.responseHelper.SendErrorResponse(w, "Error resetting password", constants.InternalServerError, err)
		return
	}

	user.Password = password_hash
	err = h.userRepo.Save(user)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error resetting password", constants.InternalServerError, err)
		return
	}

	if err := h.authRepo.DeleteByTokenAndType(data.Token, models.ResetPassword); err != nil {
		h.log.Warn().Err(err).Msg("Failed to delete reset token after password update")
	}

	h.responseHelper.SendSuccessResponse(w, "Password reset successfully", nil)
}

func (h *UserHandler) SendOtpCode(w http.ResponseWriter, r *http.Request) {
	var data models.SendOtpRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	// NOTE: should we retry this operation if it fails?
	code, err := h.authHelper.GenerateOtpCode(data.PhoneNumber)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error generating OTP code", constants.InternalServerError, err)
		return
	}

	msg := fmt.Sprintf(constants.OtpCodeMessage, code)

	// NOTE: same here, ^^^
	err = h.twilioHelper.SendSMS(data.PhoneNumber, msg)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error sending OTP code", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "OTP code sent successfully", nil)

}

func (h *UserHandler) VerifyOtpCode(w http.ResponseWriter, r *http.Request) {
	var data models.VerifyOtpRequest
	var user *models.UserModel

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	err = h.authHelper.ValidateOtpCode(data.PhoneNumber, data.Otp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error verifying OTP code", constants.InternalServerError, err)
		return
	}

	user, err = h.userRepo.FindByPhoneNumber(data.PhoneNumber)
	departmentId := helpers.GetDepartmentId(r)

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.responseHelper.SendErrorResponse(w, "Error fetching user", constants.InternalServerError, err)
			return
		}

		h.log.Info().Str("phoneNumber", h.encryptionHelper.MaskPhone(data.PhoneNumber)).Msg("User does not exist")

		userId := uuid.New().String()

		user = &models.UserModel{
			ID:          userId,
			PhoneNumber: data.PhoneNumber,
		}

		err = h.userRepo.Create(user)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user", constants.InternalServerError, err)
			return
		}

		user_role := models.DepartmentRoles{
			ID:     departmentId,
			Role:   models.User,
			UserID: userId,
		}

		err = h.departmentRoleRepo.Create(&user_role)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user role", constants.InternalServerError, err)
			return
		}

	}

	access_token, err := h.authHelper.GenerateAccessJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating access token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
		return
	}

	refresh_token, err := h.authHelper.GenerateRefreshJwtToken(user, departmentId)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating refresh token")
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
		return
	}

	h.authHelper.GenerateAccessCookie(access_token, w)
	if err := h.createSession(r, user, departmentId, refresh_token); err != nil {
		h.responseHelper.SendErrorResponse(w, "Error creating session", constants.InternalServerError, err)
		return
	}

	w.Header().Set(constants.JwtHeader, refresh_token)

	h.responseHelper.SendSuccessResponse(w, "OTP code verified successfully", nil)

}

func (h *UserHandler) RefreshAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Header.Get(constants.JwtHeader)
	if refreshToken == "" {
		h.responseHelper.SendErrorResponse(w, "Refresh token is required", constants.BadRequest, nil)
		return
	}

	claims, err := h.authHelper.ParseRefreshJwtToken(refreshToken)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid refresh token", constants.Unauthorized, err)
		return
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		h.responseHelper.SendErrorResponse(w, "Invalid refresh token claims", constants.Unauthorized, nil)
		return
	}

	departmentID, ok := claims["tid"].(string)
	if !ok || departmentID == "" {
		h.responseHelper.SendErrorResponse(w, "Invalid refresh token claims", constants.Unauthorized, nil)
		return
	}

	session, err := h.sessionRepo.FindByRefreshToken(refreshToken)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid refresh token", constants.Unauthorized, err)
		return
	}
	if session.UserID != userID || session.DepartmentID != departmentID {
		h.responseHelper.SendErrorResponse(w, "Invalid refresh token", constants.Unauthorized, nil)
		return
	}

	user, err := h.userRepo.FindById(userID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error finding user", constants.InternalServerError, err)
		return
	}

	accessToken, err := h.authHelper.GenerateAccessJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error generating access token", constants.InternalServerError, err)
		return
	}

	newRefreshToken, err := h.authHelper.GenerateRefreshJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Error generating refresh token", constants.InternalServerError, err)
		return
	}

	if err := h.sessionRepo.RevokeSession(session.ID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Error rotating refresh token", constants.InternalServerError, err)
		return
	}

	jti, _ := claims["jti"].(string)
	if jti != "" {
		exp := time.Unix(int64(claims["exp"].(float64)), 0)
		if err := h.tokenHelper.BlacklistToken(jti, exp); err != nil {
			h.log.Warn().Err(err).Msg("Failed to blacklist old refresh token JTI")
		}
	}

	if err := h.createSession(r, user, departmentID, newRefreshToken); err != nil {
		h.responseHelper.SendErrorResponse(w, "Error creating session", constants.InternalServerError, err)
		return
	}

	h.authHelper.GenerateAccessCookie(accessToken, w)
	w.Header().Set(constants.JwtHeader, newRefreshToken)
	h.responseHelper.SendSuccessResponse(w, "Access token refreshed successfully", nil)
}

func (h *UserHandler) SendMagicLinkEmail(w http.ResponseWriter, r *http.Request) {
	var data models.MagicLinkEmailRequest
	var user *models.UserModel

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	departmentId := helpers.GetDepartmentId(r)
	if departmentId == "" {
		h.responseHelper.SendErrorResponse(w, "Missing department context", constants.Unauthorized, nil)
		return
	}

	user, err = h.userRepo.FindByEmailAndDepartment(data.Email, departmentId)

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.responseHelper.SendErrorResponse(w, "Error fetching user", constants.InternalServerError, err)
			return
		}

		h.log.Info().Str("email", h.encryptionHelper.MaskEmail(data.Email)).Msg("User does not exist")

		userId := uuid.New().String()

		user = &models.UserModel{
			ID:    userId,
			Email: data.Email,
		}

		err = h.userRepo.Create(user)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user", constants.InternalServerError, err)
			return
		}

		user_role := models.DepartmentRoles{
			ID:     departmentId,
			Role:   models.User,
			UserID: userId,
		}

		err = h.departmentRoleRepo.Create(&user_role)

		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Error creating user role", constants.InternalServerError, err)
			return
		}

	}

	token := h.authHelper.GenerateAuthToken()

	tmp := models.AuthModel{
		UserID: user.ID,
		Token:  token,
		Type:   models.MagicLink,
	}

	err = h.authRepo.Create(&tmp)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
		return
	}

	baseURL := strings.TrimRight(config.AppConfig.MagicLinkBaseUrl, "/")
	tmpl_data := models.MagicEmailData{
		Name:  user.Name,
		Url:   baseURL + constants.MagicLinkVerifyEndpoint,
		Token: token,
	}

	err = h.emailHelper.SendEmail(data.Email, "magic-link", tmpl_data)

	if err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "magic link sent to", nil)
}

func (h *UserHandler) VerifyMagicLinkEmail(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token string `json:"token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid request body", constants.BadRequest, err)
		return
	}

	if data.Token == "" {
		h.log.Error().Msg("Token is empty")
		h.responseHelper.SendErrorResponse(w, "Token is required", constants.BadRequest, nil)
		return
	}

	record, err := h.authRepo.FindByTokenAndType(data.Token, models.MagicLink)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to find magic link token")
		h.responseHelper.SendErrorResponse(w, "Invalid or expired magic link", constants.BadRequest, err)
		return
	}

	user, err := h.userRepo.FindById(record.UserID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid magic link", constants.BadRequest, nil)
		return
	}

	departmentId := helpers.GetDepartmentId(r)
	if departmentId == "" {
		role, err := h.departmentRoleRepo.FindByUserID(user.ID)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to resolve department for user", constants.InternalServerError, err)
			return
		}
		departmentId = role.ID
	}

	accessToken, refreshToken, err := h.authHelper.GenerateTokens(user.ID, user.Email, user.Name, departmentId)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to generate tokens")
		h.responseHelper.SendErrorResponse(w, "Failed to generate authentication tokens", constants.InternalServerError, err)
		return
	}

	if err := h.createSession(r, user, departmentId, refreshToken); err != nil {
		h.responseHelper.SendErrorResponse(w, "Error creating session", constants.InternalServerError, err)
		return
	}

	err = h.authRepo.DeleteByTokenAndType(data.Token, models.MagicLink)
	if err != nil {
		h.log.Warn().Err(err).Msg("Failed to delete magic link token after verification")
	}

	h.authHelper.GenerateAccessCookie(accessToken, w)
	w.Header().Set(constants.JwtHeader, refreshToken)

	response := map[string]interface{}{
		"accessToken": accessToken,
		"user": map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	}

	h.responseHelper.SendSuccessResponse(w, "Magic link verified successfully", response)
}

func (h *UserHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Header.Get(constants.JwtHeader)
	if refreshToken == "" {
		cookie, err := r.Cookie(constants.AccessTokenCookie)
		if err == nil && cookie.Value != "" {
			decoded, err := h.authHelper.DecodeAccessCookie(cookie.Value)
			if err == nil && decoded != "" {
				claims, err := h.authHelper.ParseAccessJwtToken(decoded)
				if err == nil {
					if jti, ok := claims["jti"].(string); ok && jti != "" {
						exp := time.Unix(int64(claims["exp"].(float64)), 0)
						if err := h.tokenHelper.BlacklistToken(jti, exp); err != nil {
							h.log.Warn().Err(err).Msg("Failed to blacklist access token JTI on logout")
						}
					}
					if sub, ok := claims["sub"].(string); ok && sub != "" {
						_ = h.sessionRepo.RevokeAllUserSessions(sub)
					}
				}
			}
		}
	} else {
		claims, err := h.authHelper.ParseRefreshJwtToken(refreshToken)
		if err == nil {
			if jti, ok := claims["jti"].(string); ok && jti != "" {
				exp := time.Unix(int64(claims["exp"].(float64)), 0)
				if err := h.tokenHelper.BlacklistToken(jti, exp); err != nil {
					h.log.Warn().Err(err).Msg("Failed to blacklist refresh token JTI on logout")
				}
			}
			if sub, ok := claims["sub"].(string); ok && sub != "" {
				_ = h.sessionRepo.RevokeAllUserSessions(sub)
			}
		}
		session, err := h.sessionRepo.FindByRefreshToken(refreshToken)
		if err == nil && session != nil {
			_ = h.sessionRepo.RevokeSession(session.ID)
		}
	}

	h.securityLogger.LogAuthEvent(r, "logout", "", "", "success", "")

	http.SetCookie(w, &http.Cookie{
		Name:     constants.AccessTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   config.AppConfig.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set(constants.JwtHeader, "")
	h.responseHelper.SendSuccessResponse(w, "Logged out successfully", nil)
}

func (h *UserHandler) createSession(r *http.Request, user *models.UserModel, departmentID, refreshToken string) error {
	expiresAt := time.Now().Add(time.Duration(config.AppConfig.RefreshJwtExpire) * time.Hour)
	session := &models.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		DepartmentID: departmentID,
		RefreshToken: refreshToken,
		IPAddress:    getClientIP(r),
		UserAgent:    r.UserAgent(),
		ExpiresAt:    expiresAt,
	}
	return h.sessionRepo.Create(session)
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
