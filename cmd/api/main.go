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
	"uas/internal/models"
	repository "uas/internal/repositories"
	"uas/pkg/email"
	"uas/pkg/logger"
	"uas/pkg/storage/mysql"
	"uas/pkg/storage/redis"
	"uas/pkg/twilio"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var (
	db  *gorm.DB
	err error
)

func Run() {
	ctx := context.Background()
	log := logger.NewWithCtx(ctx)

	db, err = mysql.New(*log)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while connecting to database")
	}

	emailClient := email.New()
	redisClient := redis.New(*log, ctx)
	twilioClient := twilio.New()

	departmentRepo := repository.NewGormDepartmentRepository(db)
	userRepo := repository.NewGormUserRepository(db)
	passwordResetRepo := repository.NewGormAuthRepository(db)
	departmentRoleRepo := repository.NewGormDepartmentRoleRepository(db)

	redisHelper := helpers.NewRedisHelper(redisClient, log, ctx)
	authHelper := helpers.NewAuthHelper(log, departmentRepo, *redisHelper)
	responseHelper := helpers.NewResponseHelper(log)
	validatorHelper := helpers.NewValidatorHelper(log, responseHelper)
	emailHelper := helpers.NewEmailHelper(log, emailClient)
	twilioHelper := helpers.NewTwilioHelper(log, twilioClient)

	DepartmentHandler := handlers.NewDepartmentHandler(departmentRepo, log, authHelper, responseHelper, validatorHelper)
	userHandler := handlers.NewUserHandler(
		userRepo,
		passwordResetRepo,
		departmentRoleRepo,
		log,
		authHelper,
		responseHelper,
		validatorHelper,
		emailHelper,
		twilioHelper,
	)

	router := mux.NewRouter()

	traceMiddleware := middleware.NewTraceRequestMiddleware(log, authHelper)
	router.Use(traceMiddleware.Start)

	requestLogger := middleware.NewLoggerMiddleware(log)
	router.Use(requestLogger.Start)

	rateLimitMiddleware := middleware.NewRateLimitRequestMiddleware(log, redisClient)
	router.Use(rateLimitMiddleware.Start)
	router.Use(middleware.ContentTypeJSON)

	rbacMiddleware := middleware.NewRBACMiddleware(log, departmentRoleRepo)

	var AdminAccess = []models.Role{models.Admin}
	var GeneralAccess = []models.Role{models.Admin, models.User}
	// var UserAccess = []models.Role{models.User}

	router.HandleFunc(constants.OnboardTenantEndpoint, DepartmentHandler.OnboardDepartmentHandler).Methods(http.MethodPost)

	delTenant := router.Methods(http.MethodDelete).Subrouter()
	delTenant.HandleFunc(constants.DeleteTenantEndpoint, DepartmentHandler.DeleteDepartmentHandler)
	delTenant.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccess, next)
	})

	router.HandleFunc(constants.CredentialsRegisterEndpoint, userHandler.CredentialsRegisterUserHandler).Methods(http.MethodPost)
	router.HandleFunc(constants.CredentialsLoginEndpoint, userHandler.CredentialsLoginUserHandler).Methods(http.MethodPost)
	router.HandleFunc(constants.CredentialsForgotEndpoint, userHandler.CredentialsForgotPasswordHandler).Methods(http.MethodPost)
	router.HandleFunc(constants.CredentialsResetEndpoint, userHandler.CredentialsResetPasswordHandler).Methods(http.MethodPost)

	router.HandleFunc(constants.OtpSendEndpoint, userHandler.SendOtpCode).Methods(http.MethodPost)
	router.HandleFunc(constants.OtpVerifyEndpoint, userHandler.VerifyOtpCode).Methods(http.MethodPost)

	refreshToken := router.Methods(http.MethodPost).Subrouter()
	refreshToken.HandleFunc(constants.RefreshTokenEndpoint, userHandler.RefreshAccessTokenHandler)
	refreshToken.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(GeneralAccess, next)
	})

	port := fmt.Sprintf("%d", config.AppConfig.Port)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("%s:%s", config.AppConfig.Host, port),
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
