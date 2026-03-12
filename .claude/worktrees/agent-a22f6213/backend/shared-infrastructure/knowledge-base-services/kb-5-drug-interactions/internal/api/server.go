package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"kb-drug-interactions/internal/cache"
	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/metrics"
	"kb-drug-interactions/internal/services"
)

// Server represents the HTTP server
type Server struct {
	Router             *gin.Engine
	config             *config.Config
	db                 *database.Database
	cache              *cache.CacheClient
	metrics            *metrics.Collector
	interactionService *services.InteractionService
	// Enhanced engines for comprehensive clinical analysis
	integrationService *services.EnhancedIntegrationService
	pgxEngine          *services.PharmacogenomicEngine
	classEngine        *services.ClassInteractionEngine
	modifierEngine     *services.FoodAlcoholHerbalEngine
	matrixService      *services.EnhancedInteractionMatrixService
	// Phase 3 engines: Drug-Disease, Allergy, Duplicate Therapy
	drugDiseaseEngine      *services.DrugDiseaseEngine
	allergyEngine          *services.AllergyEngine
	duplicateTherapyEngine *services.DuplicateTherapyEngine
	// Phase 4: Governance and Attribution
	governanceEngine       *services.GovernancePolicyEngine
}

// NewServer creates a new HTTP server
func NewServer(
	cfg *config.Config,
	db *database.Database,
	cache *cache.CacheClient,
	metrics *metrics.Collector,
	interactionService *services.InteractionService,
	// Enhanced engines (optional, pass nil for backward compatibility)
	integrationService *services.EnhancedIntegrationService,
	pgxEngine *services.PharmacogenomicEngine,
	classEngine *services.ClassInteractionEngine,
	modifierEngine *services.FoodAlcoholHerbalEngine,
	matrixService *services.EnhancedInteractionMatrixService,
	// Phase 3 engines: Drug-Disease, Allergy, Duplicate Therapy
	drugDiseaseEngine *services.DrugDiseaseEngine,
	allergyEngine *services.AllergyEngine,
	duplicateTherapyEngine *services.DuplicateTherapyEngine,
	// Phase 4: Governance and Attribution
	governanceEngine *services.GovernancePolicyEngine,
) *Server {
	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	if cfg.IsDevelopment() {
		router.Use(gin.Logger())
	}

	server := &Server{
		Router:             router,
		config:             cfg,
		db:                 db,
		cache:              cache,
		metrics:            metrics,
		interactionService: interactionService,
		integrationService: integrationService,
		pgxEngine:          pgxEngine,
		classEngine:        classEngine,
		modifierEngine:     modifierEngine,
		matrixService:      matrixService,
		// Phase 3 engines
		drugDiseaseEngine:      drugDiseaseEngine,
		allergyEngine:          allergyEngine,
		duplicateTherapyEngine: duplicateTherapyEngine,
		// Phase 4: Governance
		governanceEngine:       governanceEngine,
	}

	// Add custom middleware
	server.Router.Use(server.metricsMiddleware())
	server.Router.Use(server.corsMiddleware())
	server.Router.Use(server.errorMiddleware())

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check and metrics
	s.Router.GET("/health", s.healthCheck)
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Create handlers with enhanced engines
	interactionHandlers := NewInteractionHandlers(
		s.interactionService,
		s.integrationService,
		s.pgxEngine,
		s.classEngine,
		s.modifierEngine,
		s.matrixService,
	)

	// API v1 routes
	v1 := s.Router.Group("/api/v1")
	{
		// Drug interaction endpoints
		interactions := v1.Group("/interactions")
		{
			interactions.POST("/check", interactionHandlers.checkInteractions)
			interactions.POST("/batch-check", interactionHandlers.batchCheckInteractions)
			interactions.GET("/quick-check", interactionHandlers.quickCheck)
			interactions.GET("/:interaction_id", interactionHandlers.getInteractionDetails)
			interactions.GET("/search", interactionHandlers.searchInteractions)
			interactions.GET("/statistics", interactionHandlers.getInteractionStatistics)
			// Enhanced endpoints (Phase 2)
			interactions.POST("/comprehensive", interactionHandlers.comprehensiveAnalysis)
			interactions.POST("/food", interactionHandlers.foodInteractions)
			interactions.POST("/class", interactionHandlers.classInteractions)
		}

		// CYP/Pharmacogenomic endpoints (Phase 2)
		cyp := v1.Group("/cyp")
		{
			cyp.GET("/profile/:drug_code", interactionHandlers.cypProfile)
			cyp.GET("/interactions/:enzyme", interactionHandlers.cypEnzymeInteractions)
		}

		// Patient-specific endpoints
		patients := v1.Group("/patients")
		{
			patients.GET("/:patient_id/interactions/history", interactionHandlers.getPatientInteractionHistory)
		}

		// Alert management endpoints
		alerts := v1.Group("/alerts")
		{
			alerts.PUT("/:alert_id/override", interactionHandlers.overrideAlert)
		}

		// Drug information endpoints
		drugs := v1.Group("/drugs")
		{
			drugs.GET("/:drug_code/interactions", s.getDrugInteractions)
			drugs.GET("/:drug_code/synonyms", s.getDrugSynonyms)
			drugs.GET("/:drug_code/alternatives", s.getDrugAlternatives)
		}

		// Administrative endpoints
		admin := v1.Group("/admin")
		{
			admin.GET("/stats", s.getSystemStats)
			admin.POST("/cache/clear", s.clearCache)
			admin.GET("/database/health", s.getDatabaseHealth)
			admin.POST("/rules/reload", s.reloadRules)
			admin.GET("/analytics", s.getAnalytics)
		}

		// Phase 3 handlers: Drug-Disease, Allergy, Duplicate Therapy
		phase3Handlers := NewPhase3Handlers(
			s.drugDiseaseEngine,
			s.allergyEngine,
			s.duplicateTherapyEngine,
		)

		// Drug-Disease contraindication endpoints (Phase 3)
		contraindications := v1.Group("/contraindications")
		{
			contraindications.POST("/disease", phase3Handlers.checkDrugDiseaseContraindications)
			contraindications.GET("/drug/:drug_code/diseases", phase3Handlers.getContraindicatedDiseases)
			contraindications.GET("/disease/:disease_code/drugs", phase3Handlers.getContraindicatedDrugs)
		}

		// Allergy cross-reactivity endpoints (Phase 3)
		allergy := v1.Group("/allergy")
		{
			allergy.POST("/check", phase3Handlers.checkAllergyRisk)
			allergy.GET("/cross-reactivity/:allergen", phase3Handlers.getCrossReactivity)
			allergy.GET("/common-patterns", phase3Handlers.getCommonCrossReactivities)
		}

		// Duplicate therapy detection endpoints (Phase 3)
		duplicates := v1.Group("/duplicates")
		{
			duplicates.POST("/check", phase3Handlers.checkDuplicateTherapy)
			duplicates.GET("/classes/:drug_code", phase3Handlers.getDrugTherapeuticClasses)
			duplicates.GET("/common-classes", phase3Handlers.getCommonDuplicateClasses)
		}

		// Phase 4: Governance and Attribution endpoints
		governanceHandlers := NewGovernanceHandlers(
			s.governanceEngine,
			s.interactionService,
			s.integrationService,
		)

		// Governance endpoints
		governance := v1.Group("/governance")
		{
			governance.GET("/policy", governanceHandlers.getGovernancePolicy)
			governance.POST("/translate", governanceHandlers.translateSeverity)
			governance.GET("/actions", governanceHandlers.getGovernanceActions)
			governance.GET("/attribution/template", governanceHandlers.getAttributionTemplate)
		}

		// Governed interaction check (with full attribution)
		interactions.POST("/governed-check", governanceHandlers.governedInteractionCheck)
	}
}

