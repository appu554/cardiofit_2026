// Package api provides the HTTP server for KB-16 Lab Interpretation Service
package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/internal/config"
	"kb-16-lab-interpretation/internal/database"
	"kb-16-lab-interpretation/pkg/baseline"
	"kb-16-lab-interpretation/pkg/fhir"
	"kb-16-lab-interpretation/pkg/governance"
	"kb-16-lab-interpretation/pkg/integration"
	"kb-16-lab-interpretation/pkg/interpretation"
	"kb-16-lab-interpretation/pkg/loinc"
	"kb-16-lab-interpretation/pkg/panels"
	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/review"
	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/trending"
	"kb-16-lab-interpretation/pkg/types"
	"kb-16-lab-interpretation/pkg/visualization"
)

// Server represents the HTTP server
type Server struct {
	config  *config.Config
	router  *gin.Engine
	httpSrv *http.Server
	log     *logrus.Entry

	// Database
	db *database.DB

	// Services
	resultStore     *store.ResultStore
	refDB           *reference.Database
	loincRepo       *loinc.Repository
	interpreter     *interpretation.Engine
	trendEngine     *trending.Engine
	baselineTracker *baseline.Tracker
	panelManager    *panels.Manager
	reviewService   *review.Service
	vizService      *visualization.Service
	fhirMapper      *fhir.Mapper

	// Integration clients
	kb2Client  *integration.KB2Client
	kb8Client  *integration.KB8Client
	kb9Client  *integration.KB9Client
	kb14Client *integration.KB14Client

	// Governance (Tier-7)
	govPublisher *governance.Publisher
	govObserver  *governance.InterpretationObserver
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, db *database.DB, log *logrus.Entry) (*Server, error) {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		config: cfg,
		router: router,
		db:     db,
		log:    log.WithField("component", "server"),
	}

	// Initialize services
	if err := s.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// initializeServices initializes all service components
func (s *Server) initializeServices() error {
	// Result store
	s.resultStore = store.NewResultStore(s.db.Postgres, s.db.Redis, s.log)

	// Reference database (in-memory hardcoded)
	s.refDB = reference.NewDatabase()

	// LOINC reference range repository (connects to shared canonical_facts DB)
	// This provides access to 6041 LOINC codes with reference ranges for Context Router
	s.loincRepo = loinc.NewRepository(s.db.Postgres, s.db.Redis, s.log)

	// Integration clients
	s.kb2Client = integration.NewKB2Client(s.config.Integration.KB2URL, s.log)
	s.kb8Client = integration.NewKB8Client(integration.KB8Config{
		BaseURL: s.config.Integration.KB8URL,
		Timeout: s.config.Integration.Timeout,
		Enabled: s.config.Integration.KB8Enabled,
	}, s.log)
	s.kb9Client = integration.NewKB9Client(integration.KB9Config{
		BaseURL: s.config.Integration.KB9URL,
		Timeout: s.config.Integration.Timeout,
		Enabled: s.config.Integration.KB9Enabled,
	}, s.log)
	s.kb14Client = integration.NewKB14Client(s.config.Integration.KB14URL, s.log)

	// Core engines
	s.interpreter = interpretation.NewEngine(s.refDB, s.resultStore, s.log)
	s.trendEngine = trending.NewEngine(s.resultStore, s.log)
	s.baselineTracker = baseline.NewTracker(s.db.Postgres, s.resultStore, s.log)
	s.panelManager = panels.NewManager(s.resultStore, s.refDB, s.kb8Client, s.log)
	s.reviewService = review.NewService(s.db.Postgres, s.kb14Client, s.log)
	s.vizService = visualization.NewService(s.resultStore, s.baselineTracker, s.refDB, s.log)
	s.fhirMapper = fhir.NewMapper()

	// Initialize governance (Tier-7) if enabled
	if s.config.Governance.Enabled {
		s.initializeGovernance()
	}

	s.log.Info("All services initialized")
	return nil
}

