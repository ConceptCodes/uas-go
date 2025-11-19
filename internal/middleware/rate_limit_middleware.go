package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
	"uas/internal/constants"
	"uas/internal/helpers"

	"github.com/rs/zerolog"
)

type RateLimitMiddleware struct {
	log           *zerolog.Logger
	metricsHelper *helpers.MetricsHelper
	clients       map[string]*ClientInfo
	mu            sync.RWMutex
	windowSize    time.Duration
	maxRequests   int
}

type ClientInfo struct {
	requests []time.Time
	mu       sync.Mutex
}

func NewRateLimitMiddleware(log *zerolog.Logger, metricsHelper *helpers.MetricsHelper) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		log:           log,
		metricsHelper: metricsHelper,
		clients:       make(map[string]*ClientInfo),
		windowSize:    time.Minute, // Default 1 minute window
		maxRequests:   100,         // Default 100 requests per window
	}
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

		// Check rate limit
		if rlm.isRateLimited(clientIP) {
			rlm.metricsHelper.RecordError("rate_limit", "TOO_MANY_REQUESTS")

			traceID := r.Context().Value(constants.RequestIdCtxKey).(string)
			rlm.log.Warn().
				Str("trace_id", traceID).
				Str("client_ip", clientIP).
				Str("endpoint", r.URL.Path).
				Msg("Rate limit exceeded")

			w.Header().Set("X-RateLimit-Limit", rlm.formatRateLimit())
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", rlm.formatResetTime())
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"RATE_LIMITED","message":"Too many requests"}`))
			return
		}

		// Record successful request
		rlm.recordRequest(clientIP)

		// Set rate limit headers
		remaining := rlm.getRemainingRequests(clientIP)
		w.Header().Set("X-RateLimit-Limit", rlm.formatRateLimit())
		w.Header().Set("X-RateLimit-Remaining", rlm.formatInt(remaining))
		w.Header().Set("X-RateLimit-Reset", rlm.formatResetTime())

		next.ServeHTTP(w, r)
	})
}

func (rlm *RateLimitMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func (rlm *RateLimitMiddleware) isRateLimited(clientIP string) bool {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	client, exists := rlm.clients[clientIP]
	if !exists {
		return false
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()

	// Clean old requests outside window
	cutoff := now.Add(-rlm.windowSize)
	validRequests := make([]time.Time, 0)
	for _, req := range client.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	client.requests = validRequests

	// Check if over limit
	return len(client.requests) >= rlm.maxRequests
}

func (rlm *RateLimitMiddleware) recordRequest(clientIP string) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	client, exists := rlm.clients[clientIP]
	if !exists {
		client = &ClientInfo{
			requests: make([]time.Time, 0),
		}
		rlm.clients[clientIP] = client
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	client.requests = append(client.requests, time.Now())
}

func (rlm *RateLimitMiddleware) getRemainingRequests(clientIP string) int {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	client, exists := rlm.clients[clientIP]
	if !exists {
		return rlm.maxRequests
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rlm.windowSize)

	validCount := 0
	for _, req := range client.requests {
		if req.After(cutoff) {
			validCount++
		}
	}

	remaining := rlm.maxRequests - validCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (rlm *RateLimitMiddleware) formatRateLimit() string {
	return rlm.formatInt(rlm.maxRequests)
}

func (rlm *RateLimitMiddleware) formatInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func (rlm *RateLimitMiddleware) formatResetTime() string {
	return fmt.Sprintf("%d", time.Now().Add(rlm.windowSize).Unix())
}

// Cleanup old client data periodically
func (rlm *RateLimitMiddleware) StartCleanup() {
	go func() {
		ticker := time.NewTicker(rlm.windowSize)
		defer ticker.Stop()

		for range ticker.C {
			rlm.cleanup()
		}
	}()
}

func (rlm *RateLimitMiddleware) cleanup() {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-2 * rlm.windowSize) // Keep 2 windows

	for clientIP, client := range rlm.clients {
		client.mu.Lock()

		// Clean old requests
		validRequests := make([]time.Time, 0)
		for _, req := range client.requests {
			if req.After(cutoff) {
				validRequests = append(validRequests, req)
			}
		}
		client.requests = validRequests

		// Remove client if no recent requests
		if len(client.requests) == 0 {
			delete(rlm.clients, clientIP)
		}

		client.mu.Unlock()
	}
}
