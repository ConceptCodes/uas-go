package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"time"
	"uas/config"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type ProfilingHandler struct {
	log    *zerolog.Logger
	db     *gorm.DB
	client *redis.Client
}

func NewProfilingHandler(
	log *zerolog.Logger,
	db *gorm.DB,
	client *redis.Client,
) *ProfilingHandler {
	return &ProfilingHandler{
		log:    log,
		db:     db,
		client: client,
	}
}

// SystemMetrics godoc
// @Summary System performance metrics
// @Description Returns detailed system performance metrics including memory, CPU, and goroutine information
// @Tags profiling
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /debug/metrics [get]
func (h *ProfilingHandler) SystemMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get goroutine count
	numGoroutines := runtime.NumGoroutine()

	// Get GC stats
	gcStats := debug.GCStats{}
	debug.ReadGCStats(&gcStats)

	// Get CPU info
	numCPU := runtime.NumCPU()
	numCgoCall := runtime.NumCgoCall()

	response := map[string]interface{}{
		"timestamp": time.Now(),
		"memory": map[string]interface{}{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"lookups":     m.Lookups,
			"mallocs":     m.Mallocs,
			"frees":       m.Frees,
			"heap": map[string]interface{}{
				"alloc":    m.HeapAlloc,
				"sys":      m.HeapSys,
				"idle":     m.HeapIdle,
				"inuse":    m.HeapInuse,
				"released": m.HeapReleased,
				"objects":  m.HeapObjects,
			},
			"stack": map[string]interface{}{
				"inuse": m.StackInuse,
				"sys":   m.StackSys,
			},
			"gc": map[string]interface{}{
				"num_gc":          m.NumGC,
				"num_forced_gc":   m.NumForcedGC,
				"gc_cpu_fraction": m.GCCPUFraction,
				"enable_gc":       m.EnableGC,
				"debug_gc":        m.DebugGC,
			},
		},
		"runtime": map[string]interface{}{
			"goroutines": numGoroutines,
			"num_cpu":    numCPU,
			"cgo_calls":  numCgoCall,
			"go_version": runtime.Version(),
			"goos":       runtime.GOOS,
			"goarch":     runtime.GOARCH,
		},
		"gc_stats": map[string]interface{}{
			"num_gc":          gcStats.NumGC,
			"total_pause":     gcStats.PauseTotal,
			"last_gc":         gcStats.LastGC,
			"pause":           gcStats.Pause,
			"pause_end":       gcStats.PauseEnd,
			"pause_quantiles": gcStats.PauseQuantiles,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DatabaseMetrics godoc
// @Summary Database performance metrics
// @Description Returns database connection pool and query performance metrics
// @Tags profiling
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /debug/db [get]
func (h *ProfilingHandler) DatabaseMetrics(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := h.db.DB()
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get database connection for metrics")
		http.Error(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	stats := sqlDB.Stats()

	response := map[string]interface{}{
		"timestamp": time.Now(),
		"mysql": map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration,
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		},
	}

	// Get Redis stats
	if h.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		info, err := h.client.Info(ctx).Result()
		if err != nil {
			h.log.Error().Err(err).Msg("Failed to get Redis info")
		} else {
			response["redis"] = map[string]interface{}{
				"info": info,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ApplicationMetrics godoc
// @Summary Application performance metrics
// @Description Returns application-specific performance metrics and configuration
// @Tags profiling
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /debug/app [get]
func (h *ProfilingHandler) ApplicationMetrics(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"timestamp": time.Now(),
		"application": map[string]interface{}{
			"name":    "uas",
			"version": "1.0.0",
			"env":     config.AppConfig.Env,
			"debug":   config.AppConfig.Debug,
		},
		"configuration": map[string]interface{}{
			"jwt_expiry":           config.AppConfig.AccessJwtExpire,
			"refresh_token_expiry": config.AppConfig.RefreshJwtExpire,
			"max_failed_attempts":  config.AppConfig.MaxFailedAttempts,
			"lockout_duration":     config.AppConfig.AccountLockMinutes,
			"max_request_size":     config.AppConfig.MaxRequestSizeMB,
			"rate_limit_capacity":  config.AppConfig.RateLimitCapacity,
			"time_unit_seconds":    config.AppConfig.TimeUnitInSeconds,
			"enable_metrics":       config.AppConfig.EnableMetrics,
			"enable_profiling":     config.AppConfig.EnableProfiling,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HeapProfile godoc
// @Summary Heap memory profile
// @Description Returns heap memory profile in pprof format
// @Tags profiling
// @Produce application/octet-stream
// @Success 200 {file} string
// @Failure 500 {object} map[string]interface{}
// @Router /debug/pprof/heap [get]
func (h *ProfilingHandler) HeapProfile(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.EnableProfiling {
		http.Error(w, "Profiling is disabled", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=heap.prof")

	pprof.WriteHeapProfile(w)
}

// GoroutineProfile godoc
// @Summary Goroutine profile
// @Description Returns goroutine profile in pprof format
// @Tags profiling
// @Produce application/octet-stream
// @Success 200 {file} string
// @Failure 500 {object} map[string]interface{}
// @Router /debug/pprof/goroutine [get]
func (h *ProfilingHandler) GoroutineProfile(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.EnableProfiling {
		http.Error(w, "Profiling is disabled", http.StatusForbidden)
		return
	}

	profile := pprof.Lookup("goroutine")
	if profile == nil {
		http.Error(w, "goroutine profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=goroutine.prof")

	profile.WriteTo(w, 0)
}

// CPUProfile godoc
// @Summary CPU profile
// @Description Returns CPU profile in pprof format
// @Tags profiling
// @Produce application/octet-stream
// @Success 200 {file} string
// @Failure 500 {object} map[string]interface{}
// @Router /debug/pprof/profile [get]
func (h *ProfilingHandler) CPUProfile(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.EnableProfiling {
		http.Error(w, "Profiling is disabled", http.StatusForbidden)
		return
	}

	// Parse duration from query parameter (default 30 seconds)
	duration := 30 * time.Second
	if d := r.URL.Query().Get("seconds"); d != "" {
		if seconds, err := strconv.Atoi(d); err == nil && seconds > 0 {
			duration = time.Duration(seconds) * time.Second
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=cpu.prof")

	if err := pprof.StartCPUProfile(w); err != nil {
		h.log.Error().Err(err).Msg("Failed to start CPU profiling")
		http.Error(w, "Failed to start CPU profiling", http.StatusInternalServerError)
		return
	}

	time.Sleep(duration)
	pprof.StopCPUProfile()
}

// Trace godoc
// @Summary Execution trace
// @Description Returns execution trace for the specified duration
// @Tags profiling
// @Produce application/octet-stream
// @Param seconds query int false "Duration in seconds (default: 1)"
// @Success 200 {file} string
// @Failure 500 {object} map[string]interface{}
// @Router /debug/pprof/trace [get]
func (h *ProfilingHandler) Trace(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.EnableProfiling {
		http.Error(w, "Profiling is disabled", http.StatusForbidden)
		return
	}

	// Parse duration from query parameter (default 1 second)
	duration := 1 * time.Second
	if d := r.URL.Query().Get("seconds"); d != "" {
		if seconds, err := strconv.Atoi(d); err == nil && seconds > 0 {
			duration = time.Duration(seconds) * time.Second
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=trace.out")

	trace.Start(w)
	time.Sleep(duration)
	trace.Stop()
}

// BlockProfile godoc
// @Summary Block profile
// @Description Returns block profile in pprof format
// @Tags profiling
// @Produce application/octet-stream
// @Success 200 {file} string
// @Failure 500 {object} map[string]interface{}
// @Router /debug/pprof/block [get]
func (h *ProfilingHandler) BlockProfile(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.EnableProfiling {
		http.Error(w, "Profiling is disabled", http.StatusForbidden)
		return
	}

	profile := pprof.Lookup("block")
	if profile == nil {
		http.Error(w, "block profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=block.prof")

	profile.WriteTo(w, 0)
}

// MutexProfile godoc
// @Summary Mutex profile
// @Description Returns mutex profile in pprof format
// @Tags profiling
// @Produce application/octet-stream
// @Success 200 {file} string
// @Failure 500 {object} map[string]interface{}
// @Router /debug/pprof/mutex [get]
func (h *ProfilingHandler) MutexProfile(w http.ResponseWriter, r *http.Request) {
	if !config.AppConfig.EnableProfiling {
		http.Error(w, "Profiling is disabled", http.StatusForbidden)
		return
	}

	profile := pprof.Lookup("mutex")
	if profile == nil {
		http.Error(w, "mutex profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=mutex.prof")

	profile.WriteTo(w, 0)
}

// Index godoc
// @Summary Profiling index
// @Description Returns index of available profiling endpoints
// @Tags profiling
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /debug/pprof [get]
func (h *ProfilingHandler) Index(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"timestamp": time.Now(),
		"endpoints": []map[string]interface{}{
			{
				"path":        "/debug/metrics",
				"method":      "GET",
				"description": "System performance metrics",
			},
			{
				"path":        "/debug/db",
				"method":      "GET",
				"description": "Database performance metrics",
			},
			{
				"path":        "/debug/app",
				"method":      "GET",
				"description": "Application performance metrics",
			},
			{
				"path":        "/debug/pprof/",
				"method":      "GET",
				"description": "pprof index page",
			},
			{
				"path":        "/debug/pprof/heap",
				"method":      "GET",
				"description": "Heap memory profile",
			},
			{
				"path":        "/debug/pprof/goroutine",
				"method":      "GET",
				"description": "Goroutine profile",
			},
			{
				"path":        "/debug/pprof/profile",
				"method":      "GET",
				"description": "CPU profile",
			},
			{
				"path":        "/debug/pprof/trace",
				"method":      "GET",
				"description": "Execution trace",
			},
			{
				"path":        "/debug/pprof/block",
				"method":      "GET",
				"description": "Block profile",
			},
			{
				"path":        "/debug/pprof/mutex",
				"method":      "GET",
				"description": "Mutex profile",
			},
			{
				"path":        "/metrics",
				"method":      "GET",
				"description": "Prometheus metrics",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers profiling routes
func (h *ProfilingHandler) RegisterRoutes(router *mux.Router) {
	// Custom metrics endpoints
	router.HandleFunc("/debug/metrics", h.SystemMetrics).Methods("GET")
	router.HandleFunc("/debug/db", h.DatabaseMetrics).Methods("GET")
	router.HandleFunc("/debug/app", h.ApplicationMetrics).Methods("GET")

	// pprof endpoints
	router.HandleFunc("/debug/pprof/", h.Index).Methods("GET")
	router.HandleFunc("/debug/pprof/heap", h.HeapProfile).Methods("GET")
	router.HandleFunc("/debug/pprof/goroutine", h.GoroutineProfile).Methods("GET")
	router.HandleFunc("/debug/pprof/profile", h.CPUProfile).Methods("GET")
	router.HandleFunc("/debug/pprof/trace", h.Trace).Methods("GET")
	router.HandleFunc("/debug/pprof/block", h.BlockProfile).Methods("GET")
	router.HandleFunc("/debug/pprof/mutex", h.MutexProfile).Methods("GET")

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())
}
