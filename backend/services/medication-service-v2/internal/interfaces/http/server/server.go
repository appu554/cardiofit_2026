package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/config"
	"medication-service-v2/internal/infrastructure/monitoring"
	"medication-service-v2/internal/interfaces/http/handlers"
	"medication-service-v2/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer wraps the HTTP server for the medication service
type HTTPServer struct {
	server   *http.Server
	router   *gin.Engine
	config   *config.Config
	services *services.Services
	logger   *zap.Logger
	metrics  *monitoring.Metrics
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	cfg *config.Config,
	services *services.Services,
	logger *zap.Logger,
	metrics *monitoring.Metrics,
) *HTTPServer {
	// Set Gin mode based on configuration
	if !cfg.Logging.Development {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	httpServer := &HTTPServer{
		config:   cfg,
		services: services,
		logger:   logger,
		metrics:  metrics,
		router:   router,
	}

	httpServer.setupMiddleware()
	httpServer.setupRoutes()
	
	httpServer.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Server.HTTP.IdleTimeout,
	}

	return httpServer
}

// setupMiddleware configures all HTTP middleware
func (s *HTTPServer) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logging middleware
	s.router.Use(middleware.RequestLogger(s.logger))

	// Metrics middleware
	s.router.Use(middleware.PrometheusMiddleware(s.metrics))

	// CORS middleware
	corsConfig := cors.Config{
		AllowOrigins:     s.config.Server.HTTP.CORS.AllowedOrigins,
		AllowMethods:     s.config.Server.HTTP.CORS.AllowedMethods,
		AllowHeaders:     s.config.Server.HTTP.CORS.AllowedHeaders,
		AllowCredentials: s.config.Server.HTTP.CORS.AllowCredentials,
		MaxAge:           s.config.Server.HTTP.CORS.MaxAge,
	}
	s.router.Use(cors.New(corsConfig))

	// Authentication middleware (for protected routes)
	authMiddleware := middleware.NewAuthMiddleware(s.logger)

	// Rate limiting middleware
	rateLimiter := middleware.NewRateLimiter(s.config, s.logger)
	s.router.Use(rateLimiter.Middleware())

	// Request timeout middleware
	s.router.Use(middleware.TimeoutMiddleware(30 * time.Second))

	// Clinical audit middleware for HIPAA compliance
	auditMiddleware := middleware.NewAuditMiddleware(s.services.AuditService, s.logger)
	
	// Apply auth and audit to API routes
	apiGroup := s.router.Group("/api/v1")
	apiGroup.Use(authMiddleware.Authenticate())
	apiGroup.Use(auditMiddleware.AuditTrail())
}

// setupRoutes configures all HTTP routes
func (s *HTTPServer) setupRoutes() {
	// Health check routes (no auth required)
	s.setupHealthRoutes()

	// Metrics endpoint (no auth required)
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes (auth required)
	s.setupAPIRoutes()

	// Documentation routes
	s.setupDocsRoutes()
}

// setupGraphQLRoutes sets up GraphQL federation endpoints
func (s *HTTPServer) setupGraphQLRoutes(graphqlServer interface{}) {
	// Note: This method would be called from main.go after GraphQL server is created
	// For now, we'll add a placeholder that can be extended
	s.logger.Info("GraphQL routes setup placeholder")
}

// setupHealthRoutes sets up health check endpoints
func (s *HTTPServer) setupHealthRoutes() {
	healthHandler := handlers.NewHealthHandler(s.services.HealthService, s.logger)
	
	health := s.router.Group("/health")
	{
		health.GET("/live", healthHandler.LivenessCheck)
		health.GET("/ready", healthHandler.ReadinessCheck) 
		health.GET("/deps", healthHandler.DependencyCheck)
		health.GET("/detailed", healthHandler.DetailedHealthCheck)
	}
}

