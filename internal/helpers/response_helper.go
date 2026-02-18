package helpers

import (
	"encoding/json"
	"net/http"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/rs/zerolog"
)

type ResponseHelper struct {
	log *zerolog.Logger
}

func NewResponseHelper(log *zerolog.Logger) *ResponseHelper {
	return &ResponseHelper{log: log}
}

func (r *ResponseHelper) SendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	response := models.SuccessResponse{
		Message: message,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		r.log.Error().Err(err).Msg("Failed to encode success response")
	}
}

func (r *ResponseHelper) SendErrorResponse(w http.ResponseWriter, message string, errorCode string, err error) {
	if err != nil {
		r.log.Error().Err(err).Msg(message)
	} else {
		r.log.Error().Str("code", errorCode).Msg(message)
	}

	response := models.ErrorResponse{
		Error:   errorCode,
		Code:    errorCode,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	switch errorCode {
	case constants.NotFound:
		w.WriteHeader(http.StatusNotFound)
	case constants.Unauthorized:
		w.WriteHeader(http.StatusUnauthorized)
	case constants.InternalServerError:
		w.WriteHeader(http.StatusInternalServerError)
	case constants.Forbidden:
		w.WriteHeader(http.StatusForbidden)
	case constants.BadRequest:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		r.log.Error().Err(encodeErr).Msg("Failed to encode error response")
	}
}
