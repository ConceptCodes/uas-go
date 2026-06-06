package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog"
)

type MfaHandler struct {
	mfaFactorRepo   repository.MfaFactorRepository
	mfaChallengeRepo repository.MfaChallengeRepository
	userRepo        repository.UserRepository
	authHelper      *helpers.AuthHelper
	mfaHelper       *helpers.MfaHelper
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
	log             *zerolog.Logger
}

func NewMfaHandler(
	mfaFactorRepo repository.MfaFactorRepository,
	mfaChallengeRepo repository.MfaChallengeRepository,
	userRepo repository.UserRepository,
	authHelper *helpers.AuthHelper,
	mfaHelper *helpers.MfaHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	log *zerolog.Logger,
) *MfaHandler {
	return &MfaHandler{
		mfaFactorRepo:   mfaFactorRepo,
		mfaChallengeRepo: mfaChallengeRepo,
		userRepo:        userRepo,
		authHelper:      authHelper,
		mfaHelper:       mfaHelper,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
		log:             log,
	}
}

func (h *MfaHandler) EnrollHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	var data models.MfaEnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	user, err := h.userRepo.FindById(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "User not found", constants.NotFound, err)
		return
	}

	factor := &models.MfaFactor{
		UserID:       userID,
		DepartmentID: departmentID,
		FactorType:   data.FactorType,
		Name:         data.Name,
	}

	var response models.MfaEnrollResponse

	switch data.FactorType {
	case models.MfaFactorTOTP:
		secret, err := h.mfaHelper.GenerateTOTPSecret()
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to generate TOTP secret", constants.InternalServerError, err)
			return
		}
		factor.Secret = secret

		codes, err := h.mfaHelper.GenerateBackupCodes(config.AppConfig.MfaBackupCodeCount)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to generate backup codes", constants.InternalServerError, err)
			return
		}

		hashedCodes := make([]string, len(codes))
		for i, c := range codes {
			hashedCodes[i] = h.mfaHelper.HashBackupCode(c)
		}
		codesJSON, _ := json.Marshal(hashedCodes)
		factor.BackupCodes = string(codesJSON)

		if err := h.mfaFactorRepo.Create(factor); err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to enroll MFA factor", constants.InternalServerError, err)
			return
		}

		response = models.MfaEnrollResponse{
			FactorID:    factor.ID,
			FactorType:  string(data.FactorType),
			Secret:      secret,
			QRCodeURI:   h.mfaHelper.GetTOTPQRCodeURI(secret, user.Email),
			BackupCodes: codes,
		}

	case models.MfaFactorSMS:
		if data.PhoneNumber == "" {
			h.responseHelper.SendErrorResponse(w, "Phone number required for SMS factor", constants.BadRequest, nil)
			return
		}
		factor.Secret = data.PhoneNumber
		factor.Name = "SMS: " + data.PhoneNumber

		codes, _ := h.mfaHelper.GenerateBackupCodes(config.AppConfig.MfaBackupCodeCount)
		hashedCodes := make([]string, len(codes))
		for i, c := range codes {
			hashedCodes[i] = h.mfaHelper.HashBackupCode(c)
		}
		codesJSON, _ := json.Marshal(hashedCodes)
		factor.BackupCodes = string(codesJSON)

		if err := h.mfaFactorRepo.Create(factor); err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to enroll MFA factor", constants.InternalServerError, err)
			return
		}

		response = models.MfaEnrollResponse{
			FactorID:    factor.ID,
			FactorType:  string(data.FactorType),
			BackupCodes: codes,
		}

	default:
		h.responseHelper.SendErrorResponse(w, "Unsupported MFA factor type", constants.BadRequest, nil)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "MFA factor enrolled. Verify by providing a valid code.", response)
}

