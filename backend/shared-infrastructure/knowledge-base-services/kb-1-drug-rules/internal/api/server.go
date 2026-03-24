// Package api provides the HTTP server and handlers for KB-1 Drug Rules Service.
// This version uses GOVERNED rules from the database - no hardcoded fallbacks.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-1-drug-rules/internal/config"
	"kb-1-drug-rules/internal/database"
	"kb-1-drug-rules/internal/models"
	"kb-1-drug-rules/internal/rules"
	"kb-1-drug-rules/internal/services"
	"kb-1-drug-rules/pkg/cache"
	"kb-1-drug-rules/pkg/kb4"
)

// Default jurisdiction when not specified
const defaultJurisdiction = "US"

// Server represents the HTTP server.
type Server struct {
	config    *config.Config
	router    *gin.Engine
	httpSrv   *http.Server
	dosing    *services.DosingService
	rules     *rules.Repository
	db        *database.DB
	cache     rules.Cache
	kb4Client *kb4.Client // KB-4 Patient Safety Service client
	log       *logrus.Entry
}

// NewServer creates a new HTTP server with database-backed governed rules.
func NewServer(cfg *config.Config) (*Server, error) {
	// Set Gin mode
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Create logger
	log := logrus.WithFields(logrus.Fields{
		"service": "kb-1-drug-rules",
		"version": "2.0.0", // Governed rules version
	})

	// Connect to PostgreSQL database
	dbCfg := database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	db, err := database.Connect(dbCfg, log)
	if err != nil {
		// Log warning but continue - server can start without DB for health checks
		log.WithError(err).Warn("Failed to connect to database - service will start in degraded mode")
	}

	// Initialize cache (Redis or NoOp)
	var ruleCache rules.Cache
	if cfg.Redis.Enabled && cfg.Cache.Enabled {
		redisCfg := cache.Config{
			Host:        cfg.Redis.Host,
			Port:        cfg.Redis.Port,
			Password:    cfg.Redis.Password,
			DB:          cfg.Redis.DB,
			MaxRetries:  cfg.Redis.MaxRetries,
			PoolSize:    cfg.Redis.PoolSize,
			DialTimeout: cfg.Redis.DialTimeout,
			ReadTimeout: cfg.Redis.ReadTimeout,
		}
		redisCache, err := cache.NewRedisCache(redisCfg, log)
		if err != nil {
			log.WithError(err).Warn("Failed to connect to Redis - caching disabled")
			ruleCache = cache.NewNoOpCache()
		} else {
			ruleCache = redisCache
		}
	} else {
		ruleCache = cache.NewNoOpCache()
	}

	// Create rules repository and dosing service
	var rulesRepo *rules.Repository
	if db != nil {
		rulesRepo = rules.NewRepository(db.DB, ruleCache, log)
	}

	var dosingService *services.DosingService
	if rulesRepo != nil {
		dosingService = services.NewDosingService(rulesRepo)
	}

	// Initialize KB-4 Patient Safety client
	var kb4Client *kb4.Client
	if cfg.KB4.Enabled {
		kb4Cfg := kb4.Config{
			BaseURL:    cfg.KB4.BaseURL,
			Timeout:    cfg.KB4.Timeout,
			MaxRetries: cfg.KB4.MaxRetries,
			RetryDelay: cfg.KB4.RetryDelay,
			Enabled:    cfg.KB4.Enabled,
		}
		kb4Client = kb4.NewClientWithConfig(kb4Cfg, log)
		log.WithField("kb4_url", cfg.KB4.BaseURL).Info("KB-4 Patient Safety integration enabled")
	} else {
		log.Warn("KB-4 Patient Safety integration disabled - safety checks will be skipped")
	}

	s := &Server{
		config:    cfg,
		router:    router,
		dosing:    dosingService,
		rules:     rulesRepo,
		db:        db,
		cache:     ruleCache,
		kb4Client: kb4Client,
		log:       log,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s, nil
}

// getJurisdiction extracts jurisdiction from request header or uses default
func (s *Server) getJurisdiction(c *gin.Context) string {
	jurisdiction := c.GetHeader("X-Patient-Jurisdiction")
	if jurisdiction == "" {
		jurisdiction = c.Query("jurisdiction")
	}
	if jurisdiction == "" {
		jurisdiction = defaultJurisdiction
	}
	return jurisdiction
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request ID middleware
	s.router.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})

	// Logging middleware
	s.router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		latency := time.Since(start)

		s.log.WithFields(logrus.Fields{
			"status":       c.Writer.Status(),
			"method":       c.Request.Method,
			"path":         path,
			"latency_ms":   latency.Milliseconds(),
			"request_id":   c.GetString("request_id"),
			"jurisdiction": s.getJurisdiction(c),
		}).Info("Request processed")
	})
}

