package helpers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type MetricsHelper struct {
	log *zerolog.Logger

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Application metrics
	activeUsers      prometheus.Gauge
	loginAttempts    *prometheus.CounterVec
	otpRequests      *prometheus.CounterVec
	registrationRate *prometheus.CounterVec
	errorRate        *prometheus.CounterVec

	// Database metrics
	dbConnections   prometheus.Gauge
	dbQueryDuration *prometheus.HistogramVec
	dbErrors        *prometheus.CounterVec

	// Redis metrics
	redisConnections prometheus.Gauge
	redisCommands    *prometheus.CounterVec
	redisDuration    *prometheus.HistogramVec
}

func NewMetricsHelper(log *zerolog.Logger) *MetricsHelper {
	return &MetricsHelper{
		log: log,

		// HTTP metrics
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 500, 1000, 5000, 10000},
			},
			[]string{"method", "endpoint"},
		),

		// Application metrics
		activeUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_users_total",
				Help: "Number of currently active users",
			},
		),
		loginAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "login_attempts_total",
				Help: "Total number of login attempts",
			},
			[]string{"method", "result"},
		),
		otpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "otp_requests_total",
				Help: "Total number of OTP requests",
			},
			[]string{"method", "result"},
		),
		registrationRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"method", "result"},
		),
		errorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "code"},
		),

		// Database metrics
		dbConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		),
		dbQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"query_type"},
		),
		dbErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"operation", "error_type"},
		),

		// Redis metrics
		redisConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
		redisCommands: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_commands_total",
				Help: "Total number of Redis commands",
			},
			[]string{"command"},
		),
		redisDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_command_duration_seconds",
				Help:    "Redis command duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"command"},
		),
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *MetricsHelper) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration, responseSize int64) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(statusCode)).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordLoginAttempt records login attempt metrics
func (m *MetricsHelper) RecordLoginAttempt(method string, success bool, duration time.Duration) {
	result := "success"
	if !success {
		result = "failed"
	}
	m.loginAttempts.WithLabelValues(method, result).Inc()
}

// RecordOTPRequest records OTP request metrics
func (m *MetricsHelper) RecordOTPRequest(method string, success bool) {
	result := "success"
	if !success {
		result = "failed"
	}
	m.otpRequests.WithLabelValues(method, result).Inc()
}

// RecordRegistration records user registration metrics
func (m *MetricsHelper) RecordRegistration(method string, success bool) {
	result := "success"
	if !success {
		result = "failed"
	}
	m.registrationRate.WithLabelValues(method, result).Inc()
}

// RecordError records application error metrics
func (m *MetricsHelper) RecordError(errorType, code string) {
	m.errorRate.WithLabelValues(errorType, code).Inc()
}

// RecordDBQuery records database query metrics
func (m *MetricsHelper) RecordDBQuery(queryType string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

// RecordDBError records database error metrics
func (m *MetricsHelper) RecordDBError(operation, errorType string) {
	m.dbErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordRedisCommand records Redis command metrics
func (m *MetricsHelper) RecordRedisCommand(command string, duration time.Duration) {
	m.redisCommands.WithLabelValues(command).Inc()
	m.redisDuration.WithLabelValues(command).Observe(duration.Seconds())
}

// UpdateActiveUsers updates the active users gauge
func (m *MetricsHelper) UpdateActiveUsers(count int) {
	m.activeUsers.Set(float64(count))
}

// UpdateDBConnections updates the database connections gauge
func (m *MetricsHelper) UpdateDBConnections(count int) {
	m.dbConnections.Set(float64(count))
}

// UpdateRedisConnections updates the Redis connections gauge
func (m *MetricsHelper) UpdateRedisConnections(count int) {
	m.redisConnections.Set(float64(count))
}

// GetHandler returns Prometheus metrics handler
func (m *MetricsHelper) GetHandler() http.Handler {
	return promhttp.Handler()
}

// RegisterMetrics registers all metrics with the default registry
func (m *MetricsHelper) RegisterMetrics() {
	prometheus.MustRegister(m.httpRequestsTotal)
	prometheus.MustRegister(m.httpRequestDuration)
	prometheus.MustRegister(m.httpResponseSize)
	prometheus.MustRegister(m.activeUsers)
	prometheus.MustRegister(m.loginAttempts)
	prometheus.MustRegister(m.otpRequests)
	prometheus.MustRegister(m.registrationRate)
	prometheus.MustRegister(m.errorRate)
	prometheus.MustRegister(m.dbConnections)
	prometheus.MustRegister(m.dbQueryDuration)
	prometheus.MustRegister(m.dbErrors)
	prometheus.MustRegister(m.redisConnections)
	prometheus.MustRegister(m.redisCommands)
	prometheus.MustRegister(m.redisDuration)

	m.log.Info().Msg("Prometheus metrics registered")
}

// MetricsMiddleware creates middleware to track HTTP metrics
func (m *MetricsHelper) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code and size
		rw := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		m.RecordHTTPRequest(
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration,
			rw.responseSize,
		)
	})
}

// metricsResponseWriter wraps http.ResponseWriter to capture metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.responseSize += int64(n)
	return n, err
}
