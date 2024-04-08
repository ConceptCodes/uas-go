package helpers

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"uas/config"
	"uas/internal/models"
	repository "uas/internal/repositories"
	"uas/pkg/logger"
	mysql "uas/pkg/storage"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func GenerateBasicAuthToken(tenantId string, tenantSecret string) string {
	return base64.StdEncoding.EncodeToString([]byte(tenantId + ":" + tenantSecret))
}

func ValidateBasicAuthToken(token string) (string, error) {
	db, err := mysql.New()

	log := logger.New()

	if err != nil {
		log.Error().Err(err).Msg("Error getting db instance")
	}

	log.Debug().Msgf("Validating token: %s", token)

	tenantRepo := repository.NewGormTenantRepository(db)

	data, err := base64.StdEncoding.DecodeString(token)

	if err != nil {
		log.Error().Err(err).Msg("Error decoding token")
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

	valid := CheckPasswordHash(tenantSecret, tenant.Secret)

	if valid {
		return tenantId, nil
	}

	return "", err
}

func GenerateJwtToken(user *models.UserModel, tenant string) (string, error) {
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

func ParseJwtToken(tokenString string) (jwt.MapClaims, error) {
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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