func (s *Server) setupRoutes() {
	// Health endpoints
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/ready", s.handleReady)

	// API v1
	v1 := s.router.Group("/v1")
	{
		// Dose calculation endpoints
		v1.POST("/calculate", s.handleCalculateDose)
		v1.POST("/calculate/weight-based", s.handleWeightBasedDose)
		v1.POST("/calculate/bsa-based", s.handleBSABasedDose)
		v1.POST("/calculate/pediatric", s.handlePediatricDose)
		v1.POST("/calculate/renal", s.handleRenalDose)
		v1.POST("/calculate/hepatic", s.handleHepaticDose)
		v1.POST("/calculate/geriatric", s.handleGeriatricDose)

		// Patient parameter endpoints
		patient := v1.Group("/patient")
		{
			patient.POST("/bsa", s.handleCalculateBSA)
			patient.POST("/ibw", s.handleCalculateIBW)
			patient.POST("/crcl", s.handleCalculateCrCl)
			patient.POST("/egfr", s.handleCalculateEGFR)
		}

		// Dose validation endpoints
		validate := v1.Group("/validate")
		{
			validate.POST("/dose", s.handleValidateDose)
			validate.GET("/max-dose", s.handleGetMaxDose)
		}

		// Dosing rules endpoints
		v1.GET("/rules", s.handleListRules)
		v1.GET("/rules/search", s.handleSearchRules)
		v1.GET("/rules/:rxnorm", s.handleGetRule)
		v1.GET("/rules/stats", s.handleGetStats)

		// Adjustment info endpoints
		adjustments := v1.Group("/adjustments")
		{
			adjustments.GET("/renal", s.handleRenalInfo)
			adjustments.GET("/hepatic", s.handleHepaticInfo)
			adjustments.GET("/age", s.handleAgeInfo)
		}

		// High-alert check endpoint
		v1.GET("/high-alert/check", s.handleHighAlertCheck)

		// High-risk categories endpoint
		v1.GET("/high-risk/categories", s.handleGetHighRiskCategories)

		// FDC (Fixed-Dose Combination) endpoints
		v1.GET("/fdc/:drug_name/components", s.handleFDCComponents)

		// Optimised dose endpoints (resistant HTN detection)
		v1.GET("/optimised-dose/:drug_name", s.handleOptimisedDose)
		v1.GET("/optimised-dose/:drug_name/check", s.handleOptimisedDoseCheck)

		// =============================================================
		// ADMIN APPROVAL WORKFLOW ENDPOINTS
		// =============================================================
		// These endpoints expose the approval workflow for pharmacist
		// and CMO review of drug rules before clinical use
		admin := v1.Group("/admin")
		{
			// View pending reviews (pharmacist queue)
			admin.GET("/pending", s.handlePendingReviews)
			// View approval workflow statistics
			admin.GET("/approval-stats", s.handleApprovalStats)
			// Approve a drug rule (requires verification flag for high-risk)
			admin.POST("/approve/:id", s.handleApproveRule)
			// Reject a drug rule
			admin.POST("/reject/:id", s.handleRejectRule)
			// Submit pharmacist review (DRAFT → REVIEWED)
			admin.POST("/review/:id", s.handleReviewRule)
			// Get rule audit history
			admin.GET("/audit/:rxnorm", s.handleRuleAudit)
			// Get rule with full governance metadata (admin view - all statuses)
			admin.GET("/rules/:rxnorm", s.handleAdminGetRule)
		}
	}

	// Also support /api/v1 prefix as shown in README
	api := s.router.Group("/api/v1")
	{
		api.POST("/calculate", s.handleCalculateDose)
		api.POST("/calculate/weight-based", s.handleWeightBasedDose)
		api.POST("/calculate/bsa-based", s.handleBSABasedDose)
		api.POST("/calculate/pediatric", s.handlePediatricDose)
		api.POST("/calculate/renal", s.handleRenalDose)
		api.POST("/validate", s.handleValidateDose)
		api.GET("/validate/max-dose", s.handleGetMaxDose)
		api.GET("/rules", s.handleListRules)
		api.GET("/rules/search", s.handleSearchRules)
		api.GET("/rules/:rxnorm", s.handleGetRule)
		api.GET("/rules/stats", s.handleGetStats)

		patient := api.Group("/patient")
		{
			patient.POST("/bsa", s.handleCalculateBSA)
			patient.POST("/ibw", s.handleCalculateIBW)
			patient.POST("/crcl", s.handleCalculateCrCl)
			patient.POST("/egfr", s.handleCalculateEGFR)
		}

		adjustments := api.Group("/adjustments")
		{
			adjustments.GET("/renal", s.handleRenalInfo)
			adjustments.GET("/hepatic", s.handleHepaticInfo)
			adjustments.GET("/age", s.handleAgeInfo)
		}

		api.GET("/high-alert/check", s.handleHighAlertCheck)
		api.GET("/high-risk/categories", s.handleGetHighRiskCategories)

		// FDC and optimised dose endpoints
		api.GET("/fdc/:drug_name/components", s.handleFDCComponents)
		api.GET("/optimised-dose/:drug_name", s.handleOptimisedDose)
		api.GET("/optimised-dose/:drug_name/check", s.handleOptimisedDoseCheck)
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Close database connection
	if s.db != nil {
		s.db.Close()
	}
	return s.httpSrv.Shutdown(ctx)
}

// checkServiceReady returns error if service is not ready for dose calculations
func (s *Server) checkServiceReady() error {
	if s.dosing == nil || s.rules == nil {
		return fmt.Errorf("service not ready: database connection required for governed rules")
	}
	return nil
}

// ============================================================================
// HEALTH HANDLERS
// ============================================================================

func (s *Server) handleHealth(c *gin.Context) {
	status := "healthy"
	if s.db == nil {
		status = "degraded"
	}
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Version:   "2.0.0",
		Service:   "kb-1-drug-rules",
	})
}

