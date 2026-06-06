package helpers

import (
	"io"
	"testing"
	"uas/config"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupPasswordTest(t *testing.T) (*PasswordHelper, func()) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		PasswordMinLength:        8,
		PasswordRequireUppercase: true,
		PasswordRequireLowercase: true,
		PasswordRequireNumber:    true,
		PasswordRequireSpecial:   true,
		PasswordHistoryCount:     5,
	}
	log := zerolog.New(io.Discard)
	h := NewPasswordHelper(&log)
	return h, func() { config.AppConfig = orig }
}

func TestPasswordValidateComplexity(t *testing.T) {
	h, cleanup := setupPasswordTest(t)
	defer cleanup()

	tests := []struct {
		name     string
		password string
		wantErr  bool
		wantCode string
	}{
		{name: "valid password", password: "Valid1#pass", wantErr: false},
		{name: "too short", password: "Ab1#x", wantErr: true, wantCode: "PASSWORD_TOO_SHORT"},
		{name: "missing uppercase", password: "valid1#pass", wantErr: true, wantCode: "PASSWORD_MISSING_UPPERCASE"},
		{name: "missing lowercase", password: "VALID1#PASS", wantErr: true, wantCode: "PASSWORD_MISSING_LOWERCASE"},
		{name: "missing number", password: "Valid#password", wantErr: true, wantCode: "PASSWORD_MISSING_NUMBER"},
		{name: "missing special", password: "Valid1password", wantErr: true, wantCode: "PASSWORD_MISSING_SPECIAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.ValidateComplexity(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				if pve, ok := err.(PasswordValidationError); ok {
					assert.Equal(t, tt.wantCode, pve.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordCheckCommonPassword(t *testing.T) {
	h, cleanup := setupPasswordTest(t)
	defer cleanup()

	tests := []struct {
		name     string
		password string
		common   bool
	}{
		{name: "common password", password: "password", common: true},
		{name: "common 123456", password: "123456", common: true},
		{name: "not common", password: "tr0ub4dour#M4n", common: false},
		{name: "common with noise", password: "password123!", common: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.common, h.CheckCommonPassword(tt.password))
		})
	}
}

func TestPasswordIsPasswordStrong(t *testing.T) {
	h, cleanup := setupPasswordTest(t)
	defer cleanup()

	tests := []struct {
		name     string
		password string
		strong   bool
	}{
		{name: "strong password", password: "Str0ng#Secure1", strong: true},
		{name: "too short", password: "Ab1#x", strong: false},
		{name: "common password", password: "password", strong: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strong, errors := h.IsPasswordStrong(tt.password)
			assert.Equal(t, tt.strong, strong)
			if !tt.strong {
				assert.NotEmpty(t, errors)
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

func TestPasswordComplexityConfigurable(t *testing.T) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		PasswordMinLength:        12,
		PasswordRequireUppercase: false,
		PasswordRequireLowercase: false,
		PasswordRequireNumber:    false,
		PasswordRequireSpecial:   false,
	}
	defer func() { config.AppConfig = orig }()

	log := zerolog.New(io.Discard)
	h := NewPasswordHelper(&log)

	err := h.ValidateComplexity("short")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "12")

	err = h.ValidateComplexity("onlylowercase")
	assert.NoError(t, err)
}
