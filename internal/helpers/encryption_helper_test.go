package helpers

import (
	"io"
	"strings"
	"testing"
	"uas/config"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEncryptionTest(t *testing.T) (*EncryptionHelper, func()) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		EncryptionKey: "0123456789abcdef0123456789abcdef",
	}
	log := zerolog.New(io.Discard)
	h, err := NewEncryptionHelper(&log)
	require.NoError(t, err)
	return h, func() { config.AppConfig = orig }
}

func TestEncryptDecrypt(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	plaintext := "Hello, World!"
	ciphertext, err := h.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := h.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptEmptyString(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	ciphertext, err := h.Encrypt("")
	require.NoError(t, err)

	decrypted, err := h.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestDecryptInvalidBase64(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	_, err := h.Decrypt("not-base64!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base64")
}

func TestDecryptTooShort(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	_, err := h.Decrypt("short")
	assert.Error(t, err)
}

func TestEncryptUniqueCiphertext(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	c1, err := h.Encrypt("same data")
	require.NoError(t, err)
	c2, err := h.Encrypt("same data")
	require.NoError(t, err)
	assert.NotEqual(t, c1, c2)
}

func TestDeriveKey(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	key1, err := h.DeriveKey("password", "salt1234")
	require.NoError(t, err)
	assert.Len(t, key1, 32)

	key2, err := h.DeriveKey("password", "different")
	require.NoError(t, err)
	assert.NotEqual(t, key1, key2)
}

func TestGenerateSalt(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	salt1, err := h.GenerateSalt()
	require.NoError(t, err)
	assert.NotEmpty(t, salt1)

	salt2, err := h.GenerateSalt()
	require.NoError(t, err)
	assert.NotEqual(t, salt1, salt2)
}

func TestMaskEmail(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "standard email", email: "user@example.com", want: "u***r@example.com"},
		{name: "short local", email: "a@b.com", want: "***@b.com"},
		{name: "no at sign", email: "noatsign", want: "n***n"},
		{name: "short no at", email: "ab", want: "***"},
		{name: "empty", email: "", want: "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, h.MaskEmail(tt.email))
		})
	}
}

func TestMaskPhone(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{name: "standard", phone: "+1234567890", want: "+1****90"},
		{name: "short", phone: "1234", want: "****"},
		{name: "very short", phone: "12", want: "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, h.MaskPhone(tt.phone))
		})
	}
}

func TestMaskName(t *testing.T) {
	h, cleanup := setupEncryptionTest(t)
	defer cleanup()

	tests := []struct {
		name   string
		input  string
		want   string
		prefix string
	}{
		{name: "standard", input: "John", want: "J***"},
		{name: "short", input: "Jo", want: "J*"},
		{name: "single char", input: "J", want: "J*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.MaskName(tt.input)
			assert.Equal(t, tt.want, result)
			assert.True(t, strings.HasPrefix(result, tt.input[:1]))
		})
	}
}

func TestNewEncryptionHelperInvalidKey(t *testing.T) {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		EncryptionKey: "too-short",
	}
	defer func() { config.AppConfig = orig }()

	log := zerolog.New(io.Discard)
	_, err := NewEncryptionHelper(&log)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}