func (s *Server) handleReady(c *gin.Context) {
	// Check database connectivity
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, models.HealthResponse{
			Status:    "not_ready",
			Timestamp: time.Now(),
			Version:   "2.0.0",
			Service:   "kb-1-drug-rules",
		})
		return
	}

	if err := s.db.Health(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.HealthResponse{
			Status:    "database_error",
			Timestamp: time.Now(),
			Version:   "2.0.0",
			Service:   "kb-1-drug-rules",
		})
		return
	}

	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "ready",
		Timestamp: time.Now(),
		Version:   "2.0.0",
		Service:   "kb-1-drug-rules",
	})
}

// ============================================================================
// DOSE CALCULATION HANDLERS
// ============================================================================

func (s *Server) handleCalculateDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.DoseCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CalculateDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	// Call KB-4 Patient Safety check if enabled
	if s.kb4Client != nil && s.kb4Client.IsEnabled() && result.Success {
		safetyVerdict := s.performSafetyCheck(ctx, result, &req)
		result.SafetyVerdict = safetyVerdict
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleWeightBasedDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.WeightBasedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CalculateWeightBasedDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleBSABasedDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.BSABasedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	// BSA calculation doesn't require drug-specific lookup
	result, err := s.dosing.CalculateBSABasedDose(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handlePediatricDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.PediatricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CalculatePediatricDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleRenalDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.RenalAdjustedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CalculateRenalAdjustedDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleHepaticDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.HepaticAdjustedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CalculateHepaticAdjustedDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleGeriatricDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.GeriatricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CalculateGeriatricDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Calculation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// PATIENT PARAMETER HANDLERS
// ============================================================================

func (s *Server) handleCalculateBSA(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.BSARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	bsa := s.dosing.CalculateBSA(req.HeightCm, req.WeightKg)
	c.JSON(http.StatusOK, models.BSAResponse{
		BSA:      bsa,
		Formula:  "Mosteller",
		HeightCm: req.HeightCm,
		WeightKg: req.WeightKg,
	})
}

func (s *Server) handleCalculateIBW(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.IBWRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ibw := s.dosing.CalculateIBW(req.HeightCm, req.Gender)
	c.JSON(http.StatusOK, models.IBWResponse{
		IBWKg:    ibw,
		Formula:  "Devine",
		HeightCm: req.HeightCm,
		Gender:   req.Gender,
	})
}

func (s *Server) handleCalculateCrCl(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.CrClRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	crcl := s.dosing.CalculateCrCl(req.Age, req.WeightKg, req.SerumCreatinine, req.Gender)
	stage, interpretation := s.dosing.GetCKDStage(crcl)

	c.JSON(http.StatusOK, models.CrClResponse{
		CrCl:           crcl,
		Formula:        "Cockcroft-Gault",
		CKDStage:       stage,
		Interpretation: interpretation,
	})
}

func (s *Server) handleCalculateEGFR(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.EGFRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	egfr := s.dosing.CalculateEGFR(req.Age, req.SerumCreatinine, req.Gender)
	stage, desc := s.dosing.GetCKDStage(egfr)

	interpretation := fmt.Sprintf("eGFR of %.1f indicates %s", egfr, desc)

	c.JSON(http.StatusOK, models.EGFRResponse{
		EGFR:           egfr,
		Formula:        "CKD-EPI 2021",
		CKDStage:       stage,
		CKDDescription: desc,
		Interpretation: interpretation,
	})
}

// ============================================================================
// DOSE VALIDATION HANDLERS
// ============================================================================

func (s *Server) handleValidateDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	var req models.DoseValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.ValidateDose(ctx, &req, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleGetMaxDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Query("rxnorm_code")
	if rxnorm == "" {
		rxnorm = c.Query("rxnorm")
	}
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "rxnorm_code required",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.GetMaxDose(ctx, rxnorm, jurisdiction)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// DOSING RULES HANDLERS
// ============================================================================

func (s *Server) handleListRules(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	// Search with empty query returns all (up to limit)
	summaries, err := s.dosing.SearchDrugs(ctx, "", jurisdiction, rules.SearchFilters{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to list rules",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, models.RulesListResponse{
		Count: len(summaries),
		Rules: summaries,
	})
}

func (s *Server) handleSearchRules(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	query := c.Query("q")
	category := c.Query("category")
	highAlert := c.Query("high_alert") == "true"

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	filters := rules.SearchFilters{
		HighAlertOnly: highAlert,
		Category:      category,
	}

	results, err := s.dosing.SearchDrugs(ctx, query, jurisdiction, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Search failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, models.RulesSearchResponse{
		Query:   query,
		Count:   len(results),
		Results: results,
	})
}

func (s *Server) handleGetRule(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Param("rxnorm")
	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	rule, err := s.rules.GetByRxNorm(ctx, rxnorm, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Message: fmt.Sprintf("No rule found for RxNorm code: %s in jurisdiction: %s", rxnorm, jurisdiction),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (s *Server) handleGetStats(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	stats, err := s.dosing.GetRepositoryStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get stats",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ============================================================================
// ADJUSTMENT INFO HANDLERS
// ============================================================================

func (s *Server) handleRenalInfo(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Query("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "rxnorm required",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	rule, err := s.rules.GetByRxNorm(ctx, rxnorm, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Success: false,
		})
		return
	}

	// Build renal adjustment info from governed rule
	response := map[string]interface{}{
		"rxnorm_code": rule.Drug.RxNormCode,
		"drug_name":   rule.Drug.Name,
		"type":        "renal",
		"renal_info":  rule.Dosing.Renal,
		"source": map[string]interface{}{
			"authority":    rule.Governance.Authority,
			"jurisdiction": rule.Governance.Jurisdiction,
			"version":      rule.Governance.Version,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleHepaticInfo(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Query("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "rxnorm required",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	rule, err := s.rules.GetByRxNorm(ctx, rxnorm, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Success: false,
		})
		return
	}

	// Build hepatic adjustment info from governed rule
	response := map[string]interface{}{
		"rxnorm_code":  rule.Drug.RxNormCode,
		"drug_name":    rule.Drug.Name,
		"type":         "hepatic",
		"hepatic_info": rule.Dosing.Hepatic,
		"source": map[string]interface{}{
			"authority":    rule.Governance.Authority,
			"jurisdiction": rule.Governance.Jurisdiction,
			"version":      rule.Governance.Version,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleAgeInfo(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Query("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "rxnorm required",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	rule, err := s.rules.GetByRxNorm(ctx, rxnorm, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Success: false,
		})
		return
	}

	// Build age adjustment info from governed rule
	response := map[string]interface{}{
		"rxnorm_code":    rule.Drug.RxNormCode,
		"drug_name":      rule.Drug.Name,
		"type":           "age",
		"pediatric_info": rule.Dosing.Pediatric,
		"geriatric_info": rule.Dosing.Geriatric,
		"source": map[string]interface{}{
			"authority":    rule.Governance.Authority,
			"jurisdiction": rule.Governance.Jurisdiction,
			"version":      rule.Governance.Version,
		},
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// HIGH-ALERT HANDLER
// ============================================================================

func (s *Server) handleHighAlertCheck(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Query("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "rxnorm required",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	result, err := s.dosing.CheckHighAlert(ctx, rxnorm, jurisdiction)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// FDC AND OPTIMISED DOSE HANDLERS
// ============================================================================

// handleFDCComponents returns the constituent drug classes for a fixed-dose combination.
// GET /v1/fdc/:drug_name/components
func (s *Server) handleFDCComponents(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	drugName := c.Param("drug_name")
	if drugName == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "drug_name path parameter required",
			Success: false,
		})
		return
	}

	mapping := s.dosing.GetFDCComponents(drugName)
	if mapping == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Not a recognised fixed-dose combination",
			Message: fmt.Sprintf("Drug '%s' is not in the FDC registry", drugName),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// handleOptimisedDose returns the maximum recommended dose for an antihypertensive.
// GET /v1/optimised-dose/:drug_name
func (s *Server) handleOptimisedDose(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	drugName := c.Param("drug_name")
	if drugName == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "drug_name path parameter required",
			Success: false,
		})
		return
	}

	opt := s.dosing.GetOptimisedDose(drugName)
	if opt == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found in optimised dose table",
			Message: fmt.Sprintf("No optimised dose data for '%s'", drugName),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, opt)
}

// handleOptimisedDoseCheck checks if a current dose is at or above the optimised dose.
// GET /v1/optimised-dose/:drug_name/check?current_dose=X
func (s *Server) handleOptimisedDoseCheck(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	drugName := c.Param("drug_name")
	if drugName == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "drug_name path parameter required",
			Success: false,
		})
		return
	}

	currentDoseStr := c.Query("current_dose")
	if currentDoseStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "current_dose query parameter required",
			Success: false,
		})
		return
	}

	var currentDose float64
	if _, err := fmt.Sscanf(currentDoseStr, "%f", &currentDose); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "current_dose must be a valid number",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	opt := s.dosing.GetOptimisedDose(drugName)
	if opt == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found in optimised dose table",
			Message: fmt.Sprintf("No optimised dose data for '%s'", drugName),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, models.OptimisedDoseCheckResponse{
		DrugName:    opt.DrugName,
		DrugClass:   opt.DrugClass,
		CurrentDose: currentDose,
		MaxDose:     opt.MaxDoseMg,
		IsOptimised: currentDose >= opt.MaxDoseMg,
	})
}

