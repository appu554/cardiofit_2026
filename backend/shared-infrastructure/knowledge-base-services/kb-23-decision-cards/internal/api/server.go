package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/services"
)

type Server struct {
	Router         *gin.Engine
	cfg            *config.Config
	db             *database.Database
	cache          *cache.CacheClient
	metrics        *metrics.Collector
	log            *zap.Logger
	templateLoader *services.TemplateLoader
	fragmentLoader *services.FragmentLoader

	// Phase 2 services (initialized in InitServices)
	confidenceTier      *services.ConfidenceTierService
	templateSelector    *services.TemplateSelector
	mcuGateManager      *services.MCUGateManager
	mcuGateCache        *services.MCUGateCache
	cardBuilder         *services.CardBuilder
	recommendComposer   *services.RecommendationComposer
	kb19Publisher       *services.KB19Publisher
	kb20Client          *services.KB20Client
	kb21Client          *services.KB21Client
	kb26BPContextClient    *services.KB26BPContextClient
	kb26TrajectoryClient   *services.KB26TrajectoryClient
	trajectoryCardMetrics  *metrics.TrajectoryCardMetrics
	hypoHandler         *services.HypoglycaemiaHandler
	behavioralHandler   *services.BehavioralGapHandler
	perturbationService *services.PerturbationService
	hysteresisEngine    *services.HysteresisEngine
	cardLifecycle       *services.CardLifecycle
	compositeService    *services.CompositeCardService
	signalCardBuilder   *services.SignalCardBuilder
	seasonalContext     *services.SeasonalContext
}

func NewServer(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.CacheClient,
	metricsCollector *metrics.Collector,
	log *zap.Logger,
	templateLoader *services.TemplateLoader,
) *Server {
	var router *gin.Engine
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New()
	} else {
		router = gin.Default()
	}

	router.Use(gin.Recovery())
	router.Use(requestLogger(log))
	router.Use(corsMiddleware())

	return &Server{
		Router:         router,
		cfg:            cfg,
		db:             db,
		cache:          cacheClient,
		metrics:        metricsCollector,
		log:            log,
		templateLoader: templateLoader,
		fragmentLoader: services.NewFragmentLoader(log),
	}
}

// InitServices creates all service dependencies.
func (s *Server) InitServices() {
	// Load fragments from templates
	s.fragmentLoader.LoadFromTemplates(s.templateLoader.List())

	// Phase 2 services
	s.confidenceTier = services.NewConfidenceTierService(s.cfg, s.log)
	s.templateSelector = services.NewTemplateSelector(s.templateLoader, s.log)
	s.mcuGateManager = services.NewMCUGateManager(s.cfg, s.log)
	s.mcuGateCache = services.NewMCUGateCache(s.cache, s.db, s.metrics, s.log)
	s.recommendComposer = services.NewRecommendationComposer(s.cfg, s.log)
	s.kb19Publisher = services.NewKB19Publisher(s.cfg, s.metrics, s.log)
	s.kb20Client = services.NewKB20Client(s.cfg, s.metrics, s.log)
	s.kb21Client = services.NewKB21Client(s.cfg, s.cache, s.metrics, s.log)
	s.kb26BPContextClient = services.NewKB26BPContextClient(
		s.cfg.KB26URL,
		s.cfg.KB26Timeout(),
		s.log,
	)
	s.trajectoryCardMetrics = metrics.NewTrajectoryCardMetrics()
	s.kb26TrajectoryClient = services.NewKB26TrajectoryClient(
		s.cfg.KB26URL,
		s.cfg.KB26Timeout(),
		s.log,
		s.trajectoryCardMetrics,
	)
	s.perturbationService = services.NewPerturbationService(s.db, s.cache, s.metrics, s.log)
	s.hysteresisEngine = services.NewHysteresisEngine(s.db, s.metrics, s.log)
	s.cardBuilder = services.NewCardBuilder(
		s.confidenceTier, s.mcuGateManager, s.recommendComposer,
		s.fragmentLoader, s.db, s.log,
		s.hysteresisEngine, s.perturbationService, s.kb21Client, s.mcuGateCache,
	)
	s.hypoHandler = services.NewHypoglycaemiaHandler(s.cfg, s.db, s.mcuGateCache, s.kb19Publisher, s.metrics, s.log)
	s.behavioralHandler = services.NewBehavioralGapHandler(s.db, s.mcuGateCache, s.kb19Publisher, s.metrics, s.log)
	s.cardLifecycle = services.NewCardLifecycle(s.db, s.mcuGateCache, s.kb19Publisher, s.log)
	s.compositeService = services.NewCompositeCardService(s.db, s.metrics, s.log)
	s.signalCardBuilder = services.NewSignalCardBuilder(s.log)

	// Load seasonal calendar for the configured market. Missing file is non-fatal.
	seasonalCalendarPath := s.cfg.SeasonalCalendarPath
	if seasonalCalendarPath == "" {
		seasonalCalendarPath = "market-configs/india/seasonal_calendar.yaml" // default
	}
	seasonalCtx, err := services.NewSeasonalContext(s.cfg.Market, seasonalCalendarPath)
	if err != nil {
		s.log.Warn("failed to load seasonal calendar, no suppression will apply", zap.Error(err))
		seasonalCtx, _ = services.NewSeasonalContext(s.cfg.Market, "")
	}
	s.seasonalContext = seasonalCtx

	s.log.Info("all services initialized")
}

// MCUGateCache returns the gate cache (used by Kafka consumer wiring in main).
func (s *Server) MCUGateCache() *services.MCUGateCache { return s.mcuGateCache }

// KB19Publisher returns the KB-19 publisher (used by Kafka consumer wiring in main).
func (s *Server) KB19Publisher() *services.KB19Publisher { return s.kb19Publisher }

// HypoHandler returns the hypoglycaemia handler (used by Kafka consumer wiring in main).
func (s *Server) HypoHandler() *services.HypoglycaemiaHandler { return s.hypoHandler }

// MetricsCollector returns the metrics collector (used by Kafka consumer wiring in main).
func (s *Server) MetricsCollector() *metrics.Collector { return s.metrics }

// Database returns the database connection (used by Kafka consumer wiring in main).
func (s *Server) Database() *database.Database { return s.db }

// KB20Client returns the KB-20 client (used by Kafka consumer wiring in main —
// Phase 6 P6-6 CKM transition handler needs it to fetch patient context).
func (s *Server) KB20Client() *services.KB20Client { return s.kb20Client }

// TemplateLoader returns the YAML card-template loader (used by Kafka consumer
// wiring in main — Phase 7 P7-A reactive renal handler needs it to look up
// renal_contraindication + renal_dose_reduce templates at card-build time).
func (s *Server) TemplateLoader() *services.TemplateLoader { return s.templateLoader }

// requestLogger middleware logs each request.
func requestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		latency := time.Since(start)
		log.Info("request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

// corsMiddleware adds CORS headers.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
