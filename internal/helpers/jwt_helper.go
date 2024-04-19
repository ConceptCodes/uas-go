package helpers

import (
	"errors"
	"time"
	"uas/config"
	"uas/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func (h *AuthHelper) GenerateAccessJwtToken(user *models.UserModel, tenant string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.Name)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":            user.ID,
		"name":          user.Name,
		"email":         user.Email,
		"department_id": tenant,
		"exp":           time.Now().Add(time.Hour * time.Duration(config.AppConfig.AccessJwtExpire)).Unix(),
	})

	token, err := t.SignedString([]byte(config.AppConfig.AccessJwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT Access token")
	}

	return token, nil
}

func (h *AuthHelper) ParseAccessJwtToken(tokenString string) (jwt.MapClaims, error) {
	h.log.Debug().Msgf("Parsing JWT Access token: %s", tokenString)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.AccessJwtSecret), nil
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

func (h *AuthHelper) GenerateRefreshJwtToken(user *models.UserModel, tenant string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.Name)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":            user.ID,
		"department_id": tenant,
		"exp":           time.Now().Add(time.Hour * time.Duration(config.AppConfig.RefreshJwtExpire)).Unix(),
	})

	token, err := t.SignedString([]byte(config.AppConfig.RefreshJwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT Refresh token")
	}

	return token, nil
}

func (h *AuthHelper) ParseRefreshJwtToken(tokenString string) (jwt.MapClaims, error) {
	h.log.Debug().Msgf("Parsing JWT Refresh token: %s", tokenString)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.RefreshJwtSecret), nil
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