// ============================================================================
// ADMIN APPROVAL WORKFLOW HANDLERS
// ============================================================================
// These handlers expose the approval workflow for pharmacist and CMO review
// of drug rules before they can be used for clinical dosing calculations.
//
// Workflow: DRAFT → REVIEWED → APPROVED → ACTIVE
//
// CRITICAL: Only ACTIVE rules are used for dose calculations!
// ============================================================================

// handlePendingReviews returns the pending review queue for pharmacists
// GET /v1/admin/pending
func (s *Server) handlePendingReviews(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	// Parse filter parameters
	filter := rules.PendingReviewFilter{
		RiskLevel:    c.Query("risk_level"),
		Jurisdiction: c.Query("jurisdiction"),
		Limit:        100,
	}

	// Parse limit if provided
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	items, err := s.rules.GetPendingReviews(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get pending reviews",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(items),
		"items":   items,
		"filter":  filter,
	})
}

// handleApprovalStats returns approval workflow statistics
// GET /v1/admin/approval-stats
func (s *Server) handleApprovalStats(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	stats, err := s.rules.GetApprovalStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get approval stats",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

// ApproveRequest for approving a drug rule
type ApproveRequest struct {
	ApprovedBy       string `json:"approved_by" binding:"required"`
	ReviewNotes      string `json:"review_notes"`
	SkipVerification bool   `json:"skip_verification"` // Required for CRITICAL/HIGH risk drugs
}

// handleApproveRule approves a drug rule for clinical use
// POST /v1/admin/approve/:id
// CRITICAL: This transitions a rule to ACTIVE status for clinical use!
func (s *Server) handleApproveRule(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Rule ID required",
			Success: false,
		})
		return
	}

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	err := s.rules.ApproveRule(ctx, ruleID, req.ApprovedBy, req.ReviewNotes, req.SkipVerification)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Failed to approve rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	s.log.WithFields(logrus.Fields{
		"rule_id":     ruleID,
		"approved_by": req.ApprovedBy,
		"verified":    req.SkipVerification,
	}).Info("Drug rule approved for clinical use")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rule approved and activated for clinical use",
		"rule_id": ruleID,
	})
}

