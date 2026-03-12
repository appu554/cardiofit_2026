// Package api provides HTTP handlers and server setup for KB-13 Quality Measures Engine.
//
// The server uses Gin framework and follows patterns established in other KB services.
// All routes are prefixed with /v1 for versioning.
package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-13-quality-measures/internal/calculator"
	"kb-13-quality-measures/internal/config"
	"kb-13-quality-measures/internal/cql"
	"kb-13-quality-measures/internal/dashboard"
	"kb-13-quality-measures/internal/integrations"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/reporter"
	"kb-13-quality-measures/internal/repository"
)

// Server represents the HTTP server for KB-13.
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	logger     *zap.Logger
	store      *models.MeasureStore

	// Phase 2/3 components
	db              *sql.DB
	cqlClient       *cql.Client
	calcEngine      *calculator.Engine
	resultRepo      *repository.ResultRepository
	careGapRepo     *repository.CareGapRepository
	careGapDetector *calculator.CareGapDetector
	dashboardSvc    *dashboard.Service
	reporterSvc     *reporter.Reporter

	// Integration clients for KB inter-service communication
	kb7Client  *integrations.KB7Client
	kb18Client *integrations.KB18Client
	kb19Client *integrations.KB19Client

	// Handlers
	calcHandlers      *CalculationHandlers
	dashboardHandlers *DashboardHandlers
	reportHandlers    *ReportHandlers
}

// ServerDependencies holds all external dependencies for the server.
type ServerDependencies struct {
	DB        *sql.DB
	CQLClient *cql.Client

	// Integration client URLs (from config)
	KB7URL  string // KB-7 Terminology Service URL
	KB18URL string // KB-18 Governance Engine URL
	KB19URL string // KB-19 Protocol Orchestrator URL
}