// setupAPIRoutes sets up the main API endpoints
func (s *HTTPServer) setupAPIRoutes() {
	// Medication handlers
	medicationHandler := handlers.NewMedicationHandler(s.services.MedicationService, s.logger)
	recipeHandler := handlers.NewRecipeHandler(s.services.RecipeService, s.logger)
	snapshotHandler := handlers.NewSnapshotHandler(s.services.SnapshotService, s.logger)
	
	// Recipe resolver handler
	recipeResolverHandler := handlers.NewRecipeResolverHandler(
		s.services.RecipeResolverService,
		s.services.RecipeTemplateService,
		s.services.RecipeCacheService,
		s.services.ConditionalRuleEngine,
	)
	
	api := s.router.Group("/api/v1")
	{
		// Medication proposal endpoints
		medications := api.Group("/medications")
		{
			medications.POST("/propose", medicationHandler.ProposeMedication)
			medications.POST("/validate/:proposalId", medicationHandler.ValidateProposal)
			medications.POST("/commit/:proposalId", medicationHandler.CommitProposal)
			medications.GET("/:proposalId", medicationHandler.GetProposal)
			medications.GET("/patient/:patientId", medicationHandler.ListPatientProposals)
			medications.GET("", medicationHandler.SearchProposals)
			medications.GET("/stats", medicationHandler.GetStatistics)
		}
		
		// Recipe endpoints
		recipes := api.Group("/recipes")
		{
			recipes.POST("/:id/resolve", recipeResolverHandler.ResolveRecipe)
			recipes.GET("/:recipeId", recipeHandler.GetRecipe)
			recipes.GET("/protocol/:protocolId", recipeHandler.GetRecipeByProtocol)
			recipes.GET("", recipeHandler.SearchRecipes)
			recipes.POST("", recipeHandler.CreateRecipe)
			recipes.PUT("/:recipeId", recipeHandler.UpdateRecipe)
			recipes.POST("/:recipeId/approve", recipeHandler.ApproveRecipe)
			recipes.DELETE("/:recipeId", recipeHandler.ArchiveRecipe)
		}
		
		// Recipe resolver endpoints
		resolver := api.Group("/resolver")
		{
			resolver.GET("/health", recipeResolverHandler.GetResolverHealth)
			resolver.GET("/protocols", recipeResolverHandler.GetProtocolResolvers)
			resolver.POST("/rules/evaluate", recipeResolverHandler.EvaluateRules)
			
			// Cache management
			cache := resolver.Group("/cache")
			{
				cache.POST("/clear", recipeResolverHandler.ClearCache)
				cache.GET("/statistics", recipeResolverHandler.GetCacheStatistics)
			}
		}
		
		// Clinical snapshot endpoints
		snapshots := api.Group("/snapshots")
		{
			snapshots.POST("", snapshotHandler.CreateSnapshot)
			snapshots.GET("/:snapshotId", snapshotHandler.GetSnapshot)
			snapshots.GET("/patient/:patientId", snapshotHandler.ListPatientSnapshots)
			snapshots.POST("/:snapshotId/validate", snapshotHandler.ValidateSnapshot)
			snapshots.POST("/:snapshotId/supersede", snapshotHandler.SupersedeSnapshot)
			snapshots.GET("", snapshotHandler.SearchSnapshots)
		}
		
		// Analytics endpoints
		analytics := api.Group("/analytics")
		{
			analytics.GET("/proposals", medicationHandler.GetProposalAnalytics)
			analytics.GET("/safety", medicationHandler.GetSafetyAnalytics)
			analytics.GET("/performance", medicationHandler.GetPerformanceMetrics)
			analytics.GET("/quality", snapshotHandler.GetDataQualityMetrics)
		}
		
		// Administrative endpoints
		admin := api.Group("/admin")
		{
			admin.GET("/cache/stats", s.getCacheStats)
			admin.POST("/cache/clear", s.clearCache)
			admin.GET("/config", s.getConfiguration)
			admin.GET("/metrics/internal", s.getInternalMetrics)
		}
	}
}

