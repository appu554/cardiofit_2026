package api

import (
	"net/http"
	"time"

	"kb-21-behavioral-intelligence/internal/cache"
	"kb-21-behavioral-intelligence/internal/config"
	"kb-21-behavioral-intelligence/internal/database"
	"kb-21-behavioral-intelligence/internal/events"
	"kb-21-behavioral-intelligence/internal/metrics"
	"kb-21-behavioral-intelligence/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server is the HTTP server for KB-21 Behavioral Intelligence Service.
type Server struct {
	Router             *gin.Engine
	config             *config.Config
	db                 *database.Database
	cache              *cache.RedisClient
	metrics            *metrics.Collector
	logger             *zap.Logger
	adherenceService   *services.AdherenceService
	engagementService  *services.EngagementService
	correlationService *services.CorrelationService
	hypoRiskService    *services.HypoRiskService
	festivalCalendar   *services.FestivalCalendar
	nudgeEngine        *services.NudgeEngine
	coldStartEngine    *services.ColdStartEngine
	gamificationEngine *services.GamificationEngine
	timingBandit       *services.TimingBandit
	eventSubscriber    *events.Subscriber
}

// NewServer creates and configures the HTTP server with all dependencies.
func NewServer(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.RedisClient,
	metricsCollector *metrics.Collector,
	logger *zap.Logger,
	adherenceSvc *services.AdherenceService,
	engagementSvc *services.EngagementService,
	correlationSvc *services.CorrelationService,
	hypoRiskSvc *services.HypoRiskService,
	festivalCal *services.FestivalCalendar,
	nudgeEngine *services.NudgeEngine,
	coldStartEngine *services.ColdStartEngine,
	gamificationEngine *services.GamificationEngine,
	timingBandit *services.TimingBandit,
	subscriber *events.Subscriber,
) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		Router:             router,
		config:             cfg,
		db:                 db,
		cache:              cacheClient,
		metrics:            metricsCollector,
		logger:             logger,
		adherenceService:   adherenceSvc,
		engagementService:  engagementSvc,
		correlationService: correlationSvc,
		hypoRiskService:    hypoRiskSvc,
		festivalCalendar:   festivalCal,
		nudgeEngine:        nudgeEngine,
		coldStartEngine:    coldStartEngine,
		gamificationEngine: gamificationEngine,
		timingBandit:       timingBandit,
		eventSubscriber:    subscriber,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.Router.Use(s.requestLogger())
	s.Router.Use(s.metricsMiddleware())
	s.Router.Use(s.corsMiddleware())
}