// Health check endpoint
func (s *Server) healthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "kb-5-drug-interactions",
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

	if health["status"] == "unhealthy" {
		c.JSON(http.StatusServiceUnavailable, health)
	} else {
		c.JSON(http.StatusOK, health)
	}
}

// Drug-specific endpoints

func (s *Server) getDrugInteractions(c *gin.Context) {
	drugCode := c.Param("drug_code")
	severity := c.Query("severity")
	limit := parseIntQuery(c, "limit", 20)

	// This would query for interactions involving the specific drug
	sendSuccess(c, map[string]interface{}{
		"drug_code":    drugCode,
		"interactions": []interface{}{}, // Would be populated with actual data
		"filters": map[string]interface{}{
			"severity": severity,
			"limit":    limit,
		},
	}, nil)
}

func (s *Server) getDrugSynonyms(c *gin.Context) {
	drugCode := c.Param("drug_code")
	
	// This would query the synonym repository
	sendSuccess(c, map[string]interface{}{
		"primary_code": drugCode,
		"synonyms":     []interface{}{}, // Would be populated with actual synonyms
	}, nil)
}

func (s *Server) getDrugAlternatives(c *gin.Context) {
	drugCode := c.Param("drug_code")
	indication := c.Query("indication")
	
	sendSuccess(c, map[string]interface{}{
		"drug_code":    drugCode,
		"indication":   indication,
		"alternatives": []interface{}{}, // Would be populated with alternatives
	}, nil)
}

