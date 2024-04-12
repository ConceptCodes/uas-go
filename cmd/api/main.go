package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"uas/config"
	"uas/internal/constants"
	"uas/internal/handlers"
	"uas/internal/helpers"
	"uas/internal/middleware"
	repository "uas/internal/repositories"
	"uas/pkg/logger"
	mysql "uas/pkg/storage"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var (
	db  *gorm.DB
	err error
)

func Run() {
	// Initialize logger
	ctx := context.Background()
	log := logger.NewWithCtx(ctx)

	// Initialize helpers/utils
	authHelper := helpers.NewAuthHelper(*log)
	responseHelper := helpers.NewResponseHelper(*log)
	validatorHelper := helpers.NewValidatorHelper(*log, *responseHelper)

	// Connect to database
	db, err = mysql.New(*log)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while connecting to database")
	}

	// Initialize repositories
	tenantRepo := repository.NewGormTenantRepository(db)
	userRepo := repository.NewGormUserRepository(db)

	// Initialize handlers
	tenantHandler := handlers.NewTenantHandler(tenantRepo, *log, *authHelper, *responseHelper, *validatorHelper)
	userHandler := handlers.NewUserHandler(userRepo, *log, *authHelper, *responseHelper, *validatorHelper)

	// Initialize router
	router := mux.NewRouter()

	// Initialize middleware
	traceMiddleware := middleware.NewTraceRequestMiddleware(*log, *authHelper)
	router.Use(traceMiddleware.Start)

	requestLogger := middleware.NewLoggerMiddleware(*log)
	router.Use(requestLogger.Start)

	router.Use(middleware.ContentTypeJSON)

	// Add routes
	router.HandleFunc(constants.OnboardTenantEndpoint, tenantHandler.OnboardTenantHandler).Methods("POST")

	router.HandleFunc(constants.RegisterEndpoint, userHandler.RegisterUserHandler).Methods("POST")
	router.HandleFunc(constants.LoginEndpoint, userHandler.LoginUserHandler).Methods("POST")

	// Initialize server
	port := fmt.Sprintf("%d", config.AppConfig.Port)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("127.0.0.1:%s", port),
		WriteTimeout: time.Duration(config.AppConfig.Timeout) * time.Second,
		ReadTimeout:  time.Duration(config.AppConfig.Timeout) * time.Second,
	}

	// Start server
	log.Debug().Msgf(constants.StartMessage, port, config.AppConfig.Env)
	err = srv.ListenAndServe()

	if err != nil {
		log.
			Fatal().
			Err(err).
			Msg("Error while starting server")
	}
}
