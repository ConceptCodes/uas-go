package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"uas/config"
	"uas/internal/models"
)

func setupHealthHandler(t *testing.T) (*HealthHandler, sqlmock.Sqlmock, *miniredis.Miniredis) {
	t.Helper()
	saveAppConfig(t)

	config.AppConfig.RefreshJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.AccessJwtSecret = "12345678901234567890123456789012"
	config.AppConfig.DbHost = "localhost"
	config.AppConfig.RedisHost = "localhost"

	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	log := newTestLogger()
	handler := NewHealthHandler(&log, gormDB, redisClient)

	return handler, sqlMock, mr
}

func TestHealthHandler_Liveness(t *testing.T) {
	handler, _, _ := setupHealthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health/alive", nil)
	rec := httptest.NewRecorder()

	handler.Liveness(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp models.HealthCheckResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Status)
	assert.Equal(t, "uas", resp.Service)
}

func TestHealthHandler_Readiness_Healthy(t *testing.T) {
	handler, sqlMock, _ := setupHealthHandler(t)

	sqlMock.ExpectQuery("SELECT 1$").
		WillReturnRows(sqlmock.NewRows([]string{"?"}).AddRow(1))

	req := httptest.NewRequest(http.MethodGet, "/health/status", nil)
	rec := httptest.NewRecorder()

	handler.Readiness(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Status    string                        `json:"status"`
		Timestamp string                        `json:"timestamp"`
		Version   string                        `json:"version"`
		Checks    map[string]HealthCheckDetails `json:"checks"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)

	assert.Equal(t, "healthy", resp.Checks["mysql"].Status)
	assert.Equal(t, "healthy", resp.Checks["redis"].Status)
	assert.Equal(t, "healthy", resp.Checks["configuration"].Status)
}

type HealthCheckDetails struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func TestHealthHandler_Readiness_MysqlDegraded(t *testing.T) {
	handler, sqlMock, _ := setupHealthHandler(t)

	sqlMock.ExpectQuery("SELECT 1$").
		WillReturnError(assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/health/status", nil)
	rec := httptest.NewRecorder()

	handler.Readiness(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp struct {
		Status    string                        `json:"status"`
		Checks    map[string]HealthCheckDetails `json:"checks"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "degraded", resp.Checks["mysql"].Status)
}

func TestHealthHandler_Readiness_ConfigurationIssues(t *testing.T) {
	handler, sqlMock, _ := setupHealthHandler(t)

	sqlMock.ExpectQuery("SELECT 1$").
		WillReturnRows(sqlmock.NewRows([]string{"?"}).AddRow(1))

	config.AppConfig.RefreshJwtSecret = "short"

	req := httptest.NewRequest(http.MethodGet, "/health/status", nil)
	rec := httptest.NewRecorder()

	handler.Readiness(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp struct {
		Status    string                        `json:"status"`
		Checks    map[string]HealthCheckDetails `json:"checks"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "unhealthy", resp.Checks["configuration"].Status)
}
