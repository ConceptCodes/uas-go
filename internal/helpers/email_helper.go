package helpers

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"errors"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/resend/resend-go/v2"
	"github.com/rs/zerolog"
)

type EmailTemplates map[string]struct {
	Subject   string
	Component func(data interface{}) string
}

type EmailHelper struct {
	logger *zerolog.Logger
	client *resend.Client
}

func NewEmailHelper(logger *zerolog.Logger, client *resend.Client) *EmailHelper {
	return &EmailHelper{logger: logger, client: client}
}

func (c EmailHelper) LoadTemplate(name string, args interface{}) (string, error) {
	cwd, _ := os.Getwd()
	templatePath := fmt.Sprintf(constants.EmailTemplatePath, cwd, name)

	_, err := os.Stat(templatePath)

	if os.IsNotExist(err) {
		_err := fmt.Errorf(constants.InvalidTemplatePathError, templatePath)
		c.logger.Error().
			Err(_err).
			Str("template", templatePath).
			Interface("args", args).
			Msg("Template does not exist")
		return "", _err
	}

	c.logger.
		Debug().
		Str("template", templatePath).
		Interface("args", args).
		Msg("Loading email template")

	parsedTemplate, _ := template.ParseFiles(templatePath)
	var tpl bytes.Buffer
	err = parsedTemplate.Execute(&tpl, args)

	if err != nil {
		err = fmt.Errorf(constants.InvalidTemplatePathError, templatePath)
		c.logger.
			Error().
			Err(err).
			Str("template", templatePath).
			Interface("args", args).
			Msg("Error while parsing email template")
		return "", err
	}

	result := tpl.String()

	if result == "" {
		err = errors.New("error while injecting variables in email template")
		c.logger.Error().
			Err(err).
			Str("template", templatePath).
			Interface("args", args).
			Msg(err.Error())

		return "", err
	}
	return result, nil
}

func (c *EmailHelper) SendEmail(email string, template string, data interface{}) error {

	var templates = EmailTemplates{
		"forgot-password": {
			Subject: constants.WelcomeEmailSubject,
			Component: func(data interface{}) string {
				tmpl, err := c.LoadTemplate("forgot-password", data.(models.ForgotPasswordData))
				if err != nil {
					return ""
				}

				return tmpl
			},
		},
	}

	templateInfo, exists := templates[template]

	if !exists {
		c.logger.Error().
			Str("template", template).
			Msg("Template not found")
	}

	subject := templateInfo.Subject
	html := templateInfo.Component(data)

	if html == "" {
		err := errors.New("error while loading email template")
		c.logger.Error().
			Err(err).
			Str("template", template).
			Msg(err.Error())
		return err
	}

	var domain = fmt.Sprintf(constants.EmailFrom, config.AppConfig.ResendEmailDomain)

	params := &resend.SendEmailRequest{
		From:    domain,
		To:      []string{email},
		Html:    html,
		Subject: subject,
	}

	sent, err := c.client.Emails.Send(params)

	if err != nil {
		c.logger.Error().
			Err(err).
			Str("email", email).
			Str("template", template).
			Msg("Error while sending email")
		return err
	}

	c.logger.Debug().
		Str("email", email).
		Str("template", template).
		Str("sent_id", sent.Id).
		Msg("Email sent successfully")

	return nil
}
