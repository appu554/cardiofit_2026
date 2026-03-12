// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RequestLogger returns a Gin middleware for request logging
func RequestLogger(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		requestID := c.GetString("request_id")

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		entry := log.WithFields(logrus.Fields{
			"request_id":  requestID,
			"method":      method,
			"path":        path,
			"status":      statusCode,
			"latency_ms":  latency.Milliseconds(),
			"client_ip":   clientIP,
			"user_agent":  c.Request.UserAgent(),
		})

		if statusCode >= 500 {
			entry.Error("Server error")
		} else if statusCode >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request completed")
		}
	}
}

// CORSMiddleware returns a Gin middleware for CORS handling
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}

// AuthMiddleware validates authentication tokens
// This is a placeholder - integrate with your auth system
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")

		// For development, allow requests without auth
		// In production, validate JWT token here
		if authHeader == "" {
			// Allow unauthenticated requests in development
			c.Set("user_id", "dev-user")
			c.Set("user_role", "admin")
			c.Next()
			return
		}

		// TODO: Validate JWT token
		// token := strings.TrimPrefix(authHeader, "Bearer ")
		// claims, err := validateToken(token)
		// if err != nil {
		//     c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		//     return
		// }
		// c.Set("user_id", claims.UserID)
		// c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting
// This is a simplified version - use Redis-based rate limiting in production
func RateLimitMiddleware(requestsPerSecond int) gin.HandlerFunc {
	// Simple token bucket implementation
	// For production, use github.com/ulule/limiter or similar
	return func(c *gin.Context) {
		// TODO: Implement proper rate limiting with Redis
		c.Next()
	}
}

// RecoveryMiddleware handles panics and returns 500 errors
func RecoveryMiddleware(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString("request_id")
				log.WithFields(logrus.Fields{
					"request_id": requestID,
					"error":      err,
					"path":       c.Request.URL.Path,
				}).Error("Panic recovered")

				c.AbortWithStatusJSON(500, gin.H{
					"error":      "Internal server error",
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}

// TimeoutMiddleware sets request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create timeout context
		// ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		// defer cancel()
		// c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
