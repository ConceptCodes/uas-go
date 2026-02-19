package helpers

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"

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

	parsedTemplate, err := template.ParseFiles(templatePath)
	if err != nil {
		c.logger.
			Error().
			Err(err).
			Str("template", templatePath).
			Interface("args", args).
			Msg("Error while reading email template")
		return "", err
	}
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
			Subject: "Reset your password",
			Component: func(data interface{}) string {
				tmpl, err := c.LoadTemplate("forgot-password", data.(models.ForgotPasswordData))
				if err != nil {
					return ""
				}

				return tmpl
			},
		},
		"reset-password": {
			Subject: "Reset your password",
			Component: func(data interface{}) string {
				tmpl, err := c.LoadTemplate("reset-password", data.(models.ForgotPasswordData))
				if err != nil {
					return ""
				}

				return tmpl
			},
		},
		"verify-email": {
			Subject: "Verify your email",
			Component: func(data interface{}) string {
				tmpl, err := c.LoadTemplate("verify-email", data.(models.VerifyEmailData))
				if err != nil {
					return ""
				}
				return tmpl
			},
		},
		"magic-link": {
			Subject: "Your magic login link",
			Component: func(data interface{}) string {
				tmpl, err := c.LoadTemplate("magic-link", data.(models.MagicEmailData))
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
		return fmt.Errorf("template not found: %s", template)
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

	if config.AppConfig.EmailProvider == "mock" {
		preview := html
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		c.logger.Info().
			Str("provider", "mock").
			Str("email", email).
			Str("template", template).
			Str("subject", subject).
			Str("preview", strings.ReplaceAll(preview, "\n", " ")).
			Msg("Mock email send")
		return nil
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
