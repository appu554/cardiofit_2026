package api

import (
	"net/http"
	"time"

	"kb-26-metabolic-digital-twin/internal/cache"
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/database"
	"kb-26-metabolic-digital-twin/internal/metrics"
	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server is the HTTP server for KB-26 Metabolic Digital Twin Service.
type Server struct {
	Router                *gin.Engine
	config                *config.Config
	db                    *database.Database
	cache                 *cache.RedisClient
	metrics               *metrics.Collector
	trajectoryMetrics     *metrics.TrajectoryMetrics
	logger                *zap.Logger
	bpContextOrchestrator *services.BPContextOrchestrator
	twinUpdater           *services.TwinUpdater
	calibrator            *services.BayesianCalibrator
	eventProcessor        *services.EventProcessor
	mriScorer             *services.MRIScorer
	preventScorer         *services.PREVENTScorer
	relapseDetector       *services.RelapseDetector
	trajectoryEngine      *services.TrajectoryEngine

	// PAI (Patient Acuity Index)
	paiRepo    *services.PAIRepository
	paiTrigger *services.PAIEventTrigger
	paiConfig  *services.PAIConfig

	// Acute-on-chronic detection (Gap 16)
	acuteRepo    *services.AcuteRepository
	acuteHandler *services.AcuteEventHandler
}

// NewServer creates and configures the HTTP server with all dependencies.
func NewServer(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.RedisClient,
	metricsCollector *metrics.Collector,
	logger *zap.Logger,
	bpContextOrchestrator *services.BPContextOrchestrator,
	twinUpdater *services.TwinUpdater,
	calibrator *services.BayesianCalibrator,
	eventProcessor *services.EventProcessor,
	mriScorer *services.MRIScorer,
	preventScorer *services.PREVENTScorer,
	relapseDetector *services.RelapseDetector,
	trajectoryPublisher services.TrajectoryPublisher,
) *Server {
	// Phase 7 P7-F: main.go injects a KafkaTrajectoryPublisher when
	// KB26_KAFKA_ENABLED=true (reusing the same feature flag as the
	// existing SignalConsumer wiring). When nil — e.g., local dev
	// without Kafka, or tests that don't exercise the publisher — the
	// server defaults to NoopTrajectoryPublisher so trajectory events
	// are silently dropped without crashing the engine.
	if trajectoryPublisher == nil {
		trajectoryPublisher = services.NoopTrajectoryPublisher{}
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	trajectoryThresholds, err := config.LoadTrajectoryThresholds(cfg.TrajectoryThresholdsPath)
	if err != nil {
		logger.Warn("failed to load trajectory thresholds, using defaults", zap.Error(err))
		trajectoryThresholds = config.DefaultTrajectoryThresholds()
	}

	trajMetrics := metrics.NewTrajectoryMetrics()

	s := &Server{
		Router:                router,
		config:                cfg,
		db:                    db,
		cache:                 cacheClient,
		metrics:               metricsCollector,
		trajectoryMetrics:     trajMetrics,
		logger:                logger,
		bpContextOrchestrator: bpContextOrchestrator,
		twinUpdater:           twinUpdater,
		calibrator:            calibrator,
		eventProcessor:        eventProcessor,
		mriScorer:             mriScorer,
		preventScorer:         preventScorer,
		relapseDetector:       relapseDetector,
		// Phase 7 P7-F: trajectoryPublisher is injected by main.go —
		// KafkaTrajectoryPublisher when KB26_KAFKA_ENABLED=true, noop otherwise.
		trajectoryEngine: services.NewTrajectoryEngine(
			trajectoryThresholds,
			trajMetrics,
			trajectoryPublisher,
			logger,
		),
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

// SetPAIServices injects the PAI repository and event trigger into the
// server after construction. Setter injection avoids further bloating
// the NewServer constructor parameter list.
func (s *Server) SetPAIServices(repo *services.PAIRepository, trigger *services.PAIEventTrigger, cfg *services.PAIConfig) {
	s.paiRepo = repo
	s.paiTrigger = trigger
	s.paiConfig = cfg
}

// SetAcuteServices injects the acute repository and event handler into the
// server after construction. Setter injection avoids further bloating
// the NewServer constructor parameter list.
func (s *Server) SetAcuteServices(repo *services.AcuteRepository, handler *services.AcuteEventHandler) {
	s.acuteRepo = repo
	s.acuteHandler = handler
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
