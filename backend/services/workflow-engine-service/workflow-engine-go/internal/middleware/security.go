package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter implements per-IP rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   *zap.Logger
}

// SecurityMiddleware provides security features
type SecurityMiddleware struct {
	rateLimiter       *RateLimiter
	trustedProxies    []string
	maxRequestSize    int64
	allowedOrigins    []string
	securityHeaders   map[string]string
	logger            *zap.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int, logger *zap.Logger) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
		logger:   logger,
	}
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(logger *zap.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		rateLimiter:    NewRateLimiter(10.0, 20, logger), // 10 requests per second, burst of 20
		trustedProxies: []string{"127.0.0.1", "::1"},
		maxRequestSize: 10 * 1024 * 1024, // 10MB max request size
		allowedOrigins: []string{"*"},     // In production, this should be specific origins
		securityHeaders: map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
			"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		},
		logger: logger,
	}
}

// RateLimitMiddleware implements rate limiting per IP
func (s *SecurityMiddleware) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := s.getClientIP(c)
		
		limiter := s.rateLimiter.getLimiter(ip)
		if !limiter.Allow() {
			s.logger.Warn("Rate limit exceeded",
				zap.String("ip", ip),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			
			c.Header("Retry-After", "60") // Suggest retry after 60 seconds
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func (s *SecurityMiddleware) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		for header, value := range s.securityHeaders {
			c.Header(header, value)
		}

		// Add HSTS header for HTTPS connections
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// RequestSizeLimit middleware limits request body size
func (s *SecurityMiddleware) RequestSizeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > s.maxRequestSize {
			s.logger.Warn("Request too large",
				zap.Int64("content_length", c.Request.ContentLength),
				zap.Int64("max_size", s.maxRequestSize),
				zap.String("ip", s.getClientIP(c)))
			
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request entity too large",
				"code":  "REQUEST_TOO_LARGE",
				"max_size": s.maxRequestSize,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateRequestMiddleware validates request structure and content
func (s *SecurityMiddleware) ValidateRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate Content-Type for POST/PUT requests
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
			contentType := c.GetHeader("Content-Type")
			if contentType == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Content-Type header is required",
					"code":  "MISSING_CONTENT_TYPE",
				})
				c.Abort()
				return
			}

			// Only allow JSON for API endpoints
			if strings.HasPrefix(c.Request.URL.Path, "/api/") && 
			   !strings.Contains(contentType, "application/json") {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error": "Only application/json is supported for API endpoints",
					"code":  "UNSUPPORTED_MEDIA_TYPE",
				})
				c.Abort()
				return
			}
		}

		// Validate critical headers for clinical workflows
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/orchestration/") {
			if c.GetHeader("X-Correlation-ID") == "" && c.Request.Method == http.MethodPost {
				s.logger.Warn("Missing correlation ID for clinical workflow",
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", s.getClientIP(c)))
			}
		}

		c.Next()
	}
}

// IPWhitelistMiddleware restricts access to specific IPs (for admin endpoints)
func (s *SecurityMiddleware) IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := s.getClientIP(c)
		
		// Check if IP is in whitelist
		allowed := false
		for _, allowedIP := range allowedIPs {
			if clientIP == allowedIP {
				allowed = true
				break
			}
		}

		if !allowed {
			s.logger.Warn("IP not whitelisted",
				zap.String("ip", clientIP),
				zap.String("path", c.Request.URL.Path))
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
				"code":  "IP_NOT_ALLOWED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HIPAAComplianceMiddleware ensures HIPAA compliance features
func (s *SecurityMiddleware) HIPAAComplianceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add HIPAA-specific headers
		c.Header("X-HIPAA-Compliant", "true")
		c.Header("X-Data-Classification", "PHI") // Protected Health Information

		// Log access to PHI for audit trail
		if strings.Contains(c.Request.URL.Path, "/patient/") || 
		   strings.Contains(c.Request.URL.Path, "/orchestration/medication") {
			s.logger.Info("PHI access",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("user_agent", c.GetHeader("User-Agent")),
				zap.String("ip", s.getClientIP(c)),
				zap.String("correlation_id", c.GetHeader("X-Correlation-ID")))
		}

		// Ensure secure connection for PHI
		if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") != "https" {
			s.logger.Warn("Insecure connection attempt for PHI",
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", s.getClientIP(c)))
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "HTTPS required for protected health information",
				"code":  "HTTPS_REQUIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper methods

// getLimiter gets or creates a rate limiter for an IP
func (r *RateLimiter) getLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(r.rate, r.burst)
		r.limiters[ip] = limiter
	}

	return limiter
}

// getClientIP extracts the real client IP considering proxies
func (s *SecurityMiddleware) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// CleanupRoutine periodically cleans up old rate limiters
func (r *RateLimiter) CleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		// In a production implementation, you'd track last access time
		// and remove limiters that haven't been used recently
		// For now, just log the cleanup attempt
		r.logger.Debug("Rate limiter cleanup",
			zap.Int("active_limiters", len(r.limiters)))
		r.mu.Unlock()
	}
}

// CustomCORS middleware with HIPAA considerations
func (s *SecurityMiddleware) CustomCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// In production, this should check against a whitelist
		if origin != "" && s.isOriginAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Correlation-ID, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Correlation-ID, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func (s *SecurityMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range s.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// Audit middleware for compliance logging
func (s *SecurityMiddleware) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Store request details for audit log
		auditData := map[string]interface{}{
			"timestamp":      startTime.UTC(),
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"query":          c.Request.URL.RawQuery,
			"ip":             s.getClientIP(c),
			"user_agent":     c.GetHeader("User-Agent"),
			"correlation_id": c.GetHeader("X-Correlation-ID"),
			"content_length": c.Request.ContentLength,
		}

		c.Next()

		// Log audit event
		duration := time.Since(startTime)
		auditData["duration_ms"] = duration.Milliseconds()
		auditData["status_code"] = c.Writer.Status()
		auditData["response_size"] = c.Writer.Size()

		// Add user information if available
		if user, exists := c.Get("user"); exists {
			if userContext, ok := user.(*UserContext); ok {
				auditData["user_id"] = userContext.UserID
				auditData["provider_id"] = userContext.ProviderID
			}
		}

		s.logger.Info("API access audit",
			zap.Any("audit", auditData))
	}
}