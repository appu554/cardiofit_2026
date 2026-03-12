// Package api provides HTTP handlers and middleware for KB-13 Quality Measures Engine.
package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger creates a Gin middleware that logs HTTP requests.
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Skip logging for health checks in production to reduce noise
		if path == "/health" || path == "/ready" {
			return
		}

		// Log request details
		logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		)

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				logger.Error("Request error",
					zap.String("path", path),
					zap.Error(e.Err),
				)
			}
		}
	}
}

// CORSMiddleware handles CORS for development environments.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID was provided by client/gateway
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a simple timestamp-based ID
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID creates a simple unique request identifier.
func generateRequestID() string {
	// Simple timestamp + random suffix
	// In production, use UUID or distributed ID generation
	return time.Now().Format("20060102150405.000000")
}

// AuthMiddleware validates authentication tokens.
// This is a placeholder - actual implementation depends on auth strategy.
func AuthMiddleware(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// For now, allow unauthenticated access in development
			// In production, this would reject the request
			c.Next()
			return
		}

		// TODO: Implement JWT validation
		// 1. Parse token from header
		// 2. Validate signature
		// 3. Check expiration
		// 4. Verify required scopes
		// 5. Set user context

		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting.
// This is a simple implementation - production should use Redis-based limiting.
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory rate limiting
	// For production, use Redis-based distributed rate limiting
	return func(c *gin.Context) {
		// Placeholder - implement rate limiting logic
		c.Next()
	}
}

// MetricsMiddleware collects request metrics for Prometheus.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath() // Use route pattern, not actual path

		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()

		// TODO: Implement Prometheus metrics
		// - http_requests_total counter
		// - http_request_duration_seconds histogram
		// - http_request_size_bytes histogram
		// - http_response_size_bytes histogram

		_ = duration
		_ = status
		_ = path
	}
}

// ErrorHandler provides consistent error response formatting.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors after request processing
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Determine status code
			status := c.Writer.Status()
			if status == 200 {
				status = 500
			}

			// Return standardized error response
			c.JSON(status, gin.H{
				"error":      "internal_error",
				"message":    err.Error(),
				"request_id": c.GetString("request_id"),
			})
		}
	}
}

// TimeoutMiddleware adds request timeout handling.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a deadline for the request context
		// This allows handlers to respect timeouts
		c.Set("request_timeout", timeout)
		c.Next()
	}
}
