package security

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// SecurityConfig holds all security-related configuration
type SecurityConfig struct {
	Authentication *AuthConfig   `json:"authentication"`
	Audit          *AuditConfig  `json:"audit"`
	RateLimit      *RateLimitConfig `json:"rate_limit"`
	License        *LicenseConfig `json:"license"`
	Security       *GeneralSecurityConfig `json:"security"`
}

// LicenseConfig holds license enforcement configuration
type LicenseConfig struct {
	JWTSecret              string        `json:"jwt_secret"`
	DefaultTokenExpiry     time.Duration `json:"default_token_expiry"`
	EnableLicenseCheck     bool          `json:"enable_license_check"`
	StrictMode             bool          `json:"strict_mode"`
	GracePeriod            time.Duration `json:"grace_period"`
	CacheExpiryMinutes     int           `json:"cache_expiry_minutes"`
}

// GeneralSecurityConfig holds general security settings
type GeneralSecurityConfig struct {
	EnableCSRFProtection   bool     `json:"enable_csrf_protection"`
	EnableCORSProtection   bool     `json:"enable_cors_protection"`
	AllowedOrigins         []string `json:"allowed_origins"`
	EnableHTTPS            bool     `json:"enable_https"`
	RequireHTTPS           bool     `json:"require_https"`
	MaxRequestSize         int64    `json:"max_request_size"`
	EnableRequestLogging   bool     `json:"enable_request_logging"`
	SessionTimeout         time.Duration `json:"session_timeout"`
	EnableIPWhitelist      bool     `json:"enable_ip_whitelist"`
	WhitelistedIPs         []string `json:"whitelisted_ips"`
	BlockSuspiciousRequests bool    `json:"block_suspicious_requests"`
	EnableGeoBlocking      bool     `json:"enable_geo_blocking"`
	BlockedCountries       []string `json:"blocked_countries"`
}

// SecurityManager coordinates all security components
type SecurityManager struct {
	licenseEnforcer *LicenseEnforcer
	rateLimiter     *RateLimiter  
	auditLogger     *AuditLogger
	authMiddleware  *AuthMiddleware
	config          *SecurityConfig
	logger          *zap.Logger
	db              *sql.DB
	redis           *redis.Client
	
	// Security metrics
	securityMetrics *SecurityMetrics
	mu              sync.RWMutex
}

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	AuthenticationAttempts   int64 `json:"authentication_attempts"`
	FailedAuthentications   int64 `json:"failed_authentications"`
	RateLimitViolations     int64 `json:"rate_limit_violations"`
	LicenseViolations       int64 `json:"license_violations"`
	BlockedRequests         int64 `json:"blocked_requests"`
	HighRiskEvents          int64 `json:"high_risk_events"`
	ActiveSessions          int64 `json:"active_sessions"`
	SuspiciousActivityCount int64 `json:"suspicious_activity_count"`
	LastSecurityIncident    *time.Time `json:"last_security_incident,omitempty"`
	TotalAuditEvents        int64 `json:"total_audit_events"`
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(db *sql.DB, redis *redis.Client, logger *zap.Logger, config *SecurityConfig) (*SecurityManager, error) {
	if config == nil {
		config = getDefaultSecurityConfig()
	}

	// Create license enforcer
	licenseEnforcer := NewLicenseEnforcer(db, logger, config.License.JWTSecret)

	// Create rate limiter
	rateLimiter := NewRateLimiter(redis, logger)

	// Create audit logger
	auditLogger := NewAuditLogger(db, logger, config.Audit)

	// Create auth middleware
	authMiddleware := NewAuthMiddleware(licenseEnforcer, rateLimiter, logger, config.Authentication)

	sm := &SecurityManager{
		licenseEnforcer: licenseEnforcer,
		rateLimiter:     rateLimiter,
		auditLogger:     auditLogger,
		authMiddleware:  authMiddleware,
		config:          config,
		logger:          logger,
		db:              db,
		redis:           redis,
		securityMetrics: &SecurityMetrics{},
	}

	// Initialize security monitoring
	sm.startSecurityMonitoring()

	return sm, nil
}

