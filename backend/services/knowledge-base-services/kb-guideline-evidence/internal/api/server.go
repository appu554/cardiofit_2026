package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"kb-guideline-evidence/internal/cache"
	"kb-guideline-evidence/internal/config"
	"kb-guideline-evidence/internal/database"
	"kb-guideline-evidence/internal/metrics"
)

// Server holds the HTTP server and its dependencies
type Server struct {
	config    *config.Config
	db        *database.Connection
	cache     *cache.CacheClient
	metrics   *metrics.Collector
	router    *gin.Engine
	server    *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *database.Connection, cache *cache.CacheClient) *Server {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create metrics collector
	metricsCollector := metrics.NewCollector()

	server := &Server{
		config:  cfg,
		db:      db,
		cache:   cache,
		metrics: metricsCollector,
		router:  gin.New(),
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logging middleware
	if !s.config.IsProduction() {
		s.router.Use(gin.Logger())
	}

	// CORS middleware
	s.router.Use(s.corsMiddleware())

	// Metrics middleware
	s.router.Use(s.metricsMiddleware())

	// Request ID middleware
	s.router.Use(s.requestIDMiddleware())

	// Authentication middleware for protected routes (if enabled)
	if s.config.RequireApproval {
		// Add JWT auth middleware here
	}
}

// setupRoutes configures all routes for the server
func (s *Server) setupRoutes() {
	// Health checks
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/ready", s.readinessCheck)
	
	// Metrics endpoint for Prometheus
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Guideline endpoints
		guidelines := v1.Group("/guidelines")
		{
			guidelines.GET("", s.listGuidelines)
			guidelines.GET("/:guideline_id", s.getGuideline)
			guidelines.GET("/condition/:condition", s.getGuidelinesByCondition)
			guidelines.GET("/search", s.searchGuidelines)
		}

		// Recommendation endpoints
		recommendations := v1.Group("/recommendations")
		{
			recommendations.GET("/:rec_id", s.getRecommendation)
			recommendations.GET("/domain/:domain", s.getRecommendationsByDomain)
			recommendations.GET("/links", s.getRecommendationsWithLinks)
		}

		// Regional endpoints
		regional := v1.Group("/regional")
		{
			regional.GET("/profiles", s.getRegionalProfiles)
			regional.GET("/profiles/:region", s.getRegionalProfile)
		}

		// Clinical query endpoint
		v1.POST("/clinical-query", s.clinicalQuery)

		// Cross-KB validation endpoints
		validation := v1.Group("/validation")
		{
			validation.GET("/links/:rec_id", s.validateLinks)
			validation.POST("/batch-validate", s.batchValidateLinks)
			validation.GET("/report", s.linkageReport)
		}

		// Admin endpoints (if configured)
		if !s.config.IsProduction() {
			admin := v1.Group("/admin")
			{
				admin.GET("/stats", s.getStats)
				admin.POST("/cache/clear", s.clearCache)
				admin.POST("/cache/warm", s.warmCache)
				admin.GET("/metrics-summary", s.getMetricsSummary)
			}
		}
	}

	// Serve OpenAPI documentation
	s.router.Static("/docs", "./docs")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.config.GetServerAddress(),
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.WriteTimeout) * time.Second,
	}

	log.Printf("Starting KB-3 Guideline Evidence server on %s", s.server.Addr)

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	s.waitForShutdown()

	return nil
}

// waitForShutdown waits for interrupt signal and gracefully shuts down the server
func (s *Server) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}

// Middleware functions

// corsMiddleware configures CORS headers
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.GetCorsOrigins() {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// metricsMiddleware records HTTP request metrics
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// Record metrics
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		s.metrics.RecordRequest(
			c.Request.Method,
			c.FullPath(),
			statusCode,
			duration,
		)

		if responseSize > 0 {
			s.metrics.RecordResponseSize(c.FullPath(), responseSize)
		}
	}
}

