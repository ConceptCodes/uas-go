package helpers

import (
	"io"
	"testing"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupJWTTests(t *testing.T) func() {
	orig := config.AppConfig
	config.AppConfig = config.Config{
		AccessJwtSecret:  "this-is-a-32-byte-test-secret-key!!",
		RefreshJwtSecret: "this-is-another-32-byte-test-secret!",
		JwtIssuer:        "test-issuer",
		JwtAudience:      "test-audience",
		AccessJwtExpire:  1,
		RefreshJwtExpire: 24,
	}
	return func() { config.AppConfig = orig }
}

func TestParseJWTValid(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	claims := jwt.MapClaims{
		constants.JwtSubKey: "user-123",
		constants.JwtJtiKey: uuid.New().String(),
		constants.JwtTidKey: "tenant-abc",
		"iss":               config.AppConfig.JwtIssuer,
		"aud":               config.AppConfig.JwtAudience,
		"iat":               time.Now().Unix(),
		"nbf":               time.Now().Unix(),
		"exp":               time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.AccessJwtSecret))
	require.NoError(t, err)

	parsed, err := parseJWT(tokenString, config.AppConfig.AccessJwtSecret)
	require.NoError(t, err)
	assert.Equal(t, "user-123", parsed[constants.JwtSubKey])
	assert.Equal(t, "tenant-abc", parsed[constants.JwtTidKey])
}

func TestParseJWTWrongSecret(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte("different-secret-key-that-is-32-bytes!!"))
	require.NoError(t, err)

	_, err = parseJWT(tokenString, config.AppConfig.AccessJwtSecret)
	assert.Error(t, err)
}

func TestParseJWTExpiredToken(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(config.AppConfig.AccessJwtSecret))
	require.NoError(t, err)

	_, err = parseJWT(tokenString, config.AppConfig.AccessJwtSecret)
	assert.Error(t, err)
}

func TestParseJWTInvalidSigningMethod(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(config.AppConfig.AccessJwtSecret))
	require.NoError(t, err)

	_, err = parseJWT(tokenString, config.AppConfig.AccessJwtSecret)
	assert.Error(t, err)
}

func TestParseJWTInvalidToken(t *testing.T) {
	_, err := parseJWT("not-a-token", "secret-key-32-bytes-long!!")
	assert.Error(t, err)
}

func TestAuthHelperJWTTokenGeneration(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	log := zerolog.New(io.Discard)
	h := &AuthHelper{log: &log}

	user := createTestUser()

	accessToken, err := h.GenerateAccessJwtToken(user, "tenant-abc")
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	claims, err := h.ParseAccessJwtToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-id-123", claims[constants.JwtSubKey])
	assert.Equal(t, "tenant-abc", claims[constants.JwtTidKey])
	assert.Equal(t, []interface{}{"pwd"}, claims["amr"])
}

func TestAuthHelperJWTRefreshToken(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	log := zerolog.New(io.Discard)
	h := &AuthHelper{log: &log}

	user := createTestUser()

	refreshToken, err := h.GenerateRefreshJwtToken(user, "tenant-abc")
	require.NoError(t, err)
	assert.NotEmpty(t, refreshToken)

	claims, err := h.ParseRefreshJwtToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, "user-id-123", claims[constants.JwtSubKey])
}

func TestAuthHelperGenerateTokens(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	log := zerolog.New(io.Discard)
	h := &AuthHelper{log: &log}

	accessToken, refreshToken, err := h.GenerateTokens("user-id-123", "test@test.com", "Test User", "tenant-abc")
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	accessClaims, err := h.ParseAccessJwtToken(accessToken)
	require.NoError(t, err)
	refreshClaims, err := h.ParseRefreshJwtToken(refreshToken)
	require.NoError(t, err)

	assert.Equal(t, "user-id-123", accessClaims[constants.JwtSubKey])
	assert.Equal(t, "user-id-123", refreshClaims[constants.JwtSubKey])
}

func TestAuthHelperGenerateTokensWithAMR(t *testing.T) {
	cleanup := setupJWTTests(t)
	defer cleanup()

	log := zerolog.New(io.Discard)
	h := &AuthHelper{log: &log}

	user := createTestUser()
	amr := []string{"pwd", "otp"}

	accessToken, err := h.GenerateAccessJwtTokenWithAMR(user, "tenant-abc", amr)
	require.NoError(t, err)

	claims, err := h.ParseAccessJwtToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"pwd", "otp"}, claims["amr"])
}

func createTestUser() *models.UserModel {
	return &models.UserModel{
		ID:    "user-id-123",
		Email: "test@test.com",
		Name:  "Test User",
	}
}
