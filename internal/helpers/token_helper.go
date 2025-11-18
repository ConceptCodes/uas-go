package helpers

import (
	"fmt"
	"time"
	"uas/config"

	"github.com/rs/zerolog"
)

type TokenHelper struct {
	log         *zerolog.Logger
	redisHelper *RedisHelper
}

func NewTokenHelper(log *zerolog.Logger, redisHelper *RedisHelper) *TokenHelper {
	return &TokenHelper{
		log:         log,
		redisHelper: redisHelper,
	}
}

func (t *TokenHelper) BlacklistToken(tokenID string, expiresAt time.Time) error {
	key := fmt.Sprintf("token_blacklist:%s", tokenID)
	ttl := time.Until(expiresAt)
	if ttl < 0 {
		ttl = 0
	}
	return t.redisHelper.SetData(key, "blacklisted", ttl)
}

func (t *TokenHelper) IsTokenBlacklisted(tokenID string) (bool, error) {
	key := fmt.Sprintf("token_blacklist:%s", tokenID)
	data, err := t.redisHelper.GetData(key)
	if err != nil {
		return false, nil
	}
	return data == "blacklisted", nil
}

func (t *TokenHelper) RevokeAllUserTokens(userID string) error {
	key := fmt.Sprintf("user_tokens_revoked:%s", userID)
	ttl := time.Duration(config.AppConfig.RefreshJwtExpire) * time.Hour
	return t.redisHelper.SetData(key, "revoked", ttl)
}

func (t *TokenHelper) AreUserTokensRevoked(userID string) (bool, error) {
	key := fmt.Sprintf("user_tokens_revoked:%s", userID)
	data, err := t.redisHelper.GetData(key)
	if err != nil {
		return false, nil
	}
	return data == "revoked", nil
}