func (h *MfaHandler) VerifyEnrollHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	var data models.MfaVerifyEnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	factor, err := h.mfaFactorRepo.FindByID(data.FactorID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Factor not found", constants.NotFound, err)
		return
	}

	if factor.UserID != userID {
		h.responseHelper.SendErrorResponse(w, "Factor does not belong to user", constants.Forbidden, nil)
		return
	}

	var valid bool
	switch factor.FactorType {
	case models.MfaFactorTOTP:
		valid = h.mfaHelper.ValidateTOTPCode(factor.Secret, data.Code)
	case models.MfaFactorSMS:
		otpKey := "mfa:sms:" + userID + ":" + data.FactorID
		err := h.authHelper.ValidateOtpCode(otpKey, data.Code)
		valid = err == nil
	default:
		h.responseHelper.SendErrorResponse(w, "Unsupported factor type", constants.BadRequest, nil)
		return
	}

	if !valid {
		h.responseHelper.SendErrorResponse(w, "Invalid verification code", constants.BadRequest, nil)
		return
	}

	if count, _ := h.mfaFactorRepo.CountActiveByUserID(userID, departmentID); count == 0 {
		factor.IsPrimary = true
	}

	h.responseHelper.SendSuccessResponse(w, "MFA factor verified successfully", models.MfaFactorResponse{
		ID:         factor.ID,
		FactorType: string(factor.FactorType),
		Name:       factor.Name,
		IsPrimary:  factor.IsPrimary,
		CreatedAt:  factor.CreatedAt,
	})
}

func (h *MfaHandler) ListFactorsHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	factors, err := h.mfaFactorRepo.FindByUserID(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to list factors", constants.InternalServerError, err)
		return
	}

	var response []models.MfaFactorResponse
	for _, f := range factors {
		response = append(response, models.MfaFactorResponse{
			ID:         f.ID,
			FactorType: string(f.FactorType),
			Name:       f.Name,
			IsPrimary:  f.IsPrimary,
			LastUsedAt: f.LastUsedAt,
			CreatedAt:  f.CreatedAt,
		})
	}

	h.responseHelper.SendSuccessResponse(w, "MFA factors retrieved", response)
}

func (h *MfaHandler) DisableHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	var data models.MfaDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	user, err := h.userRepo.FindById(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "User not found", constants.NotFound, err)
		return
	}

	if !h.authHelper.CheckPasswordHash(data.Password, user.Password) {
		h.responseHelper.SendErrorResponse(w, "Invalid password", constants.Unauthorized, nil)
		return
	}

	if err := h.mfaFactorRepo.Delete(data.FactorID, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to disable MFA factor", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "MFA factor disabled successfully", nil)
}

func (h *MfaHandler) ChallengeVerifyHandler(w http.ResponseWriter, r *http.Request) {
	var data models.MfaChallengeVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	userID, departmentID, err := h.mfaHelper.ValidateMFATempToken(data.TempToken)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "MFA session expired or invalid", constants.Unauthorized, err)
		return
	}

	var factor *models.MfaFactor
	if data.FactorID != "" {
		factor, err = h.mfaFactorRepo.FindByID(data.FactorID, departmentID)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Factor not found", constants.NotFound, err)
			return
		}
		if factor.UserID != userID {
			h.responseHelper.SendErrorResponse(w, "Factor does not belong to user", constants.Forbidden, nil)
			return
		}
	} else {
		factor, err = h.mfaFactorRepo.GetPrimaryFactor(userID, departmentID)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "No MFA factor configured", constants.BadRequest, err)
			return
		}
	}

	var valid bool
	switch factor.FactorType {
	case models.MfaFactorTOTP:
		valid = h.mfaHelper.ValidateTOTPCode(factor.Secret, data.Code)
	case models.MfaFactorSMS:
		otpKey := "mfa:sms:" + userID + ":" + factor.ID
		err := h.authHelper.ValidateOtpCode(otpKey, data.Code)
		valid = err == nil
	default:
		h.responseHelper.SendErrorResponse(w, "Unsupported factor type", constants.BadRequest, nil)
		return
	}

	if !valid {
		h.responseHelper.SendErrorResponse(w, "Invalid MFA code", constants.Unauthorized, nil)
		return
	}

	h.mfaFactorRepo.UpdateLastUsed(factor.ID, departmentID)

	user, err := h.userRepo.FindById(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "User not found", constants.InternalServerError, err)
		return
	}

	accessToken, err := h.authHelper.GenerateAccessJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate access token", constants.InternalServerError, err)
		return
	}

	refreshToken, err := h.authHelper.GenerateRefreshJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate refresh token", constants.InternalServerError, err)
		return
	}

	h.authHelper.GenerateAccessCookie(accessToken, w)
	w.Header().Set(constants.JwtHeader, refreshToken)

	h.responseHelper.SendSuccessResponse(w, "MFA verification successful", nil)
}