// RejectRequest for rejecting a drug rule
type RejectRequest struct {
	RejectedBy      string `json:"rejected_by" binding:"required"`
	RejectionReason string `json:"rejection_reason" binding:"required"`
}

// handleRejectRule rejects a drug rule
// POST /v1/admin/reject/:id
func (s *Server) handleRejectRule(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Rule ID required",
			Success: false,
		})
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	err := s.rules.RejectRule(ctx, ruleID, req.RejectedBy, req.RejectionReason)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Failed to reject rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	s.log.WithFields(logrus.Fields{
		"rule_id":     ruleID,
		"rejected_by": req.RejectedBy,
		"reason":      req.RejectionReason,
	}).Info("Drug rule rejected")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rule rejected and retired",
		"rule_id": ruleID,
	})
}

// ReviewRequest for pharmacist review submission
type ReviewRequest struct {
	ReviewedBy  string `json:"reviewed_by" binding:"required"`
	ReviewNotes string `json:"review_notes" binding:"required"`
	// Pharmacist review checklist
	DosingVerified       bool `json:"dosing_verified"`
	RenalVerified        bool `json:"renal_verified"`
	HepaticVerified      bool `json:"hepatic_verified"`
	InteractionsVerified bool `json:"interactions_verified"`
	SafetyVerified       bool `json:"safety_verified"`
}

