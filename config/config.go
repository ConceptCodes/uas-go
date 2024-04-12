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

	JwtSecret string `env:"JWT_SECRET" envDefault:"jwt_secret"`
	JwtExpire int    `env:"JWT_EXPIRE" envDefault:"24"`
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