// setupDocsRoutes sets up documentation endpoints
func (s *HTTPServer) setupDocsRoutes() {
	docs := s.router.Group("/docs")
	{
		docs.GET("/api", s.getAPIDocumentation)
		docs.GET("/openapi.json", s.getOpenAPISpec)
		docs.GET("/health", s.getHealthDocumentation)
	}
}

// Administrative handlers
func (s *HTTPServer) getCacheStats(c *gin.Context) {
	// Implementation would get cache statistics
	c.JSON(http.StatusOK, gin.H{
		"cache_stats": "Implementation needed",
	})
}

func (s *HTTPServer) clearCache(c *gin.Context) {
	// Implementation would clear specified caches
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache clearing functionality needs implementation",
	})
}

func (s *HTTPServer) getConfiguration(c *gin.Context) {
	// Return sanitized configuration (no secrets)
	sanitizedConfig := map[string]interface{}{
		"service": map[string]interface{}{
			"name":    s.config.Service.Name,
			"version": s.config.Service.Version,
			"port":    s.config.Service.Port,
		},
		"performance": map[string]interface{}{
			"max_concurrent_calculations": s.config.Performance.MaxConcurrentCalculations,
			"cache_ttl":                  s.config.Performance.CacheTTL,
			"snapshot_expiry_hours":      s.config.Performance.SnapshotExpiryHours,
		},
		"clinical_engine": map[string]interface{}{
			"timeout":     s.config.ClinicalEngine.Timeout,
			"max_retries": s.config.ClinicalEngine.MaxRetries,
			"performance_targets": s.config.ClinicalEngine.PerformanceTargets,
		},
	}
	
	c.JSON(http.StatusOK, sanitizedConfig)
}

func (s *HTTPServer) getInternalMetrics(c *gin.Context) {
	healthMetrics := s.metrics.GetHealthMetrics()
	c.JSON(http.StatusOK, healthMetrics)
}

// Documentation handlers
func (s *HTTPServer) getAPIDocumentation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name": "Medication Service V2 API",
		"version": s.config.Service.Version,
		"description": "FHIR-compliant medication management service with Recipe & Snapshot architecture",
		"endpoints": map[string]interface{}{
			"medications": "/api/v1/medications",
			"recipes":     "/api/v1/recipes", 
			"snapshots":   "/api/v1/snapshots",
			"analytics":   "/api/v1/analytics",
			"health":      "/health",
			"metrics":     "/metrics",
		},
		"authentication": "Bearer token required for /api/v1/* endpoints",
		"rate_limits": map[string]string{
			"default": "100 requests per minute",
			"clinical_calculations": "50 requests per minute",
		},
	})
}

func (s *HTTPServer) getOpenAPISpec(c *gin.Context) {
	// This would return the OpenAPI 3.0 specification
	c.JSON(http.StatusOK, gin.H{
		"openapi": "3.0.0",
		"info": gin.H{
			"title":   "Medication Service V2 API",
			"version": s.config.Service.Version,
		},
		"message": "Full OpenAPI specification needs implementation",
	})
}

func (s *HTTPServer) getHealthDocumentation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"health_endpoints": map[string]string{
			"/health/live":     "Liveness probe - returns 200 if service is running",
			"/health/ready":    "Readiness probe - returns 200 if service is ready to accept requests",
			"/health/deps":     "Dependency health check - status of external dependencies",
			"/health/detailed": "Detailed health information including metrics and diagnostics",
		},
		"monitoring": map[string]string{
			"/metrics": "Prometheus metrics endpoint",
		},
	})
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.String("address", s.server.Addr),
		zap.String("service", s.config.Service.Name),
		zap.String("version", s.config.Service.Version),
	)
	
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	
	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}
	
	s.logger.Info("HTTP server shutdown complete")
	return nil
}

// GetServer returns the underlying HTTP server (for testing)
func (s *HTTPServer) GetServer() *http.Server {
	return s.server
}

// GetRouter returns the underlying Gin router (for testing)
func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}