// handleReviewRule submits pharmacist review (DRAFT → REVIEWED)
// POST /v1/admin/review/:id
func (s *Server) handleReviewRule(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Rule ID required",
			Success: false,
		})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	// Update rule to REVIEWED status
	query := `
		UPDATE drug_rules
		SET approval_status = 'REVIEWED',
		    reviewed_by = $2,
		    reviewed_at = NOW(),
		    review_notes = $3
		WHERE id = $1 AND approval_status = 'DRAFT'
	`

	result, err := s.db.DB.ExecContext(ctx, query, ruleID, req.ReviewedBy, req.ReviewNotes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to submit review",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Rule not found or not in DRAFT status",
			Success: false,
		})
		return
	}

	s.log.WithFields(logrus.Fields{
		"rule_id":     ruleID,
		"reviewed_by": req.ReviewedBy,
	}).Info("Drug rule reviewed by pharmacist")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Review submitted, rule awaiting CMO approval",
		"rule_id": ruleID,
		"status":  "REVIEWED",
	})
}

// handleRuleAudit returns audit history for a drug rule
// GET /v1/admin/audit/:rxnorm
func (s *Server) handleRuleAudit(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Param("rxnorm")
	if rxnorm == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "RxNorm code required",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	history, err := s.rules.GetRuleHistory(ctx, rxnorm, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get audit history",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"rxnorm_code": rxnorm,
		"count":       len(history),
		"history":     history,
	})
}

// handleAdminGetRule returns a rule with all statuses (admin view)
// GET /v1/admin/rules/:rxnorm
// Unlike the regular endpoint, this returns rules regardless of approval status
func (s *Server) handleAdminGetRule(c *gin.Context) {
	if err := s.checkServiceReady(); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "Service unavailable",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	rxnorm := c.Param("rxnorm")
	ctx := c.Request.Context()
	jurisdiction := s.getJurisdiction(c)

	// Use admin query (activeOnly = false)
	rule, err := s.rules.GetByRxNormWithStatus(ctx, rxnorm, jurisdiction, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve rule",
			Message: err.Error(),
			Success: false,
		})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Drug not found",
			Message: fmt.Sprintf("No rule found for RxNorm code: %s", rxnorm),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// ============================================================================
