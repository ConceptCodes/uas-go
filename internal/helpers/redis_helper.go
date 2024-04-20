package helpers

import (
	"context"
	"time"
	"uas/internal/constants"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type RedisHelper struct {
	client *redis.Client
	log    *zerolog.Logger
	ctx    context.Context
}

func NewRedisHelper(client *redis.Client, log *zerolog.Logger, ctx context.Context) *RedisHelper {
	return &RedisHelper{client: client, log: log, ctx: ctx}
}

func (r *RedisHelper) GetData(key string) (string, error) {
	r.log.
		Debug().
		Str("key", key).
		Msgf("Getting key %s from redis", key)

	return r.client.Get(r.ctx, key).Result()
}

func (r *RedisHelper) SetData(key string, value string, ttl time.Duration) error {
	r.log.
		Debug().
		Str("key", key).
		Msgf("Setting key %s in redis", key)

	return r.client.Set(r.ctx, key, value, ttl|constants.DefaultRedisTtl).Err()
}
