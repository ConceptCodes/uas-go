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
	"uas/internal/services"
	"uas/pkg/email"
	"uas/pkg/logger"
	"uas/pkg/storage/mysql"
	"uas/pkg/storage/redis"
	"uas/pkg/twilio"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
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
	if err := db.AutoMigrate(
		&models.DepartmentModel{},
		&models.UserModel{},
		&models.DepartmentRoles{},
		&models.AuthModel{},
		&models.DepartmentConfig{},
		&models.PasswordHistory{},
		&models.SecurityEvent{},
		&models.Session{},
		&models.AuditLog{},
	); err != nil {
		log.Fatal().Err(err).Msg("Error while auto-migrating database schema")
	}

	emailClient := email.New()
	redisClient := redis.New(*log, ctx)
	twilioClient := twilio.New()

	departmentRepo := repository.NewGormDepartmentRepository(db)
	userRepo := repository.NewGormUserRepository(db)
	passwordResetRepo := repository.NewGormAuthRepository(db)
	departmentRoleRepo := repository.NewGormDepartmentRoleRepository(db)
	passwordHistoryRepo := repository.NewGormPasswordHistoryRepository(db)
	securityAuditRepo := repository.NewGormSecurityAuditRepository(db)
	auditLogRepo := repository.NewGormAuditLogRepository(db)
	sessionRepo := repository.NewGormSessionRepository(db)

	redisHelper := helpers.NewRedisHelper(redisClient, log, ctx)
	authHelper := helpers.NewAuthHelper(log, departmentRepo, *redisHelper)
	responseHelper := helpers.NewResponseHelper(log)
	validatorHelper := helpers.NewValidatorHelper(log, responseHelper)
	emailHelper := helpers.NewEmailHelper(log, emailClient)
	twilioHelper := helpers.NewTwilioHelper(log, twilioClient)
	loginAttemptHelper := helpers.NewLoginAttemptHelper(redisHelper, log)
	passwordHelper := helpers.NewPasswordHelper(log)
	securityLoggerHelper := helpers.NewSecurityLoggerHelper(log, securityAuditRepo)
	auditHelper := helpers.NewAuditHelper(log, auditLogRepo)
	tokenHelper := helpers.NewTokenHelper(log, redisHelper)
	encryptionHelper, err := helpers.NewEncryptionHelper(log)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while creating encryption helper")
	}
	metricsHelper := helpers.NewMetricsHelper(log)
	retentionService := services.NewRetentionService(auditLogRepo, log)
	_ = tokenHelper
	_ = securityLoggerHelper

	DepartmentHandler := handlers.NewDepartmentHandler(departmentRepo, log, authHelper, responseHelper, validatorHelper)
	healthHandler := handlers.NewHealthHandler(log, db, redisClient)
	auditHandler := handlers.NewAuditHandler(auditLogRepo, log, responseHelper, validatorHelper)
	userHandler := handlers.NewUserHandler(
		userRepo,
		passwordResetRepo,
		departmentRoleRepo,
		departmentRepo,
		sessionRepo,
		passwordHistoryRepo,
		log,
		authHelper,
		responseHelper,
		validatorHelper,
		emailHelper,
		twilioHelper,
		loginAttemptHelper,
		passwordHelper,
		encryptionHelper,
	)

	router := mux.NewRouter()

	traceMiddleware := middleware.NewTraceRequestMiddleware(log, authHelper)
	router.Use(traceMiddleware.Start)

	auditMiddleware := middleware.NewAuditMiddleware(log, auditHelper)
	router.Use(auditMiddleware.Log)

	requestLogger := middleware.NewLoggerMiddleware(log)
	router.Use(requestLogger.Start)

	rateLimitMiddleware := middleware.NewRateLimitMiddleware(log, metricsHelper)
	rateLimitMiddleware.StartCleanup()
	router.Use(rateLimitMiddleware.Handle)

	startAuditRetentionCleanup(retentionService, log)

	securityHeadersMiddleware := middleware.NewSecurityHeadersMiddleware(log)
	router.Use(securityHeadersMiddleware.Start)

	requestSizeMiddleware := middleware.NewRequestSizeMiddleware(log)
	router.Use(requestSizeMiddleware.Start)

	router.Use(middleware.ContentTypeJSON)

	rbacMiddleware := middleware.NewRBACMiddleware(log, departmentRoleRepo, authHelper)
	endpointRateLimitMiddleware := middleware.NewEndpointRateLimitMiddleware(log, redisClient)

	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsRegisterEndpoint, 5, "ip")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsLoginEndpoint, 5, "ip")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsForgotEndpoint, 3, "email")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsVerifyEndpoint, 5, "email")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsRegisterEndpointV2, 5, "ip")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsLoginEndpointV2, 5, "ip")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsForgotEndpointV2, 3, "email")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.CredentialsVerifyEndpointV2, 5, "email")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.OtpSendEndpoint, 3, "phone")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.OtpVerifyEndpoint, 5, "phone")
	endpointRateLimitMiddleware.RegisterEndpoint(constants.MagicLinkSendEndpoint, 3, "email")

	var AdminAccess = []models.Role{models.Admin}
	// var UserAccess = []models.Role{models.User}

	router.HandleFunc(constants.OnboardTenantEndpoint, DepartmentHandler.OnboardDepartmentHandler).Methods(http.MethodPost)
	healthHandler.RegisterRoutes(router)

	delTenant := router.Methods(http.MethodDelete).Subrouter()
	delTenant.HandleFunc(constants.DeleteTenantEndpoint, DepartmentHandler.DeleteDepartmentHandler)
	delTenant.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccess, next)
	})

	registerSub := router.Methods(http.MethodPost).Subrouter()
	registerSub.HandleFunc(constants.CredentialsRegisterEndpoint, userHandler.CredentialsRegisterUserHandler)
	registerSub.HandleFunc(constants.CredentialsRegisterEndpointV2, userHandler.CredentialsRegisterUserHandler)
	registerSub.Use(endpointRateLimitMiddleware.Start)

	loginSub := router.Methods(http.MethodPost).Subrouter()
	loginSub.HandleFunc(constants.CredentialsLoginEndpoint, userHandler.CredentialsLoginUserHandler)
	loginSub.HandleFunc(constants.CredentialsLoginEndpointV2, userHandler.CredentialsLoginUserHandler)
	loginSub.Use(endpointRateLimitMiddleware.Start)

	forgotSub := router.Methods(http.MethodPost).Subrouter()
	forgotSub.HandleFunc(constants.CredentialsForgotEndpoint, userHandler.CredentialsForgotPasswordHandler)
	forgotSub.HandleFunc(constants.CredentialsForgotEndpointV2, userHandler.CredentialsForgotPasswordHandler)
	forgotSub.Use(endpointRateLimitMiddleware.Start)

	verifySub := router.Methods(http.MethodPost).Subrouter()
	verifySub.HandleFunc(constants.CredentialsVerifyEndpoint, userHandler.CredentialsVerifyEmailHandler)
	verifySub.HandleFunc(constants.CredentialsVerifyEndpointV2, userHandler.CredentialsVerifyEmailHandler)
	verifySub.Use(endpointRateLimitMiddleware.Start)

	router.HandleFunc(constants.CredentialsResetEndpoint, userHandler.CredentialsResetPasswordHandler).Methods(http.MethodPost)
	router.HandleFunc(constants.CredentialsResetEndpointV2, userHandler.CredentialsResetPasswordHandler).Methods(http.MethodPost)

	// Magic link endpoints
	magicLinkSendSub := router.Methods(http.MethodPost).Subrouter()
	magicLinkSendSub.HandleFunc(constants.MagicLinkSendEndpoint, userHandler.SendMagicLinkEmail)
	magicLinkSendSub.Use(endpointRateLimitMiddleware.Start)

	router.HandleFunc(constants.MagicLinkVerifyEndpoint, userHandler.VerifyMagicLinkEmail).Methods(http.MethodGet)

	otpSendSub := router.Methods(http.MethodPost).Subrouter()
	otpSendSub.HandleFunc(constants.OtpSendEndpoint, userHandler.SendOtpCode)
	otpSendSub.Use(endpointRateLimitMiddleware.Start)

	otpVerifySub := router.Methods(http.MethodPost).Subrouter()
	otpVerifySub.HandleFunc(constants.OtpVerifyEndpoint, userHandler.VerifyOtpCode)
	otpVerifySub.Use(endpointRateLimitMiddleware.Start)

	refreshToken := router.Methods(http.MethodPost).Subrouter()
	refreshToken.HandleFunc(constants.RefreshTokenEndpoint, userHandler.RefreshAccessTokenHandler)

	auditSub := router.PathPrefix(constants.ApiPrefix + "/audit").Subrouter()
	auditSub.HandleFunc("/logs", auditHandler.GetAuditLogs).Methods(http.MethodGet)
	auditSub.HandleFunc("/logs/{id}", auditHandler.GetAuditLogByID).Methods(http.MethodGet)
	auditSub.HandleFunc("/stats", auditHandler.GetAuditStats).Methods(http.MethodGet)
	auditSub.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccess, next)
	})

	port := fmt.Sprintf("%d", config.AppConfig.Port)
	srv := &http.Server{
		Handler:           router,
		Addr:              fmt.Sprintf("%s:%s", config.AppConfig.Host, port),
		WriteTimeout:      time.Duration(config.AppConfig.Timeout) * time.Second,
		ReadTimeout:       time.Duration(config.AppConfig.Timeout) * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
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

func startAuditRetentionCleanup(retentionService *services.RetentionService, log *zerolog.Logger) {
	intervalHours := config.AppConfig.AuditRetentionCleanupHours
	if intervalHours <= 0 {
		log.Info().Int("hours", intervalHours).Msg("Audit retention cleanup disabled")
		return
	}

	runCleanup := func() {
		if err := retentionService.CleanupOldAuditLogs(); err != nil {
			log.Error().Err(err).Msg("Audit retention cleanup failed")
			return
		}
		log.Info().Msg("Audit retention cleanup completed")
	}

	go func() {
		runCleanup()

		ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			runCleanup()
		}
	}()
}
