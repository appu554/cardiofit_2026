package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDefaultAuthConfig(t *testing.T) {
	config := DefaultAuthConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, "X-API-Key", config.APIKeyName)
	assert.NotNil(t, config.APIKeys)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/ready")
	assert.Contains(t, config.SkipPaths, "/metrics")
}

func TestDefaultLoggingConfig(t *testing.T) {
	config := DefaultLoggingConfig()

	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/ready")
	assert.False(t, config.LogRequestBody)
	assert.False(t, config.LogResponseBody)
	assert.Equal(t, 1024, config.MaxBodyLogSize)
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 100, config.RequestsPerMin)
	assert.Equal(t, 20, config.BurstSize)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.False(t, config.UseRedis)
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &AuthConfig{Enabled: false}

	router := gin.New()
	router.Use(AuthMiddleware(config, logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_SkipPaths(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &AuthConfig{
		Enabled:   true,
		SkipPaths: []string{"/health", "/ready"},
		APIKeys:   map[string]string{},
	}

	router := gin.New()
	router.Use(AuthMiddleware(config, logger))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &AuthConfig{
		Enabled:    true,
		APIKeyName: "X-API-Key",
		APIKeys: map[string]string{
			"test-key-123": "test-service",
		},
		SkipPaths: []string{},
	}

	router := gin.New()
	router.Use(AuthMiddleware(config, logger))
	router.GET("/test", func(c *gin.Context) {
		serviceName, _ := c.Get("service_name")
		c.JSON(http.StatusOK, gin.H{"service": serviceName})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &AuthConfig{
		Enabled:    true,
		APIKeyName: "X-API-Key",
		APIKeys: map[string]string{
			"valid-key": "service",
		},
		SkipPaths: []string{},
	}

	router := gin.New()
	router.Use(AuthMiddleware(config, logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoggingMiddleware_SkipPaths(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &LoggingConfig{
		SkipPaths: []string{"/health"},
	}

	router := gin.New()
	router.Use(LoggingMiddleware(config, logger))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggingMiddleware_RequestID(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := DefaultLoggingConfig()
	config.SkipPaths = []string{}

	router := gin.New()
	router.Use(LoggingMiddleware(config, logger))
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestCorrelationMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(CorrelationMiddleware())
	router.GET("/test", func(c *gin.Context) {
		correlationID := c.GetString("correlation_id")
		spanID := c.GetString("span_id")
		c.JSON(http.StatusOK, gin.H{
			"correlation_id": correlationID,
			"span_id":        spanID,
		})
	})

	// Test with no incoming correlation ID
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
	assert.NotEmpty(t, w.Header().Get("X-Span-ID"))

	// Test with incoming correlation ID
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Correlation-ID", "existing-correlation-id")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "existing-correlation-id", w2.Header().Get("X-Correlation-ID"))
}

func TestInMemoryRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:        true,
		RequestsPerMin: 10,
		BurstSize:      5,
	}

	limiter := NewInMemoryRateLimiter(config)
	defer limiter.Stop()

	// First requests should be allowed
	for i := 0; i < 5; i++ {
		allowed, remaining, _ := limiter.Allow(nil, "test-client")
		assert.True(t, allowed, "Request %d should be allowed", i)
		assert.GreaterOrEqual(t, remaining, 0)
	}

	// Exhaust the bucket
	for i := 0; i < 10; i++ {
		limiter.Allow(nil, "test-client")
	}

	// Should be rate limited
	allowed, _, retryAfter := limiter.Allow(nil, "test-client")
	assert.False(t, allowed)
	assert.Greater(t, retryAfter, time.Duration(0))
}

func TestRateLimitMiddleware_Disabled(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &RateLimitConfig{
		Enabled: false,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter, config, logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitMiddleware_SkipPaths(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &RateLimitConfig{
		Enabled:        true,
		RequestsPerMin: 1,
		BurstSize:      1,
		SkipPaths:      []string{"/health"},
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter, config, logger))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Health endpoint should not be rate limited
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimitMiddleware_Headers(t *testing.T) {
	logger := logrus.New().WithField("test", true)
	config := &RateLimitConfig{
		Enabled:        true,
		RequestsPerMin: 100,
		BurstSize:      20,
		SkipPaths:      []string{},
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter, config, logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
}

func TestIsDataAccessEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/patients/123", true},
		{"/api/v1/enrollments", true},
		{"/api/v1/evaluate", true},
		{"/api/v1/registries", false},
		{"/health", false},
		{"/api/v1/stats", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isDataAccessEndpoint(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateJWT(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		secret   string
		expected bool
	}{
		{
			name:     "empty token with empty secret",
			token:    "",
			secret:   "",
			expected: true, // When no secret configured, auth is bypassed
		},
		{
			name:     "valid token with empty secret",
			token:    "valid-token-12345",
			secret:   "",
			expected: true,
		},
		{
			name:     "short token with secret",
			token:    "short",
			secret:   "secret",
			expected: false,
		},
		{
			name:     "valid token with secret",
			token:    "long-valid-token-12345",
			secret:   "secret",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateJWT(tt.token, tt.secret)
			assert.Equal(t, tt.expected, result)
		})
	}
}