func (h *MfaHandler) RecoverHandler(w http.ResponseWriter, r *http.Request) {
	var data models.MfaRecoverRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	userID, departmentID, err := h.mfaHelper.ValidateMFATempToken(data.TempToken)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "MFA session expired or invalid", constants.Unauthorized, err)
		return
	}

	factors, err := h.mfaFactorRepo.FindByUserID(userID, departmentID)
	if err != nil || len(factors) == 0 {
		h.responseHelper.SendErrorResponse(w, "No MFA factors configured", constants.BadRequest, nil)
		return
	}

	for _, factor := range factors {
		if factor.BackupCodes == "" {
			continue
		}
		valid, remainingCodes, err := h.mfaHelper.VerifyBackupCode(data.BackupCode, factor.BackupCodes)
		if err != nil {
			continue
		}
		if valid {
			h.mfaFactorRepo.UpdateBackupCodes(factor.ID, departmentID, remainingCodes)

			user, err := h.userRepo.FindById(userID, departmentID)
			if err != nil {
				h.responseHelper.SendErrorResponse(w, "User not found", constants.InternalServerError, err)
				return
			}

			accessToken, err := h.authHelper.GenerateAccessJwtToken(user, departmentID)
			if err != nil {
				h.responseHelper.SendErrorResponse(w, "Failed to generate access token", constants.InternalServerError, err)
				return
			}

			refreshToken, err := h.authHelper.GenerateRefreshJwtToken(user, departmentID)
			if err != nil {
				h.responseHelper.SendErrorResponse(w, "Failed to generate refresh token", constants.InternalServerError, err)
				return
			}

			h.authHelper.GenerateAccessCookie(accessToken, w)
			w.Header().Set(constants.JwtHeader, refreshToken)

			h.responseHelper.SendSuccessResponse(w, "MFA recovery successful", nil)
			return
		}
	}

	h.responseHelper.SendErrorResponse(w, "Invalid backup code", constants.Unauthorized, nil)
}

func (h *MfaHandler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	count, err := h.mfaFactorRepo.CountActiveByUserID(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to check MFA status", constants.InternalServerError, err)
		return
	}

	factors, _ := h.mfaFactorRepo.FindByUserID(userID, departmentID)
	var factorResponses []models.MfaFactorResponse
	for _, f := range factors {
		factorResponses = append(factorResponses, models.MfaFactorResponse{
			ID:         f.ID,
			FactorType: string(f.FactorType),
			Name:       f.Name,
			IsPrimary:  f.IsPrimary,
			LastUsedAt: f.LastUsedAt,
			CreatedAt:  f.CreatedAt,
		})
	}

	h.responseHelper.SendSuccessResponse(w, "MFA status retrieved", map[string]interface{}{
		"mfaEnabled": count > 0,
		"factors":    factorResponses,
	})
}

func NewMfaRequiredError() *models.AppError {
	return &models.AppError{
		Code:       constants.MFARequired,
		Message:    "MFA verification required",
		HTTPStatus: http.StatusTooManyRequests, // Using 429 to signal MFA needed, client checks code
		Timestamp:  time.Now(),
	}
}