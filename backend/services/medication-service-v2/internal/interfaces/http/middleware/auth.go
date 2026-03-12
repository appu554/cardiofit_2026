package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// JWTAuth provides JWT authentication middleware for HTTP requests
type JWTAuth struct {
	secret        string
	publicKey     *rsa.PublicKey
	logger        *zap.Logger
	
	// Skip authentication for these paths
	skipAuthPaths map[string]bool
}

// Claims represents JWT claims structure
type Claims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Scopes    []string `json:"scopes"`
	jwt.RegisteredClaims
}

// AuthContext holds authentication information
type AuthContext struct {
	UserID    string
	Username  string
	Email     string
	Roles     []string
	Scopes    []string
	Claims    *Claims
}

// AuthContextKey is the key for storing auth context in Gin context
const AuthContextKey = "auth_context"

// NewJWTAuth creates a new JWT authentication middleware
func NewJWTAuth(secret string, logger *zap.Logger) *JWTAuth {
	// Paths that don't require authentication
	skipPaths := map[string]bool{
		"/health":       true,
		"/health/ready": true,
		"/health/live":  true,
		"/metrics":      true,
		"/docs":         true,
		"/":             true,
	}

	return &JWTAuth{
		secret:        secret,
		logger:        logger,
		skipAuthPaths: skipPaths,
	}
}

// SetPublicKey sets the RSA public key for JWT verification
func (j *JWTAuth) SetPublicKey(publicKeyPEM string) error {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to parse PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an RSA public key")
	}

	j.publicKey = rsaPublicKey
	return nil
}

// Middleware returns the JWT authentication middleware function
func (j *JWTAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path requires authentication
		if j.skipAuthPaths[c.Request.URL.Path] || strings.HasPrefix(c.Request.URL.Path, "/docs/") {
			c.Next()
			return
		}

		// Extract and validate JWT token
		authCtx, err := j.authenticate(c)
		if err != nil {
			j.logger.Warn("HTTP Authentication failed", 
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
				zap.Error(err))

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_failed",
				"message": err.Error(),
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Store auth context in Gin context
		c.Set(AuthContextKey, authCtx)

		// Log successful authentication
		j.logger.Debug("HTTP Authentication successful",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("user_id", authCtx.UserID),
			zap.String("username", authCtx.Username),
			zap.String("client_ip", c.ClientIP()))

		// Log HIPAA audit event
		j.logHIPAAAuditEvent(c, authCtx)

		c.Next()
	}
}

// authenticate extracts and validates JWT token from HTTP request
func (j *JWTAuth) authenticate(c *gin.Context) (*AuthContext, error) {
	// Extract authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header is required")
	}

	// Parse Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// Parse and validate JWT token
	token, err := j.parseToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Create auth context
	authCtx := &AuthContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
		Scopes:   claims.Scopes,
		Claims:   claims,
	}

	return authCtx, nil
}

// parseToken parses and validates JWT token
func (j *JWTAuth) parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			// HMAC signing
			return []byte(j.secret), nil
		case *jwt.SigningMethodRSA:
			// RSA signing
			if j.publicKey == nil {
				return nil, fmt.Errorf("RSA public key not configured")
			}
			return j.publicKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})
}

// GetAuthContext extracts auth context from Gin context
func GetAuthContext(c *gin.Context) (*AuthContext, bool) {
	value, exists := c.Get(AuthContextKey)
	if !exists {
		return nil, false
	}
	
	authCtx, ok := value.(*AuthContext)
	return authCtx, ok
}

// HasRole checks if user has required role
func (a *AuthContext) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasScope checks if user has required scope
func (a *AuthContext) HasScope(scope string) bool {
	for _, s := range a.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyRole checks if user has any of the required roles
func (a *AuthContext) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if a.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if user has all required roles
func (a *AuthContext) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !a.HasRole(role) {
			return false
		}
	}
	return true
}

// IsExpired checks if token is expired
func (a *AuthContext) IsExpired() bool {
	if a.Claims == nil {
		return true
	}
	return time.Now().After(a.Claims.ExpiresAt.Time)
}

// RequireRole creates middleware that requires specific roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, ok := GetAuthContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "Authentication required",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		if !authCtx.HasAnyRole(roles...) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("Requires one of roles: %v", roles),
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireScopes creates middleware that requires specific scopes
func RequireScopes(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, ok := GetAuthContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "Authentication required",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		for _, scope := range scopes {
			if !authCtx.HasScope(scope) {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "insufficient_permissions",
					"message": fmt.Sprintf("Missing required scope: %s", scope),
					"code":    "FORBIDDEN",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireAllRoles creates middleware that requires all specified roles
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, ok := GetAuthContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "Authentication required",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		if !authCtx.HasAllRoles(roles...) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("Requires all roles: %v", roles),
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// logHIPAAAuditEvent logs HIPAA compliance audit events for HTTP requests
func (j *JWTAuth) logHIPAAAuditEvent(c *gin.Context, authCtx *AuthContext) {
	auditData := map[string]interface{}{
		"event_type":    "http_authentication",
		"event_action":  "access_granted",
		"timestamp":     time.Now().UTC(),
		"service":       "medication-service-v2",
		"component":     "http-auth",
		"method":        c.Request.Method,
		"path":          c.Request.URL.Path,
		"client_ip":     c.ClientIP(),
		"user_agent":    c.Request.UserAgent(),
	}

	if authCtx != nil {
		auditData["user_id"] = authCtx.UserID
		auditData["username"] = authCtx.Username
		auditData["roles"] = authCtx.Roles
		auditData["scopes"] = authCtx.Scopes
	}

	// Extract additional headers for audit
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		auditData["x_forwarded_for"] = forwardedFor
	}
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		auditData["x_real_ip"] = realIP
	}
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		auditData["request_id"] = requestID
	}

	j.logger.Info("HIPAA Audit Event",
		zap.String("audit_type", "access_control"),
		zap.Any("audit_data", auditData))
}

// ValidateTokenMiddleware validates token without requiring authentication
// Useful for optional authentication scenarios
func ValidateTokenMiddleware(secret string, logger *zap.Logger) gin.HandlerFunc {
	auth := NewJWTAuth(secret, logger)
	
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		authCtx, err := auth.authenticate(c)
		if err != nil {
			logger.Debug("Token validation failed", zap.Error(err))
			c.Next()
			return
		}

		c.Set(AuthContextKey, authCtx)
		c.Next()
	}
}

// APIKeyAuth provides API key authentication as an alternative to JWT
type APIKeyAuth struct {
	validAPIKeys map[string]string // key -> user_id
	logger       *zap.Logger
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(apiKeys map[string]string, logger *zap.Logger) *APIKeyAuth {
	return &APIKeyAuth{
		validAPIKeys: apiKeys,
		logger:       logger,
	}
}

// Middleware returns the API key authentication middleware
func (a *APIKeyAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get API key from header or query parameter
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "api_key_required",
				"message": "API key is required",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		userID, valid := a.validAPIKeys[apiKey]
		if !valid {
			a.logger.Warn("Invalid API key used",
				zap.String("client_ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path))

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_api_key",
				"message": "Invalid API key",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Create basic auth context for API key
		authCtx := &AuthContext{
			UserID:   userID,
			Username: "api_user",
			Roles:    []string{"api_user"},
			Scopes:   []string{"read", "write"},
		}

		c.Set(AuthContextKey, authCtx)

		a.logger.Debug("API key authentication successful",
			zap.String("user_id", userID),
			zap.String("client_ip", c.ClientIP()))

		c.Next()
	}
}