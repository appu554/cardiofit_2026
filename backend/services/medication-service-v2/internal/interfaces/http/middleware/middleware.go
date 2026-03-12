package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Logger creates a Gin middleware for structured logging
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		fields := []zap.Field{
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		}

		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		if param.StatusCode >= 400 {
			logger.Error("HTTP request completed with error", fields...)
		} else {
			logger.Info("HTTP request completed", fields...)
		}

		return "" // Return empty string as we're using structured logging
	})
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	globalLimiter *rate.Limiter
	clientLimiters map[string]*rate.Limiter
	mu            sync.RWMutex
	rps           rate.Limit
	burst         int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		globalLimiter:  rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
		clientLimiters: make(map[string]*rate.Limiter),
		rps:           rate.Limit(requestsPerSecond / 10), // Per-client limit is 1/10 of global
		burst:         burst,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check global rate limit first
		if !rl.globalLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Global rate limit exceeded",
				"code":    "TOO_MANY_REQUESTS",
				"retry_after": "60s",
			})
			c.Abort()
			return
		}

		// Check per-client rate limit
		clientIP := c.ClientIP()
		clientLimiter := rl.getClientLimiter(clientIP)
		
		if !clientLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Client rate limit exceeded",
				"code":    "TOO_MANY_REQUESTS",
				"retry_after": "60s",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getClientLimiter returns a rate limiter for the specific client
func (rl *RateLimiter) getClientLimiter(clientIP string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.clientLimiters[clientIP]
	rl.mu.RUnlock()

	if !exists {
		limiter = rate.NewLimiter(rl.rps, rl.burst)
		rl.mu.Lock()
		rl.clientLimiters[clientIP] = limiter
		rl.mu.Unlock()
	}

	return limiter
}

// Cleanup removes old client limiters to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if len(rl.clientLimiters) > 10000 {
		// Clear half of the map
		newMap := make(map[string]*rate.Limiter)
		count := 0
		for k, v := range rl.clientLimiters {
			if count < 5000 {
				newMap[k] = v
				count++
			} else {
				break
			}
		}
		rl.clientLimiters = newMap
	}
}

// Metrics middleware for collecting HTTP metrics
type Metrics struct {
	serviceName     string
	requestCount    map[string]map[int]int64  // path -> status -> count
	requestDuration map[string]time.Duration // path -> total duration
	mu              sync.RWMutex
}

// NewMetrics creates a new metrics middleware
func NewMetrics(serviceName string) *Metrics {
	return &Metrics{
		serviceName:     serviceName,
		requestCount:    make(map[string]map[int]int64),
		requestDuration: make(map[string]time.Duration),
	}
}

// Middleware returns the metrics collection middleware
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		c.Next()
		
		duration := time.Since(startTime)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := c.Writer.Status()

		m.mu.Lock()
		defer m.mu.Unlock()

		// Initialize path metrics if not exists
		if m.requestCount[path] == nil {
			m.requestCount[path] = make(map[int]int64)
		}

		// Update metrics
		m.requestCount[path][status]++
		m.requestDuration[path] += duration
	}
}

// GetMetrics returns current metrics snapshot
func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := map[string]interface{}{
		"service": m.serviceName,
		"paths":   make(map[string]interface{}),
	}

	for path, statusCounts := range m.requestCount {
		pathMetrics := map[string]interface{}{
			"status_codes":   statusCounts,
			"total_duration": m.requestDuration[path],
		}

		// Calculate total requests for this path
		totalRequests := int64(0)
		for _, count := range statusCounts {
			totalRequests += count
		}
		pathMetrics["total_requests"] = totalRequests

		if totalRequests > 0 {
			pathMetrics["average_duration"] = m.requestDuration[path] / time.Duration(totalRequests)
		}

		metrics["paths"].(map[string]interface{})[path] = pathMetrics
	}

	return metrics
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Remove server information
		c.Header("Server", "")
		
		c.Next()
	}
}

