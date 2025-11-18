package config

import (
	"uas/pkg/logger"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	Host    string `env:"HOST" envDefault:"127.0.0.1"`
	Port    int    `env:"PORT" envDefault:"8080"`
	Timeout int    `env:"HTTP_TIMEOUT" envDefault:"15"`

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

	RefreshJwtSecret string `env:"REFRESH_JWT_SECRET" envDefault:"CHANGE_ME_REFRESH_SECRET_MIN_32_CHARS"`
	RefreshJwtExpire int    `env:"REFRESH_JWT_EXPIRE" envDefault:"24"`

	AccessJwtSecret string `env:"ACCESS_JWT_SECRET" envDefault:"CHANGE_ME_ACCESS_SECRET_MIN_32_CHARS"`
	AccessJwtExpire int    `env:"ACCESS_JWT_EXPIRE" envDefault:"15"`

	RateLimitCapacity int `env:"RATE_LIMIT_CAPACITY" envDefault:"100"`
	TimeUnitInSeconds int `env:"TIME_UNIT_IN_SECONDS" envDefault:"60"`

	CookieBlockKey string `env:"COOKIE_BLOCK_KEY" envDefault:"CHANGE_ME_COOKIE_BLOCK_MIN_32_CHARS"`
	CookieHashKey  string `env:"COOKIE_HASH_KEY" envDefault:"CHANGE_ME_COOKIE_HASH_MIN_32_CHARS"`
	CookieDomain   string `env:"COOKIE_DOMAIN" envDefault:""`
	CookieSecure   bool   `env:"COOKIE_SECURE" envDefault:"true"`
	CookieSameSite string `env:"COOKIE_SAMESITE" envDefault:"Strict"`

	MaxFailedAttempts  int `env:"MAX_FAILED_ATTEMPTS" envDefault:"5"`
	AccountLockMinutes int `env:"ACCOUNT_LOCK_MINUTES" envDefault:"15"`
	LoginRateLimitMins int `env:"LOGIN_RATE_LIMIT_MINUTES" envDefault:"1"`
	OtpRateLimitMins   int `env:"OTP_RATE_LIMIT_MINUTES" envDefault:"5"`

	TwilioAccountSid  string `env:"TWILIO_ACCOUNT_SID" envDefault:"twilio_account_sid"`
	TwilioAuthToken   string `env:"TWILIO_AUTH_TOKEN" envDefault:"twilio_auth_token"`
	TwilioPhoneNumber string `env:"TWILIO_PHONE_NUMBER" envDefault:"twilio_phone_number"`

	OtpExpire int `env:"OTP_EXPIRE" envDefault:"5"`

	PasswordMinLength        int  `env:"PASSWORD_MIN_LENGTH" envDefault:"12"`
	PasswordRequireUppercase bool `env:"PASSWORD_REQUIRE_UPPERCASE" envDefault:"true"`
	PasswordRequireLowercase bool `env:"PASSWORD_REQUIRE_LOWERCASE" envDefault:"true"`
	PasswordRequireNumber    bool `env:"PASSWORD_REQUIRE_NUMBER" envDefault:"true"`
	PasswordRequireSpecial   bool `env:"PASSWORD_REQUIRE_SPECIAL" envDefault:"true"`
	PasswordHistoryCount     int  `env:"PASSWORD_HISTORY_COUNT" envDefault:"5"`

	MaxRequestSizeMB int `env:"MAX_REQUEST_SIZE_MB" envDefault:"10"`
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
