package helpers

import (
	"fmt"
	"regexp"
	"uas/config"
	"unicode"

	"github.com/rs/zerolog"
)

type PasswordHelper struct {
	log *zerolog.Logger
}

func NewPasswordHelper(log *zerolog.Logger) *PasswordHelper {
	return &PasswordHelper{log: log}
}

type PasswordValidationError struct {
	Message string
	Code    string
}

func (e PasswordValidationError) Error() string {
	return e.Message
}

func (h *PasswordHelper) ValidateComplexity(password string) error {
	if len(password) < config.AppConfig.PasswordMinLength {
		return PasswordValidationError{
			Message: fmt.Sprintf("Password must be at least %d characters long", config.AppConfig.PasswordMinLength),
			Code:    "PASSWORD_TOO_SHORT",
		}
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case !unicode.IsLetter(char) && !unicode.IsDigit(char):
			hasSpecial = true
		}
	}

	if config.AppConfig.PasswordRequireUppercase && !hasUpper {
		return PasswordValidationError{
			Message: "Password must contain at least one uppercase letter",
			Code:    "PASSWORD_MISSING_UPPERCASE",
		}
	}

	if config.AppConfig.PasswordRequireLowercase && !hasLower {
		return PasswordValidationError{
			Message: "Password must contain at least one lowercase letter",
			Code:    "PASSWORD_MISSING_LOWERCASE",
		}
	}

	if config.AppConfig.PasswordRequireNumber && !hasNumber {
		return PasswordValidationError{
			Message: "Password must contain at least one number",
			Code:    "PASSWORD_MISSING_NUMBER",
		}
	}

	if config.AppConfig.PasswordRequireSpecial && !hasSpecial {
		return PasswordValidationError{
			Message: "Password must contain at least one special character",
			Code:    "PASSWORD_MISSING_SPECIAL",
		}
	}

	return nil
}

func (h *PasswordHelper) CheckCommonPassword(password string) bool {
	commonPasswords := map[string]bool{
		"password": true,
		"123456":   true,
		"12345678": true,
		"qwerty":   true,
		"abc123":   true,
		"monkey":   true,
		"1234567":  true,
		"letmein":  true,
		"trustno1": true,
		"dragon":   true,
		"baseball": true,
		"111111":   true,
		"iloveyou": true,
		"master":   true,
		"sunshine": true,
		"ashley":   true,
		"bailey":   true,
		"shadow":   true,
		"123123":   true,
		"654321":   true,
	}

	lowerPassword := regexp.MustCompile("[^a-z0-9]+").ReplaceAllString(password, "")
	return commonPasswords[lowerPassword]
}

func (h *PasswordHelper) IsPasswordStrong(password string) (bool, []string) {
	errors := []string{}

	if err := h.ValidateComplexity(password); err != nil {
		errors = append(errors, err.Error())
	}

	if h.CheckCommonPassword(password) {
		errors = append(errors, "Password is too common")
	}

	return len(errors) == 0, errors
}
