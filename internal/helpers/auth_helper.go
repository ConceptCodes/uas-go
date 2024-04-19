package helpers

import (
	"encoding/base64"
	"errors"
	"strings"
	repository "uas/internal/repositories"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

type AuthHelper struct {
	log        *zerolog.Logger
	tenantRepo repository.TenantRepository
}

func NewAuthHelper(log *zerolog.Logger, tenantRepo repository.TenantRepository) *AuthHelper {
	return &AuthHelper{log: log, tenantRepo: tenantRepo}
}

func (h *AuthHelper) GenerateBasicAuthToken(tenantId string, tenantSecret string) string {
	h.log.Debug().Msgf("Generating basic auth token for tenant: %s", tenantId)
	return base64.StdEncoding.EncodeToString([]byte(tenantId + ":" + tenantSecret))
}

func (h *AuthHelper) ValidateBasicAuthToken(token string) (string, error) {
	h.log.Debug().Msgf("Validating token: %s", token)

	data, err := base64.StdEncoding.DecodeString(token)

	if err != nil {
		h.log.Error().Err(err).Msg("Error decoding token")
		return "", err
	}

	parts := strings.Split(string(data), ":")
	if len(parts) < 2 {
		return "", errors.New("invalid token format")
	}

	tenantId := parts[0]
	tenantSecret := parts[1]

	tenant, err := h.tenantRepo.FindById(tenantId)

	if err != nil {
		return "", err
	}

	valid := h.CheckPasswordHash(tenantSecret, tenant.Secret)

	if valid {
		return tenantId, nil
	}

	return "", err
}

func (h *AuthHelper) HashPassword(password string) (string, error) {
	h.log.Debug().Msg("Hashing password")
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (h *AuthHelper) CheckPasswordHash(password, hash string) bool {
	h.log.Debug().Msg("Checking password hash")
	if password == "" || hash == "" {
		h.log.Error().Msg("Password or hash is empty")
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