// initializeGovernance sets up the Tier-7 governance event system
func (s *Server) initializeGovernance() {
	cfg := s.config.Governance

	// Create publisher config
	pubConfig := governance.PublisherConfig{
		RedisEnabled:    s.config.Redis.Enabled,
		CriticalChannel: cfg.CriticalChannel,
		StandardChannel: cfg.StandardChannel,
		AuditChannel:    cfg.AuditChannel,
		MaxRetries:      3,
		RetryInterval:   100 * time.Millisecond,
		BufferSize:      cfg.BufferSize,
		FlushInterval:   5 * time.Second,
		AsyncPublish:    cfg.AsyncPublish,
		AuditEnabled:    cfg.AuditEnabled,
		MetricsEnabled:  s.config.Metrics.Enabled,
	}

	// Create publisher
	s.govPublisher = governance.NewPublisher(pubConfig, s.db.Redis, s.log)

	// Create observer config
	obsConfig := governance.ObserverConfig{
		CriticalLabEvents: true,
		PanicLabEvents:    true,
		DeltaCheckEvents:  true,
		PatternDetection:  true,
		TrendingEvents:    true,
		CareGapEvents:     true,

		PanicAckSLAMin:       cfg.PanicAckSLAMin,
		CriticalAckSLAMin:    cfg.CriticalAckSLAMin,
		HighAckSLAMin:        cfg.HighAckSLAMin,
		PanicReviewSLAMin:    cfg.PanicAckSLAMin * 2,
		CriticalReviewSLAMin: cfg.CriticalAckSLAMin * 2,
		HighReviewSLAMin:     cfg.HighAckSLAMin * 2,

		PanicEscalation:    "rapid_response",
		CriticalEscalation: "attending_physician",
		HighEscalation:     "care_team",
	}

	// Create observer
	s.govObserver = governance.NewInterpretationObserverWithConfig(s.govPublisher, s.log, obsConfig)

	// Start the publisher
	if err := s.govPublisher.Start(context.Background()); err != nil {
		s.log.WithError(err).Warn("Failed to start governance publisher")
	} else {
		s.log.Info("Tier-7 governance event system initialized")
	}
}

// setupMiddleware configures middleware stack
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(RecoveryMiddleware(s.log))

	// Request ID
	s.router.Use(RequestIDMiddleware())

	// CORS
	s.router.Use(CORSMiddleware())

	// Logging
	s.router.Use(LoggingMiddleware(s.log))

	// Metrics
	if s.config.Metrics.Enabled {
		s.router.Use(MetricsMiddleware())
	}

	// Client service identification
	s.router.Use(ClientServiceMiddleware())

	// Timeout
	s.router.Use(TimeoutMiddleware(s.config.Server.WriteTimeout))
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health endpoints
	s.router.GET("/health", s.health)
	s.router.GET("/ready", s.ready)

	// Metrics endpoint
	if s.config.Metrics.Enabled {
		s.router.GET(s.config.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Results Management
		results := v1.Group("/results")
		{
			results.POST("", s.storeResult)
			results.GET("/:id", s.getResult)
			results.POST("/batch", s.storeBatch)
		}

		// Patient Results
		patients := v1.Group("/patients/:patientId")
		{
			patients.GET("/results", s.getPatientResults)
			patients.GET("/results/:code", s.getPatientResultsByCode)
		}

		// Interpretation
		interpret := v1.Group("/interpret")
		{
			interpret.POST("", s.interpretResult)
			interpret.POST("/batch", s.interpretBatch)
		}

		// Trending
		trendingGroup := v1.Group("/trending/:patientId")
		{
			trendingGroup.GET("", s.getAllTrends)
			trendingGroup.GET("/:code", s.getTrend)
			trendingGroup.GET("/:code/multi", s.getMultiWindowTrend)
			trendingGroup.GET("/:code/enhanced", s.getTrendEnhanced)
			trendingGroup.GET("/:code/enhanced/multi", s.getMultiWindowTrendEnhanced)
			trendingGroup.GET("/:code/predict", s.getPredictions)
			trendingGroup.GET("/:code/context", s.getLabContext)
		}

		// Baselines
		baselines := v1.Group("/baselines/:patientId")
		{
			baselines.GET("", s.getPatientBaselines)
			baselines.GET("/:code", s.getBaseline)
			baselines.POST("/:code", s.setManualBaseline)
			baselines.POST("/:code/calculate", s.calculateBaseline)
		}

		// Panels
		panelRoutes := v1.Group("/panels")
		{
			panelRoutes.GET("", s.listPanels)
			panelRoutes.GET("/definitions/:type", s.getPanelDefinition)
			panelRoutes.POST("/patient/:patientId/assemble/:type", s.assemblePanel)
			panelRoutes.GET("/patient/:patientId/detect", s.detectAvailablePanels)
		}

		// Review Workflow
		reviewRoutes := v1.Group("/review")
		{
			reviewRoutes.GET("/pending", s.getPendingReviews)
			reviewRoutes.GET("/critical", s.getCriticalQueue)
			reviewRoutes.POST("/acknowledge", s.acknowledgeResult)
			reviewRoutes.POST("/complete", s.completeReview)
			reviewRoutes.GET("/stats", s.getReviewStats)
		}

		// Visualization
		charts := v1.Group("/charts/:patientId")
		{
			charts.GET("/:code", s.getChartData)
		}

		sparklines := v1.Group("/sparklines/:patientId")
		{
			sparklines.GET("/:code", s.getSparkline)
		}

		v1.GET("/dashboard/:patientId", s.getDashboard)

		// Reference Data
		ref := v1.Group("/reference")
		{
			ref.GET("/tests", s.listTests)
			ref.GET("/tests/:code", s.getTestDefinition)
		}

		// LOINC Reference Ranges (from shared canonical_facts DB)
		// Used by Context Router for DDI threshold evaluation
		loincRoutes := v1.Group("/loinc")
		{
			loincRoutes.GET("/reference-ranges/:code", s.getLOINCReferenceRange)
			loincRoutes.GET("/reference-ranges/:code/context", s.getLOINCReferenceRangeWithContext)
			loincRoutes.GET("/ddi-relevant", s.getDDIRelevantLOINCRanges)
			loincRoutes.GET("/categories", s.getLOINCCategories)
			loincRoutes.GET("/categories/:category", s.getLOINCByCategory)
			loincRoutes.GET("/search", s.searchLOINC)
			loincRoutes.GET("/stats", s.getLOINCStats)
		}

		// Care Gaps (KB-9 Integration)
		careGaps := v1.Group("/care-gaps")
		{
			careGaps.GET("/patient/:patientId", s.getPatientCareGaps)
			careGaps.POST("/patient/:patientId/identify", s.identifyLabCareGaps)
			careGaps.POST("/report", s.reportCareGap)
		}
	}

	// FHIR endpoints
	fhirGroup := s.router.Group("/fhir")
	{
		fhirGroup.GET("/Observation", s.searchObservations)
		fhirGroup.GET("/Observation/:id", s.getObservation)
		fhirGroup.GET("/DiagnosticReport/:patientId/:panelType", s.getDiagnosticReport)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpSrv = &http.Server{
		Addr:         ":" + strconv.Itoa(s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	s.log.WithField("port", s.config.Server.Port).Info("Starting HTTP server")
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

// =============================================================================
// HEALTH ENDPOINTS
// =============================================================================

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "kb-16-lab-interpretation",
		"version": "1.0.0",
	})
}

func (s *Server) ready(c *gin.Context) {
	ctx := c.Request.Context()
	dbHealth := s.db.Health(ctx)

	allHealthy := true
	if dbHealth["postgres"] != "healthy" {
		allHealthy = false
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"ready":    allHealthy,
		"database": dbHealth,
	})
}

