package email

import (
	"fmt"
	"sync"
	"time"

	"uas/config"
	"uas/internal/constants"

	"github.com/resend/resend-go/v2"
	"github.com/rs/zerolog"
)

var (
	once   sync.Once
	client *resend.Client
)

type EmailClient struct {
	logger *zerolog.Logger
}

func NewEmailClient(logger *zerolog.Logger) *EmailClient {
	return &EmailClient{logger: logger}
}

func New() *resend.Client {
	once.Do(func() {
		client = resend.NewClient(config.AppConfig.ResendApiKey)
	})

	return client
}

func (e *EmailClient) HealthCheck(r *resend.Client) bool {
	e.logger.Debug().Msgf(constants.HealthCheckMessage, "resend")

	_, err := r.Emails.Send(&resend.SendEmailRequest{
		From:    config.AppConfig.EmailFrom,
		To:      []string{"delivered@resend.dev"},
		Html:    fmt.Sprintf("Health check triggered @ %s", time.Now().Format(time.RFC3339)),
		Subject: "Health check",
	})

	if err != nil {
		e.logger.
			Error().
			Err(err).
			Msgf(constants.HealthCheckError, "resend")
		return false
	}

	e.logger.Info().Msg("Resend is up")
	return true
}
