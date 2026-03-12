package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/interfaces/http/handlers"
	"medication-service-v2/internal/interfaces/http/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP REST API server
type Server struct {
	router     *gin.Engine
	logger     *zap.Logger
	services   *services.Services
	config     ServerConfig
	httpServer *http.Server
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port                int           `mapstructure:"port"`
	Host                string        `mapstructure:"host"`
	ReadTimeout         time.Duration `mapstructure:"read_timeout"`
	WriteTimeout        time.Duration `mapstructure:"write_timeout"`
	IdleTimeout         time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes      int           `mapstructure:"max_header_bytes"`
	EnableCORS          bool          `mapstructure:"enable_cors"`
	AllowedOrigins      []string      `mapstructure:"allowed_origins"`
	AllowedMethods      []string      `mapstructure:"allowed_methods"`
	AllowedHeaders      []string      `mapstructure:"allowed_headers"`
	EnableAuth          bool          `mapstructure:"enable_auth"`
	AuthSecret          string        `mapstructure:"auth_secret"`
	EnableRateLimit     bool          `mapstructure:"enable_rate_limit"`
	RateLimitRPS        float64       `mapstructure:"rate_limit_rps"`
	RateLimitBurst      int           `mapstructure:"rate_limit_burst"`
	EnableMetrics       bool          `mapstructure:"enable_metrics"`
	EnablePprof         bool          `mapstructure:"enable_pprof"`
	TrustedProxies      []string      `mapstructure:"trusted_proxies"`
}

// DefaultServerConfig returns default HTTP server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:           8080,
		Host:           "0.0.0.0",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
		EnableCORS:     true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		EnableAuth:     true,
		EnableRateLimit: true,
		RateLimitRPS:   1000.0, // 1000 requests per second
		RateLimitBurst: 100,    // Burst of 100
		EnableMetrics:  true,
		EnablePprof:    false, // Disabled by default for security
		TrustedProxies: []string{},
	}
}

