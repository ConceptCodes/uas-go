package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"uas/internal/helpers"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type RateLimitMiddleware struct {
	log           *zerolog.Logger
	metricsHelper *helpers.MetricsHelper
	rdb           *redis.Client
	windowSize    time.Duration
	maxRequests   int
}

func NewRateLimitMiddleware(log *zerolog.Logger, metricsHelper *helpers.MetricsHelper) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		log:           log,
		metricsHelper: metricsHelper,
		windowSize:    time.Minute,
		maxRequests:   100,
	}
}

func (rlm *RateLimitMiddleware) WithRedis(rdb *redis.Client) *RateLimitMiddleware {
	rlm.rdb = rdb
	return rlm
}

func (rlm *RateLimitMiddleware) SetWindowSize(window time.Duration) {
	rlm.windowSize = window
}

func (rlm *RateLimitMiddleware) SetMaxRequests(max int) {
	rlm.maxRequests = max
}

func (rlm *RateLimitMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := rlm.getClientIP(r)

		if rlm.isRateLimited(r.Context(), clientIP) {
			rlm.metricsHelper.RecordError("rate_limit", "TOO_MANY_REQUESTS")

			traceID := helpers.TraceIDFromContext(r.Context())
			rlm.log.Warn().
				Str("trace_id", traceID).
				Str("client_ip", clientIP).
				Str("endpoint", r.URL.Path).
				Msg("Rate limit exceeded")

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rlm.maxRequests))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(time.Now().Add(rlm.windowSize).Unix())))
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"RATE_LIMITED","message":"Too many requests"}`))
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rlm.maxRequests))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rlm.maxRequests-1))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(time.Now().Add(rlm.windowSize).Unix())))

		next.ServeHTTP(w, r)
	})
}

func (rlm *RateLimitMiddleware) getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (rlm *RateLimitMiddleware) isRateLimited(ctx interface{}, clientIP string) bool {
	if rlm.rdb == nil {
		return false
	}

	key := fmt.Sprintf("global_rate_limit:%s", clientIP)
	limiter := redis_rate.NewLimiter(rlm.rdb)
	limit := redis_rate.Limit{
		Rate:   rlm.maxRequests,
		Burst:  rlm.maxRequests,
		Period: rlm.windowSize,
	}

	res, err := limiter.Allow(ctx.(context.Context), key, limit)
	if err != nil {
		rlm.log.Error().Err(err).Str("client_ip", clientIP).Msg("Redis rate limit check failed, allowing request")
		return false
	}

	return res.Allowed == 0
}

func (rlm *RateLimitMiddleware) StartCleanup() {
}