// HIPAAAudit provides HIPAA compliance audit logging for HTTP requests
func HIPAAAudit(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Log request start
		auditData := map[string]interface{}{
			"event_type":    "http_request",
			"event_action":  "request_started",
			"timestamp":     startTime.UTC(),
			"service":       "medication-service-v2",
			"component":     "http-server",
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"query":         c.Request.URL.RawQuery,
			"client_ip":     c.ClientIP(),
			"user_agent":    c.Request.UserAgent(),
		}

		// Extract additional headers
		if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
			auditData["x_forwarded_for"] = forwardedFor
		}
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			auditData["request_id"] = requestID
		}

		logger.Info("HIPAA Audit - Request Start",
			zap.String("audit_type", "access"),
			zap.Any("audit_data", auditData))

		c.Next()

		// Log request completion
		duration := time.Since(startTime)
		completionData := map[string]interface{}{
			"event_type":     "http_request",
			"event_action":   "request_completed",
			"timestamp":      time.Now().UTC(),
			"service":        "medication-service-v2",
			"component":      "http-server",
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"status_code":    c.Writer.Status(),
			"duration_ms":    duration.Milliseconds(),
			"response_size":  c.Writer.Size(),
			"client_ip":      c.ClientIP(),
		}

		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			completionData["request_id"] = requestID
		}

		// Extract user information if available
		if authCtx, exists := GetAuthContext(c); exists {
			completionData["user_id"] = authCtx.UserID
			completionData["username"] = authCtx.Username
			completionData["user_roles"] = authCtx.Roles
		}

		// Log errors separately for HIPAA compliance
		if c.Writer.Status() >= 400 {
			logger.Error("HIPAA Audit - Request Error",
				zap.String("audit_type", "error"),
				zap.Any("audit_data", completionData))
		} else {
			logger.Info("HIPAA Audit - Request Completed",
				zap.String("audit_type", "access"),
				zap.Any("audit_data", completionData))
		}
	}
}

// Timeout middleware sets a timeout for requests
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a timeout context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace the request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to receive the result
		finished := make(chan struct{})
		
		go func() {
			defer close(finished)
			c.Next()
		}()

		select {
		case <-finished:
			// Request completed successfully
		case <-ctx.Done():
			// Request timed out
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "request_timeout",
				"message": fmt.Sprintf("Request timeout after %v", timeout),
				"code":    "REQUEST_TIMEOUT",
			})
			c.Abort()
		}
	}
}

// RequestSizeLimit limits the size of request bodies
func RequestSizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "request_too_large",
				"message": fmt.Sprintf("Request body too large. Maximum size is %d bytes", maxSize),
				"code":    "REQUEST_TOO_LARGE",
			})
			c.Abort()
			return
		}

		// Limit the request body reader
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// CORS middleware configuration for complex CORS scenarios
type CORSConfig struct {
	AllowAllOrigins  bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// CustomCORS creates a custom CORS middleware
func CustomCORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if config.AllowAllOrigins || contains(config.AllowOrigins, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Methods", joinStrings(config.AllowMethods, ", "))
				c.Header("Access-Control-Allow-Headers", joinStrings(config.AllowHeaders, ", "))
				
				if config.AllowCredentials {
					c.Header("Access-Control-Allow-Credentials", "true")
				}
				
				if config.MaxAge > 0 {
					c.Header("Access-Control-Max-Age", strconv.Itoa(int(config.MaxAge.Seconds())))
				}
			}
			
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Handle actual requests
		if config.AllowAllOrigins {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if contains(config.AllowOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", joinStrings(config.ExposeHeaders, ", "))
		}

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Next()
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinStrings(slice []string, separator string) string {
	if len(slice) == 0 {
		return ""
	}
	
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += separator + slice[i]
	}
	return result
}

// ValidationMiddleware provides request validation
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic validation checks
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "content_type_required",
					"message": "Content-Type header is required",
					"code":    "BAD_REQUEST",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}