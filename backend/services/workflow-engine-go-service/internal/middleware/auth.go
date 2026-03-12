package middleware

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// AuthMiddleware handles JWT token validation and user context
type AuthMiddleware struct {
	publicKey    *rsa.PublicKey
	authService  string
	logger       *zap.Logger
	skipPaths    map[string]bool
	requiredRole string
}

// Claims represents JWT claims structure
type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ProviderID  string   `json:"provider_id,omitempty"`
	Specialty   string   `json:"specialty,omitempty"`
	jwt.RegisteredClaims
}

// UserContext contains user information for request context
type UserContext struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ProviderID  string   `json:"provider_id,omitempty"`
	Specialty   string   `json:"specialty,omitempty"`
	IsProvider  bool     `json:"is_provider"`
	IsAdmin     bool     `json:"is_admin"`
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authServiceURL string, publicKey *rsa.PublicKey, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		publicKey:   publicKey,
		authService: authServiceURL,
		logger:      logger,
		skipPaths: map[string]bool{
			"/health":     true,
			"/metrics":    true,
			"/playground": true,
			"/graphql":    false, // GraphQL needs auth
		},
		requiredRole: "provider", // Default required role for clinical workflows
	}
}

// ValidateToken middleware validates JWT tokens
func (a *AuthMiddleware) ValidateToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if a.skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			a.logger.Warn("Missing authorization header",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("remote_addr", c.ClientIP()))
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
				"code":  "AUTH_MISSING",
			})
			c.Abort()
			return
		}

		// Parse Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			a.logger.Warn("Invalid authorization header format",
				zap.String("path", c.Request.URL.Path))
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "AUTH_INVALID_FORMAT",
			})
			c.Abort()
			return
		}

		// Validate and parse JWT token
		userContext, err := a.validateJWT(tokenString)
		if err != nil {
			a.logger.Warn("Token validation failed",
				zap.String("path", c.Request.URL.Path),
				zap.Error(err))
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "AUTH_INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		// Check if user has required role for clinical workflows
		if !a.hasRequiredRole(userContext) {
			a.logger.Warn("Insufficient permissions",
				zap.String("user_id", userContext.UserID),
				zap.Strings("user_roles", userContext.Roles),
				zap.String("required_role", a.requiredRole))
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions for clinical workflow operations",
				"code":  "AUTH_INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		// Add user context to request
		c.Set("user", userContext)
		c.Set("user_id", userContext.UserID)
		c.Set("provider_id", userContext.ProviderID)

		// Add user context to request context for downstream services
		ctx := context.WithValue(c.Request.Context(), "user", userContext)
		c.Request = c.Request.WithContext(ctx)

		a.logger.Info("Authentication successful",
			zap.String("user_id", userContext.UserID),
			zap.String("provider_id", userContext.ProviderID),
			zap.Strings("roles", userContext.Roles),
			zap.String("path", c.Request.URL.Path))

		c.Next()
	}
}

// RequireRole creates middleware that requires specific roles
func (a *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userContext, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		user := userContext.(*UserContext)
		
		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range user.Roles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			a.logger.Warn("Role check failed",
				zap.String("user_id", user.UserID),
				zap.Strings("user_roles", user.Roles),
				zap.Strings("required_roles", roles))
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient role permissions",
				"code":  "AUTH_INSUFFICIENT_ROLE",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission creates middleware that requires specific permissions
func (a *AuthMiddleware) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userContext, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		user := userContext.(*UserContext)
		
		// Check if user has any of the required permissions
		hasPermission := false
		for _, requiredPerm := range permissions {
			for _, userPerm := range user.Permissions {
				if userPerm == requiredPerm {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			a.logger.Warn("Permission check failed",
				zap.String("user_id", user.UserID),
				zap.Strings("user_permissions", user.Permissions),
				zap.Strings("required_permissions", permissions))
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "AUTH_INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// validateJWT validates and parses a JWT token
func (a *AuthMiddleware) validateJWT(tokenString string) (*UserContext, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	// Validate required claims
	if claims.UserID == "" {
		return nil, errors.New("missing user_id in token")
	}

	// Build user context
	userContext := &UserContext{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		ProviderID:  claims.ProviderID,
		Specialty:   claims.Specialty,
		IsProvider:  a.hasRole(claims.Roles, "provider"),
		IsAdmin:     a.hasRole(claims.Roles, "admin"),
	}

	return userContext, nil
}

// hasRequiredRole checks if user has the required role for the service
func (a *AuthMiddleware) hasRequiredRole(user *UserContext) bool {
	// Admins always have access
	if user.IsAdmin {
		return true
	}

	// Check for required role
	return a.hasRole(user.Roles, a.requiredRole)
}

// hasRole checks if user has a specific role
func (a *AuthMiddleware) hasRole(roles []string, targetRole string) bool {
	for _, role := range roles {
		if role == targetRole {
			return true
		}
	}
	return false
}

// GetUserFromContext extracts user context from Gin context
func GetUserFromContext(c *gin.Context) (*UserContext, error) {
	userContext, exists := c.Get("user")
	if !exists {
		return nil, errors.New("user context not found")
	}

	user, ok := userContext.(*UserContext)
	if !ok {
		return nil, errors.New("invalid user context type")
	}

	return user, nil
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(c *gin.Context) (string, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return "", err
	}
	return user.UserID, nil
}

// GetProviderIDFromContext extracts provider ID from context
func GetProviderIDFromContext(c *gin.Context) (string, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return "", err
	}
	
	if user.ProviderID == "" {
		return "", errors.New("user is not a healthcare provider")
	}
	
	return user.ProviderID, nil
}

// IsProviderRequest checks if the request is from a healthcare provider
func IsProviderRequest(c *gin.Context) bool {
	user, err := GetUserFromContext(c)
	if err != nil {
		return false
	}
	return user.IsProvider
}

// IsAdminRequest checks if the request is from an administrator
func IsAdminRequest(c *gin.Context) bool {
	user, err := GetUserFromContext(c)
	if err != nil {
		return false
	}
	return user.IsAdmin
}