// NewServer creates a new HTTP server instance
func NewServer(
	logger *zap.Logger,
	services *services.Services,
	config ServerConfig,
) *Server {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	return &Server{
		logger:   logger,
		services: services,
		config:   config,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.setupRouter()

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	s.logger.Info("Starting HTTP server", 
		zap.String("address", addr),
		zap.Bool("cors_enabled", s.config.EnableCORS),
		zap.Bool("auth_enabled", s.config.EnableAuth),
		zap.Bool("rate_limit_enabled", s.config.EnableRateLimit),
		zap.Bool("metrics_enabled", s.config.EnableMetrics))

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		s.logger.Info("Shutting down HTTP server")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// setupRouter configures the Gin router with all routes and middleware
func (s *Server) setupRouter() {
	s.router = gin.New()

	// Set trusted proxies
	if len(s.config.TrustedProxies) > 0 {
		s.router.SetTrustedProxies(s.config.TrustedProxies)
	}

	// Global middleware
	s.setupGlobalMiddleware()

	// Setup routes
	s.setupRoutes()
}

// setupGlobalMiddleware configures global middleware
func (s *Server) setupGlobalMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(middleware.Logger(s.logger))

	// CORS middleware
	if s.config.EnableCORS {
		corsConfig := cors.Config{
			AllowOrigins:     s.config.AllowedOrigins,
			AllowMethods:     s.config.AllowedMethods,
			AllowHeaders:     s.config.AllowedHeaders,
			ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-Total-Count"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}
		s.router.Use(cors.New(corsConfig))
	}

	// Request ID middleware
	s.router.Use(middleware.RequestID())

	// Rate limiting middleware
	if s.config.EnableRateLimit {
		rateLimiter := middleware.NewRateLimiter(s.config.RateLimitRPS, s.config.RateLimitBurst)
		s.router.Use(rateLimiter.Middleware())
	}

	// Metrics middleware
	if s.config.EnableMetrics {
		metricsMiddleware := middleware.NewMetrics("medication_service_v2")
		s.router.Use(metricsMiddleware.Middleware())
	}

	// Security headers middleware
	s.router.Use(middleware.SecurityHeaders())

	// HIPAA audit middleware
	s.router.Use(middleware.HIPAAAudit(s.logger))
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check routes (no auth required)
	s.router.GET("/health", handlers.NewHealthHandler(s.services.HealthService, s.logger).HealthCheck)
	s.router.GET("/health/ready", handlers.NewHealthHandler(s.services.HealthService, s.logger).ReadinessCheck)
	s.router.GET("/health/live", handlers.NewHealthHandler(s.services.HealthService, s.logger).LivenessCheck)

	// Metrics endpoint (if enabled)
	if s.config.EnableMetrics {
		s.router.GET("/metrics", handlers.NewMetricsHandler(s.logger).GetMetrics)
	}

	// API version 1 routes
	v1 := s.router.Group("/api/v1")
	
	// Apply authentication middleware to API routes
	if s.config.EnableAuth {
		authMiddleware := middleware.NewJWTAuth(s.config.AuthSecret, s.logger)
		v1.Use(authMiddleware.Middleware())
	}

	s.setupV1Routes(v1)

	// FHIR R4 routes (separate group for FHIR compliance)
	fhir := s.router.Group("/fhir/r4")
	if s.config.EnableAuth {
		authMiddleware := middleware.NewJWTAuth(s.config.AuthSecret, s.logger)
		fhir.Use(authMiddleware.Middleware())
	}
	
	s.setupFHIRRoutes(fhir)

	// Documentation routes
	s.router.Static("/docs", "./docs")
	s.router.GET("/", s.redirectToSwagger)
}

// setupV1Routes configures API v1 routes
func (s *Server) setupV1Routes(v1 *gin.RouterGroup) {
	// Create handlers
	medicationHandler := handlers.NewMedicationProposalHandler(s.services.MedicationService, s.logger)
	recipeHandler := handlers.NewRecipeHandler(s.services.RecipeResolverIntegration, s.logger)
	snapshotHandler := handlers.NewSnapshotHandler(s.services.SnapshotService, s.logger)
	workflowHandler := handlers.NewWorkflowHandler(s.services.WorkflowOrchestratorService, s.logger)
	clinicalEngineHandler := handlers.NewClinicalEngineHandler(s.services.ClinicalEngineService, s.logger)
	knowledgeBaseHandler := handlers.NewKnowledgeBaseHandler(s.services, s.logger)

	// Medication Proposal Management Routes
	medicationRoutes := v1.Group("/medication-proposals")
	{
		medicationRoutes.POST("", medicationHandler.CreateProposal)
		medicationRoutes.GET("/:id", medicationHandler.GetProposal)
		medicationRoutes.PUT("/:id", medicationHandler.UpdateProposal)
		medicationRoutes.DELETE("/:id", medicationHandler.DeleteProposal)
		medicationRoutes.GET("", medicationHandler.ListProposals)
		medicationRoutes.POST("/:id/validate", medicationHandler.ValidateProposal)
		medicationRoutes.POST("/:id/commit", medicationHandler.CommitProposal)
		medicationRoutes.GET("/:id/history", medicationHandler.GetProposalHistory)
	}

	// Recipe Resolver Routes
	recipeRoutes := v1.Group("/recipes")
	{
		recipeRoutes.POST("/resolve", recipeHandler.ResolveRecipe)
		recipeRoutes.GET("/templates", recipeHandler.ListTemplates)
		recipeRoutes.GET("/templates/:id", recipeHandler.GetTemplate)
		recipeRoutes.POST("/templates", recipeHandler.CreateTemplate)
		recipeRoutes.PUT("/templates/:id", recipeHandler.UpdateTemplate)
		recipeRoutes.DELETE("/templates/:id", recipeHandler.DeleteTemplate)
		recipeRoutes.POST("/validate", recipeHandler.ValidateRecipe)
	}

	// Snapshot Management Routes
	snapshotRoutes := v1.Group("/snapshots")
	{
		snapshotRoutes.POST("", snapshotHandler.CreateSnapshot)
		snapshotRoutes.GET("/:id", snapshotHandler.GetSnapshot)
		snapshotRoutes.PUT("/:id", snapshotHandler.UpdateSnapshot)
		snapshotRoutes.DELETE("/:id", snapshotHandler.DeleteSnapshot)
		snapshotRoutes.GET("", snapshotHandler.QuerySnapshots)
		snapshotRoutes.GET("/patient/:patient_id", snapshotHandler.GetPatientSnapshots)
	}

	// Workflow Orchestration Routes
	workflowRoutes := v1.Group("/workflows")
	{
		workflowRoutes.POST("", workflowHandler.StartWorkflow)
		workflowRoutes.GET("/:id", workflowHandler.GetWorkflowStatus)
		workflowRoutes.PUT("/:id", workflowHandler.UpdateWorkflow)
		workflowRoutes.POST("/:id/cancel", workflowHandler.CancelWorkflow)
		workflowRoutes.GET("", workflowHandler.ListWorkflows)
		workflowRoutes.GET("/:id/steps", workflowHandler.GetWorkflowSteps)
	}

	// Clinical Engine Routes
	clinicalRoutes := v1.Group("/clinical")
	{
		clinicalRoutes.POST("/dosage/calculate", clinicalEngineHandler.CalculateDosage)
		clinicalRoutes.POST("/risk/assess", clinicalEngineHandler.AssessRisk)
		clinicalRoutes.POST("/safety/check", clinicalEngineHandler.PerformSafetyChecks)
		clinicalRoutes.POST("/rules/evaluate", clinicalEngineHandler.EvaluateRules)
		clinicalRoutes.GET("/interactions", clinicalEngineHandler.CheckInteractions)
		clinicalRoutes.GET("/contraindications", clinicalEngineHandler.CheckContraindications)
	}

	// Knowledge Base Routes
	knowledgeRoutes := v1.Group("/knowledge")
	{
		knowledgeRoutes.POST("/query", knowledgeBaseHandler.QueryKnowledge)
		knowledgeRoutes.GET("/evidence", knowledgeBaseHandler.GetEvidenceSources)
		knowledgeRoutes.GET("/guidelines", knowledgeBaseHandler.GetGuidelines)
		knowledgeRoutes.GET("/drug-info/:drug_name", knowledgeBaseHandler.GetDrugInformation)
		knowledgeRoutes.GET("/protocols", knowledgeBaseHandler.GetProtocols)
	}

	// Cache Management Routes (admin only)
	cacheRoutes := v1.Group("/cache")
	cacheRoutes.Use(middleware.RequireRole("admin"))
	{
		cacheRoutes.DELETE("", handlers.NewCacheHandler(s.services, s.logger).ClearCache)
		cacheRoutes.GET("/stats", handlers.NewCacheHandler(s.services, s.logger).GetCacheStats)
		cacheRoutes.DELETE("/:key", handlers.NewCacheHandler(s.services, s.logger).DeleteCacheKey)
	}

	// Admin routes
	adminRoutes := v1.Group("/admin")
	adminRoutes.Use(middleware.RequireRole("admin"))
	{
		adminRoutes.GET("/stats", handlers.NewAdminHandler(s.services, s.logger).GetSystemStats)
		adminRoutes.POST("/maintenance", handlers.NewAdminHandler(s.services, s.logger).PerformMaintenance)
		adminRoutes.GET("/audit-log", handlers.NewAdminHandler(s.services, s.logger).GetAuditLog)
	}
}

// setupFHIRRoutes configures FHIR R4 compliant routes
func (s *Server) setupFHIRRoutes(fhir *gin.RouterGroup) {
	fhirHandler := handlers.NewFHIRHandler(s.services, s.logger)

	// FHIR MedicationRequest resource
	fhir.POST("/MedicationRequest", fhirHandler.CreateMedicationRequest)
	fhir.GET("/MedicationRequest/:id", fhirHandler.GetMedicationRequest)
	fhir.PUT("/MedicationRequest/:id", fhirHandler.UpdateMedicationRequest)
	fhir.DELETE("/MedicationRequest/:id", fhirHandler.DeleteMedicationRequest)
	fhir.GET("/MedicationRequest", fhirHandler.SearchMedicationRequests)

	// FHIR Medication resource
	fhir.GET("/Medication/:id", fhirHandler.GetMedication)
	fhir.GET("/Medication", fhirHandler.SearchMedications)

	// FHIR Patient resource (limited operations)
	fhir.GET("/Patient/:id", fhirHandler.GetPatient)

	// FHIR Observation resource (for lab values, vital signs)
	fhir.GET("/Observation/:id", fhirHandler.GetObservation)
	fhir.GET("/Observation", fhirHandler.SearchObservations)

	// FHIR Bundle operations
	fhir.POST("/", fhirHandler.ProcessBundle)

	// FHIR metadata
	fhir.GET("/metadata", fhirHandler.GetCapabilityStatement)
}

// redirectToSwagger redirects root path to Swagger documentation
func (s *Server) redirectToSwagger(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/docs/swagger/index.html")
}

// RegisterSwaggerRoutes registers Swagger documentation routes
func (s *Server) RegisterSwaggerRoutes() {
	// This would typically be done with swaggo/gin-swagger
	// For now, we serve static documentation
	s.router.Static("/docs/swagger", "./docs/swagger")
}

// GetRouter returns the Gin router instance (useful for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// SetupTestRouter creates a router for testing purposes
func (s *Server) SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.setupGlobalMiddleware()
	s.setupRoutes()
	return s.router
}