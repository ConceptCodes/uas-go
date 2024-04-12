package mysql

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"uas/config"
	"uas/internal/constants"
)

var (
	db   *gorm.DB
	once sync.Once
)

func New(l zerolog.Logger) (*gorm.DB, error) {
	var err error
	l.Debug().Msg("Connecting to MySQL")

	once.Do(func() {
		db, err = gorm.Open(mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				config.AppConfig.DbUser,
				config.AppConfig.DbPass,
				config.AppConfig.DbHost,
				config.AppConfig.DbPort,
				config.AppConfig.DbName,
			),
		}), &gorm.Config{
			Logger: logger.New(
				&l,
				logger.Config{
					LogLevel:             logger.Info,
					Colorful:             config.AppConfig.Env == constants.LocalEnv,
					ParameterizedQueries: true,
				},
			),
		})

	})
	return db, err
}

func Close(l zerolog.Logger) {
	sqlDB, _ := db.DB()

	err := sqlDB.Close()

	if err != nil {
		l.Error().Err(err).Msg("Error closing db")
		return
	}
}

func HealthCheck(l zerolog.Logger) bool {
	l.Debug().Msgf(constants.HealthCheckMessage, "mysql")

	sqlDB, _ := db.DB()

	err := sqlDB.Ping()

	if err != nil {
		l.
			Error().
			Err(err).
			Msgf(constants.HealthCheckError, "mysql")
		return false
	}

	l.Info().Msg("MySQL is up")
	return true
}
