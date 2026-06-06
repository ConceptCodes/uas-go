package helpers

import (
	"io"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	s, err := GenerateRandomString(16)
	require.NoError(t, err)
	assert.Len(t, s, 16)

	s2, err := GenerateRandomString(16)
	require.NoError(t, err)
	assert.NotEqual(t, s, s2)
}

func TestGenerateRandomStringDifferentLengths(t *testing.T) {
	for _, length := range []int{8, 16, 32, 64} {
		s, err := GenerateRandomString(length)
		require.NoError(t, err)
		assert.Len(t, s, length)
	}
}

func TestGenerateBasicAuthToken(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := &AuthHelper{log: &log}

	token := h.GenerateBasicAuthToken("tenant-abc", "secret-123")
	assert.NotEmpty(t, token)
	assert.True(t, strings.HasPrefix(token, "dGVuYW50LWFiYzpzZWNyZXQtMTIz") ||
		strings.Contains(token, "tenant"))
}

func TestGenerateAuthToken(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := &AuthHelper{log: &log}

	token := h.GenerateAuthToken()
	assert.NotEmpty(t, token)

	token2 := h.GenerateAuthToken()
	assert.NotEqual(t, token, token2)
}
