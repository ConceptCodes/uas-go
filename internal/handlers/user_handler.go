package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog/log"
)

type UserHandler struct {
	userRepo repository.UserRepository
}

func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
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
// @Router /users [post]
func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var data models.CredentialsRegisterRequest

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		helpers.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
	}

	helpers.ValidateStruct(w, &data)

	password_hash, err := helpers.HashPassword(data.Password)
	err_message := fmt.Sprintf(constants.CreateEntityError, "User")

	if err != nil {
		log.Error().Err(err).Msg("Error hashing password")
		helpers.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	user := &models.UserModel{
		Name:     data.Name,
		Email:    data.Email,
		Password: password_hash,
	}

	err = h.userRepo.Save(user)

	if err != nil {
		helpers.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	token, err := helpers.GenerateJwtToken(user, "1")

	if err != nil {
		log.Error().Err(err).Msg("Error generating token")
		helpers.SendErrorResponse(w, err_message, constants.InternalServerError, err)
	}

	res := &models.JwtTokenResponse{
		Token: token,
	}

	helpers.SendSuccessResponse(w, "User registered successfully", res)
}
