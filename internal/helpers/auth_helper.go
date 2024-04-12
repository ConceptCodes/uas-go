package helpers

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"uas/config"
	"uas/internal/models"
	repository "uas/internal/repositories"
	"uas/pkg/storage/mysql"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

type AuthHelper struct {
	log *zerolog.Logger
}

func NewAuthHelper(log *zerolog.Logger) *AuthHelper {
	return &AuthHelper{log: log}
}

func (h *AuthHelper) GenerateBasicAuthToken(tenantId string, tenantSecret string) string {
	h.log.Debug().Msgf("Generating basic auth token for tenant: %s", tenantId)
	return base64.StdEncoding.EncodeToString([]byte(tenantId + ":" + tenantSecret))
}

func (h *AuthHelper) ValidateBasicAuthToken(token string) (string, error) {
	db, err := mysql.New(h.log)

	if err != nil {
		h.log.Error().Err(err).Msg("Error getting db instance")
	}

	h.log.Debug().Msgf("Validating token: %s", token)

	tenantRepo := repository.NewGormTenantRepository(db)

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

	tenant, err := tenantRepo.FindById(tenantId)

	if err != nil {
		return "", err
	}

	valid := h.CheckPasswordHash(tenantSecret, tenant.Secret)

	if valid {
		return tenantId, nil
	}

	return "", err
}

func (h *AuthHelper) GenerateJwtToken(user *models.UserModel, tenant string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.Name)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name":          user.Name,
		"email":         user.Email,
		"department_id": tenant,
		"exp":           time.Now().Add(time.Hour * time.Duration(config.AppConfig.JwtExpire)).Unix(),
	})

	token, err := t.SignedString([]byte(config.AppConfig.JwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT token")
	}

	return token, nil
}

func (h *AuthHelper) ParseJwtToken(tokenString string) (jwt.MapClaims, error) {
	h.log.Debug().Msgf("Parsing JWT token: %s", tokenString)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("error extracting claims")
	}

	return claims, nil
}

func (h *AuthHelper) HashPassword(password string) (string, error) {
	h.log.Debug().Msg("Hashing password")
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (h *AuthHelper) CheckPasswordHash(password, hash string) bool {
	h.log.Debug().Msg("Checking password hash")
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
