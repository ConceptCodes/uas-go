package helpers

import (
	"errors"
	"fmt"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (h *AuthHelper) GenerateAccessJwtTokenWithAMR(user *models.UserModel, tenant string, amr []string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.ID)
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		constants.JwtSubKey: user.ID,
		constants.JwtJtiKey: uuid.New().String(),
		constants.JwtTidKey: tenant,
		"iss":               config.AppConfig.JwtIssuer,
		"aud":               config.AppConfig.JwtAudience,
		"iat":               now.Unix(),
		"nbf":               now.Unix(),
		"exp":               now.Add(time.Hour * time.Duration(config.AppConfig.AccessJwtExpire)).Unix(),
		"amr":               amr,
	})

	token, err := t.SignedString([]byte(config.AppConfig.AccessJwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT Access token")
	}

	return token, nil
}

func (h *AuthHelper) GenerateAccessJwtToken(user *models.UserModel, tenant string) (string, error) {
	return h.GenerateAccessJwtTokenWithAMR(user, tenant, []string{"pwd"})
}

func (h *AuthHelper) GenerateRefreshJwtTokenWithAMR(user *models.UserModel, tenant string, amr []string) (string, error) {
	h.log.Debug().Msgf("Generating JWT token for user: %s", user.ID)
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		constants.JwtSubKey: user.ID,
		constants.JwtJtiKey: uuid.New().String(),
		constants.JwtTidKey: tenant,
		"iss":               config.AppConfig.JwtIssuer,
		"aud":               config.AppConfig.JwtAudience,
		"iat":               now.Unix(),
		"nbf":               now.Unix(),
		"exp":               now.Add(time.Hour * time.Duration(config.AppConfig.RefreshJwtExpire)).Unix(),
		"amr":               amr,
	})

	token, err := t.SignedString([]byte(config.AppConfig.RefreshJwtSecret))
	if err != nil {
		return "", errors.New("error generating JWT Refresh token")
	}

	return token, nil
}

func (h *AuthHelper) GenerateRefreshJwtToken(user *models.UserModel, tenant string) (string, error) {
	return h.GenerateRefreshJwtTokenWithAMR(user, tenant, []string{"pwd"})
}

func (h *AuthHelper) ParseAccessJwtToken(tokenString string) (jwt.MapClaims, error) {
	claims, err := parseJWT(tokenString, config.AppConfig.AccessJwtSecret)
	if err != nil {
		return nil, err
	}
	if h.tokenHelper != nil {
		jti, ok := claims[constants.JwtJtiKey].(string)
		if ok && jti != "" {
			blacklisted, _ := h.tokenHelper.IsTokenBlacklisted(jti)
			if blacklisted {
				return nil, errors.New("token has been revoked")
			}
		}
	}
	return claims, nil
}

func (h *AuthHelper) ParseRefreshJwtToken(tokenString string) (jwt.MapClaims, error) {
	claims, err := parseJWT(tokenString, config.AppConfig.RefreshJwtSecret)
	if err != nil {
		return nil, err
	}
	if h.tokenHelper != nil {
		jti, ok := claims[constants.JwtJtiKey].(string)
		if ok && jti != "" {
			blacklisted, _ := h.tokenHelper.IsTokenBlacklisted(jti)
			if blacklisted {
				return nil, errors.New("token has been revoked")
			}
		}
	}
	return claims, nil
}

func (h *AuthHelper) GenerateTokens(userID, userEmail, userName, tenant string) (string, string, error) {
	h.log.Debug().Msgf("Generating JWT tokens for user: %s", userID)

	user := &models.UserModel{
		ID:    userID,
		Email: userEmail,
		Name:  userName,
	}

	accessToken, err := h.GenerateAccessJwtToken(user, tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

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
