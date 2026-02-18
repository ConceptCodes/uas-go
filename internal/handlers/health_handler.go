package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/models"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type HealthHandler struct {
	log         *zerolog.Logger
	db          *gorm.DB
	redisClient *redis.Client
}

func NewHealthHandler(
	log *zerolog.Logger,
	db *gorm.DB,
	redisClient *redis.Client,
) *HealthHandler {
	return &HealthHandler{
		log:         log,
		db:          db,
		redisClient: redisClient,
	}
}

type HealthStatus struct {
	Status    string                      `json:"status"`
	Timestamp time.Time                   `json:"timestamp"`
	Version   string                      `json:"version"`
	Uptime    string                      `json:"uptime"`
	Checks    map[string]DependencyHealth `json:"checks"`
}

type DependencyHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
}

// Liveness godoc
// @Summary Liveness probe
// @Description Simple liveness check that always returns 200 OK
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthCheckResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /health/alive [get]
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	response := models.HealthCheckResponse{
		Service: "uas",
		Status:  true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Readiness godoc
// @Summary Readiness probe
// @Description Checks if the service is ready to accept traffic
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthStatus
// @Failure 503 {object} models.ErrorResponse
// @Router /health/status [get]
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Check all dependencies
	checks := make(map[string]DependencyHealth)
	overallStatus := "healthy"

	// Check MySQL
	mysqlStatus := h.checkMySQL()
	checks["mysql"] = mysqlStatus
	if mysqlStatus.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check Redis
	redisStatus := h.checkRedis()
	checks["redis"] = redisStatus
	if redisStatus.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check configuration
	configStatus := h.checkConfiguration()
	checks["configuration"] = configStatus
	if configStatus.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Calculate uptime
	uptime := time.Since(startTime).String()

	response := HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   "1.0.0", // This should be dynamic
		Uptime:    uptime,
		Checks:    checks,
	}

	// Set appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)

	// Log health check result
	h.log.Info().
		Str("overall_status", overallStatus).
		Str("mysql_status", mysqlStatus.Status).
		Str("redis_status", redisStatus.Status).
		Str("config_status", configStatus.Status).
		Msg("Health check completed")
}

func (h *HealthHandler) checkMySQL() DependencyHealth {
	start := time.Now()

	// Get database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get database connection")
		return DependencyHealth{
			Status:  "unhealthy",
			Message: "Failed to get database connection",
			Details: err.Error(),
		}
	}

	// Ping MySQL
	if err := sqlDB.Ping(); err != nil {
		h.log.Error().Err(err).Msg("Database ping failed")
		return DependencyHealth{
			Status:  "unhealthy",
			Message: "Database ping failed",
			Details: err.Error(),
		}
	}

	// Check query performance
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result int
	err = sqlDB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		h.log.Error().Err(err).Msg("Database query failed")
		return DependencyHealth{
			Status:  "degraded",
			Message: "Database query failed",
			Details: err.Error(),
		}
	}

	duration := time.Since(start)
	h.log.Info().Msg("Database health check completed")

	return DependencyHealth{
		Status:  "healthy",
		Message: "Database is responding normally",
		Details: fmt.Sprintf("Query completed in %v", duration),
	}
}

func (h *HealthHandler) checkRedis() DependencyHealth {
	start := time.Now()

	// Get Redis client
	client := h.redisClient
	if client == nil {
		h.log.Error().Msg("Redis client is nil")
		return DependencyHealth{
			Status:  "unhealthy",
			Message: "Redis client not available",
		}
	}

	// Ping Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		h.log.Error().Err(err).Msg("Redis ping failed")
		return DependencyHealth{
			Status:  "unhealthy",
			Message: "Redis ping failed",
			Details: err.Error(),
		}
	}

	// Test basic operations
	testKey := "health_check_" + time.Now().Format("20060102")

	// Test SET operation
	if err := client.Set(ctx, testKey, "ok", 10*time.Second).Err(); err != nil {
		h.log.Error().Err(err).Msg("Redis SET operation failed")
		return DependencyHealth{
			Status:  "degraded",
			Message: "Redis SET operation failed",
			Details: err.Error(),
		}
	}

	// Test GET operation
	val, err := client.Get(ctx, testKey).Result()
	if err != nil || val != "ok" {
		h.log.Error().Err(err).Msg("Redis GET operation failed")
		return DependencyHealth{
			Status:  "degraded",
			Message: "Redis GET operation failed",
			Details: err.Error(),
		}
	}

	// Cleanup
	client.Del(ctx, testKey)

	duration := time.Since(start)
	h.log.Info().Msg("Redis health check completed")

	return DependencyHealth{
		Status:  "healthy",
		Message: "Redis is responding normally",
		Details: fmt.Sprintf("Operations completed in %v", duration),
	}
}

func (h *HealthHandler) checkConfiguration() DependencyHealth {
	// Check critical configuration
	issues := []string{}

	// Check JWT secrets
	if len(config.AppConfig.RefreshJwtSecret) < 32 {
		issues = append(issues, "REFRESH_JWT_SECRET too short")
	}
	if len(config.AppConfig.AccessJwtSecret) < 32 {
		issues = append(issues, "ACCESS_JWT_SECRET too short")
	}

	// Check database configuration
	if config.AppConfig.DbHost == "" {
		issues = append(issues, "DB_HOST not configured")
	}
	if config.AppConfig.RedisHost == "" {
		issues = append(issues, "REDIS_HOST not configured")
	}

	// Check security settings
	if config.AppConfig.Env == "production" {
		if !config.AppConfig.CookieSecure {
			issues = append(issues, "COOKIE_SECURE must be true in production")
		}
		if config.AppConfig.MaxFailedAttempts < 1 {
			issues = append(issues, "MAX_FAILED_ATTEMPTS must be at least 1")
		}
	}

	if len(issues) > 0 {
		return DependencyHealth{
			Status:  "unhealthy",
			Message: "Configuration issues detected",
			Details: "Issues: " + strings.Join(issues, ", "),
		}
	}

	return DependencyHealth{
		Status:  "healthy",
		Message: "Configuration is valid",
	}
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc(constants.HealthCheckEndpoint, h.Liveness).Methods("GET")
	router.HandleFunc(constants.ReadinessEndpoint, h.Readiness).Methods("GET")
}
