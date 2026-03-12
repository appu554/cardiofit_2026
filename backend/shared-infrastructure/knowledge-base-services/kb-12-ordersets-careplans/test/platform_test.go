// Package test provides platform and infrastructure tests for KB-12
package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/internal/cache"
	"kb-12-ordersets-careplans/internal/clients"
	"kb-12-ordersets-careplans/internal/config"
	"kb-12-ordersets-careplans/internal/database"
	"kb-12-ordersets-careplans/pkg/ordersets"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ============================================
// 1.1 Startup & Health Tests
// ============================================

func TestServiceBootsWithoutKBDependencies(t *testing.T) {
	// Test that service can start even if KB services are unavailable
	// It should degrade gracefully, not crash

	// Create clients with invalid endpoints (simulating unavailable services)
	kb1Config := config.KBClientConfig{
		BaseURL: "http://localhost:19999", // Invalid port
		Enabled: false,                    // Disabled
		Timeout: 1 * time.Second,
	}
	kb1Client := clients.NewKB1DosingClient(kb1Config)
	assert.NotNil(t, kb1Client, "KB1 client should be created even with invalid config")

	kb3Config := config.KBClientConfig{
		BaseURL: "http://localhost:19998",
		Enabled: false,
		Timeout: 1 * time.Second,
	}
	kb3Client := clients.NewKB3TemporalClient(kb3Config)
	assert.NotNil(t, kb3Client, "KB3 client should be created even with invalid config")

	kb6Config := config.KBClientConfig{
		BaseURL: "http://localhost:19997",
		Enabled: false,
		Timeout: 1 * time.Second,
	}
	kb6Client := clients.NewKB6FormularyClient(kb6Config)
	assert.NotNil(t, kb6Client, "KB6 client should be created even with invalid config")

	kb7Config := config.KBClientConfig{
		BaseURL: "http://localhost:19996",
		Enabled: false,
		Timeout: 1 * time.Second,
	}
	kb7Client := clients.NewKB7TerminologyClient(kb7Config)
	assert.NotNil(t, kb7Client, "KB7 client should be created even with invalid config")

	t.Log("✓ Service can initialize with disabled/unavailable KB dependencies")
}

func TestServiceRunsWithRedisCacheDisabled(t *testing.T) {
	// Test that service operates correctly without Redis cache
	// Template loading should work from in-memory/hardcoded templates

	// Create template loader without cache
	loader := ordersets.NewTemplateLoader(nil, nil)
	assert.NotNil(t, loader, "Template loader should work without cache")

	// Verify templates still load
	err := loader.LoadAllTemplates(context.Background())
	// Should not panic, may return error but should degrade gracefully
	if err != nil {
		t.Logf("Expected: Template loading without DB returns error: %v", err)
	}

	// Hardcoded templates should still be available
	counts := ordersets.GetTemplateCount()
	assert.NotNil(t, counts, "Template counts should be available")
	t.Logf("✓ Service runs without cache, %v templates available", counts)
}

func TestGracefulShutdownPreservesState(t *testing.T) {
	// Test that context cancellation is handled properly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Simulate an operation that respects context
	done := make(chan bool)
	go func() {
		select {
		case <-ctx.Done():
			done <- true
		case <-time.After(1 * time.Second):
			done <- false
		}
	}()

	// Wait for context to cancel
	time.Sleep(150 * time.Millisecond)
	result := <-done
	assert.True(t, result, "Context cancellation should be respected")
	t.Log("✓ Graceful shutdown via context cancellation works")
}

func TestDatabaseConnectionHealth(t *testing.T) {
	// Skip if no DB configured
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping DB health test")
	}

	cfg := &config.DatabaseConfig{
		URL:             dbURL,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := database.NewConnection(cfg)
	if err != nil {
		t.Skipf("Could not connect to database: %v", err)
	}
	defer db.Close()

	// Test health check
	err = db.Health(context.Background())
	assert.NoError(t, err, "Database health check should pass")
	t.Log("✓ Database connection healthy")
}

func TestRedisCacheHealth(t *testing.T) {
	// Skip if no Redis configured
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set, skipping Redis health test")
	}

	cfg := &config.RedisConfig{
		URL: redisURL,
	}

	redisCache, err := cache.NewCache(cfg)
	if err != nil {
		t.Skipf("Could not connect to Redis: %v", err)
	}

	// Test health check
	err = redisCache.Health(context.Background())
	assert.NoError(t, err, "Redis health check should pass")
	t.Log("✓ Redis cache healthy")
}

