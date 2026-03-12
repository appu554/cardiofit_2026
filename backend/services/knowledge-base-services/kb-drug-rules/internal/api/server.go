package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-drug-rules/internal/cache"
	"kb-drug-rules/internal/conversion"
	"kb-drug-rules/internal/governance"
	"kb-drug-rules/internal/metrics"
	"kb-drug-rules/internal/validation"
)

// Server represents the API server with TOML support
type Server struct {
	db                  *gorm.DB
	cache               cache.KB1CacheInterface
	governance          governance.Engine
	metrics             metrics.Collector
	logger              *logrus.Logger
	validator           *validation.EnhancedTOMLValidator
	converter           *conversion.FormatConverter
	calculationHandlers *CalculationHandlers
	governanceHandlers  *GovernanceHandlers
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	DB         *gorm.DB
	Cache      cache.KB1CacheInterface
	Governance governance.Engine
	Metrics    metrics.Collector
	Logger     *logrus.Logger
}

// NewServer creates a new API server instance with TOML support
func NewServer(config *ServerConfig) *Server {
	server := &Server{
		db:                  config.DB,
		cache:               config.Cache,
		governance:          config.Governance,
		metrics:             config.Metrics,
		logger:              config.Logger,
		calculationHandlers: NewCalculationHandlers(),
		governanceHandlers:  NewGovernanceHandlers(),
	}

	// Initialize TOML components
	server.validator = validation.NewEnhancedTOMLValidator()
	server.converter = conversion.NewFormatConverter()
	// Note: lockManager and diffService would be initialized with proper dependencies
	// For now, we'll handle them in the handlers

	return server
}

// RegisterRoutes registers all API routes
func (s *Server) RegisterRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", s.healthCheck)
	router.GET("/ready", s.readinessCheck)

	// Metrics endpoint
	router.GET("/metrics", s.metricsHandler)

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Drug rules endpoints
		v1.GET("/items/:drug_id", s.getDrugRules)
		v1.POST("/validate", s.validateRules)
		v1.POST("/hotload", s.hotloadRules)
		v1.POST("/promote", s.promoteVersion)

		// TOML Workflow Endpoints (Complete Pipeline)
		v1.POST("/toml/process", s.processTOMLWorkflow)        // Complete TOML workflow
		v1.POST("/toml/validate", s.validateTOMLOnly)          // TOML validation only
		v1.POST("/toml/convert", s.convertTOMLToJSON)          // Format conversion only
		v1.GET("/toml/rules/:drug_id", s.getTOMLRule)          // Get rule in TOML format

		// Enhanced TOML endpoints (from enhanced_handlers.go)
		v1.POST("/validate-toml", s.validateTOMLRules)
		v1.POST("/convert", s.convertFormat)
		v1.POST("/hotload-toml", s.hotloadTOMLRules)
		v1.POST("/batch-load", s.batchLoadRules)
		v1.GET("/versions/:drug_id/history", s.getVersionHistory)
		v1.POST("/rollback", s.rollbackVersion)

		// Governance endpoints
		v1.POST("/governance/submit", s.submitForApproval)
		v1.POST("/governance/review", s.reviewSubmission)
		v1.GET("/governance/status/:ticket_id", s.getApprovalStatus)

		// Management endpoints
		v1.GET("/versions/:drug_id", s.listVersions)
		v1.DELETE("/items/:drug_id/:version", s.deleteVersion)
		v1.GET("/regions", s.listSupportedRegions)
		v1.GET("/stats", s.getServiceStats)

		// ===== DOSE CALCULATION ENDPOINTS (KB-1 README compliance) =====
		// Dose Calculation
		v1.POST("/calculate", s.calculationHandlers.CalculateDose)
		v1.POST("/calculate/weight-based", s.calculationHandlers.CalculateWeightBased)
		v1.POST("/calculate/bsa-based", s.calculationHandlers.CalculateBSABased)
		v1.POST("/calculate/pediatric", s.calculationHandlers.CalculatePediatric)
		v1.POST("/calculate/renal", s.calculationHandlers.CalculateRenalAdjusted)

		// Dose Validation
		v1.POST("/validate/dose", s.calculationHandlers.ValidateDose)
		v1.GET("/validate/max-dose", s.calculationHandlers.GetMaxDose)

		// Patient Parameter Calculations
		v1.POST("/patient/bsa", s.calculationHandlers.CalculateBSAEndpoint)
		v1.POST("/patient/ibw", s.calculationHandlers.CalculateIBWEndpoint)
		v1.POST("/patient/crcl", s.calculationHandlers.CalculateCrClEndpoint)
		v1.POST("/patient/egfr", s.calculationHandlers.CalculateEGFREndpoint)

		// Dosing Rules
		v1.GET("/rules", s.calculationHandlers.ListRules)
		v1.GET("/rules/:rxnorm", s.calculationHandlers.GetRule)
		v1.GET("/rules/search", s.calculationHandlers.SearchRules)

		// ===== GOVERNANCE-ENHANCED ENDPOINTS (Tier-7 Compliance) =====
		// These endpoints wrap standard calculations with governance severity mapping
		// and evidence provenance for legal/regulatory compliance
		govGroup := v1.Group("/governance")
		{
			// Governance-enhanced dose validation with severity mapping
			govGroup.POST("/validate", s.governanceHandlers.GovernanceValidateDose)
			// Governance-enhanced dose calculation with provenance
			govGroup.POST("/calculate", s.governanceHandlers.GovernanceCalculateDose)
			// Get all governance severity levels and actions
			govGroup.GET("/severities", s.governanceHandlers.GetGovernanceSeverities)
			// Get evidence provenance for a specific drug
			govGroup.GET("/provenance/:rxnorm_code", s.governanceHandlers.GetEvidenceProvenance)
		}
	}

	// Admin routes (protected)
	admin := router.Group("/admin")
	admin.Use(s.adminAuthMiddleware())
	{
		admin.POST("/cache/invalidate", s.invalidateCache)
		admin.GET("/cache/stats", s.getCacheStats)
		admin.POST("/governance/override", s.governanceOverride)
		admin.GET("/audit", s.getAuditLog)
	}
}

