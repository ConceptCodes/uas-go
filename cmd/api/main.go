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
	log.Warn().Msg("AutoMigrate disabled — schema must be applied via SQL migrations")

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
	mfaFactorRepo := repository.NewGormMfaFactorRepository(db)
	mfaChallengeRepo := repository.NewGormMfaChallengeRepository(db)
	idPRepo := repository.NewGormIdentityProviderRepository(db)
	identityRepo := repository.NewGormUserIdentityRepository(db)
	webhookEndpointRepo := repository.NewGormWebhookEndpointRepository(db)
	webhookDeliveryRepo := repository.NewGormWebhookDeliveryRepository(db)

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
	mfaHelper := helpers.NewMfaHelper(log, redisHelper)
	ssoHelper := helpers.NewSSOHelper(log)
	webhookHelper := helpers.NewWebhookHelper(log, webhookEndpointRepo, webhookDeliveryRepo)

	authHelper.WithTokenHelper(tokenHelper)

	DepartmentHandler := handlers.NewDepartmentHandler(departmentRepo, sessionRepo, passwordHistoryRepo, passwordResetRepo, userRepo, departmentRoleRepo, log, authHelper, responseHelper, validatorHelper)
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
		tokenHelper,
		securityLoggerHelper,
		mfaFactorRepo,
		mfaHelper,
		webhookHelper,
	)
	mfaHandler := handlers.NewMfaHandler(
		mfaFactorRepo,
		mfaChallengeRepo,
		userRepo,
		authHelper,
		mfaHelper,
		responseHelper,
		validatorHelper,
		log,
	)
	ssoHandler := handlers.NewSsoHandler(
		idPRepo,
		identityRepo,
		userRepo,
		departmentRoleRepo,
		sessionRepo,
		authHelper,
		ssoHelper,
		mfaHelper,
		responseHelper,
		validatorHelper,
		log,
	)
	webhookHandler := handlers.NewWebhookHandler(
		webhookEndpointRepo,
		webhookDeliveryRepo,
		webhookHelper,
		responseHelper,
		validatorHelper,
		log,
	)
	samlHandler := handlers.NewSamlHandler(
		idPRepo,
		identityRepo,
		userRepo,
		departmentRoleRepo,
		sessionRepo,
		authHelper,
		mfaHelper,
		responseHelper,
		log,
	)

	adminHandler := handlers.NewAdminHandler(
		userRepo,
		departmentRepo,
		sessionRepo,
		passwordHistoryRepo,
		passwordHelper,
		authHelper,
		responseHelper,
		validatorHelper,
		log,
	)

	router := mux.NewRouter()

	errorMiddleware := middleware.NewErrorMiddleware(log)
	router.Use(errorMiddleware.Handle)

	traceMiddleware := middleware.NewTraceRequestMiddleware(log, authHelper)
	router.Use(traceMiddleware.Start)

	tenantResolverMiddleware := middleware.NewTenantResolverMiddleware(log)
	router.Use(tenantResolverMiddleware.Resolve)

	csrfMiddleware := middleware.NewCSRFMiddleware(log)
	router.Use(csrfMiddleware.Protect)

	auditMiddleware := middleware.NewAuditMiddleware(log, auditHelper)
	router.Use(auditMiddleware.Log)

	requestLogger := middleware.NewLoggerMiddleware(log)
	router.Use(requestLogger.Start)

	rateLimitMiddleware := middleware.NewRateLimitMiddleware(log, metricsHelper).WithRedis(redisClient)
	router.Use(rateLimitMiddleware.Handle)

	if config.AppConfig.EnableMetrics {
		metricsHelper.RegisterMetrics()
		responseTimeMiddleware := middleware.NewResponseTimeMiddleware(log, metricsHelper)
		router.Use(responseTimeMiddleware.Handle)
		router.Handle("/metrics", metricsHelper.GetHandler()).Methods(http.MethodGet)
		log.Info().Msg("Metrics middleware and /metrics endpoint enabled")
	}

	startAuditRetentionCleanup(retentionService, log)

	securityHeadersMiddleware := middleware.NewSecurityHeadersMiddleware(log)
	router.Use(securityHeadersMiddleware.Start)

	requestSizeMiddleware := middleware.NewRequestSizeMiddleware(log)
	router.Use(requestSizeMiddleware.Start)

	router.Use(middleware.ContentTypeJSON)

	rbacMiddleware := middleware.NewRBACMiddleware(log, departmentRoleRepo, authHelper)
	endpointRateLimitMiddleware := middleware.NewEndpointRateLimitMiddleware(log, redisClient).WithAuthHelper(authHelper)

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

	router.HandleFunc(constants.OnboardTenantEndpoint, DepartmentHandler.OnboardDepartmentHandler).Methods(http.MethodPost)
	healthHandler.RegisterRoutes(router)

	// Tenant-required routes — all use the tenant auth + endpoint rate limiting
	tenantSub := router.PathPrefix(constants.ApiPrefix).Subrouter()
	tenantSub.Use(endpointRateLimitMiddleware.Start)

	delTenant := tenantSub.Methods(http.MethodDelete).Subrouter()
	delTenant.HandleFunc(constants.DeleteTenantEndpoint, DepartmentHandler.DeleteDepartmentHandler)
	delTenant.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccess, next)
	})

	tenantSub.HandleFunc(constants.CredentialsRegisterEndpoint, userHandler.CredentialsRegisterUserHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsRegisterEndpointV2, userHandler.CredentialsRegisterUserHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsLoginEndpoint, userHandler.CredentialsLoginUserHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsLoginEndpointV2, userHandler.CredentialsLoginUserHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsForgotEndpoint, userHandler.CredentialsForgotPasswordHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsForgotEndpointV2, userHandler.CredentialsForgotPasswordHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsVerifyEndpoint, userHandler.CredentialsVerifyEmailHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsVerifyEndpointV2, userHandler.CredentialsVerifyEmailHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsResetEndpoint, userHandler.CredentialsResetPasswordHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.CredentialsResetEndpointV2, userHandler.CredentialsResetPasswordHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MagicLinkSendEndpoint, userHandler.SendMagicLinkEmail).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MagicLinkVerifyEndpoint, userHandler.VerifyMagicLinkEmail).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.OtpSendEndpoint, userHandler.SendOtpCode).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.OtpVerifyEndpoint, userHandler.VerifyOtpCode).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.RefreshTokenEndpoint, userHandler.RefreshAccessTokenHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.LogoutEndpoint, userHandler.LogoutHandler).Methods(http.MethodPost)

	auditSub := tenantSub.PathPrefix("/audit").Subrouter()
	auditSub.HandleFunc("/logs", auditHandler.GetAuditLogs).Methods(http.MethodGet)
	auditSub.HandleFunc("/logs/{id}", auditHandler.GetAuditLogByID).Methods(http.MethodGet)
	auditSub.HandleFunc("/stats", auditHandler.GetAuditStats).Methods(http.MethodGet)
	auditSub.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccess, next)
	})

	var AdminAccessRoles = []models.Role{models.Admin, models.TenantAdmin}

	tenantSub.HandleFunc(constants.MfaEnrollEndpoint, mfaHandler.EnrollHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MfaVerifyEnrollEndpoint, mfaHandler.VerifyEnrollHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MfaFactorsEndpoint, mfaHandler.ListFactorsHandler).Methods(http.MethodGet)
	tenantSub.HandleFunc(constants.MfaDisableEndpoint, mfaHandler.DisableHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MfaChallengeVerifyEndpoint, mfaHandler.ChallengeVerifyHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MfaRecoverEndpoint, mfaHandler.RecoverHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.MfaStatusEndpoint, mfaHandler.StatusHandler).Methods(http.MethodGet)

	// SSO routes — public login/callback, admin for provider CRUD
	tenantSub.HandleFunc(constants.SsoLoginEndpoint, ssoHandler.LoginHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.SsoCallbackEndpoint, ssoHandler.CallbackHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.SsoIdentitiesEndpoint, ssoHandler.ListIdentitiesHandler).Methods(http.MethodGet)
	tenantSub.HandleFunc(constants.SsoIdentityEndpoint, ssoHandler.DeleteIdentityHandler).Methods(http.MethodDelete)

	ssoAdminSub := tenantSub.PathPrefix("/sso/providers").Subrouter()
	ssoAdminSub.HandleFunc("", ssoHandler.ListProvidersHandler).Methods(http.MethodGet)
	ssoAdminSub.HandleFunc("", ssoHandler.CreateProviderHandler).Methods(http.MethodPost)
	ssoAdminSub.HandleFunc("/{id}", ssoHandler.GetProviderHandler).Methods(http.MethodGet)
	ssoAdminSub.HandleFunc("/{id}", ssoHandler.UpdateProviderHandler).Methods(http.MethodPut)
	ssoAdminSub.HandleFunc("/{id}", ssoHandler.DeleteProviderHandler).Methods(http.MethodDelete)
	ssoAdminSub.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccessRoles, next)
	})

	// SAML routes
	tenantSub.HandleFunc(constants.SamlLoginEndpoint, samlHandler.LoginHandler).Methods(http.MethodGet, http.MethodPost)
	tenantSub.HandleFunc(constants.SamlACSEndpoint, samlHandler.AssertionConsumerServiceHandler).Methods(http.MethodPost)
	tenantSub.HandleFunc(constants.SamlMetadataEndpoint, samlHandler.GetMetadataHandler).Methods(http.MethodGet)

	// Webhook routes — admin for endpoint CRUD
	webhookEndpointSub := tenantSub.PathPrefix("/webhooks/endpoints").Subrouter()
	webhookEndpointSub.HandleFunc("", webhookHandler.CreateEndpointHandler).Methods(http.MethodPost)
	webhookEndpointSub.HandleFunc("", webhookHandler.ListEndpointsHandler).Methods(http.MethodGet)
	webhookEndpointSub.HandleFunc("/{id}", webhookHandler.GetEndpointHandler).Methods(http.MethodGet)
	webhookEndpointSub.HandleFunc("/{id}", webhookHandler.UpdateEndpointHandler).Methods(http.MethodPut)
	webhookEndpointSub.HandleFunc("/{id}", webhookHandler.DeleteEndpointHandler).Methods(http.MethodDelete)
	webhookEndpointSub.HandleFunc("/{id}/secret", webhookHandler.RotateSecretHandler).Methods(http.MethodPost)
	webhookEndpointSub.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccessRoles, next)
	})

	tenantSub.HandleFunc(constants.WebhookDeliveriesEndpoint, webhookHandler.ListDeliveriesHandler).Methods(http.MethodGet)
	tenantSub.HandleFunc(constants.WebhookDeliveryEndpoint, webhookHandler.GetDeliveryHandler).Methods(http.MethodGet)
	tenantSub.HandleFunc(constants.WebhookRetryEndpoint, webhookHandler.RetryDeliveryHandler).Methods(http.MethodPost)

	// Admin API — requires platform_admin or tenant_admin
	adminSub := tenantSub.PathPrefix("/admin").Subrouter()
	adminSub.HandleFunc("/users", adminHandler.ListUsersHandler).Methods(http.MethodGet)
	adminSub.HandleFunc("/users/{id}", adminHandler.GetUserHandler).Methods(http.MethodGet)
	adminSub.HandleFunc("/users/{id}", adminHandler.UpdateUserHandler).Methods(http.MethodPut)
	adminSub.HandleFunc("/users/{id}/reset-password", adminHandler.ResetUserPasswordHandler).Methods(http.MethodPost)
	adminSub.HandleFunc("/users/{id}/sessions", adminHandler.RevokeUserSessionsHandler).Methods(http.MethodDelete)
	adminSub.Use(func(next http.Handler) http.Handler {
		return rbacMiddleware.Authorize(AdminAccessRoles, next)
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