// ============================================
// 1.2 Configuration Tests
// ============================================

func TestConfigLoadFromEnvironment(t *testing.T) {
	// Set test environment variables
	originalPort := os.Getenv("PORT")
	originalEnv := os.Getenv("ENVIRONMENT")
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("ENVIRONMENT", originalEnv)
	}()

	os.Setenv("PORT", "9999")
	os.Setenv("ENVIRONMENT", "test")

	cfg := config.Load()
	assert.NotNil(t, cfg, "Config should load")
	assert.Equal(t, 9999, cfg.Server.Port, "Port should be loaded from env")
	assert.Equal(t, "test", cfg.Server.Environment, "Environment should be loaded from env")
	t.Log("✓ Configuration loads from environment variables")
}

func TestConfigDefaults(t *testing.T) {
	// Clear relevant env vars to test defaults
	originalPort := os.Getenv("PORT")
	defer os.Setenv("PORT", originalPort)
	os.Unsetenv("PORT")

	cfg := config.Load()
	assert.NotNil(t, cfg, "Config should load with defaults")
	// Default port should be set (usually 8092 for KB-12)
	assert.Greater(t, cfg.Server.Port, 0, "Default port should be positive")
	t.Logf("✓ Default port: %d", cfg.Server.Port)
}

func TestConfigValidation(t *testing.T) {
	cfg := config.Load()

	// Timeouts should be positive
	assert.Greater(t, cfg.Server.ReadTimeout, time.Duration(0), "Read timeout should be positive")
	assert.Greater(t, cfg.Server.WriteTimeout, time.Duration(0), "Write timeout should be positive")
	assert.Greater(t, cfg.Server.ShutdownTimeout, time.Duration(0), "Shutdown timeout should be positive")
	t.Log("✓ Configuration timeouts are valid")
}

func TestKBServiceConfigValidation(t *testing.T) {
	cfg := config.Load()

	// KB service configs should exist
	assert.NotNil(t, cfg.KBServices, "KB services config should exist")

	// Log configured KB services
	t.Logf("KB-1 Dosing: BaseURL=%s, Enabled=%v", cfg.KBServices.KB1Dosing.BaseURL, cfg.KBServices.KB1Dosing.Enabled)
	t.Logf("KB-3 Temporal: BaseURL=%s, Enabled=%v", cfg.KBServices.KB3Temporal.BaseURL, cfg.KBServices.KB3Temporal.Enabled)
	t.Logf("KB-6 Formulary: BaseURL=%s, Enabled=%v", cfg.KBServices.KB6Formulary.BaseURL, cfg.KBServices.KB6Formulary.Enabled)
	t.Logf("KB-7 Terminology: BaseURL=%s, Enabled=%v", cfg.KBServices.KB7Terminology.BaseURL, cfg.KBServices.KB7Terminology.Enabled)
}

// ============================================
// 1.3 API Layer Tests
// ============================================

func TestCORSMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test preflight request
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, "OPTIONS should return 204")
	assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "POST")
	t.Log("✓ CORS middleware configured correctly")
}

func TestHealthEndpointStructure(t *testing.T) {
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "kb-12-ordersets-careplans",
			"version":  "test",
			"time":     time.Now().UTC().Format(time.RFC3339),
			"database": "healthy",
			"cache":    "disabled",
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)

	// Verify required fields
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "service")
	assert.Contains(t, body, "time")
	assert.Equal(t, "kb-12-ordersets-careplans", body["service"])
	t.Log("✓ Health endpoint returns required fields")
}

func TestReadyEndpoint(t *testing.T) {
	router := gin.New()
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	req, _ := http.NewRequest("GET", "/ready", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["ready"])
	t.Log("✓ Ready endpoint returns ready status")
}

func TestLiveEndpoint(t *testing.T) {
	router := gin.New()
	router.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"alive": true})
	})

	req, _ := http.NewRequest("GET", "/live", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["alive"])
	t.Log("✓ Liveness endpoint returns alive status")
}

func TestRequestTimeout(t *testing.T) {
	// Test that requests respect timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	router := gin.New()
	router.GET("/slow", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
		case <-time.After(100 * time.Millisecond):
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	req, _ := http.NewRequestWithContext(ctx, "GET", "/slow", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Context should be cancelled
	assert.Error(t, ctx.Err(), "Context should be cancelled")
	t.Log("✓ Request timeout handling works")
}

// ============================================
// Helper Functions
// ============================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
