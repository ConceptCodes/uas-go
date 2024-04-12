package redis

import (
	"context"
	"fmt"
	"sync"

	"uas/config"
	"uas/internal/constants"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

var (
	once        sync.Once
	redisClient *redis.Client
)

type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
	log    zerolog.Logger
}

func New(l zerolog.Logger) *RedisClient {
	once.Do(func() {
		l.Debug().Msg("Connecting to redis")

		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", config.AppConfig.RedisHost, config.AppConfig.RedisPort),
			Password: config.AppConfig.RedisPassword,
			DB:       config.AppConfig.RedisDB,
		})

		ctx := context.Background()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			l.Error().Err(err).Msg("Error while connecting to redis")
		}
	})

	return &RedisClient{
		Client: redisClient,
		Ctx:    context.Background(),
		log:    l,
	}
}

func (r *RedisClient) HealthCheck() bool {
	r.log.
		Debug().
		Msgf(fmt.Sprintf(constants.HealthCheckMessage, "redis"))

	res := r.Client.Ping(r.Ctx).Err()

	if res != nil {
		r.log.
			Error().
			Err(res).
			Msgf(constants.HealthCheckError, "redis")
		return false
	}

	r.log.Info().Msg("Redis is up")

	return true
}

func (r *RedisClient) Close() error {
	r.log.
		Debug().
		Msg("Closing redis connection")
	return r.Client.Close()
}
