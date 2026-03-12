// Package api provides the REST API layer for KB-8 Calculator Service.
//
// The API uses Gin framework with middleware for logging, recovery,
// CORS, and metrics. All endpoints return JSON with proper error codes.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"kb-8-calculator-service/internal/calculator"
	"kb-8-calculator-service/internal/config"
)

// Server represents the API server.
type Server struct {
	cfg     *config.Config
	service *calculator.Service
	logger  *zap.Logger
	router  *gin.Engine
}

// NewServer creates a new API server.
func NewServer(cfg *config.Config, service *calculator.Service, logger *zap.Logger) *Server {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		cfg:     cfg,
		service: service,
		logger:  logger,
		router:  gin.New(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

// setupMiddleware configures middleware for the server.
func (s *Server) setupMiddleware() {
	// Recovery middleware - recovers from panics
	s.router.Use(gin.Recovery())

	// Custom logging middleware using zap
	s.router.Use(s.loggingMiddleware())

	// CORS middleware
	s.router.Use(s.corsMiddleware())

	// Request ID middleware
	s.router.Use(s.requestIDMiddleware())
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health endpoints (no prefix)
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)
	s.router.GET("/live", s.liveHandler)

	// Metrics endpoint (Prometheus)
	if s.cfg.MetricsEnabled {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Calculator info
		v1.GET("/calculators", s.listCalculatorsHandler)

		// Calculator endpoints
		calc := v1.Group("/calculate")
		{
			// P0 Calculators (Critical for dosing)
			calc.POST("/egfr", s.calculateEGFRHandler)
			calc.POST("/crcl", s.calculateCrClHandler)
			calc.POST("/bmi", s.calculateBMIHandler)

			// P1 Calculators (Clinical scores)
			calc.POST("/sofa", s.calculateSOFAHandler)
			calc.POST("/qsofa", s.calculateQSOFAHandler)
			calc.POST("/cha2ds2vasc", s.calculateCHA2DS2VAScHandler)
			calc.POST("/hasbled", s.calculateHASBLEDHandler)
			calc.POST("/ascvd", s.calculateASCVDHandler)

			// Batch calculation
			calc.POST("/batch", s.calculateBatchHandler)
		}
	}

	// Playground (development only)
	if s.cfg.PlaygroundEnabled && s.cfg.IsDevelopment() {
		s.router.GET("/playground", s.playgroundHandler)
	}
}

// loggingMiddleware creates a zap-based logging middleware.
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after request completes
		latency := time.Since(start)
		status := c.Writer.Status()

		// Skip logging health checks at info level
		if path == "/health" || path == "/live" || path == "/ready" {
			s.logger.Debug("request",
				zap.String("path", path),
				zap.Int("status", status),
				zap.Duration("latency", latency),
			)
			return
		}

		s.logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("request_id", c.GetString("request_id")),
		)
	}
}

// corsMiddleware adds CORS headers for development.
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestIDMiddleware adds request ID tracking.
func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID creates a simple request ID.
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}

// APIError represents an API error response.
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// respondError sends an error response.
func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIError{
		Code:      code,
		Message:   message,
		RequestID: c.GetString("request_id"),
	})
}

// respondSuccess sends a success response.
func respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}
