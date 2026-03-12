package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/services"
)

type Server struct {
	Router  *gin.Engine
	Config  *config.Config
	DB      *database.Database
	Cache   *cache.CacheClient
	Metrics *metrics.Collector
	Log     *zap.Logger

	NodeLoader             *services.NodeLoader
	SessionService         *services.SessionService
	BayesianEngine         *services.BayesianEngine
	SafetyEngine           *services.SafetyEngine
	QuestionOrchestrator   *services.QuestionOrchestrator
	CMApplicator           *services.CMApplicator
	SessionContextProvider *services.SessionContextProvider
	GuidelineClient        *services.GuidelineClient
	MedicationSafety       *services.MedicationSafetyProvider
	TelemetryWriter        *services.TelemetryWriter
	OutcomePublisher       *services.OutcomePublisher
	CalibrationManager     *services.CalibrationManager
	CrossNodeSafety        *services.CrossNodeSafety
	ContradictionDetector  *services.ContradictionDetector
	TransitionEvaluator    *services.TransitionEvaluator

	// Gap-fix additions (CC-1, BAY-10, BAY-11, E01, E03)
	SCEService         *services.SCEService
	ExpertPanelService *services.ExpertPanelService
	EventPublisher     *services.EventPublisherFacade
	KafkaPublisher     services.KafkaPublisher
	AcuityScorer       *services.AcuityScorer
	TierCService       *services.TierCService
}

func NewServer(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.CacheClient,
	metricsCollector *metrics.Collector,
	log *zap.Logger,
	nodeLoader *services.NodeLoader,
) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	if cfg.IsDevelopment() {
		router.Use(gin.Logger())
	}
	router.Use(corsMiddleware())
	router.Use(metricsMiddleware(metricsCollector))

	return &Server{
		Router: router, Config: cfg, DB: db,
		Cache: cacheClient, Metrics: metricsCollector,
		Log: log, NodeLoader: nodeLoader,
	}
}

func (s *Server) InitServices() {
	s.CMApplicator = services.NewCMApplicator(s.Log)
	s.BayesianEngine = services.NewBayesianEngine(s.Log, s.Metrics)
	s.SafetyEngine = services.NewSafetyEngine(s.Log, s.Metrics)
	s.QuestionOrchestrator = services.NewQuestionOrchestrator(s.Log, s.Metrics)
	s.SessionContextProvider = services.NewSessionContextProvider(s.Config, s.Log, s.Metrics)
	s.GuidelineClient = services.NewGuidelineClient(s.Config, s.Log)
	s.MedicationSafety = services.NewMedicationSafetyProvider(s.Config, s.Log)
	s.TelemetryWriter = services.NewTelemetryWriter(s.Config, s.Log)
	s.OutcomePublisher = services.NewOutcomePublisher(s.Config, s.Log, s.Metrics)
	s.CalibrationManager = services.NewCalibrationManager(s.DB, s.Cache, s.Log, s.Metrics)
	s.CrossNodeSafety = services.NewCrossNodeSafety(s.Config.NodesDir, s.Log)
	s.ContradictionDetector = services.NewContradictionDetector(s.Log)
	s.TransitionEvaluator = services.NewTransitionEvaluator(s.Log)
	s.AcuityScorer = services.NewAcuityScorer(s.Log)

	// BAY-11: Kafka publishing
	s.KafkaPublisher = s.initKafkaPublisher()
	s.EventPublisher = services.NewEventPublisherFacade(s.KafkaPublisher, s.Log)

	// CC-1: Safety Constraint Engine (in-process sidecar)
	s.SCEService = services.NewSCEService(s.SafetyEngine, s.NodeLoader, s.KafkaPublisher, s.Log)

	// E01: Expert Panel calibration workflow
	s.ExpertPanelService = services.NewExpertPanelService(s.DB, s.Cache, s.Log, s.Metrics)

	// E03: Tier C data-driven calibration (Month 18+, N≥200)
	s.TierCService = services.NewTierCService(s.DB, s.Cache, s.Log, s.Metrics)

	s.SessionService = services.NewSessionService(
		s.DB, s.Cache, s.Log, s.Metrics,
		s.NodeLoader, s.BayesianEngine, s.SafetyEngine,
		s.QuestionOrchestrator, s.CMApplicator,
		s.SessionContextProvider, s.GuidelineClient,
		s.MedicationSafety, s.TelemetryWriter, s.OutcomePublisher,
		s.CrossNodeSafety,
		s.ContradictionDetector, s.TransitionEvaluator,
	)
}