// Middleware functions

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.WithFields(logrus.Fields{
			"status":     param.StatusCode,
			"method":     param.Method,
			"path":       param.Path,
			"ip":         param.ClientIP,
			"latency":    param.Latency,
			"user_agent": param.Request.UserAgent(),
			"request_id": param.Request.Header.Get("X-Request-ID"),
		}).Info("HTTP Request")
		return ""
	})
}

// MetricsMiddleware collects metrics for HTTP requests
func MetricsMiddleware(metrics metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Record metrics
		duration := time.Since(start)
		metrics.RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, ETag, Cache-Control")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds request ID to context
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement rate limiting logic
		// For now, just pass through
		c.Next()
	}
}

// adminAuthMiddleware validates admin authentication
func (s *Server) adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement admin authentication
		// For now, just check for admin header
		if c.GetHeader("X-Admin-Token") == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Admin authentication required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Helper functions

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production, use UUID or similar
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// respondWithError sends an error response
func (s *Server) respondWithError(c *gin.Context, statusCode int, message string, details map[string]string) {
	requestID, _ := c.Get("request_id")
	
	response := gin.H{
		"error":   http.StatusText(statusCode),
		"message": message,
	}
	
	if requestID != nil {
		response["request_id"] = requestID
	}
	
	if details != nil {
		response["details"] = details
	}
	
	c.JSON(statusCode, response)
}

// respondWithSuccess sends a success response
func (s *Server) respondWithSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// setCacheHeaders sets appropriate cache headers
func (s *Server) setCacheHeaders(c *gin.Context, etag string, maxAge int) {
	c.Header("ETag", etag)
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	
	// Check if client has cached version
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
}

// validateContentType validates request content type
func (s *Server) validateContentType(c *gin.Context, expectedType string) bool {
	contentType := c.GetHeader("Content-Type")
	if contentType != expectedType {
		s.respondWithError(c, http.StatusUnsupportedMediaType, 
			fmt.Sprintf("Expected Content-Type: %s", expectedType), nil)
		return false
	}
	return true
}

// extractUserInfo extracts user information from request
func (s *Server) extractUserInfo(c *gin.Context) map[string]string {
	return map[string]string{
		"user_id":   c.GetHeader("X-User-ID"),
		"user_role": c.GetHeader("X-User-Role"),
		"client_ip": c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}
}