// =============================================================================
// RESULTS MANAGEMENT HANDLERS
// =============================================================================

func (s *Server) storeResult(c *gin.Context) {
	var req types.StoreResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	result, err := s.resultStore.Store(c.Request.Context(), &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusCreated, result)
}

func (s *Server) getResult(c *gin.Context) {
	id := c.Param("id")

	result, err := s.resultStore.GetByID(c.Request.Context(), id)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", "Result not found")
		return
	}

	s.successResponse(c, http.StatusOK, result)
}

func (s *Server) storeBatch(c *gin.Context) {
	var req struct {
		Results []types.StoreResultRequest `json:"results" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	results, err := s.resultStore.StoreBatch(c.Request.Context(), req.Results)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusCreated, gin.H{
		"stored": len(results),
		"results": results,
	})
}

func (s *Server) getPatientResults(c *gin.Context) {
	patientID := c.Param("patientId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	results, total, err := s.resultStore.GetByPatient(c.Request.Context(), patientID, limit, offset)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponseWithMeta(c, http.StatusOK, results, &types.APIMeta{
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (s *Server) getPatientResultsByCode(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "365"))

	results, err := s.resultStore.GetByPatientAndCode(c.Request.Context(), patientID, code, days)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, results)
}

// =============================================================================
// INTERPRETATION HANDLERS
// =============================================================================

func (s *Server) interpretResult(c *gin.Context) {
	var req types.InterpretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	ctx := c.Request.Context()

	// Get patient context if not provided
	patientCtx := req.PatientContext
	if patientCtx == nil && s.kb2Client != nil {
		intCtx, err := s.kb2Client.GetPatientContext(ctx, req.Result.PatientID)
		if err == nil && intCtx != nil {
			patientCtx = s.convertPatientContext(intCtx)
		}
	}

	result, err := s.interpreter.Interpret(ctx, &req.Result, patientCtx)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "INTERPRET_ERROR", err.Error())
		return
	}

	// Add trending if requested
	if req.IncludeTrending {
		trend, _ := s.trendEngine.AnalyzeMultiWindow(ctx, req.Result.PatientID, req.Result.Code)
		if len(trend) > 0 {
			// Use the 30-day window as default
			if t, ok := trend["30d"]; ok {
				result.Trending = t
			}
		}
	}

	// Add baseline comparison if requested
	if req.IncludeBaseline {
		if req.Result.ValueNumeric != nil {
			result.BaselineCompare, _ = s.baselineTracker.CompareToBaseline(ctx, req.Result.PatientID, req.Result.Code, *req.Result.ValueNumeric)
		}
	}

	RecordInterpretation()
	if result.Interpretation.IsCritical {
		RecordCriticalValue()
	}
	if result.Interpretation.IsPanic {
		RecordPanicValue()
	}

	// Emit governance events for critical/panic values (Tier-7)
	if s.govObserver != nil {
		provenance := s.buildProvenance(&req.Result, patientCtx)
		go func() {
			if err := s.govObserver.OnInterpretation(context.Background(), result, provenance); err != nil {
				s.log.WithError(err).Warn("Failed to emit governance event")
			}
		}()
	}

	s.successResponse(c, http.StatusOK, result)
}

func (s *Server) interpretBatch(c *gin.Context) {
	var req types.BatchInterpretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	ctx := c.Request.Context()
	results := make([]types.InterpretedResult, 0, len(req.Results))

	for _, labResult := range req.Results {
		interpreted, err := s.interpreter.Interpret(ctx, &labResult, req.PatientContext)
		if err != nil {
			s.log.WithError(err).Warn("Failed to interpret result")
			continue
		}
		results = append(results, *interpreted)
		RecordInterpretation()
	}

	// Emit governance events for batch results (Tier-7)
	if s.govObserver != nil && len(results) > 0 {
		provenance := s.buildProvenance(&req.Results[0], req.PatientContext)
		go func() {
			if err := s.govObserver.OnBatchInterpretation(context.Background(), results, provenance); err != nil {
				s.log.WithError(err).Warn("Failed to emit batch governance events")
			}
		}()
	}

	s.successResponse(c, http.StatusOK, gin.H{
		"interpreted": len(results),
		"results":     results,
	})
}

// =============================================================================
// TRENDING HANDLERS
// =============================================================================

func (s *Server) getTrend(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	window := c.DefaultQuery("window", "30d")

	// Convert window to days
	windowDays := 30
	if days, ok := trending.StandardWindows[window]; ok {
		windowDays = days.Days
	}

	trend, err := s.trendEngine.AnalyzeTrend(c.Request.Context(), patientID, code, windowDays)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "TREND_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, trend)
}

func (s *Server) getMultiWindowTrend(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")

	trend, err := s.trendEngine.AnalyzeMultiWindow(c.Request.Context(), patientID, code)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "TREND_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, trend)
}

func (s *Server) getAllTrends(c *gin.Context) {
	patientID := c.Param("patientId")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	trends, err := s.trendEngine.GetAllTrends(c.Request.Context(), patientID, days)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "TREND_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, trends)
}

// getTrendEnhanced returns trend analysis with clinical context and multi-horizon predictions
func (s *Server) getTrendEnhanced(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	window := c.DefaultQuery("window", "30d")

	// Convert window to days
	windowDays := 30
	if days, ok := trending.StandardWindows[window]; ok {
		windowDays = days.Days
	}

	enhanced, err := s.trendEngine.AnalyzeTrendEnhanced(c.Request.Context(), patientID, code, windowDays)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "TREND_ERROR", err.Error())
		return
	}

	if enhanced == nil {
		s.errorResponse(c, http.StatusNotFound, "INSUFFICIENT_DATA", "Not enough data points for trend analysis")
		return
	}

	s.successResponse(c, http.StatusOK, enhanced)
}

// getMultiWindowTrendEnhanced returns comprehensive multi-window analysis with clinical intelligence
func (s *Server) getMultiWindowTrendEnhanced(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")

	analysis, err := s.trendEngine.AnalyzeMultiWindowEnhanced(c.Request.Context(), patientID, code)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "TREND_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, analysis)
}

// getPredictions returns multi-horizon predictions with confidence intervals
func (s *Server) getPredictions(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	window := c.DefaultQuery("window", "90d")

	// Get historical data for predictions
	windowDays := 90
	if days, ok := trending.StandardWindows[window]; ok {
		windowDays = days.Days
	}

	// Get trend analysis which includes predictions
	trend, err := s.trendEngine.AnalyzeTrend(c.Request.Context(), patientID, code, windowDays)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "TREND_ERROR", err.Error())
		return
	}

	if trend == nil || len(trend.DataPoints) < 4 {
		s.errorResponse(c, http.StatusBadRequest, "INSUFFICIENT_DATA",
			"At least 4 data points required for predictions")
		return
	}

	// Generate multi-horizon predictions
	predEngine := trending.NewPredictionEngine()
	predictions := predEngine.PredictMultiHorizon(trend.DataPoints)

	if predictions == nil {
		s.errorResponse(c, http.StatusBadRequest, "PREDICTION_FAILED",
			"Unable to generate predictions from available data")
		return
	}

	// Add lab context to response
	labContext, _ := trending.GetLabContext(code)

	s.successResponse(c, http.StatusOK, gin.H{
		"patient_id":   patientID,
		"test_code":    code,
		"predictions":  predictions,
		"lab_context":  labContext,
		"data_points":  len(trend.DataPoints),
		"window_days":  windowDays,
	})
}

// getLabContext returns clinical interpretation context for a lab test
func (s *Server) getLabContext(c *gin.Context) {
	code := c.Param("code")

	labContext, found := trending.GetLabContext(code)
	if !found {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND",
			"No clinical context available for lab code: "+code)
		return
	}

	// Also return the volatility threshold
	volatilityThreshold := trending.GetVolatilityThreshold(code)

	s.successResponse(c, http.StatusOK, gin.H{
		"lab_context":          labContext,
		"volatility_threshold": volatilityThreshold,
	})
}

// =============================================================================
// BASELINE HANDLERS
// =============================================================================

func (s *Server) getPatientBaselines(c *gin.Context) {
	patientID := c.Param("patientId")

	baselines, err := s.baselineTracker.GetAllBaselines(c.Request.Context(), patientID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "BASELINE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, baselines)
}

func (s *Server) getBaseline(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")

	baseline, err := s.baselineTracker.GetBaseline(c.Request.Context(), patientID, code)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", "Baseline not found")
		return
	}

	s.successResponse(c, http.StatusOK, baseline)
}

func (s *Server) setManualBaseline(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")

	var req types.SetBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	baseline, err := s.baselineTracker.SetManualBaseline(c.Request.Context(), patientID, code, req.Mean, req.StdDev, req.Notes)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "BASELINE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, baseline)
}

func (s *Server) calculateBaseline(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "365"))

	baseline, err := s.baselineTracker.CalculateBaseline(c.Request.Context(), patientID, code, days)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "BASELINE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, baseline)
}

// =============================================================================
// PANEL HANDLERS
// =============================================================================

func (s *Server) listPanels(c *gin.Context) {
	panels := s.panelManager.ListPanelDefinitions()
	s.successResponse(c, http.StatusOK, panels)
}

func (s *Server) getPanelDefinition(c *gin.Context) {
	panelType := types.PanelType(c.Param("type"))

	panel, err := s.panelManager.GetPanelDefinition(panelType)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", "Panel type not found")
		return
	}

	s.successResponse(c, http.StatusOK, panel)
}

func (s *Server) assemblePanel(c *gin.Context) {
	patientID := c.Param("patientId")
	panelType := types.PanelType(c.Param("type"))
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	ctx := c.Request.Context()

	// Assemble panel using the manager
	panel, err := s.panelManager.AssemblePanel(ctx, patientID, panelType, days)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "ASSEMBLY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, panel)
}

func (s *Server) detectAvailablePanels(c *gin.Context) {
	patientID := c.Param("patientId")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	available, err := s.panelManager.DetectAvailablePanels(c.Request.Context(), patientID, days)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, available)
}

// =============================================================================
// REVIEW HANDLERS
// =============================================================================

func (s *Server) getPendingReviews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	priority := c.Query("priority")

	filters := types.PendingReviewFilters{
		Limit:    limit,
		Page:     page,
		Priority: priority,
	}

	pending, total, err := s.reviewService.GetPendingReviews(c.Request.Context(), filters)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponseWithMeta(c, http.StatusOK, pending, &types.APIMeta{
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}

func (s *Server) getCriticalQueue(c *gin.Context) {
	critical, err := s.reviewService.GetCriticalQueue(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, critical)
}

func (s *Server) acknowledgeResult(c *gin.Context) {
	var req types.AcknowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	err := s.reviewService.Acknowledge(c.Request.Context(), req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "ACKNOWLEDGE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, gin.H{"acknowledged": true})
}

func (s *Server) completeReview(c *gin.Context) {
	var req types.CompleteReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	err := s.reviewService.CompleteReview(c.Request.Context(), req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "REVIEW_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, gin.H{"completed": true})
}

func (s *Server) getReviewStats(c *gin.Context) {
	stats, err := s.reviewService.GetReviewStats(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, stats)
}

// =============================================================================
// VISUALIZATION HANDLERS
// =============================================================================

func (s *Server) getChartData(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	window := c.DefaultQuery("window", "30d")

	chart, err := s.vizService.GenerateChartData(c.Request.Context(), patientID, code, window)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "CHART_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, chart)
}

func (s *Server) getSparkline(c *gin.Context) {
	patientID := c.Param("patientId")
	code := c.Param("code")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	sparkline, err := s.vizService.GenerateSparkline(c.Request.Context(), patientID, code, days)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "SPARKLINE_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, sparkline)
}

func (s *Server) getDashboard(c *gin.Context) {
	patientID := c.Param("patientId")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	dashboard, err := s.vizService.GenerateDashboard(c.Request.Context(), patientID, days)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "DASHBOARD_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, dashboard)
}

// =============================================================================
// REFERENCE DATA HANDLERS
// =============================================================================

func (s *Server) listTests(c *gin.Context) {
	category := c.Query("category")
	tests := s.refDB.ListTests(category)
	s.successResponse(c, http.StatusOK, tests)
}

func (s *Server) getTestDefinition(c *gin.Context) {
	code := c.Param("code")

	test := s.refDB.GetTest(code)
	if test == nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", "Test not found")
		return
	}

	s.successResponse(c, http.StatusOK, test)
}

// =============================================================================
// CARE GAP HANDLERS (KB-9 Integration)
// =============================================================================

func (s *Server) getPatientCareGaps(c *gin.Context) {
	patientID := c.Param("patientId")

	if s.kb9Client == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "KB9_UNAVAILABLE", "KB-9 Care Gaps service not configured")
		return
	}

	// Get measures from query param or default to all lab-related measures
	measuresParam := c.QueryArray("measures")
	var measures []integration.MeasureType
	if len(measuresParam) > 0 {
		for _, m := range measuresParam {
			measures = append(measures, integration.MeasureType(m))
		}
	} else {
		// Default to lab-related measures
		measures = []integration.MeasureType{
			integration.MeasureCMS122DiabetesHbA1c,
			integration.MeasureCMS165BPControl,
			integration.MeasureCMS69BMIScreening,
		}
	}

	report, err := s.kb9Client.GetPatientCareGaps(c.Request.Context(), patientID, measures)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "CARE_GAP_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, report)
}

func (s *Server) identifyLabCareGaps(c *gin.Context) {
	patientID := c.Param("patientId")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "365"))

	if s.kb9Client == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "KB9_UNAVAILABLE", "KB-9 Care Gaps service not configured")
		return
	}

	ctx := c.Request.Context()

	// Get patient lab history
	results, _, err := s.resultStore.GetByPatient(ctx, patientID, 100, 0)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	// Filter to requested time window and convert to lab history
	cutoff := time.Now().AddDate(0, 0, -days)
	labHistory := make([]integration.LabHistoryEntry, 0)
	for _, r := range results {
		if r.CollectedAt.After(cutoff) && r.ValueNumeric != nil {
			labHistory = append(labHistory, integration.LabHistoryEntry{
				Code:        r.Code,
				Value:       *r.ValueNumeric,
				Unit:        r.Unit,
				CollectedAt: r.CollectedAt,
			})
		}
	}

	// Identify care gaps based on lab history
	gaps, err := s.kb9Client.IdentifyLabCareGaps(ctx, patientID, labHistory)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "CARE_GAP_ERROR", err.Error())
		return
	}

	// Emit governance events for identified care gaps (Tier-7)
	if s.govObserver != nil && len(gaps) > 0 {
		go func() {
			for _, gap := range gaps {
				if err := s.govObserver.OnCareGapIdentified(context.Background(), patientID, string(gap.MeasureType), gap.LabName, gap.DaysOverdue, nil); err != nil {
					s.log.WithError(err).Warn("Failed to emit care gap governance event")
				}
			}
		}()
	}

	s.successResponse(c, http.StatusOK, gin.H{
		"patient_id":     patientID,
		"gaps_identified": len(gaps),
		"gaps":           gaps,
		"lab_history_analyzed": len(labHistory),
	})
}

func (s *Server) reportCareGap(c *gin.Context) {
	var req integration.LabBasedCareGap
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if s.kb9Client == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "KB9_UNAVAILABLE", "KB-9 Care Gaps service not configured")
		return
	}

	err := s.kb9Client.ReportLabBasedCareGap(c.Request.Context(), &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "REPORT_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, gin.H{"reported": true})
}

// =============================================================================
// FHIR HANDLERS
// =============================================================================

func (s *Server) searchObservations(c *gin.Context) {
	patientID := c.Query("patient")
	code := c.Query("code")

	if patientID == "" {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "patient parameter required")
		return
	}

	ctx := c.Request.Context()
	var results []types.LabResult
	var err error

	if code != "" {
		results, err = s.resultStore.GetByPatientAndCode(ctx, patientID, code, 365)
	} else {
		results, _, err = s.resultStore.GetByPatient(ctx, patientID, 100, 0)
	}

	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	observations := make([]*fhir.Observation, len(results))
	for i, r := range results {
		observations[i] = s.fhirMapper.ToObservation(&r, nil)
	}

	c.JSON(http.StatusOK, gin.H{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        len(observations),
		"entry":        observations,
	})
}

func (s *Server) getObservation(c *gin.Context) {
	id := c.Param("id")

	result, err := s.resultStore.GetByID(c.Request.Context(), id)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", "Observation not found")
		return
	}

	obs := s.fhirMapper.ToObservation(result, nil)
	c.JSON(http.StatusOK, obs)
}

func (s *Server) getDiagnosticReport(c *gin.Context) {
	patientID := c.Param("patientId")
	panelType := types.PanelType(c.Param("panelType"))

	ctx := c.Request.Context()

	// Get recent results
	results, err := s.resultStore.GetRecentByPatient(ctx, patientID, 7)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	// Interpret and assemble panel
	interpreted := make([]types.InterpretedResult, 0)
	for _, r := range results {
		result, _ := s.interpreter.Interpret(ctx, &r, nil)
		if result != nil {
			interpreted = append(interpreted, *result)
		}
	}

	panel, err := s.panelManager.AssemblePanel(ctx, patientID, panelType, 7)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "ASSEMBLY_ERROR", err.Error())
		return
	}

	report := s.fhirMapper.ToDiagnosticReport(panel)
	c.JSON(http.StatusOK, report)
}

// =============================================================================
// RESPONSE HELPERS
// =============================================================================

func (s *Server) successResponse(c *gin.Context, status int, data interface{}) {
	c.JSON(status, types.APIResponse{
		Success: true,
		Data:    data,
	})
}

func (s *Server) successResponseWithMeta(c *gin.Context, status int, data interface{}, meta *types.APIMeta) {
	c.JSON(status, types.APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func (s *Server) errorResponse(c *gin.Context, status int, code, message string) {
	c.JSON(status, types.APIResponse{
		Success: false,
		Error: &types.APIError{
			Code:    code,
			Message: message,
		},
	})
}

// =============================================================================
// TYPE CONVERSION HELPERS
// =============================================================================

// buildProvenance creates a provenance record for governance events
func (s *Server) buildProvenance(result *types.LabResult, patientCtx *types.PatientContext) *governance.EventProvenance {
	builder := governance.NewProvenanceBuilder("1.0.0")

	// Add reference range used
	if s.refDB != nil {
		var age int
		var sex string
		if patientCtx != nil {
			age = patientCtx.Age
			sex = patientCtx.Sex
		}
		ranges := s.refDB.GetRanges(result.Code, age, sex)
		if ranges != nil {
			builder.AddReferenceRange(
				result.Code,
				"KB-16",
				"1.0.0",
				ranges.Low,
				ranges.High,
				ranges.CriticalLow,
				ranges.CriticalHigh,
				age > 0,
				sex != "",
			)
		}
	}

	return builder.Build()
}

// convertPatientContext converts integration.PatientContext to types.PatientContext
func (s *Server) convertPatientContext(intCtx *integration.PatientContext) *types.PatientContext {
	if intCtx == nil {
		return nil
	}

	// Convert conditions
	conditions := make([]types.Condition, len(intCtx.Conditions))
	for i, c := range intCtx.Conditions {
		conditions[i] = types.Condition{
			Code:   c.Code,
			Name:   c.Display,
			System: "SNOMED", // Default system
		}
	}

	// Convert medications
	medications := make([]types.Medication, len(intCtx.Medications))
	for i, m := range intCtx.Medications {
		medications[i] = types.Medication{
			RxNormCode: m.Code,
			Name:       m.Display,
		}
	}

	return &types.PatientContext{
		PatientID:   intCtx.PatientID,
		Age:         intCtx.Age,
		Sex:         intCtx.Sex,
		Conditions:  conditions,
		Medications: medications,
		Phenotypes:  intCtx.Phenotypes,
	}
}

// =============================================================================
// LOINC REFERENCE RANGE HANDLERS
// =============================================================================
// These endpoints expose the loinc_reference_ranges table from the shared
// canonical_facts database. The Context Router uses these to evaluate DDI
// context thresholds against patient lab values.
// =============================================================================

// getLOINCReferenceRange retrieves reference range for a LOINC code
// GET /api/v1/loinc/reference-ranges/:code
func (s *Server) getLOINCReferenceRange(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_LOINC_CODE", "LOINC code is required")
		return
	}

	ref, err := s.loincRepo.GetByLOINCCode(c.Request.Context(), code)
	if err != nil {
		s.log.WithError(err).WithField("loinc_code", code).Error("Failed to get LOINC reference range")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	if ref == nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("No reference range found for LOINC code: %s", code))
		return
	}

	s.successResponse(c, http.StatusOK, ref.ToResponse())
}

// getLOINCReferenceRangeWithContext retrieves reference range with age/sex specificity
// GET /api/v1/loinc/reference-ranges/:code/context?age=45&sex=male
func (s *Server) getLOINCReferenceRangeWithContext(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_LOINC_CODE", "LOINC code is required")
		return
	}

	// Parse optional age and sex parameters
	age := 0
	if ageStr := c.Query("age"); ageStr != "" {
		if parsed, err := strconv.Atoi(ageStr); err == nil {
			age = parsed
		}
	}
	sex := c.DefaultQuery("sex", "all")

	ref, err := s.loincRepo.GetByLOINCCodeWithContext(c.Request.Context(), code, age, sex)
	if err != nil {
		s.log.WithError(err).WithField("loinc_code", code).Error("Failed to get LOINC reference range with context")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	if ref == nil {
		s.errorResponse(c, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("No reference range found for LOINC code: %s", code))
		return
	}

	s.successResponse(c, http.StatusOK, ref.ToResponse())
}

// getDDIRelevantLOINCRanges retrieves all LOINC codes relevant for DDI context evaluation
// GET /api/v1/loinc/ddi-relevant
func (s *Server) getDDIRelevantLOINCRanges(c *gin.Context) {
	refs, err := s.loincRepo.GetDDIRelevantRanges(c.Request.Context())
	if err != nil {
		s.log.WithError(err).Error("Failed to get DDI-relevant LOINC ranges")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	// Convert to response format
	responses := make([]*loinc.LOINCReferenceResponse, len(refs))
	for i := range refs {
		responses[i] = refs[i].ToResponse()
	}

	s.successResponseWithMeta(c, http.StatusOK, responses, &types.APIMeta{
		Total: len(responses),
	})
}

// getLOINCCategories retrieves all LOINC categories
// GET /api/v1/loinc/categories
func (s *Server) getLOINCCategories(c *gin.Context) {
	categories, err := s.loincRepo.ListCategories(c.Request.Context())
	if err != nil {
		s.log.WithError(err).Error("Failed to list LOINC categories")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, gin.H{
		"categories": categories,
		"count":      len(categories),
	})
}

// getLOINCByCategory retrieves all LOINC codes for a category
// GET /api/v1/loinc/categories/:category
func (s *Server) getLOINCByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_CATEGORY", "Category is required")
		return
	}

	refs, err := s.loincRepo.GetByCategory(c.Request.Context(), category)
	if err != nil {
		s.log.WithError(err).WithField("category", category).Error("Failed to get LOINC by category")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	// Convert to response format
	responses := make([]*loinc.LOINCReferenceResponse, len(refs))
	for i := range refs {
		responses[i] = refs[i].ToResponse()
	}

	s.successResponseWithMeta(c, http.StatusOK, responses, &types.APIMeta{
		Total: len(responses),
	})
}

// searchLOINC searches LOINC codes by component name
// GET /api/v1/loinc/search?q=potassium&limit=50
func (s *Server) searchLOINC(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		s.errorResponse(c, http.StatusBadRequest, "INVALID_QUERY", "Search query 'q' is required")
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	refs, err := s.loincRepo.SearchByName(c.Request.Context(), query, limit)
	if err != nil {
		s.log.WithError(err).WithField("query", query).Error("Failed to search LOINC")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	// Convert to response format
	responses := make([]*loinc.LOINCReferenceResponse, len(refs))
	for i := range refs {
		responses[i] = refs[i].ToResponse()
	}

	s.successResponseWithMeta(c, http.StatusOK, responses, &types.APIMeta{
		Total: len(responses),
	})
}

// getLOINCStats retrieves LOINC repository statistics
// GET /api/v1/loinc/stats
func (s *Server) getLOINCStats(c *gin.Context) {
	stats, err := s.loincRepo.GetStatistics(c.Request.Context())
	if err != nil {
		s.log.WithError(err).Error("Failed to get LOINC stats")
		s.errorResponse(c, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	s.successResponse(c, http.StatusOK, stats)
}
