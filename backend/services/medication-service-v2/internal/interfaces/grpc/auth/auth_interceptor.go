package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor provides JWT authentication for gRPC services
type AuthInterceptor struct {
	secret    string
	publicKey *rsa.PublicKey
	logger    *zap.Logger
	
	// Skip authentication for these methods
	skipAuthMethods map[string]bool
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

// AuthContext holds authentication information in request context
type AuthContext struct {
	UserID    string
	Username  string
	Email     string
	Roles     []string
	Scopes    []string
	Claims    *Claims
}

// ContextKey is the key type for storing auth context
type ContextKey string

const (
	// AuthContextKey is used to store auth context in request context
	AuthContextKey ContextKey = "auth_context"
)

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor(secret string, logger *zap.Logger) *AuthInterceptor {
	// Methods that don't require authentication
	skipMethods := map[string]bool{
		"/medication.v1.MedicationService/HealthCheck": true,
		// Add other public methods as needed
	}

	return &AuthInterceptor{
		secret:          secret,
		logger:          logger,
		skipAuthMethods: skipMethods,
	}
}

// SetPublicKey sets the RSA public key for JWT verification
func (a *AuthInterceptor) SetPublicKey(publicKeyPEM string) error {
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

	a.publicKey = rsaPublicKey
	return nil
}

// UnaryServerInterceptor returns a unary server interceptor for authentication
func (a *AuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method requires authentication
		if a.skipAuthMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract and validate JWT token
		authCtx, err := a.authenticate(ctx)
		if err != nil {
			a.logger.Warn("Authentication failed", 
				zap.String("method", info.FullMethod),
				zap.Error(err))
			return nil, err
		}

		// Add auth context to request context
		newCtx := context.WithValue(ctx, AuthContextKey, authCtx)

		// Log successful authentication
		a.logger.Debug("Authentication successful",
			zap.String("method", info.FullMethod),
			zap.String("user_id", authCtx.UserID),
			zap.String("username", authCtx.Username))

		return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor for authentication
func (a *AuthInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method requires authentication
		if a.skipAuthMethods[info.FullMethod] {
			return handler(srv, stream)
		}

		// Extract and validate JWT token
		authCtx, err := a.authenticate(stream.Context())
		if err != nil {
			a.logger.Warn("Authentication failed for stream", 
				zap.String("method", info.FullMethod),
				zap.Error(err))
			return err
		}

		// Create new stream with auth context
		newCtx := context.WithValue(stream.Context(), AuthContextKey, authCtx)
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          newCtx,
		}

		// Log successful authentication
		a.logger.Debug("Stream authentication successful",
			zap.String("method", info.FullMethod),
			zap.String("user_id", authCtx.UserID),
			zap.String("username", authCtx.Username))

		return handler(srv, wrappedStream)
	}
}

// authenticate extracts and validates JWT token from request context
func (a *AuthInterceptor) authenticate(ctx context.Context) (*AuthContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	// Extract authorization header
	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	// Parse Bearer token
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// Parse and validate JWT token
	token, err := a.parseToken(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
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
func (a *AuthInterceptor) parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			// HMAC signing
			return []byte(a.secret), nil
		case *jwt.SigningMethodRSA:
			// RSA signing
			if a.publicKey == nil {
				return nil, fmt.Errorf("RSA public key not configured")
			}
			return a.publicKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})
}

// GetAuthContext extracts auth context from request context
func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx, ok
}

// RequireRole checks if user has required role
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

// wrappedServerStream wraps grpc.ServerStream to provide custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// Authorization middleware helpers

// RequireRoles creates an interceptor that requires specific roles
func RequireRoles(roles ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		authCtx, ok := GetAuthContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "authentication required")
		}

		if !authCtx.HasAnyRole(roles...) {
			return nil, status.Errorf(codes.PermissionDenied, 
				"insufficient permissions: requires one of %v", roles)
		}

		return handler(ctx, req)
	}
}

// RequireScopes creates an interceptor that requires specific scopes
func RequireScopes(scopes ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		authCtx, ok := GetAuthContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "authentication required")
		}

		for _, scope := range scopes {
			if !authCtx.HasScope(scope) {
				return nil, status.Errorf(codes.PermissionDenied, 
					"insufficient permissions: missing scope %s", scope)
			}
		}

		return handler(ctx, req)
	}
}

// HIPAA audit logging for authentication events
func (a *AuthInterceptor) logHIPAAAuditEvent(ctx context.Context, event string, authCtx *AuthContext) {
	auditData := map[string]interface{}{
		"event_type":    "authentication",
		"event_action":  event,
		"timestamp":     time.Now().UTC(),
		"service":       "medication-service-v2",
		"component":     "grpc-auth",
	}

	if authCtx != nil {
		auditData["user_id"] = authCtx.UserID
		auditData["username"] = authCtx.Username
		auditData["roles"] = authCtx.Roles
	}

	// Extract client information if available
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
			auditData["user_agent"] = userAgent[0]
		}
		if clientIP := md.Get("x-forwarded-for"); len(clientIP) > 0 {
			auditData["client_ip"] = clientIP[0]
		}
	}

	a.logger.Info("HIPAA Audit Event",
		zap.String("audit_type", "authentication"),
		zap.Any("audit_data", auditData))
}