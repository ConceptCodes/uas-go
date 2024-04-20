package helpers

import (
	"encoding/json"
	"fmt"
	"uas/config"

	"github.com/rs/zerolog"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioHelper struct {
	twilioClient *twilio.RestClient
	log          *zerolog.Logger
}

func NewTwilioHelper(log *zerolog.Logger, twilioClient *twilio.RestClient) *TwilioHelper {
	return &TwilioHelper{
		twilioClient: twilioClient,
		log:          log,
	}
}

func (t *TwilioHelper) SendSMS(to, from, body string) error {

	if to == "" || from == "" || body == "" {
		return fmt.Errorf("to, from, and body are required")
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(config.AppConfig.TwilioPhoneNumber)
	params.SetBody(body)

	resp, err := t.twilioClient.Api.CreateMessage(params)

	if err != nil {
		t.log.Error().Err(err).Msg("Error while sending SMS")
		return err
	} else {
		response, _ := json.Marshal(*resp)
		t.log.Debug().Bytes("response", response).Msgf("SMS sent successfully")
	}

	return nil
}
