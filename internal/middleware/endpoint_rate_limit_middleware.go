package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type RateLimitConfig struct {
	RequestsPerMinute int
	Identifier        string
}

type EndpointRateLimitMiddleware struct {
	log    *zerolog.Logger
	rdb    *redis.Client
	config map[string]RateLimitConfig
}

func NewEndpointRateLimitMiddleware(log *zerolog.Logger, rdb *redis.Client) *EndpointRateLimitMiddleware {
	return &EndpointRateLimitMiddleware{
		log:    log,
		rdb:    rdb,
		config: make(map[string]RateLimitConfig),
	}
}

func (m *EndpointRateLimitMiddleware) RegisterEndpoint(path string, requestsPerMinute int, identifier string) {
	m.config[path] = RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		Identifier:        identifier,
	}
}

func (m *EndpointRateLimitMiddleware) Start(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config, exists := m.config[r.URL.Path]
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		var identifier string

		switch config.Identifier {
		case "ip":
			ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
			identifier = ipAddress
		case "email":
			identifier = r.FormValue("email")
			if identifier == "" {
				m.log.Warn().Str("path", r.URL.Path).Msg("Email identifier requested but not provided")
				http.Error(w, "Email required", http.StatusBadRequest)
				return
			}
		case "phone":
			identifier = r.FormValue("phoneNumber")
			if identifier == "" {
				m.log.Warn().Str("path", r.URL.Path).Msg("Phone identifier requested but not provided")
				http.Error(w, "Phone number required", http.StatusBadRequest)
				return
			}
		case "user":
			claims, err := m.extractUserClaims(r)
			if err != nil {
				m.log.Warn().Err(err).Str("path", r.URL.Path).Msg("User identifier requested but JWT not found")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			userId, ok := claims["userId"].(string)
			if !ok {
				m.log.Warn().Str("path", r.URL.Path).Msg("User identifier requested but userId claim not found")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			identifier = userId
		default:
			ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
			identifier = ipAddress
		}

		key := fmt.Sprintf("rate_limit:%s:%s", r.URL.Path, identifier)
		limiter := redis_rate.NewLimiter(m.rdb)
		limit := redis_rate.Limit{
			Rate:   config.RequestsPerMinute,
			Burst:  1,
			Period: time.Minute,
		}
		res, err := limiter.Allow(r.Context(), key, limit)

		if err != nil {
			m.log.Error().Err(err).Str("path", r.URL.Path).Msg("Error checking endpoint rate limit")
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		h := w.Header()
		h.Set("RateLimit-Remaining", strconv.Itoa(res.Remaining))

		if res.Allowed == 0 {
			seconds := int(res.RetryAfter / time.Second)
			h.Set("RateLimit-RetryAfter", strconv.Itoa(seconds))
			m.log.Warn().Str("path", r.URL.Path).Str("identifier", identifier).Int("retry_after", seconds).Msg("Endpoint rate limit exceeded")
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		m.log.Debug().Str("path", r.URL.Path).Int("remaining", res.Remaining).Msg("Endpoint rate limit check passed")
		next.ServeHTTP(w, r)
	})
}

func (m *EndpointRateLimitMiddleware) extractUserClaims(r *http.Request) (map[string]interface{}, error) {
	_, err := r.Cookie("access-token")
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (m *EndpointRateLimitMiddleware) RateLimitByIP(limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
			key := fmt.Sprintf("rate_limit_by_ip:%s", ipAddress)

			limiter := redis_rate.NewLimiter(m.rdb)
			limitCfg := redis_rate.Limit{
				Rate:   limit,
				Burst:  1,
				Period: time.Minute,
			}
			res, err := limiter.Allow(r.Context(), key, limitCfg)

			if err != nil {
				m.log.Error().Err(err).Msg("Error checking IP rate limit")
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
				return
			}

			if res.Allowed == 0 {
				seconds := int(res.RetryAfter / time.Second)
				w.Header().Set("RateLimit-RetryAfter", strconv.Itoa(seconds))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
