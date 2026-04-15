package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/config"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/metrics"
	"kb-patient-profile/internal/services"

	"go.uber.org/zap"
)

// Server holds the HTTP server and all injected dependencies.
type Server struct {
	Router  *gin.Engine
	config  *config.Config
	db      *database.Database
	cache   *cache.Client
	metrics *metrics.Collector

	patientService    *services.PatientService
	labService        *services.LabService
	medicationService *services.MedicationService
	stratumEngine     *services.StratumEngine
	cmRegistry        *services.CMRegistry
	adrService        *services.ADRService
	pipelineService   *services.PipelineService
	projectionService *services.ProjectionService
	loincRegistry     *services.LOINCRegistry
	protocolService   *services.ProtocolService
	protocolRegistry  *services.ProtocolRegistry
	eventBus          services.EventPublisher
	bpReadingQuery    *services.BPReadingQuery

	// Phase 8 P8-3: optional CGM status fetcher backed by a KB-26
	// HTTP client. When nil (local dev, tests, or deployments where
	// KB-26 is not reachable), the summary-context handler populates
	// HasCGM=false and downstream card generation falls back to the
	// HbA1c glycaemic path. Set via SetKB26CGMFetcher from main.go.
	kb26CGMFetcher services.CGMStatusFetcher

	logger *zap.Logger
}

// NewServer constructs the HTTP server with all dependencies injected.
func NewServer(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.Client,
	metricsCollector *metrics.Collector,
	patientSvc *services.PatientService,
	labSvc *services.LabService,
	medSvc *services.MedicationService,
	stratumEng *services.StratumEngine,
	cmReg *services.CMRegistry,
	adrSvc *services.ADRService,
	pipelineSvc *services.PipelineService,
	projectionSvc *services.ProjectionService,
	loincReg *services.LOINCRegistry,
	protocolSvc *services.ProtocolService,
	protocolReg *services.ProtocolRegistry,
	eventBus *services.EventBus,
	logger *zap.Logger,
) *Server {
	router := gin.New()
	router.Use(gin.Recovery())
	if cfg.IsDevelopment() {
		router.Use(gin.Logger())
	}

	s := &Server{
		Router:            router,
		config:            cfg,
		db:                db,
		cache:             cacheClient,
		metrics:           metricsCollector,
		patientService:    patientSvc,
		labService:        labSvc,
		medicationService: medSvc,
		stratumEngine:     stratumEng,
		cmRegistry:        cmReg,
		adrService:        adrSvc,
		pipelineService:   pipelineSvc,
		projectionService: projectionSvc,
		loincRegistry:     loincReg,
		protocolService:   protocolSvc,
		protocolRegistry:  protocolReg,
		eventBus:          eventBus,
		bpReadingQuery:    services.NewBPReadingQuery(db.DB),
		logger:            logger,
	}

	s.Router.Use(s.metricsMiddleware())
	s.Router.Use(s.corsMiddleware())
	s.setupRoutes()

	return s
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		s.metrics.RequestDuration.WithLabelValues(c.Request.Method, c.FullPath(), status).Observe(duration)
		s.metrics.RequestTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SetKB26CGMFetcher injects the KB-26 CGM status fetcher after server
// construction. Matches the existing pattern used for KB-25 client
// injection on protocolService. Called from main.go once the
// KB26Client is instantiated. Phase 8 P8-3.
func (s *Server) SetKB26CGMFetcher(fetcher services.CGMStatusFetcher) {
	s.kb26CGMFetcher = fetcher
}
