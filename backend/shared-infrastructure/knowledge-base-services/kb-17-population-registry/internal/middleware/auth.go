// Package middleware provides HTTP middleware for KB-17 Population Registry
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled     bool
	APIKeyName  string
	APIKeys     map[string]string // key -> service name
	JWTSecret   string
	SkipPaths   []string // paths to skip authentication
}

// DefaultAuthConfig returns default auth configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		Enabled:    false, // Disabled by default for development
		APIKeyName: "X-API-Key",
		APIKeys:    make(map[string]string),
		SkipPaths: []string{
			"/health",
			"/ready",
			"/metrics",
		},
	}
}

// AuthMiddleware creates authentication middleware
func AuthMiddleware(config *AuthConfig, logger *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if auth is disabled
		if !config.Enabled {
			c.Next()
			return
		}

		// Skip for excluded paths
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path || strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Check for API key
		apiKey := c.GetHeader(config.APIKeyName)
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != "" {
			if serviceName, valid := config.APIKeys[apiKey]; valid {
				c.Set("authenticated", true)
				c.Set("auth_method", "api_key")
				c.Set("service_name", serviceName)
				c.Next()
				return
			}
		}

		// Check for Bearer token
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if validateJWT(token, config.JWTSecret) {
				c.Set("authenticated", true)
				c.Set("auth_method", "jwt")
				c.Next()
				return
			}
		}

		// Authentication failed
		logger.WithFields(logrus.Fields{
			"path":      c.Request.URL.Path,
			"method":    c.Request.Method,
			"client_ip": c.ClientIP(),
		}).Warn("Authentication failed")

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Valid authentication required",
		})
	}
}

// validateJWT validates a JWT token (simplified implementation)
func validateJWT(token, secret string) bool {
	// In production, use proper JWT validation library
	// This is a placeholder that accepts any non-empty token when secret is set
	if secret == "" {
		return true // No secret configured, allow all tokens
	}
	return token != "" && len(token) > 10
}

// ServiceAuthConfig holds service-to-service auth configuration
type ServiceAuthConfig struct {
	ServiceName string
	ServiceKey  string
}

// ServiceAuthMiddleware creates middleware for service-to-service auth
func ServiceAuthMiddleware(config *ServiceAuthConfig, logger *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceKey := c.GetHeader("X-Service-Key")
		serviceName := c.GetHeader("X-Service-Name")

		if serviceKey == config.ServiceKey && serviceName != "" {
			c.Set("calling_service", serviceName)
			c.Set("is_service_call", true)
			c.Next()
			return
		}

		// Allow non-service calls to proceed (they'll be handled by regular auth)
		c.Set("is_service_call", false)
		c.Next()
	}
}

// RequireRole creates middleware that requires a specific role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Role information not available",
			})
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Invalid role format",
			})
			return
		}

		for _, role := range roles {
			if roleStr == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Insufficient permissions",
		})
	}
}
