package api

import (
	"fmt"
	"net/http"
	"time"

	"uas/config"
	"uas/internal/constants"
	"uas/pkg/logger"
	mysql "uas/pkg/storage"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

var (
	db  *gorm.DB
	err error
	log *zerolog.Logger
)

func connectToDB() {
	db, err = mysql.New()

	if err != nil {
		log.Fatal().Err(err).Msg("Error while connecting to database")
	}
}

func Run() {
	log = logger.New()

	router := mux.NewRouter()
	port := fmt.Sprintf("%d", config.AppConfig.Port)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("127.0.0.1:%s", port),
		WriteTimeout: time.Duration(config.AppConfig.Timeout) * time.Second,
		ReadTimeout:  time.Duration(config.AppConfig.Timeout) * time.Second,
	}

	log.Debug().Msgf(constants.StartMessage, port, config.AppConfig.Env)

	err = srv.ListenAndServe()

	if err != nil {
		log.
			Fatal().
			Err(err).
			Msg("Error while starting server")
	}
}
