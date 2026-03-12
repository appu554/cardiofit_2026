package security

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret           string        `json:"jwt_secret"`
	APIKeys             []string      `json:"api_keys"`
	RequireAuth         bool          `json:"require_auth"`
	TokenExpiry         time.Duration `json:"token_expiry"`
	RefreshTokenExpiry  time.Duration `json:"refresh_token_expiry"`
	AllowAnonymousRead  bool          `json:"allow_anonymous_read"`
	TrustedProxies      []string      `json:"trusted_proxies"`
}

// UserContext represents authenticated user context
type UserContext struct {
	UserID       string                 `json:"user_id"`
	Email        string                 `json:"email,omitempty"`
	Organization string                 `json:"organization,omitempty"`
	Scopes       []string               `json:"scopes"`
	IsAnonymous  bool                   `json:"is_anonymous"`
	AuthMethod   string                 `json:"auth_method"` // jwt, api_key, anonymous
	Claims       map[string]interface{} `json:"claims,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
}

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	licenseEnforcer *LicenseEnforcer
	rateLimiter     *RateLimiter
	logger          *zap.Logger
	config          *AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(licenseEnforcer *LicenseEnforcer, rateLimiter *RateLimiter, logger *zap.Logger, config *AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		licenseEnforcer: licenseEnforcer,
		rateLimiter:     rateLimiter,
		logger:          logger,
		config:          config,
	}
}

// AuthenticationMiddleware performs authentication
func (am *AuthMiddleware) AuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Extract user context
		userCtx, err := am.extractUserContext(c)
		if err != nil {
			am.logger.Warn("Authentication failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication_failed",
				"message": "Invalid or missing authentication credentials",
			})
			c.Abort()
			return
		}

		// Check if authentication is required
		if am.config.RequireAuth && userCtx.IsAnonymous {
			operation := am.getOperationFromPath(c.Request.URL.Path)
			
			// Allow anonymous read operations if configured
			if am.config.AllowAnonymousRead && am.isReadOperation(operation) {
				userCtx.UserID = "anonymous"
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "authentication_required",
					"message": "Authentication is required for this operation",
				})
				c.Abort()
				return
			}
		}

		// Store user context in request context
		ctx = context.WithValue(ctx, "user", userCtx)
		c.Request = c.Request.WithContext(ctx)

		// Log authentication
		am.logger.Debug("User authenticated",
			zap.String("user_id", userCtx.UserID),
			zap.String("auth_method", userCtx.AuthMethod),
			zap.String("ip_address", userCtx.IPAddress))

		c.Next()
	}
}

// RateLimitingMiddleware performs rate limiting
func (am *AuthMiddleware) RateLimitingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Get user context
		userCtx := am.getUserContextFromRequest(c)
		if userCtx == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal_error",
				"message": "Failed to get user context",
			})
			c.Abort()
			return
		}

		// Determine rate limiting key
		rateLimitKey := am.getRateLimitKey(userCtx, c)
		operation := am.getOperationFromPath(c.Request.URL.Path)

		// Check rate limits
		result, err := am.rateLimiter.CheckLimit(ctx, rateLimitKey, operation)
		if err != nil {
			am.logger.Error("Rate limit check failed", zap.Error(err))
			// Continue on error (fail open)
		} else if !result.Allowed {
			am.logger.Warn("Rate limit exceeded",
				zap.String("user_id", userCtx.UserID),
				zap.String("key", rateLimitKey),
				zap.String("operation", operation),
				zap.String("limit_type", result.LimitType))

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", result.RequestsUsed+result.Remaining))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime.Unix()))
			
			if result.RetryAfter > 0 {
				c.Header("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate_limit_exceeded",
				"message": fmt.Sprintf("Rate limit exceeded for %s operations", result.LimitType),
				"retry_after": result.RetryAfter.Seconds(),
				"limit_type": result.LimitType,
			})
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		if result != nil {
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime.Unix()))
		}

		c.Next()
	}
}

// AuthorizationMiddleware performs authorization checks
func (am *AuthMiddleware) AuthorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Get user context
		userCtx := am.getUserContextFromRequest(c)
		if userCtx == nil || userCtx.IsAnonymous {
			// Anonymous users handled in authentication middleware
			c.Next()
			return
		}

		// Extract system and operation from request
		system := am.getSystemFromRequest(c)
		operation := am.getOperationFromPath(c.Request.URL.Path)

		// Check license and access permissions
		if system != "" {
			err := am.licenseEnforcer.ValidateAccess(ctx, userCtx.UserID, system, operation)
			if err != nil {
				am.logger.Warn("Authorization failed",
					zap.String("user_id", userCtx.UserID),
					zap.String("system", system),
					zap.String("operation", operation),
					zap.Error(err))

				c.JSON(http.StatusForbidden, gin.H{
					"error": "access_denied",
					"message": err.Error(),
					"system": system,
					"operation": operation,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// AuditLoggingMiddleware logs all requests for audit purposes
func (am *AuthMiddleware) AuditLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Log request details
		userCtx := am.getUserContextFromRequest(c)
		duration := time.Since(start)
		
		logFields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if userCtx != nil {
			logFields = append(logFields,
				zap.String("user_id", userCtx.UserID),
				zap.String("auth_method", userCtx.AuthMethod),
				zap.String("organization", userCtx.Organization),
			)
		}

		// Add system information if available
		if system := am.getSystemFromRequest(c); system != "" {
			logFields = append(logFields, zap.String("system", system))
		}

		// Log at appropriate level based on status
		if c.Writer.Status() >= 500 {
			am.logger.Error("Request completed", logFields...)
		} else if c.Writer.Status() >= 400 {
			am.logger.Warn("Request completed", logFields...)
		} else {
			am.logger.Info("Request completed", logFields...)
		}
	}
}

// extractUserContext extracts user context from request
func (am *AuthMiddleware) extractUserContext(c *gin.Context) (*UserContext, error) {
	// Try JWT token first
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return am.validateJWTToken(token, c)
		}
	}

	// Try API key
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return am.validateAPIKey(apiKey, c)
	}

	// Try API key in query parameter
	if apiKey := c.Query("api_key"); apiKey != "" {
		return am.validateAPIKey(apiKey, c)
	}

	// Return anonymous user context
	return &UserContext{
		UserID:      "anonymous",
		IsAnonymous: true,
		AuthMethod:  "anonymous",
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		Scopes:      []string{"terminology:public:read"},
	}, nil
}

// validateJWTToken validates a JWT token and extracts user context
func (am *AuthMiddleware) validateJWTToken(tokenString string, c *gin.Context) (*UserContext, error) {
	claims, err := am.licenseEnforcer.ValidateJWTToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT token: %w", err)
	}

	// Extract user information from claims
	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	organization, _ := claims["org"].(string)
	
	// Extract scopes
	var scopes []string
	if scopesInterface, ok := claims["scopes"]; ok {
		if scopesSlice, ok := scopesInterface.([]interface{}); ok {
			for _, scope := range scopesSlice {
				if scopeStr, ok := scope.(string); ok {
					scopes = append(scopes, scopeStr)
				}
			}
		}
	}

	if userID == "" {
		return nil, fmt.Errorf("missing user ID in token claims")
	}

	return &UserContext{
		UserID:       userID,
		Email:        email,
		Organization: organization,
		Scopes:       scopes,
		IsAnonymous:  false,
		AuthMethod:   "jwt",
		Claims:       claims,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	}, nil
}

// validateAPIKey validates an API key and returns user context
func (am *AuthMiddleware) validateAPIKey(apiKey string, c *gin.Context) (*UserContext, error) {
	// Check against configured API keys using constant-time comparison
	validKey := false
	for _, validAPIKey := range am.config.APIKeys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validAPIKey)) == 1 {
			validKey = true
			break
		}
	}

	if !validKey {
		return nil, fmt.Errorf("invalid API key")
	}

	// For API keys, use a generic service account context
	return &UserContext{
		UserID:       fmt.Sprintf("api_key_%s", apiKey[:8]), // Use first 8 chars for identification
		Organization: "api_client",
		Scopes:       []string{"terminology:*"}, // API keys get full access
		IsAnonymous:  false,
		AuthMethod:   "api_key",
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	}, nil
}

// getUserContextFromRequest gets user context from request context
func (am *AuthMiddleware) getUserContextFromRequest(c *gin.Context) *UserContext {
	if userCtx, exists := c.Request.Context().Value("user").(*UserContext); exists {
		return userCtx
	}
	return nil
}

// getRateLimitKey determines the key to use for rate limiting
func (am *AuthMiddleware) getRateLimitKey(userCtx *UserContext, c *gin.Context) string {
	if userCtx.IsAnonymous {
		// Use IP address for anonymous users
		return fmt.Sprintf("ip:%s", c.ClientIP())
	}
	
	// Use user ID for authenticated users
	return fmt.Sprintf("user:%s", userCtx.UserID)
}

// getOperationFromPath extracts operation type from request path
func (am *AuthMiddleware) getOperationFromPath(path string) string {
	// Map paths to operations
	if strings.Contains(path, "/lookup") || strings.Contains(path, "/concepts/") {
		return "lookup"
	}
	if strings.Contains(path, "/search") {
		return "search"
	}
	if strings.Contains(path, "/expand") || strings.Contains(path, "/$expand") {
		return "expand"
	}
	if strings.Contains(path, "/validate") || strings.Contains(path, "/$validate-code") {
		return "validate"
	}
	if strings.Contains(path, "/batch") {
		return "batch"
	}
	
	return "lookup" // default operation
}

// getSystemFromRequest extracts terminology system from request
func (am *AuthMiddleware) getSystemFromRequest(c *gin.Context) string {
	// Try path parameter first
	if system := c.Param("system"); system != "" {
		return system
	}
	
	// Try query parameter
	if system := c.Query("system"); system != "" {
		return system
	}
	
	// Try to extract from request body or other headers
	// This would need to be implemented based on your API design
	
	return ""
}

// isReadOperation checks if an operation is read-only
func (am *AuthMiddleware) isReadOperation(operation string) bool {
	readOperations := map[string]bool{
		"lookup":   true,
		"search":   true,
		"expand":   true,
		"validate": true,
	}
	
	return readOperations[operation]
}

// HealthCheckMiddleware allows health checks to bypass authentication
func (am *AuthMiddleware) HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/readiness" {
			c.Next()
			return
		}
		
		// Continue with normal middleware chain
		c.Next()
	}
}