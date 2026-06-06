package helpers

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Email       string `json:"email" validate:"required,validEmail"`
	Phone       string `json:"phone" validate:"validPhone"`
	NoSQL       string `json:"noSql" validate:"noSQLKeywords"`
	Required    string `json:"required" validate:"required"`
}

func TestValidatorNoSQLKeywords(t *testing.T) {
	log := zerolog.New(io.Discard)
	respHelper := NewResponseHelper(&log)
	v := NewValidatorHelper(&log, respHelper)

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "normal text", value: "hello world", valid: true},
		{name: "with null byte", value: "hello\x00world", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s := testStruct{Email: "test@test.com", Phone: "+1234567890", NoSQL: tt.value, Required: "x"}
			ok := v.ValidateStruct(w, &s)
			assert.Equal(t, tt.valid, ok)
		})
	}
}

func TestValidatorValidEmail(t *testing.T) {
	log := zerolog.New(io.Discard)
	respHelper := NewResponseHelper(&log)
	v := NewValidatorHelper(&log, respHelper)

	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{name: "valid email", email: "user@example.com", valid: true},
		{name: "valid plus", email: "user+tag@example.com", valid: true},
		{name: "invalid no at", email: "invalid", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s := testStruct{Email: tt.email, Phone: "+1234567890", NoSQL: "hello", Required: "x"}
			ok := v.ValidateStruct(w, &s)
			assert.Equal(t, tt.valid, ok)
		})
	}
}

func TestValidatorValidPhone(t *testing.T) {
	log := zerolog.New(io.Discard)
	respHelper := NewResponseHelper(&log)
	v := NewValidatorHelper(&log, respHelper)

	tests := []struct {
		name  string
		phone string
		valid bool
	}{
		{name: "valid E.164", phone: "+1234567890", valid: true},
		{name: "valid no plus", phone: "1234567890", valid: true},
		{name: "single digit", phone: "1", valid: false},
		{name: "with letters", phone: "+abc", valid: false},
		{name: "empty string", phone: "", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s := testStruct{Email: "test@test.com", Phone: tt.phone, NoSQL: "hello", Required: "x"}
			ok := v.ValidateStruct(w, &s)
			assert.Equal(t, tt.valid, ok)
		})
	}
}

func TestValidatorMissingRequiredField(t *testing.T) {
	log := zerolog.New(io.Discard)
	respHelper := NewResponseHelper(&log)
	v := NewValidatorHelper(&log, respHelper)

	w := httptest.NewRecorder()
	s := testStruct{Email: "test@test.com", Phone: "+1234567890", NoSQL: "hello", Required: ""}
	ok := v.ValidateStruct(w, &s)
	require.False(t, ok)
	assert.Equal(t, 400, w.Code)
}
