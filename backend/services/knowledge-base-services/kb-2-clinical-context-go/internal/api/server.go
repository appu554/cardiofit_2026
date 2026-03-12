package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/redis/go-redis/v9"

	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/metrics"
	"kb-2-clinical-context-go/internal/services"
)

// Server represents the API server
type Server struct {
	config *ServerConfig
	
	// Services
	contextService   *services.ContextService
	phenotypeEngine  *services.PhenotypeEngine
	riskService     *services.RiskAssessmentService
	treatmentService *services.TreatmentPreferenceService
	
	// Metrics
	metrics *metrics.PrometheusMetrics
}

// ServerConfig holds server configuration and dependencies
type ServerConfig struct {
	Config           *config.Config
	MetricsCollector *metrics.PrometheusMetrics
	
	// Database clients
	mongoClient *mongo.Client
	redisClient *redis.Client
	
	// Services
	ContextService   *services.ContextService
	PhenotypeEngine  *services.PhenotypeEngine
	RiskService     *services.RiskAssessmentService
	TreatmentService *services.TreatmentPreferenceService
}

// NewServer creates a new API server instance
func NewServer(config ServerConfig) *Server {
	server := &Server{
		config:           &config,
		contextService:   config.ContextService,
		phenotypeEngine:  config.PhenotypeEngine,
		riskService:     config.RiskService,
		treatmentService: config.TreatmentService,
		metrics:         config.MetricsCollector,
	}
	
	// Set up service dependencies
	server.contextService.SetServiceDependencies(
		server.phenotypeEngine,
		server.riskService,
		server.treatmentService,
	)
	
	return server
}

// RegisterRoutes registers all API routes
func (s *Server) RegisterRoutes(router *gin.Engine) {
	// Middleware for request tracking
	router.Use(s.requestTrackingMiddleware())
	
	// Health endpoints
	router.GET("/health", s.healthCheck)
	router.GET("/ready", s.readinessCheck)
	
	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	// API documentation
	router.GET("/v1/docs", s.apiDocumentation)
	
	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Phenotype endpoints
		phenotypes := v1.Group("/phenotypes")
		{
			phenotypes.GET("", s.getAvailablePhenotypes)
			phenotypes.POST("/evaluate", s.evaluatePhenotypes)
			phenotypes.POST("/explain", s.explainPhenotypes)
		}
		
		// Risk assessment endpoints
		risk := v1.Group("/risk")
		{
			risk.POST("/assess", s.assessRisk)
		}
		
		// Treatment preference endpoints
		treatment := v1.Group("/treatment")
		{
			treatment.POST("/preferences", s.evaluateTreatmentPreferences)
		}
		
		// Context assembly endpoints
		context := v1.Group("/context")
		{
			context.POST("/assemble", s.assembleContext)
			context.GET("/history/:patient_id", s.getContextHistory)
		}
	}
}

// requestTrackingMiddleware tracks requests for metrics
func (s *Server) requestTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.metrics.IncrementConcurrentRequests()
		defer s.metrics.DecrementConcurrentRequests()
		
		timer := s.metrics.StartTimer(c.Request.Method, c.FullPath())
		defer timer.ObserveDuration()
		
		c.Next()
		
		status := c.Writer.Status()
		s.metrics.RecordRequest(c.Request.Method, c.FullPath(), string(rune(status)))
	}
}

// readinessCheck checks if service is ready to accept traffic
func (s *Server) readinessCheck(c *gin.Context) {
	// Check if all required services are initialized
	ready := true
	checks := make(map[string]string)
	
	if s.phenotypeEngine == nil {
		ready = false
		checks["phenotype_engine"] = "not_initialized"
	} else {
		checks["phenotype_engine"] = "ready"
	}
	
	if s.riskService == nil {
		ready = false
		checks["risk_service"] = "not_initialized"
	} else {
		checks["risk_service"] = "ready"
	}
	
	if s.treatmentService == nil {
		ready = false
		checks["treatment_service"] = "not_initialized"
	} else {
		checks["treatment_service"] = "ready"
	}
	
	if s.contextService == nil {
		ready = false
		checks["context_service"] = "not_initialized"
	} else {
		checks["context_service"] = "ready"
	}
	
	status := "ready"
	statusCode := 200
	if !ready {
		status = "not_ready"
		statusCode = 503
	}
	
	response := gin.H{
		"status": status,
		"checks": checks,
	}
	
	c.JSON(statusCode, response)
}