package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/services"
)

// SignalHandlerGroup groups all HTTP handlers for PM/MD signal nodes.
type SignalHandlerGroup struct {
	monitoringLoader    *services.MonitoringNodeLoader
	deteriorationLoader *services.DeteriorationNodeLoader
	monitoringEngine    *services.MonitoringNodeEngine
	deteriorationEngine *services.DeteriorationNodeEngine
	cascade             *services.SignalCascade
	publisher           *services.SignalPublisher
	db                  *database.Database
	cache               *cache.CacheClient
	cfg                 *config.Config
	log                 *zap.Logger
}

// NewSignalHandlerGroup constructs a SignalHandlerGroup, wiring all signal-related
// services from their dependencies.
func NewSignalHandlerGroup(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.CacheClient,
	kafkaPublisher services.KafkaPublisher,
	log *zap.Logger,
	_ *metrics.Collector, // reserved for future metrics instrumentation
) *SignalHandlerGroup {
	kb26Client := services.NewKB26Client(cfg.KB26URL, cfg.KB26Timeout(), log)

	resolver := services.NewDataResolver(
		cfg.KB20URL,
		kb26Client,
		nil, // no cache adapter wired at this layer; DataResolver handles nil gracefully
		cfg.KB26StalenessThreshold(),
		log,
	)

	evaluator := services.NewExpressionEvaluator()
	trajectory := services.NewTrajectoryComputer(log)

	monLoader := services.NewMonitoringNodeLoader(cfg.MonitoringNodesDir, log)
	if err := monLoader.Load(); err != nil {
		log.Warn("signal_handler_group: failed to load monitoring nodes on startup",
			zap.String("dir", cfg.MonitoringNodesDir),
			zap.Error(err),
		)
	}

	deterLoader := services.NewDeteriorationNodeLoader(cfg.DeteriorationNodesDir, log)
	if err := deterLoader.Load(); err != nil {
		log.Warn("signal_handler_group: failed to load deterioration nodes on startup",
			zap.String("dir", cfg.DeteriorationNodesDir),
			zap.Error(err),
		)
	}

	monEngine := services.NewMonitoringNodeEngine(monLoader, resolver, evaluator, trajectory, db.DB, log)
	deterEngine := services.NewDeteriorationNodeEngine(deterLoader, resolver, trajectory, kb26Client, evaluator, db.DB, log)

	cascade := services.NewSignalCascade(monLoader, deterLoader, deterEngine, log)

	publisher := services.NewSignalPublisher(
		cfg.KB23URL,
		kafkaPublisher,
		cfg.KafkaSignalTopic,
		cfg.SignalPublisherRetryCount,
		time.Duration(cfg.SignalPublisherRetryDelaySec)*time.Second,
		db.DB,
		log,
	)

	return &SignalHandlerGroup{
		monitoringLoader:    monLoader,
		deteriorationLoader: deterLoader,
		monitoringEngine:    monEngine,
		deteriorationEngine: deterEngine,
		cascade:             cascade,
		publisher:           publisher,
		db:                  db,
		cache:               cacheClient,
		cfg:                 cfg,
		log:                 log.With(zap.String("component", "signal-handler-group")),
	}
}

// RegisterRoutes registers all signal-related routes under the given router group.
// All routes are prefixed with /signals relative to the provided group.
func (g *SignalHandlerGroup) RegisterRoutes(router *gin.RouterGroup) {
	signals := router.Group("/signals")

	// Event ingestion endpoints
	signals.POST("/events/observation", g.handleObservation)
	signals.POST("/events/twin-state-update", g.handleTwinStateUpdate)
	signals.POST("/events/checkin-response", g.handleCheckinResponse)

	// Patient signal query endpoints
	signals.GET("/patients/:id/signals", g.handleGetPatientSignals)
	signals.GET("/patients/:id/signals/:nodeId", g.handleGetSignalHistory)
	signals.GET("/patients/:id/deterioration-summary", g.handleGetDeteriorationSummary)

	// Node definition listing endpoints
	signals.GET("/nodes/monitoring", g.handleListMonitoringNodes)
	signals.GET("/nodes/monitoring/:nodeId", g.handleGetMonitoringNode)
	signals.GET("/nodes/deterioration", g.handleListDeteriorationNodes)
	signals.GET("/nodes/deterioration/:nodeId", g.handleGetDeteriorationNode)
}
