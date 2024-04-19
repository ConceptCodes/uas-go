package config

import (
	"uas/pkg/logger"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	Port    int `env:"PORT" envDefault:"8080"`
	Timeout int `env:"HTTP_TIMEOUT" envDefault:"15"`

	DbHost string `env:"DB_HOST" envDefault:"mysql_host"`
	DbPort int    `env:"DB_PORT" envDefault:"5432"`
	DbUser string `env:"DB_USER" envDefault:"mysql_user"`
	DbPass string `env:"DB_PASS" envDefault:"mysql_password"`
	DbName string `env:"DB_NAME" envDefault:"mysql_db"`

	RedisHost     string `env:"REDIS_HOST" envDefault:"redis_host"`
	RedisPort     int    `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword string `env:"REDIS_PASSWORD" envDefault:"redis_password"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	Env string `env:"ENV" envDefault:"development"`

	ResendApiKey      string `env:"RESEND_API_KEY" envDefault:"resend"`
	EmailFrom         string `env:"EMAIL_FROM" envDefault:"example@gmail.com"`
	ResendEmailDomain string `env:"RESEND_EMAIL_DOMAIN" envDefault:"resend.dev"`

	RefreshJwtSecret string `env:"REFRESH_JWT_SECRET" envDefault:"refresh_jwt_secret"`
	RefreshJwtExpire int    `env:"REFRESH_JWT_EXPIRE" envDefault:"24"`

	AccessJwtSecret string `env:"ACCESS_JWT_SECRET" envDefault:"access_jwt_secret"`
	AccessJwtExpire int    `env:"ACCESS_JWT_EXPIRE" envDefault:"15"`

	RateLimitCapacity int `env:"RATE_LIMIT_CAPACITY" envDefault:"100"`
	TimeUnitInSeconds int `env:"TIME_UNIT_IN_SECONDS" envDefault:"60"`

	CookieBlockKey string `env:COOKIE_BLOCK_KEY" envDefault:"cookie_block_key"`
	CookieHashKey  string `env:COOKIE_HASH_KEY" envDefault:"cookie_hash_key"`
}

var AppConfig = Config{}

func init() {
	log := logger.New()
	log.Debug().Msg("Loading env vars")

	err := godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Error while loading env vars")
	}

	err = env.Parse(&AppConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while parsing env vars")
	}
}
