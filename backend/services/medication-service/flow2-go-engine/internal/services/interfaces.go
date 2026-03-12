package services

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// CacheService interface defines caching operations
type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
}

// MetricsService interface defines metrics collection operations
type MetricsService interface {
	// Flow 2 specific metrics
	RecordFlow2Execution(duration time.Duration, status string, recipesExecuted int)
	RecordMedicationIntelligence(duration time.Duration, intelligenceScore float64)
	RecordDoseOptimization(duration time.Duration, optimizationScore float64)
	RecordSafetyValidation(duration time.Duration, safetyStatus string)
	IncrementFlow2Errors()
	
	// HTTP metrics
	RecordHTTPRequest(method, path string, statusCode int, duration time.Duration)
	
	// Cache metrics
	IncrementCacheHits(cacheLevel string)
	IncrementCacheMisses(cacheLevel string)
	
	// Rust engine metrics
	IncrementRustEngineFailures()
	RecordRustEngineLatency(duration time.Duration)
	
	// Prometheus handler
	PrometheusHandler(c *gin.Context)
	
	// Registry access
	GetRegistry() *prometheus.Registry
}

// HealthService interface defines health check operations
type HealthService interface {
	HealthCheck(c *gin.Context)
	ReadinessCheck(c *gin.Context)
	LivenessCheck(c *gin.Context)
	
	// Component health checks
	CheckRustEngine(ctx context.Context) error
	CheckRedis(ctx context.Context) error
	CheckContextService(ctx context.Context) error
	CheckMedicationAPI(ctx context.Context) error
	
	// Health status
	IsHealthy() bool
	GetHealthStatus() map[string]interface{}
}
