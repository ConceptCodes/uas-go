package helpers

import (
	"errors"
	"fmt"
	"time"
	"uas/config"
	"uas/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func (h *AuthHelper) GenerateAccessJwtToken(user *models.UserModel, tenant string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.Name)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":       user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"departmentId": tenant,
		"exp":          time.Now().Add(time.Hour * time.Duration(config.AppConfig.AccessJwtExpire)).Unix(),
	})

	token, err := t.SignedString([]byte(config.AppConfig.AccessJwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT Access token")
	}

	return token, nil
}

func (h *AuthHelper) ParseAccessJwtToken(tokenString string) (jwt.MapClaims, error) {
	return parseJWT(tokenString, config.AppConfig.AccessJwtSecret)
}

func (h *AuthHelper) GenerateRefreshJwtToken(user *models.UserModel, tenant string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.Name)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":       user.ID,
		"departmentId": tenant,
		"exp":          time.Now().Add(time.Hour * time.Duration(config.AppConfig.RefreshJwtExpire)).Unix(),
	})

	token, err := t.SignedString([]byte(config.AppConfig.RefreshJwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT Refresh token")
	}

	return token, nil
}

func (h *AuthHelper) ParseRefreshJwtToken(tokenString string) (jwt.MapClaims, error) {
	return parseJWT(tokenString, config.AppConfig.RefreshJwtSecret)
}

func (h *AuthHelper) GenerateTokens(userID, userEmail, userName, tenant string) (string, string, error) {
	h.log.Debug().Msgf("Generating JWT tokens for user: %s", userName)

	// Create user model for token generation
	user := &models.UserModel{
		ID:    userID,
		Email: userEmail,
		Name:  userName,
	}

	// Generate access token
	accessToken, err := h.GenerateAccessJwtToken(user, tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := h.GenerateRefreshJwtToken(user, tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func parseJWT(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("error extracting claims")
	}

	return claims, nil
}