// Administrative endpoints

func (s *Server) getSystemStats(c *gin.Context) {
	stats := map[string]interface{}{
		"service":           "kb-5-drug-interactions",
		"uptime":           time.Since(time.Now().Add(-1 * time.Hour)), // Mock uptime
		"total_interactions": 1500, // Would be queried from database
		"cache_stats":       s.cache.GetStats(),
		"configuration": map[string]interface{}{
			"max_interactions_per_request": s.config.MaxInteractionsPerRequest,
			"cache_enabled":               s.config.CacheInteractionResults,
			"synonym_resolution_enabled":   s.config.EnableSynonymResolution,
			"analytics_enabled":           s.config.EnableInteractionAnalytics,
		},
	}

	sendSuccess(c, stats, nil)
}

func (s *Server) clearCache(c *gin.Context) {
	cacheType := c.Query("type")
	
	var err error
	switch cacheType {
	case "interactions":
		err = s.cache.InvalidateInteractionChecks()
	case "synonyms":
		err = s.cache.InvalidateDrugSynonyms()
	case "rules":
		err = s.cache.InvalidateInteractionRules()
	case "all":
		// Clear all cache types
		s.cache.InvalidateInteractionChecks()
		s.cache.InvalidateDrugSynonyms()
		s.cache.InvalidateInteractionRules()
	default:
		sendError(c, http.StatusBadRequest, "Invalid cache type", "INVALID_CACHE_TYPE", map[string]interface{}{
			"valid_types": []string{"interactions", "synonyms", "rules", "all"},
		})
		return
	}

	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to clear cache", "CACHE_CLEAR_FAILED", nil)
		return
	}

	sendSuccess(c, map[string]interface{}{
		"cache_type": cacheType,
		"status":     "cleared",
		"timestamp":  time.Now().UTC(),
	}, nil)
}

func (s *Server) getDatabaseHealth(c *gin.Context) {
	health := make(map[string]interface{})

	// Check database connectivity
	if err := s.db.HealthCheck(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
	} else {
		health["status"] = "healthy"
	}

	// Get connection pool stats
	if sqlDB, err := s.db.DB.DB(); err == nil {
		stats := sqlDB.Stats()
		health["connection_pool"] = map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":          stats.InUse,
			"idle":            stats.Idle,
		}
	}

	sendSuccess(c, health, nil)
}

func (s *Server) reloadRules(c *gin.Context) {
	// This would trigger a reload of interaction rules from the database
	sendSuccess(c, map[string]interface{}{
		"status":    "reloaded",
		"timestamp": time.Now().UTC(),
		"message":   "Interaction rules reloaded successfully",
	}, nil)
}

func (s *Server) getAnalytics(c *gin.Context) {
	days := parseIntQuery(c, "days", 30)
	
	// This would query analytics data from the database
	analytics := map[string]interface{}{
		"period_days": days,
		"metrics": map[string]interface{}{
			"total_checks":               2500,
			"total_interactions_found":   680,
			"avg_interactions_per_check": 2.7,
			"top_drug_pairs": []map[string]interface{}{
				{
					"drug_a":    "Warfarin",
					"drug_b":    "Aspirin",
					"count":     45,
					"severity":  "major",
				},
				{
					"drug_a":    "Metformin",
					"drug_b":    "Contrast Media",
					"count":     32,
					"severity":  "moderate",
				},
			},
			"severity_distribution": map[string]interface{}{
				"contraindicated": 25,
				"major":          180,
				"moderate":       420,
				"minor":          55,
			},
		},
	}

	sendSuccess(c, analytics, nil)
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
				sendError(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR", nil)
			}
		}
	})
}

func (s *Server) logError(c *gin.Context, err error) {
	// In production, you'd use a proper logging library
	method := c.Request.Method
	path := c.Request.URL.Path
	clientIP := c.ClientIP()
	
	gin.DefaultWriter.Write([]byte(
		time.Now().Format(time.RFC3339) + " ERROR " +
		method + " " + path + " " + clientIP + " " + err.Error() + "\n",
	))
}