// GetMiddlewareChain returns the complete security middleware chain
func (sm *SecurityManager) GetMiddlewareChain() []gin.HandlerFunc {
	middlewares := []gin.HandlerFunc{
		// Security headers
		sm.securityHeadersMiddleware(),
		
		// Health check bypass
		sm.authMiddleware.HealthCheckMiddleware(),
		
		// CORS protection
		sm.corsMiddleware(),
		
		// Request size limiting
		sm.requestSizeLimiter(),
		
		// IP filtering
		sm.ipFilteringMiddleware(),
		
		// Audit logging (should be early in chain)
		sm.authMiddleware.AuditLoggingMiddleware(),
		
		// Authentication
		sm.authMiddleware.AuthenticationMiddleware(),
		
		// Rate limiting
		sm.authMiddleware.RateLimitingMiddleware(),
		
		// Authorization
		sm.authMiddleware.AuthorizationMiddleware(),
		
		// Request validation
		sm.requestValidationMiddleware(),
	}

	return middlewares
}

// securityHeadersMiddleware adds security headers
func (sm *SecurityManager) securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		if sm.config.Security.RequireHTTPS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		// Remove server identification
		c.Header("Server", "")
		
		c.Next()
	}
}

// corsMiddleware handles CORS protection
func (sm *SecurityManager) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.Security.EnableCORSProtection {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range sm.config.Security.AllowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.AbortWithStatus(200)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}

		if !allowed && origin != "" {
			sm.auditLogger.LogSecurityError(c.Request.Context(), "", "cors_violation", 
				"CORS policy violation", map[string]interface{}{
					"origin": origin,
					"allowed_origins": sm.config.Security.AllowedOrigins,
				})
			c.AbortWithStatus(403)
			return
		}

		c.Next()
	}
}

// requestSizeLimiter limits request body size
func (sm *SecurityManager) requestSizeLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sm.config.Security.MaxRequestSize > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, sm.config.Security.MaxRequestSize)
		}
		c.Next()
	}
}

// ipFilteringMiddleware filters requests by IP
func (sm *SecurityManager) ipFilteringMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// IP whitelist check
		if sm.config.Security.EnableIPWhitelist {
			allowed := false
			for _, allowedIP := range sm.config.Security.WhitelistedIPs {
				if clientIP == allowedIP {
					allowed = true
					break
				}
			}
			
			if !allowed {
				sm.auditLogger.LogSecurityError(c.Request.Context(), "", "ip_blocked",
					"Request from non-whitelisted IP", map[string]interface{}{
						"client_ip": clientIP,
						"whitelisted_ips": sm.config.Security.WhitelistedIPs,
					})
				c.AbortWithStatus(403)
				return
			}
		}

		// Check for suspicious patterns
		if sm.config.Security.BlockSuspiciousRequests {
			if sm.isSuspiciousRequest(c) {
				sm.incrementSecurityMetric("suspicious_activity_count")
				sm.auditLogger.LogSecurityError(c.Request.Context(), "", "suspicious_request",
					"Suspicious request pattern detected", map[string]interface{}{
						"client_ip": clientIP,
						"user_agent": c.Request.UserAgent(),
						"path": c.Request.URL.Path,
					})
				c.AbortWithStatus(403)
				return
			}
		}

		c.Next()
	}
}

// requestValidationMiddleware validates incoming requests
func (sm *SecurityManager) requestValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate request method
		allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		methodAllowed := false
		for _, method := range allowedMethods {
			if c.Request.Method == method {
				methodAllowed = true
				break
			}
		}

		if !methodAllowed {
			sm.auditLogger.LogSecurityError(c.Request.Context(), "", "invalid_method",
				"Invalid HTTP method", map[string]interface{}{
					"method": c.Request.Method,
					"allowed_methods": allowedMethods,
				})
			c.AbortWithStatus(405)
			return
		}

		// Additional request validation can be added here
		
		c.Next()
	}
}