// NewServer creates a new HTTP server with all routes configured.
func NewServer(cfg *config.Config, logger *zap.Logger, deps *ServerDependencies) (*Server, error) {
	// Set Gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Create measure store and load measures
	store := models.NewMeasureStore()
	if err := store.LoadMeasuresFromDirectory(cfg.Server.MeasuresPath); err != nil {
		logger.Warn("Failed to load measures from directory",
			zap.String("path", cfg.Server.MeasuresPath),
			zap.Error(err),
		)
		// Continue - measures can be loaded later or directory may not exist yet
	}

	logger.Info("Measure store initialized",
		zap.Int("measures_loaded", store.Count()),
	)

	s := &Server{
		config: cfg,
		router: router,
		logger: logger,
		store:  store,
	}

	// Initialize Phase 2/3 components if dependencies provided
	if deps != nil {
		s.db = deps.DB
		s.cqlClient = deps.CQLClient

		// Create calculator engine
		if deps.CQLClient != nil {
			s.calcEngine = calculator.NewEngine(deps.CQLClient, store, &cfg.Calculator, logger)
			s.calcHandlers = NewCalculationHandlers(s.calcEngine)
			logger.Info("Calculator engine initialized")
		}

		// Create care gap detector
		if deps.CQLClient != nil {
			s.careGapDetector = calculator.NewCareGapDetector(deps.CQLClient, logger)
			logger.Info("Care gap detector initialized")
		}

		// Create repositories if database provided
		if deps.DB != nil {
			s.resultRepo = repository.NewResultRepository(deps.DB, logger)
			s.careGapRepo = repository.NewCareGapRepository(deps.DB, logger)
			logger.Info("Repositories initialized")

			// Create dashboard service
			s.dashboardSvc = dashboard.NewService(
				deps.DB,
				s.resultRepo,
				s.careGapRepo,
				store,
				logger,
			)
			s.dashboardHandlers = NewDashboardHandlers(s.dashboardSvc)
			logger.Info("Dashboard service initialized")

			// Create reporter service and handlers
			s.reporterSvc = reporter.NewReporter(s.resultRepo, logger)
			s.reportHandlers = NewReportHandlers(s.reporterSvc, store)
			logger.Info("Reporter service initialized")
		}

		// Initialize KB integration clients
		if deps.KB7URL != "" {
			s.kb7Client = integrations.NewKB7Client(deps.KB7URL, logger)
			logger.Info("KB-7 Terminology client initialized", zap.String("url", deps.KB7URL))
		}
		if deps.KB18URL != "" {
			s.kb18Client = integrations.NewKB18Client(deps.KB18URL, logger)
			logger.Info("KB-18 Governance client initialized", zap.String("url", deps.KB18URL))
		}
		if deps.KB19URL != "" {
			s.kb19Client = integrations.NewKB19Client(deps.KB19URL, logger)
			logger.Info("KB-19 Protocol client initialized", zap.String("url", deps.KB19URL))
		}
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return s, nil
}

// setupMiddleware configures middleware for the router.
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request logging middleware
	s.router.Use(RequestLogger(s.logger))

	// CORS middleware for development
	if s.config.Server.Environment != "production" {
		s.router.Use(CORSMiddleware())
	}
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health endpoints (no auth required)
	s.router.GET("/health", s.HealthCheck)
	s.router.GET("/ready", s.ReadinessCheck)

	// Metrics endpoint (if enabled)
	if s.config.Metrics.Enabled {
		s.router.GET(s.config.Metrics.Path, s.MetricsHandler)
	}

	// API v1 routes
	v1 := s.router.Group("/v1")
	{
		// Measure definition routes
		measures := v1.Group("/measures")
		{
			measures.GET("", s.ListMeasures)
			measures.GET("/:id", s.GetMeasure)
			measures.GET("/search", s.SearchMeasures)
			measures.GET("/by-program/:program", s.GetMeasuresByProgram)
			measures.GET("/by-domain/:domain", s.GetMeasuresByDomain)
			measures.POST("/reload", s.ReloadMeasures)
		}

		// Benchmark routes
		benchmarks := v1.Group("/benchmarks")
		{
			benchmarks.GET("/:measureId", s.GetBenchmarks)
			benchmarks.GET("/:measureId/:year", s.GetBenchmarkByYear)
		}

		// Calculation routes (Phase 2) - use handlers if available
		if s.calcHandlers != nil {
			s.calcHandlers.RegisterRoutes(v1)
		} else {
			// Fallback placeholder routes
			calculate := v1.Group("/calculations")
			{
				calculate.POST("/measure/:id", s.notImplemented("Calculations"))
				calculate.POST("/measure/:id/async", s.notImplemented("Async calculations"))
				calculate.GET("/jobs/:jobId", s.notImplemented("Job tracking"))
				calculate.POST("/batch", s.notImplemented("Batch calculations"))
			}
		}

		// Report routes (Phase 2) - use handlers if available
		if s.reportHandlers != nil {
			s.reportHandlers.RegisterRoutes(v1)
		} else {
			// Fallback placeholder routes
			reports := v1.Group("/reports")
			{
				reports.GET("", s.ListReports)
				reports.GET("/:id", s.GetReport)
				reports.POST("/generate", s.GenerateReport)
			}
		}

		// Care gap routes (Phase 3)
		gaps := v1.Group("/care-gaps")
		{
			gaps.GET("", s.ListCareGaps)
			gaps.GET("/:id", s.GetCareGap)
			gaps.GET("/by-measure/:measureId", s.GetCareGapsByMeasure)
			gaps.GET("/by-patient/:patientId", s.GetCareGapsByPatient)
			gaps.GET("/summary/:measureId", s.GetCareGapSummary)
			gaps.PUT("/:id/status", s.UpdateCareGapStatus)
			gaps.POST("/identify/:measureId", s.IdentifyCareGaps)
		}

		// Dashboard routes (Phase 3) - use handlers if available
		if s.dashboardHandlers != nil {
			s.dashboardHandlers.RegisterRoutes(v1)
		} else {
			// Fallback placeholder routes
			dash := v1.Group("/dashboard")
			{
				dash.GET("/overview", s.notImplemented("Dashboard overview"))
				dash.GET("/measures", s.notImplemented("Measure performance"))
				dash.GET("/programs", s.notImplemented("Program summaries"))
				dash.GET("/domains", s.notImplemented("Domain summaries"))
				dash.GET("/trends/:measureId", s.notImplemented("Trend data"))
				dash.GET("/care-gaps", s.notImplemented("Care gap dashboard"))
			}
		}
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.Int("port", s.config.Server.Port),
		zap.String("environment", s.config.Server.Environment),
	)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// GetStore returns the measure store for external access.
func (s *Server) GetStore() *models.MeasureStore {
	return s.store
}

// notImplemented returns a handler for not yet implemented features.
func (s *Server) notImplemented(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not_implemented",
			"message": fmt.Sprintf("%s requires database connection", feature),
		})
	}
}

// --- Care Gap Handlers ---

// ListCareGaps returns a list of all care gaps.
func (s *Server) ListCareGaps(c *gin.Context) {
	if s.careGapRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	gaps, err := s.careGapRepo.GetOpenGaps(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"gaps":  gaps,
		"count": len(gaps),
	})
}