// requestIDMiddleware adds a unique request ID to each request
func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// Health check endpoints

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   s.config.KBVersion,
		"service":   "kb3-guideline-evidence",
	}

	// Check database health
	if err := s.db.HealthCheck(); err != nil {
		health["status"] = "unhealthy"
		health["database_error"] = err.Error()
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	// Check cache health
	if err := s.cache.HealthCheck(); err != nil {
		health["status"] = "degraded"
		health["cache_error"] = err.Error()
	}

	statusCode := http.StatusOK
	if health["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if health["status"] == "degraded" {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, health)
}

// readinessCheck returns server readiness status
func (s *Server) readinessCheck(c *gin.Context) {
	ready := map[string]interface{}{
		"ready":     true,
		"timestamp": time.Now().UTC(),
		"checks":    make(map[string]interface{}),
	}

	checks := ready["checks"].(map[string]interface{})

	// Check database connectivity
	if err := s.db.HealthCheck(); err != nil {
		checks["database"] = map[string]interface{}{
			"ready": false,
			"error": err.Error(),
		}
		ready["ready"] = false
	} else {
		checks["database"] = map[string]interface{}{
			"ready": true,
		}
	}

	// Check cache connectivity
	if err := s.cache.HealthCheck(); err != nil {
		checks["cache"] = map[string]interface{}{
			"ready": false,
			"error": err.Error(),
		}
		// Cache is not critical for readiness, so don't mark as not ready
	} else {
		checks["cache"] = map[string]interface{}{
			"ready": true,
		}
	}

	statusCode := http.StatusOK
	if !ready["ready"].(bool) {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, ready)
}

// Admin endpoints

// getStats returns various server statistics
func (s *Server) getStats(c *gin.Context) {
	stats := map[string]interface{}{
		"timestamp":      time.Now().UTC(),
		"database_stats": s.db.GetStats(),
		"cache_stats":    s.cache.GetStats(),
		"config": map[string]interface{}{
			"version":           s.config.KBVersion,
			"default_region":    s.config.DefaultRegion,
			"supported_regions": s.config.SupportedRegions,
			"cache_ttl":         s.config.CacheTTL,
		},
	}

	c.JSON(http.StatusOK, stats)
}

// clearCache clears all cache entries
func (s *Server) clearCache(c *gin.Context) {
	cacheType := c.Query("type")
	
	var err error
	if cacheType == "" {
		// Clear all caches
		patterns := []string{
			cache.GuidelineCacheKeyPrefix + "*",
			cache.RecommendationCacheKeyPrefix + "*",
			cache.SearchCacheKeyPrefix + "*",
			cache.CrossKBCacheKeyPrefix + "*",
		}
		
		for _, pattern := range patterns {
			if e := s.cache.DeletePattern(pattern); e != nil {
				err = e
				break
			}
		}
	} else {
		// Clear specific cache type
		var pattern string
		switch cacheType {
		case "guidelines":
			pattern = cache.GuidelineCacheKeyPrefix + "*"
		case "recommendations":
			pattern = cache.RecommendationCacheKeyPrefix + "*"
		case "search":
			pattern = cache.SearchCacheKeyPrefix + "*"
		case "cross_kb":
			pattern = cache.CrossKBCacheKeyPrefix + "*"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cache type"})
			return
		}
		
		err = s.cache.DeletePattern(pattern)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to clear cache: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Cache cleared successfully",
		"type":      cacheType,
		"timestamp": time.Now().UTC(),
	})
}

// warmCache pre-loads frequently accessed data into cache
func (s *Server) warmCache(c *gin.Context) {
	// Implementation would depend on specific caching strategy
	c.JSON(http.StatusOK, gin.H{
		"message":   "Cache warming started",
		"timestamp": time.Now().UTC(),
	})
}

// getMetricsSummary returns a summary of key metrics
func (s *Server) getMetricsSummary(c *gin.Context) {
	summary := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"cache_performance": map[string]interface{}{
			"guideline_hit_rate":      s.metrics.GetCacheHitRate("guideline"),
			"recommendation_hit_rate": s.metrics.GetCacheHitRate("recommendation"),
			"search_hit_rate":         s.metrics.GetCacheHitRate("search"),
		},
	}

	c.JSON(http.StatusOK, summary)
}

// Utility functions

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("kb3-%d-%d", time.Now().UnixNano(), os.Getpid())
}

// parseIntQuery parses an integer query parameter with default value
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// parseBoolQuery parses a boolean query parameter with default value
func parseBoolQuery(c *gin.Context, key string, defaultValue bool) bool {
	if value := c.Query(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// sendError sends a standardized error response
func (s *Server) sendError(c *gin.Context, statusCode int, message string, code string, details map[string]interface{}) {
	errorResponse := ErrorResponse{
		Error:     message,
		Code:      code,
		Details:   details,
		RequestID: c.GetString("request_id"),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(statusCode, errorResponse)
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	Metadata  interface{} `json:"metadata,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// sendSuccess sends a standardized success response
func (s *Server) sendSuccess(c *gin.Context, data interface{}, metadata interface{}) {
	response := SuccessResponse{
		Data:      data,
		Metadata:  metadata,
		RequestID: c.GetString("request_id"),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}