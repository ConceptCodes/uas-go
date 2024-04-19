package handlers

import (
	"net/http"
	"sync"
	"uas/internal/helpers"
	"uas/internal/models"
	"uas/pkg/storage/mysql"

	"github.com/rs/zerolog"
)

type HealthHandler struct {
	logger          *zerolog.Logger
	responseHelper  *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
}

type Service struct {
	Name string
	Fn   func() bool
}

func NewHealthHandler(
	logger *zerolog.Logger,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
) *HealthHandler {
	return &HealthHandler{
		logger:          logger,
		responseHelper:  responseHelper,
		validatorHelper: validatorHelper,
	}
}

func (h *HealthHandler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {

	services := []Service{
		{Name: "MySQL", Fn: func() bool { return mysql.HealthCheck(*h.logger) }},
	}

	var wg sync.WaitGroup
	responses := make([]models.HealthCheckResponse, len(services))

	wg.Add(len(services))

	for i, service := range services {
		go func(i int, service Service) {
			defer wg.Done()
			responses[i] = Runner(service.Name, service.Fn)
		}(i, service)
	}

	wg.Wait()

	h.responseHelper.SendSuccessResponse(w, "Service report", responses)
	return
}

func Runner(name string, fn func() bool) models.HealthCheckResponse {
	return models.HealthCheckResponse{
		Service: name,
		Status:  fn(),
	}
}
