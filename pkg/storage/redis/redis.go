package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"uas/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

var (
	once   sync.Once
	client *redis.Client
)

func New(l zerolog.Logger, ctx context.Context) *redis.Client {
	once.Do(func() {
		l.Debug().Msg("Connecting to redis")

		options := &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", config.AppConfig.RedisHost, config.AppConfig.RedisPort),
			Password: config.AppConfig.RedisPassword,
			DB:       config.AppConfig.RedisDB,
		}

		// Add TLS configuration if enabled
		if config.AppConfig.EnableRedisTLS {
			options.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		client = redis.NewClient(options)

		ctx := context.Background()

		_, err := client.Ping(ctx).Result()
		if err != nil {
			l.Error().Err(err).Msg("Error while connecting to redis")
		}
	})

	return client
}