// KB-4 PATIENT SAFETY INTEGRATION
// ============================================================================

// performSafetyCheck calls KB-4 Patient Safety Service to evaluate the calculated dose
// against comprehensive safety criteria including:
// - Black Box Warnings, Contraindications, Dose/Age Limits
// - Pregnancy/Lactation Safety, High-Alert Status
// - Beers Criteria, Anticholinergic Burden, Lab Requirements
func (s *Server) performSafetyCheck(ctx context.Context, result *models.DoseCalculationResult, req *models.DoseCalculationRequest) *models.KB4SafetyVerdict {
	if s.kb4Client == nil {
		return nil
	}

	// Build KB-4 safety check request
	safetyReq := &kb4.SafetyCheckRequest{
		Drug: kb4.DrugInfo{
			RxNormCode: result.RxNormCode,
			DrugName:   result.DrugName,
		},
		ProposedDose: result.RecommendedDose,
		DoseUnit:     result.Unit,
		Frequency:    result.Frequency,
		Route:        result.Route,
		Patient:      s.buildKB4PatientContext(&req.Patient),
	}

	// Call KB-4 safety check
	safetyResp, err := s.kb4Client.Check(ctx, safetyReq)
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"rxnorm_code": result.RxNormCode,
			"drug_name":   result.DrugName,
		}).Warn("KB-4 safety check failed - proceeding without safety verdict")
		return nil
	}

	// Convert KB-4 response to verdict model
	verdict := &models.KB4SafetyVerdict{
		Safe:             safetyResp.Safe,
		BlockPrescribing: safetyResp.BlockPrescribing,
		RequiresAction:   safetyResp.RequiresAction,
		IsHighAlertDrug:  safetyResp.IsHighAlertDrug,
		TotalAlerts:      safetyResp.TotalAlerts,
		CriticalAlerts:   safetyResp.CriticalAlerts,
		HighAlerts:       safetyResp.HighAlerts,
		CheckedAt:        safetyResp.CheckedAt,
		KB4RequestID:     safetyResp.RequestID,
	}

	// Convert alerts
	for _, alert := range safetyResp.Alerts {
		verdict.Alerts = append(verdict.Alerts, models.KB4SafetyAlert{
			Type:                   string(alert.Type),
			Severity:               string(alert.Severity),
			Title:                  alert.Title,
			Message:                alert.Message,
			RequiresAcknowledgment: alert.RequiresAcknowledgment,
			CanOverride:            alert.CanOverride,
			ClinicalRationale:      alert.ClinicalRationale,
			Recommendations:        alert.Recommendations,
			References:             alert.References,
		})
	}

	// Log safety check result
	s.log.WithFields(logrus.Fields{
		"rxnorm_code":       result.RxNormCode,
		"safe":              verdict.Safe,
		"block_prescribing": verdict.BlockPrescribing,
		"total_alerts":      verdict.TotalAlerts,
		"critical_alerts":   verdict.CriticalAlerts,
		"kb4_request_id":    verdict.KB4RequestID,
	}).Info("KB-4 safety check completed")

	return verdict
}

// buildKB4PatientContext converts KB-1 patient parameters to KB-4 patient context
func (s *Server) buildKB4PatientContext(patient *models.PatientParameters) kb4.PatientContext {
	if patient == nil {
		return kb4.PatientContext{}
	}

	// Map ChildPughScore (int) to ChildPughClass (string A/B/C) if class not provided
	childPughClass := patient.ChildPughClass
	if childPughClass == "" && patient.ChildPughScore > 0 {
		switch {
		case patient.ChildPughScore <= 6:
			childPughClass = "A"
		case patient.ChildPughScore <= 9:
			childPughClass = "B"
		default:
			childPughClass = "C"
		}
	}

	return kb4.PatientContext{
		Age:            float64(patient.Age),
		AgeUnit:        "years", // KB-1 assumes years
		WeightKg:       patient.WeightKg,
		HeightCm:       patient.HeightCm,
		Gender:         patient.Gender,
		EGFR:           patient.EGFR,
		ChildPughScore: childPughClass,
		// Note: KB-1 PatientParameters does not currently include:
		// - IsPregnant, PregnancyTrimester, IsLactating
		// - CreatinineClearance (uses EGFR instead)
		// - Conditions, CurrentMedications, Allergies
		// These can be extended in PatientParameters if needed for full safety checking
	}
}
