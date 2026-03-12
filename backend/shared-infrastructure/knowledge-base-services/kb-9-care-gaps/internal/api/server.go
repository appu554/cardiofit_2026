// Package api provides HTTP handlers and routing for KB-9 Care Gaps Service.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"kb-9-care-gaps/internal/cache"
	"kb-9-care-gaps/internal/caregaps"
	"kb-9-care-gaps/internal/config"
)

// Server represents the HTTP API server.
type Server struct {
	config          *config.Config
	careGapsService *caregaps.Service
	cache           *cache.Cache
	logger          *zap.Logger
	router          *gin.Engine
	startTime       time.Time
}

// NewServer creates a new API server instance.
func NewServer(cfg *config.Config, careGapsService *caregaps.Service, redisCache *cache.Cache, logger *zap.Logger) *Server {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))
	router.Use(corsMiddleware())

	s := &Server{
		config:          cfg,
		careGapsService: careGapsService,
		cache:           redisCache,
		logger:          logger,
		router:          router,
		startTime:       time.Now(),
	}

	// Setup routes
	s.setupRoutes()

	return s
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.router
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health & monitoring endpoints
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/ready", s.handleReady)
	s.router.GET("/live", s.handleLive)

	// Prometheus metrics
	if s.config.MetricsEnabled {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Care gaps endpoints
		v1.POST("/care-gaps", s.handleGetCareGaps)

		// Measure evaluation endpoints
		v1.POST("/measure/evaluate", s.handleEvaluateMeasure)
		v1.POST("/measure/evaluate-population", s.handleEvaluatePopulation)

		// Measure information endpoints
		v1.GET("/measures", s.handleListMeasures)
		v1.GET("/measures/:type", s.handleGetMeasure)

		// Gap management endpoints
		v1.POST("/gaps/:gapId/addressed", s.handleGapAddressed)
		v1.POST("/gaps/:gapId/dismiss", s.handleDismissGap)
		v1.POST("/gaps/:gapId/snooze", s.handleSnoozeGap)
	}

	// FHIR Operations (Da Vinci DEQM)
	fhir := s.router.Group("/fhir")
	{
		fhir.POST("/Measure/$care-gaps", s.handleFHIRCareGaps)
		fhir.POST("/Measure/:measureId/$evaluate-measure", s.handleFHIREvaluateMeasure)
	}

	// GraphQL endpoint (if enabled)
	if s.config.FederationEnabled {
		s.router.POST("/graphql", s.handleGraphQL)
		if s.config.PlaygroundEnabled {
			s.router.GET("/graphql", s.handleGraphQLPlayground)
		}
	}
}

// requestLogger creates a logging middleware.
func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// Log request details
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		)
	}
}

// corsMiddleware adds CORS headers.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