func (s *Server) initKafkaPublisher() services.KafkaPublisher {
	if s.Config.KafkaEnabled {
		pub, err := services.NewKafkaGoPublisher(
			s.Config.KafkaBootstrapServers,
			s.Config.KafkaClientID,
			s.Log,
		)
		if err != nil {
			s.Log.Error("BAY-11: failed to init Kafka publisher, falling back to log-only",
				zap.Error(err),
			)
		} else {
			s.Log.Info("BAY-11: Kafka publisher enabled",
				zap.String("bootstrap", s.Config.KafkaBootstrapServers),
			)
			return pub
		}
	} else {
		s.Log.Info("BAY-11: Kafka publisher disabled (log-only mode)")
	}
	return services.NewLogOnlyPublisher(s.Log)
}

func (s *Server) RegisterRoutes() {
	s.Router.GET("/health", s.healthHandler)
	s.Router.GET("/readiness", s.readinessHandler)
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	s.Router.POST("/internal/nodes/reload", s.reloadNodesHandler)

	v1 := s.Router.Group("/api/v1")
	{
		v1.POST("/sessions", s.createSessionHandler)
		v1.GET("/sessions/:id", s.getSessionHandler)
		v1.POST("/sessions/:id/answers", s.submitAnswerHandler)
		v1.POST("/sessions/:id/suspend", s.suspendSessionHandler)
		v1.POST("/sessions/:id/resume", s.resumeSessionHandler)
		v1.POST("/sessions/:id/complete", s.completeSessionHandler)
		v1.GET("/sessions/:id/differential", s.getDifferentialHandler)
		v1.GET("/sessions/:id/safety", s.getSafetyFlagsHandler)
		v1.GET("/snapshots/:session_id", s.getSnapshotHandler)
		v1.GET("/nodes", s.listNodesHandler)
		v1.GET("/nodes/:node_id", s.getNodeHandler)
		v1.POST("/calibration/feedback", s.calibrationFeedbackHandler)
		v1.GET("/calibration/status/:node_id", s.calibrationStatusHandler)
		v1.POST("/calibration/import-golden", s.importGoldenHandler)

		// BAY-10: SCE escalation webhook, multi-complaint init, CI/CD node validation
		v1.POST("/session/escalate", s.escalateSessionHandler)
		v1.POST("/session/multi-init", s.multiInitHandler)
		v1.POST("/node/validate", s.validateNodeHandler)

		// E01: Expert panel calibration endpoints
		v1.POST("/calibration/expert-review", s.expertReviewHandler)
		v1.GET("/calibration/expert-history/:node_id", s.expertReviewHistoryHandler)

		// E03: Tier C data-driven calibration endpoints
		v1.POST("/calibration/tier-c/compute", s.tierCComputeHandler)
		v1.POST("/calibration/tier-c/approve", s.tierCApproveHandler)
	}
}

func (s *Server) healthHandler(c *gin.Context) {
	checks := map[string]string{}
	healthy := true
	if err := s.DB.Health(); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		checks["database"] = "healthy"
	}
	if err := s.Cache.Health(); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		checks["redis"] = "healthy"
	}
	checks["nodes_loaded"] = fmt.Sprintf("%d", len(s.NodeLoader.List()))
	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": map[bool]string{true: "healthy", false: "unhealthy"}[healthy], "checks": checks})
}

func (s *Server) readinessHandler(c *gin.Context) {
	if err := s.DB.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) reloadNodesHandler(c *gin.Context) {
	if err := s.NodeLoader.Reload(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.CrossNodeSafety.Load(); err != nil {
		s.Log.Warn("failed to reload cross-node triggers", zap.Error(err))
	}
	c.JSON(http.StatusOK, gin.H{"message": "nodes reloaded", "count": len(s.NodeLoader.List())})
}

func corsMiddleware() gin.HandlerFunc {
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

func metricsMiddleware(m *metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.Method == "POST" && strings.HasSuffix(c.Request.URL.Path, "/answers") {
			m.AnswerLatency.Observe(float64(time.Since(start).Milliseconds()))
		}
	}
}
