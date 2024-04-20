package twilio

import (
	"sync"
	"uas/config"

	"github.com/rs/zerolog"

	"github.com/twilio/twilio-go"
)

var (
	once   sync.Once
	client *twilio.RestClient
)

type TwilioClient struct {
	logger *zerolog.Logger
}

func NewTwilioClient(logger *zerolog.Logger) *TwilioClient {
	return &TwilioClient{logger: logger}
}

func New() *twilio.RestClient {
	once.Do(func() {
		client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: config.AppConfig.TwilioAccountSid,
			Password: config.AppConfig.TwilioAuthToken,
		})
	})

	return client
}