func (s *Server) setupRoutes() {
	// Infrastructure endpoints
	s.Router.GET("/health", s.healthCheck)
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Root-level compatibility routes for inter-KB communication.
	// KB-23 constructs URLs as {KB21_URL}/patient/{id}/adherence/htn
	// where KB21_URL is the bare host (e.g. http://localhost:8133),
	// so these routes must exist without the /api/v1 prefix.
	s.Router.GET("/patient/:patient_id/adherence/htn", s.getHTNAdherence)
	s.Router.GET("/patient/:patient_id/adherence/htn/gate", s.getHTNAdherenceGate)

	// Festival status — root-level for KB-20 perturbation integration (P4)
	s.Router.GET("/festival-status", s.getFestivalStatus)

	// API v1
	v1 := s.Router.Group("/api/v1")
	{
		// Interaction recording
		v1.POST("/patient/:patient_id/interaction", s.recordInteraction)

		// Adherence endpoints
		adherence := v1.Group("/patient/:patient_id/adherence")
		{
			adherence.GET("", s.getAdherence)
			adherence.POST("/recompute", s.recomputeAdherence)

			// Antihypertensive adherence (Amendment 4, Wave 2)
			// KB-23 card_builder consumes these for HTN decision card gating.
			adherence.GET("/htn", s.getHTNAdherence)
			adherence.GET("/htn/gate", s.getHTNAdherenceGate)
		}

		// Adherence weights for KB-22 (Finding F-06)
		v1.GET("/patient/:patient_id/adherence-weights", s.getAdherenceWeights)

		// Engagement endpoints
		engagement := v1.Group("/patient/:patient_id/engagement")
		{
			engagement.GET("", s.getEngagementProfile)
			engagement.POST("/recompute", s.recomputeEngagement)
		}

		// Loop trust for V-MCU (Finding F-01)
		v1.GET("/patient/:patient_id/loop-trust", s.getLoopTrust)

		// Outcome correlation (Finding F-04)
		correlation := v1.Group("/patient/:patient_id/outcome-correlation")
		{
			correlation.GET("", s.getLatestCorrelation)
			correlation.GET("/history", s.getCorrelationHistory)
		}

		// Hypoglycemia risk (Finding F-03)
		v1.GET("/patient/:patient_id/hypo-risk", s.evaluateHypoRisk)

		// Answer reliability for KB-22 HPI session (R-03)
		v1.GET("/patient/:patient_id/answer-reliability", s.getAnswerReliability)

		// Event webhook endpoints (dev mode — Kafka replacement)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/lab-result", s.webhookLabResult)
			webhooks.POST("/medication-changed", s.webhookMedicationChanged)
		}

		// Festival status (P4 perturbation integration for KB-20)
		v1.GET("/festival-status", s.getFestivalStatus)

		// Analytics endpoints (Finding F-11)
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/phenotype-distribution", s.getPhenotypeDistribution)
			analytics.GET("/question-effectiveness", s.getQuestionEffectiveness)
			analytics.GET("/cohort", s.getCohortSnapshots)
		}

		// BCE v1.0 Nudge Engine
		nudge := v1.Group("/patient/:patient_id/nudge")
		{
			nudge.POST("/select", s.selectNudge)
			nudge.POST("/outcome", s.observeNudgeOutcome)
		}
		v1.GET("/patient/:patient_id/techniques", s.getTechniqueEffectiveness)
		v1.GET("/patient/:patient_id/motivation-phase", s.getMotivationPhase)

		// Cold-start phenotype (E1)
		v1.POST("/patient/:patient_id/intake-profile", s.submitIntakeProfile)
		v1.GET("/patient/:patient_id/cold-start-phenotype", s.getColdStartPhenotype)

		// Gamification (E2)
		v1.GET("/patient/:patient_id/streaks", s.getPatientStreaks)
		v1.GET("/patient/:patient_id/milestones", s.getPatientMilestones)

		// Timing optimization (E4)
		v1.GET("/patient/:patient_id/optimal-timing", s.getOptimalDeliveryTime)
	}
}

// --- Infrastructure handlers ---

func (s *Server) healthCheck(c *gin.Context) {
	checks := map[string]interface{}{}

	dbStatus := "healthy"
	if err := s.db.HealthCheck(); err != nil {
		dbStatus = "unhealthy"
	}
	checks["database"] = map[string]string{"status": dbStatus}

	if s.cache != nil {
		cacheStatus := "healthy"
		if err := s.cache.Ping(); err != nil {
			cacheStatus = "unhealthy"
		}
		checks["cache"] = map[string]string{"status": cacheStatus}
	}

	overallStatus := "healthy"
	if dbStatus == "unhealthy" {
		overallStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   s.config.ServiceName,
		"version":   "1.0.0",
		"checks":    checks,
	})
}

// --- Middleware ---

func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		s.logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
		}
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// --- Response helpers ---

func sendSuccess(c *gin.Context, data interface{}, metadata map[string]interface{}) {
	resp := gin.H{
		"success": true,
		"data":    data,
	}
	if metadata != nil {
		resp["metadata"] = metadata
	}
	c.JSON(http.StatusOK, resp)
}

func sendError(c *gin.Context, statusCode int, message, code string, details map[string]interface{}) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// --- Festival status handler (P4 perturbation) ---

// getFestivalStatus returns the active festival window for a region, if any.
// Used by KB-20 to populate P4 (festival fasting) perturbation fields.
// GET /festival-status?region=NORTH (default: ALL)
func (s *Server) getFestivalStatus(c *gin.Context) {
	if s.festivalCalendar == nil {
		c.JSON(http.StatusOK, gin.H{
			"active": false,
		})
		return
	}

	region := c.DefaultQuery("region", "ALL")
	now := time.Now().UTC()

	window := s.festivalCalendar.GetActiveFestival(now, region)
	if window == nil {
		c.JSON(http.StatusOK, gin.H{
			"active": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":       true,
		"name":         window.Name,
		"fasting_type": string(window.FastingType),
		"start":        window.Start.Format(time.RFC3339),
		"end":          window.End.Format(time.RFC3339),
		"core_start":   window.CoreStart.Format(time.RFC3339),
		"core_end":     window.CoreEnd.Format(time.RFC3339),
	})
}