// isSuspiciousRequest checks if a request exhibits suspicious patterns
func (sm *SecurityManager) isSuspiciousRequest(c *gin.Context) bool {
	userAgent := c.Request.UserAgent()
	path := c.Request.URL.Path
	
	// Check for common attack patterns
	suspiciousPatterns := []string{
		"sqlmap", "nikto", "nmap", "dirb", "gobuster", "ffuf",
		"../", "../../", "<script", "javascript:", "eval(",
		"union select", "drop table", "insert into", "delete from",
		"<iframe", "onerror=", "onload=", "onclick=",
	}

	// Check user agent
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(userAgent), pattern) {
			return true
		}
	}

	// Check path
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(path), pattern) {
			return true
		}
	}

	// Check for empty user agent (often indicates bot/script)
	if userAgent == "" {
		return true
	}

	return false
}

// startSecurityMonitoring starts background security monitoring
func (sm *SecurityManager) startSecurityMonitoring() {
	go sm.monitorSecurityMetrics()
	go sm.cleanupExpiredData()
	go sm.detectAnomalies()
}

// monitorSecurityMetrics periodically updates security metrics
func (sm *SecurityManager) monitorSecurityMetrics() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.updateSecurityMetrics()
		}
	}
}

// updateSecurityMetrics updates security metrics from various sources
func (sm *SecurityManager) updateSecurityMetrics() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	ctx := context.Background()

	// Update authentication metrics
	authQuery := `
		SELECT 
			COUNT(*) as total_attempts,
			COUNT(CASE WHEN result = 'failure' THEN 1 END) as failed_attempts
		FROM audit_events 
		WHERE event_type = 'authentication' 
		AND timestamp > NOW() - INTERVAL '1 hour'
	`
	
	var totalAttempts, failedAttempts int64
	sm.db.QueryRowContext(ctx, authQuery).Scan(&totalAttempts, &failedAttempts)
	
	sm.securityMetrics.AuthenticationAttempts = totalAttempts
	sm.securityMetrics.FailedAuthentications = failedAttempts

	// Update rate limit violations
	rateLimitQuery := `
		SELECT COUNT(*) 
		FROM audit_events 
		WHERE event_type = 'rate_limit' 
		AND timestamp > NOW() - INTERVAL '1 hour'
	`
	sm.db.QueryRowContext(ctx, rateLimitQuery).Scan(&sm.securityMetrics.RateLimitViolations)

	// Update high risk events
	highRiskQuery := `
		SELECT COUNT(*) 
		FROM audit_events 
		WHERE risk_score >= 70
		AND timestamp > NOW() - INTERVAL '1 hour'
	`
	sm.db.QueryRowContext(ctx, highRiskQuery).Scan(&sm.securityMetrics.HighRiskEvents)

	// Log metrics
	sm.logger.Info("Security metrics updated",
		zap.Int64("auth_attempts", sm.securityMetrics.AuthenticationAttempts),
		zap.Int64("failed_auth", sm.securityMetrics.FailedAuthentications),
		zap.Int64("rate_limit_violations", sm.securityMetrics.RateLimitViolations),
		zap.Int64("high_risk_events", sm.securityMetrics.HighRiskEvents))
}

// cleanupExpiredData cleans up expired security data
func (sm *SecurityManager) cleanupExpiredData() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.performCleanup()
		}
	}
}

// performCleanup removes expired audit logs and cached data
func (sm *SecurityManager) performCleanup() {
	ctx := context.Background()
	retentionPeriod := sm.config.Audit.RetentionPeriod

	// Clean up old audit events
	deleteQuery := `
		DELETE FROM audit_events 
		WHERE timestamp < NOW() - INTERVAL '%d hours'
	`
	
	result, err := sm.db.ExecContext(ctx, fmt.Sprintf(deleteQuery, int(retentionPeriod.Hours())))
	if err != nil {
		sm.logger.Error("Failed to cleanup audit events", zap.Error(err))
	} else {
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			sm.logger.Info("Cleaned up expired audit events", 
				zap.Int64("rows_deleted", rowsAffected))
		}
	}

	// Clear license and user caches
	sm.licenseEnforcer.ClearCache()
}

