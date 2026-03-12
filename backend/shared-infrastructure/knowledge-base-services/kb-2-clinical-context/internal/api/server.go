package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"kb-clinical-context/api/handlers"
	"kb-clinical-context/internal/cache"
	"kb-clinical-context/internal/config"
	"kb-clinical-context/internal/database"
	"kb-clinical-context/internal/graphql"
	"kb-clinical-context/internal/metrics"
	"kb-clinical-context/internal/services"
)

type Server struct {
	Router         *gin.Engine
	config         *config.Config
	db             *database.Database
	cache          *cache.MultiTierCache
	metrics        *metrics.Collector
	contextService *services.ContextService
	graphqlHandler *graphql.GraphQLHandler
	logger         *zap.Logger
}

func NewServer(
	cfg *config.Config,
	db *database.Database,
	cache *cache.MultiTierCache,
	metrics *metrics.Collector,
	contextService *services.ContextService,
	logger *zap.Logger,
) *Server {
	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	if cfg.IsDevelopment() {
		router.Use(gin.Logger())
	}

	// Initialize GraphQL handler
	graphqlHandler, err := graphql.NewGraphQLHandler(contextService)
	if err != nil {
		log.Printf("Warning: Failed to initialize GraphQL handler: %v", err)
	}

	server := &Server{
		Router:         router,
		config:         cfg,
		db:             db,
		cache:          cache,
		metrics:        metrics,
		contextService: contextService,
		graphqlHandler: graphqlHandler,
		logger:         logger,
	}

	// Add custom middleware
	server.Router.Use(server.metricsMiddleware())
	server.Router.Use(server.corsMiddleware())
	server.Router.Use(server.errorMiddleware())

	// Setup routes
	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	// Health check and metrics
	s.Router.GET("/health", s.healthCheck)
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// GraphQL Federation endpoint
	if s.graphqlHandler != nil {
		s.Router.POST("/api/federation", s.graphqlHandler.ServeHTTP)
		s.Router.GET("/api/federation", s.graphqlHandler.HandleSDL)
		s.Router.OPTIONS("/api/federation", s.graphqlHandler.ServeHTTP)

		// Also provide GraphQL at /graphql for direct access
		s.Router.POST("/graphql", s.graphqlHandler.ServeHTTP)
		s.Router.GET("/graphql", s.graphqlHandler.HandleIntrospection)
		s.Router.OPTIONS("/graphql", s.graphqlHandler.ServeHTTP)
	}

	// Create handlers
	contextHandlers := NewContextHandlers(s.contextService)
	phenotypeHandlers := handlers.NewPhenotypeHandlers(s.contextService, s.logger)

	// API v1 routes
	v1 := s.Router.Group("/api/v1")
	{
		// Context building endpoints
		context := v1.Group("/context")
		{
			context.POST("/build", contextHandlers.buildContext)
			context.GET("/:patient_id/history", contextHandlers.getContextHistory)
			context.GET("/statistics", contextHandlers.getContextStats)
		}

		// Phenotype endpoints using proper MongoDB handlers
		phenotypes := v1.Group("/phenotypes")
		{
			phenotypes.POST("/detect", contextHandlers.detectPhenotypes)
			phenotypes.GET("/definitions", phenotypeHandlers.GetPhenotypeDefinitions)
			phenotypes.GET("/validate", phenotypeHandlers.ValidatePhenotypes)
			phenotypes.GET("/engine/stats", phenotypeHandlers.GetEngineStats)
			phenotypes.POST("/reload", phenotypeHandlers.ReloadPhenotypes)
			phenotypes.POST("/test", phenotypeHandlers.TestPhenotypeExpression)
			phenotypes.GET("/health", phenotypeHandlers.HealthCheck)
		}

		// Risk assessment endpoints
		risk := v1.Group("/risk")
		{
			risk.POST("/assess", contextHandlers.assessRisk)
		}

		// Care gaps endpoints
		careGaps := v1.Group("/care-gaps")
		{
			careGaps.GET("/:patient_id", contextHandlers.identifyCareGaps)
		}

		// Administrative endpoints
		admin := v1.Group("/admin")
		{
			admin.GET("/health", contextHandlers.getSystemHealth)
			admin.POST("/cache/clear", contextHandlers.clearContextCache)
		}
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "kb-2-clinical-context",
		"version":   "1.0.0",
		"checks":    make(map[string]interface{}),
	}

	checks := health["checks"].(map[string]interface{})

	// Database health check
	if err := s.db.HealthCheck(); err != nil {
		checks["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "unhealthy"
	} else {
		checks["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Cache health check
	if err := s.cache.HealthCheck(); err != nil {
		checks["cache"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "unhealthy"
	} else {
		checks["cache"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Additional service-specific health checks
	checks["mongodb_collections"] = map[string]interface{}{
		"status": "healthy",
		"collections": []string{
			"phenotype_definitions",
			"patient_contexts",
		},
	}

	checks["cache_keys"] = map[string]interface{}{
		"status": "healthy",
		"types": []string{
			"patient_contexts",
			"phenotypes",
			"risk_assessments",
		},
	}

	if health["status"] == "unhealthy" {
		c.JSON(http.StatusServiceUnavailable, health)
	} else {
		c.JSON(http.StatusOK, health)
	}
}

// Middleware functions

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// Record metrics
		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()

		s.metrics.RecordRequest(method, path, status, duration)
		s.metrics.RecordResponseSize(path, c.Writer.Size())
	})
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

func (s *Server) errorMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Log the error
			s.logError(c, err.Err)

			// Return appropriate error response if not already sent
			if c.Writer.Status() == http.StatusOK {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"message": "Internal server error",
						"code":    "INTERNAL_ERROR",
					},
				})
			}
		}
	})
}

func (s *Server) logError(c *gin.Context, err error) {
	method := c.Request.Method
	path := c.Request.URL.Path
	clientIP := c.ClientIP()

	log.Printf("%s ERROR %s %s %s %s",
		time.Now().Format(time.RFC3339),
		method,
		path,
		clientIP,
		err.Error(),
	)
}