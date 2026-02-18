package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	EncryptionKey      string `env:"ENCRYPTION_KEY" envDefault:"CHANGE_ME_32_BYTE_ENCRYPTION_KEY"`
	EnableDBEncryption bool   `env:"ENABLE_DB_ENCRYPTION" envDefault:"false"`
	EnableRedisTLS     bool   `env:"ENABLE_REDIS_TLS" envDefault:"false"`
	EnableMySQLTLS     bool   `env:"ENABLE_MYSQL_TLS" envDefault:"false"`

	// Performance monitoring
	EnableMetrics   bool `env:"ENABLE_METRICS" envDefault:"true"`
	EnableProfiling bool `env:"ENABLE_PROFILING" envDefault:"false"`
	Debug           bool `env:"DEBUG" envDefault:"false"`

	// Additional configuration for profiling
	JwtExpiry          int `env:"JWT_EXPIRY" envDefault:"15"`
	RefreshTokenExpiry int `env:"REFRESH_TOKEN_EXPIRY" envDefault:"24"`
	LockoutDuration    int `env:"LOCKOUT_DURATION" envDefault:"15"`
	SessionTimeout     int `env:"SESSION_TIMEOUT" envDefault:"60"`
	MaxRequestSize     int `env:"MAX_REQUEST_SIZE" envDefault:"10485760"`
	RateLimitRequests  int `env:"RATE_LIMIT_REQUESTS" envDefault:"100"`
	RateLimitWindow    int `env:"RATE_LIMIT_WINDOW" envDefault:"60"`

	// Magic link configuration
	MagicLinkBaseUrl string `env:"MAGIC_LINK_BASE_URL" envDefault:"http://localhost:8080"`

	// CORS configuration
	CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" envDefault:"http://localhost:3000,http://localhost:5173"`

	// Audit retention cleanup schedule
	AuditRetentionCleanupHours int `env:"AUDIT_RETENTION_CLEANUP_HOURS" envDefault:"24"`
}

var AppConfig = Config{}

func init() {
	log := logger.New()
	log.Debug().Msg("Loading env vars")

	// Load environment-specific config file
	environment := os.Getenv("ENV")
	if environment == "" {
		environment = "development"
	}

	// Try to load environment-specific config first
	envFile := filepath.Join("config", fmt.Sprintf(".env.%s", environment))
	if err := godotenv.Load(envFile); err != nil {
		log.Debug().Err(err).Msgf("Could not load %s, using defaults", envFile)
	}

	// Load base .env file as fallback
	if err := godotenv.Load(); err != nil {
		log.Debug().Err(err).Msg("Could not load .env file, using environment variables only")
	}

	err := env.Parse(&AppConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while parsing env vars")
	}

	log.Info().Str("environment", environment).Msg("Configuration loaded successfully")

	// Validate configuration
	if err := validateConfig(); err != nil {
		log.Fatal().Err(err).Msg("Configuration validation failed")
	}
}

func validateConfig() error {
	var errors []string

	// Validate required secrets
	if len(AppConfig.RefreshJwtSecret) < 32 {
		errors = append(errors, "REFRESH_JWT_SECRET must be at least 32 characters")
	}
	if len(AppConfig.AccessJwtSecret) < 32 {
		errors = append(errors, "ACCESS_JWT_SECRET must be at least 32 characters")
	}
	if len(AppConfig.CookieBlockKey) < 32 {
		errors = append(errors, "COOKIE_BLOCK_KEY must be at least 32 characters")
	}
	if len(AppConfig.CookieHashKey) < 32 {
		errors = append(errors, "COOKIE_HASH_KEY must be at least 32 characters")
	}

	// Validate encryption key if enabled
	if AppConfig.EnableDBEncryption {
		if len(AppConfig.EncryptionKey) < 32 {
			errors = append(errors, "ENCRYPTION_KEY must be at least 32 characters when DB encryption is enabled")
		}
	}

	// Validate email format
	if AppConfig.EmailFrom != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(AppConfig.EmailFrom) {
			errors = append(errors, "EMAIL_FROM must be a valid email address")
		}
	}

	// Validate port range
	if AppConfig.Port < 1 || AppConfig.Port > 65535 {
		errors = append(errors, "PORT must be between 1 and 65535")
	}
	if AppConfig.DbPort < 1 || AppConfig.DbPort > 65535 {
		errors = append(errors, "DB_PORT must be between 1 and 65535")
	}
	if AppConfig.RedisPort < 1 || AppConfig.RedisPort > 65535 {
		errors = append(errors, "REDIS_PORT must be between 1 and 65535")
	}

	// Validate JWT expiration times
	if AppConfig.RefreshJwtExpire < 1 || AppConfig.RefreshJwtExpire > 168 { // max 1 week
		errors = append(errors, "REFRESH_JWT_EXPIRE must be between 1 and 168 hours")
	}
	if AppConfig.AccessJwtExpire < 1 || AppConfig.AccessJwtExpire > 24 { // max 24 hours
		errors = append(errors, "ACCESS_JWT_EXPIRE must be between 1 and 24 hours")
	}

	// Validate password policy
	if AppConfig.PasswordMinLength < 8 {
		errors = append(errors, "PASSWORD_MIN_LENGTH must be at least 8 characters")
	}

	// Validate rate limiting
	if AppConfig.RateLimitCapacity < 1 {
		errors = append(errors, "RATE_LIMIT_CAPACITY must be at least 1")
	}
	if AppConfig.TimeUnitInSeconds < 1 {
		errors = append(errors, "TIME_UNIT_IN_SECONDS must be at least 1")
	}

	// Validate security settings
	if AppConfig.MaxFailedAttempts < 1 {
		errors = append(errors, "MAX_FAILED_ATTEMPTS must be at least 1")
	}
	if AppConfig.AccountLockMinutes < 1 {
		errors = append(errors, "ACCOUNT_LOCK_MINUTES must be at least 1")
	}

	// Validate production-specific settings
	if AppConfig.Env == "production" {
		if strings.Contains(AppConfig.RefreshJwtSecret, "CHANGE_ME") ||
			strings.Contains(AppConfig.AccessJwtSecret, "CHANGE_ME") ||
			strings.Contains(AppConfig.CookieBlockKey, "CHANGE_ME") ||
			strings.Contains(AppConfig.CookieHashKey, "CHANGE_ME") {
			errors = append(errors, "Production environment cannot use default secret values")
		}

		if !AppConfig.CookieSecure {
			errors = append(errors, "COOKIE_SECURE must be true in production")
		}

		if AppConfig.CookieSameSite != "Strict" && AppConfig.CookieSameSite != "Lax" {
			errors = append(errors, "COOKIE_SAMESITE must be 'Strict' or 'Lax' in production")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ReloadConfiguration manually triggers configuration reload
func ReloadConfiguration() error {
	// Reload environment variables
	environment := os.Getenv("ENV")
	if environment == "" {
		environment = "development"
	}

	// Load environment-specific config
	envFile := filepath.Join("config", ".env."+environment)
	godotenv.Load(envFile)
	godotenv.Load()

	// Parse new configuration
	var newConfig Config
	if err := env.Parse(&newConfig); err != nil {
		log := logger.New()
		log.Error().Err(err).Msg("Error parsing reloaded config")
		return err
	}

	// Validate new configuration
	if err := validateConfig(); err != nil {
		log := logger.New()
		log.Error().Err(err).Msg("Configuration validation failed during reload")
		return err
	}

	// Update global config
	AppConfig = newConfig

	log := logger.New()
	log.Info().Msg("Configuration reloaded successfully")
	return nil
}