// detectAnomalies monitors for security anomalies
func (sm *SecurityManager) detectAnomalies() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.checkForAnomalies()
		}
	}
}

// checkForAnomalies checks for security anomalies
func (sm *SecurityManager) checkForAnomalies() {
	ctx := context.Background()

	// Check for unusual authentication failures
	failureThreshold := int64(50) // 50 failures in 5 minutes
	if sm.securityMetrics.FailedAuthentications > failureThreshold {
		sm.auditLogger.LogSecurityError(ctx, "system", "anomaly_detected",
			"Unusual number of authentication failures detected",
			map[string]interface{}{
				"failure_count": sm.securityMetrics.FailedAuthentications,
				"threshold": failureThreshold,
				"time_window": "5 minutes",
			})
	}

	// Check for high number of rate limit violations
	rateLimitThreshold := int64(100)
	if sm.securityMetrics.RateLimitViolations > rateLimitThreshold {
		sm.auditLogger.LogSecurityError(ctx, "system", "anomaly_detected",
			"Unusual number of rate limit violations",
			map[string]interface{}{
				"violation_count": sm.securityMetrics.RateLimitViolations,
				"threshold": rateLimitThreshold,
			})
	}
}

// GetSecurityStatus returns current security status
func (sm *SecurityManager) GetSecurityStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"security_enabled": true,
		"authentication_required": sm.config.Authentication.RequireAuth,
		"rate_limiting_enabled": true,
		"audit_logging_enabled": sm.config.Audit.EnableAuditLogging,
		"license_enforcement_enabled": sm.config.License.EnableLicenseCheck,
		"metrics": sm.securityMetrics,
		"last_updated": time.Now(),
	}
}

// incrementSecurityMetric safely increments a security metric
func (sm *SecurityManager) incrementSecurityMetric(metric string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	switch metric {
	case "authentication_attempts":
		sm.securityMetrics.AuthenticationAttempts++
	case "failed_authentications":
		sm.securityMetrics.FailedAuthentications++
	case "rate_limit_violations":
		sm.securityMetrics.RateLimitViolations++
	case "license_violations":
		sm.securityMetrics.LicenseViolations++
	case "blocked_requests":
		sm.securityMetrics.BlockedRequests++
	case "high_risk_events":
		sm.securityMetrics.HighRiskEvents++
	case "suspicious_activity_count":
		sm.securityMetrics.SuspiciousActivityCount++
	}
}

// getDefaultSecurityConfig returns default security configuration
func getDefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Authentication: &AuthConfig{
			RequireAuth:         true,
			TokenExpiry:         time.Hour * 24,
			RefreshTokenExpiry:  time.Hour * 24 * 7,
			AllowAnonymousRead:  false,
		},
		Audit: &AuditConfig{
			EnableAuditLogging:   true,
			LogLevel:             SeverityInfo,
			RetentionPeriod:      365 * 24 * time.Hour,
			EnableRealTimeAlerts: true,
			HighRiskThreshold:    80,
			EnableComplianceMode: true,
		},
		License: &LicenseConfig{
			EnableLicenseCheck:  true,
			StrictMode:          true,
			GracePeriod:         time.Hour * 24,
			CacheExpiryMinutes:  60,
		},
		Security: &GeneralSecurityConfig{
			EnableCSRFProtection:    true,
			EnableCORSProtection:    true,
			AllowedOrigins:          []string{"*"},
			EnableHTTPS:             true,
			RequireHTTPS:            false,
			MaxRequestSize:          10 * 1024 * 1024, // 10MB
			EnableRequestLogging:    true,
			SessionTimeout:          time.Hour * 8,
			BlockSuspiciousRequests: true,
		},
	}
}