// GetCareGap returns a specific care gap by ID.
func (s *Server) GetCareGap(c *gin.Context) {
	if s.careGapRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	id := c.Param("id")
	gap, err := s.careGapRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gap)
}

// GetCareGapsByMeasure returns care gaps for a specific measure.
func (s *Server) GetCareGapsByMeasure(c *gin.Context) {
	if s.careGapRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	measureID := c.Param("measureId")
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	gaps, err := s.careGapRepo.GetByMeasure(c.Request.Context(), measureID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"measure_id": measureID,
		"gaps":       gaps,
		"count":      len(gaps),
	})
}

// GetCareGapsByPatient returns care gaps for a specific patient.
func (s *Server) GetCareGapsByPatient(c *gin.Context) {
	if s.careGapRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	patientID := c.Param("patientId")
	gaps, err := s.careGapRepo.GetByPatient(c.Request.Context(), patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"gaps":       gaps,
		"count":      len(gaps),
	})
}

// GetCareGapSummary returns summary statistics for a measure's care gaps.
func (s *Server) GetCareGapSummary(c *gin.Context) {
	if s.careGapRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	measureID := c.Param("measureId")
	summary, err := s.careGapRepo.GetSummaryByMeasure(c.Request.Context(), measureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// --- Report Handlers (Placeholders) ---

func (s *Server) ListReports(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "Reports will be available in a future release",
	})
}

func (s *Server) GetReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "Reports will be available in a future release",
	})
}

func (s *Server) GenerateReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "Report generation will be available in a future release",
	})
}

// --- Additional Handlers ---

// ReloadMeasures reloads measure definitions from disk.
func (s *Server) ReloadMeasures(c *gin.Context) {
	if err := s.store.LoadMeasuresFromDirectory(s.config.Server.MeasuresPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reload_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Measures reloaded successfully",
		"measures_count": s.store.Count(),
	})
}

// UpdateCareGapStatus updates the status of a care gap.
func (s *Server) UpdateCareGapStatus(c *gin.Context) {
	if s.careGapRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := models.CareGapStatus(req.Status)
	if status != models.CareGapStatusOpen &&
		status != models.CareGapStatusInProgress &&
		status != models.CareGapStatusClosed &&
		status != models.CareGapStatusDeferred {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_status",
			"message": "Status must be: open, in-progress, closed, or deferred",
		})
		return
	}

	if err := s.careGapRepo.UpdateStatus(c.Request.Context(), id, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Care gap status updated",
		"id":      id,
		"status":  req.Status,
	})
}

// IdentifyCareGaps triggers care gap identification for a measure.
func (s *Server) IdentifyCareGaps(c *gin.Context) {
	if s.careGapDetector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "not_available",
			"message": "Care gap detection requires CQL engine connection",
		})
		return
	}

	measureID := c.Param("measureId")

	// Get measure definition
	measure := s.store.GetMeasure(measureID)
	if measure == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "measure_not_found",
			"message": "Measure not found: " + measureID,
		})
		return
	}

	// Build detection request
	// Note: In production, DenominatorPatientIDs and NumeratorPatientIDs would
	// come from a prior calculation result. This endpoint initiates detection
	// which can be integrated with the calculator engine for full population data.
	detectionReq := &calculator.DetectionRequest{
		MeasureID: measureID,
		Measure:   measure,
		// Patient IDs should be populated from calculation results or request body
		DenominatorPatientIDs: []string{},
		NumeratorPatientIDs:   []string{},
	}

	// Check if patient IDs provided in request body
	var reqBody struct {
		DenominatorPatientIDs []string `json:"denominator_patient_ids"`
		NumeratorPatientIDs   []string `json:"numerator_patient_ids"`
	}
	if err := c.ShouldBindJSON(&reqBody); err == nil {
		detectionReq.DenominatorPatientIDs = reqBody.DenominatorPatientIDs
		detectionReq.NumeratorPatientIDs = reqBody.NumeratorPatientIDs
	}

	// Detect care gaps
	gaps, err := s.careGapDetector.DetectCareGaps(c.Request.Context(), detectionReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save gaps if repository available
	if s.careGapRepo != nil && len(gaps) > 0 {
		if err := s.careGapRepo.SaveBatch(c.Request.Context(), gaps); err != nil {
			s.logger.Warn("Failed to save identified care gaps",
				zap.String("measure_id", measureID),
				zap.Error(err),
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"measure_id": measureID,
		"gaps_found": len(gaps),
		"gaps":       gaps,
	})
}
