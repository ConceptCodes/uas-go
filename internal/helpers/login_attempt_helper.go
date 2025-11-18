package helpers

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uas/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type LoginAttemptHelper struct {
	redisHelper *RedisHelper
	log         *zerolog.Logger
}

func NewLoginAttemptHelper(redisHelper *RedisHelper, log *zerolog.Logger) *LoginAttemptHelper {
	return &LoginAttemptHelper{
		redisHelper: redisHelper,
		log:         log,
	}
}

func (h *LoginAttemptHelper) RecordFailedAttempt(identifier string) error {
	key := fmt.Sprintf("failed_login:%s", identifier)
	count, err := h.redisHelper.IncrData(key)
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Error recording failed login attempt")
		return err
	}

	ttl := time.Duration(config.AppConfig.AccountLockMinutes) * time.Minute
	err = h.redisHelper.ExpireData(key, ttl)
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Error setting TTL for failed login attempt")
		return err
	}

	if count >= int64(config.AppConfig.MaxFailedAttempts) {
		h.log.Warn().Str("identifier", identifier).Int64("attempts", count).Msg("Account locked due to too many failed login attempts")
		lockKey := fmt.Sprintf("account_lock:%s", identifier)
		err = h.redisHelper.SetData(lockKey, "locked", ttl)
		if err != nil {
			h.log.Error().Err(err).Str("identifier", identifier).Msg("Error locking account")
			return err
		}
	}

	return nil
}

func (h *LoginAttemptHelper) ClearFailedAttempts(identifier string) error {
	key := fmt.Sprintf("failed_login:%s", identifier)
	err := h.redisHelper.DeleteData(key)
	if err != nil {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Error clearing failed login attempts")
		return err
	}
	return nil
}

func (h *LoginAttemptHelper) IsAccountLocked(identifier string) (bool, error) {
	lockKey := fmt.Sprintf("account_lock:%s", identifier)
	data, err := h.redisHelper.GetData(lockKey)
	if err != nil && !errors.Is(err, redis.Nil) {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Error checking account lock status")
		return false, err
	}
	return data == "locked", nil
}

func (h *LoginAttemptHelper) GetFailedAttemptCount(identifier string) (int64, error) {
	key := fmt.Sprintf("failed_login:%s", identifier)
	count, err := h.redisHelper.GetDataInt(key)
	if err != nil && !errors.Is(err, redis.Nil) {
		h.log.Error().Err(err).Str("identifier", identifier).Msg("Error getting failed login count")
		return 0, err
	}
	return count, nil
}

func (h *LoginAttemptHelper) GetProgressiveDelay(attemptCount int64) time.Duration {
	delays := []time.Duration{
		0 * time.Second,
		2 * time.Second,
		5 * time.Second,
		15 * time.Second,
		60 * time.Second,
	}

	index := int(attemptCount)
	if index >= len(delays) {
		index = len(delays) - 1
	}

	return delays[index]
}

func (h *LoginAttemptHelper) ApplyProgressiveDelay(ctx context.Context, delayDuration time.Duration) {
	if delayDuration > 0 {
		select {
		case <-time.After(delayDuration):
		case <-ctx.Done():
			h.log.Info().Msg("Progressive delay cancelled due to context deadline")
		}
	}
}
