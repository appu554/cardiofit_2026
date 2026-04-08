package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	ingestmetrics "github.com/cardiofit/ingestion-service/internal/metrics"

	abdmadapter "github.com/cardiofit/ingestion-service/internal/adapters/abdm"
	"github.com/cardiofit/ingestion-service/internal/adapters/ehr"
	"github.com/cardiofit/ingestion-service/internal/adapters/labs"
	"github.com/cardiofit/ingestion-service/internal/adapters/wearables"
	"github.com/cardiofit/ingestion-service/internal/config"
	"github.com/cardiofit/ingestion-service/internal/dlq"
	fhirmapper "github.com/cardiofit/ingestion-service/internal/fhir"
	kafkapkg "github.com/cardiofit/ingestion-service/internal/kafka"
	outboxpkg "github.com/cardiofit/ingestion-service/internal/outbox"
	"github.com/cardiofit/ingestion-service/internal/pipeline"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Server holds the HTTP server and all dependencies.
type Server struct {
	Router        *gin.Engine
	config        *config.Config
	db            *pgxpool.Pool
	redis         *redis.Client
	fhirClient    *fhirclient.Client
	logger        *zap.Logger
	orchestrator  *pipeline.Orchestrator
	outboxPublisher *outboxpkg.Publisher
	kafkaProducer   *kafkapkg.Producer
	topicRouter     *kafkapkg.TopicRouter
	dlqPublisher  dlq.Publisher
	dlqReplay     *dlq.ReplayHandler
	labHandler    *labs.Handler
	ehrHandler    *ehr.Handler
	abdmHandler     *abdmadapter.HIUHandler
	wearableHandler *wearables.Handler
	dlqResolver     *dlq.Resolver
}

// NewServer creates and configures the HTTP server with all dependencies.
func NewServer(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	fhirClient *fhirclient.Client,
	logger *zap.Logger,
) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Initialize pipeline components
	normalizer := pipeline.NewNormalizer(logger)
	validator := pipeline.NewValidator(logger)

	// DLQ publisher
	var dlqPub dlq.Publisher
	if db != nil {
		dlqPub = dlq.NewPostgresPublisher(db, logger)
	} else {
		dlqPub = dlq.NewMemoryPublisher(logger)
	}

	// FHIR mapper and Kafka router
	mapper := fhirmapper.NewCompositeMapper(logger)
	topicRouter := kafkapkg.NewTopicRouter(logger)

	// Pipeline orchestrator
	orchestrator := pipeline.NewOrchestrator(normalizer, validator, mapper, topicRouter, dlqPub, logger)

	// Outbox publisher (preferred path — replaces direct Kafka writes)
	var outboxPublisher *outboxpkg.Publisher
	if cfg.Outbox.Enabled {
		dbURL := cfg.Outbox.DatabaseURL
		if dbURL == "" {
			dbURL = cfg.Database.URL
		}
		sdkClient, err := outboxpkg.NewOutboxClient(dbURL, cfg.Outbox.GRPCAddress, cfg.Outbox.DefaultPriority)
		if err != nil {
			logger.Error("outbox SDK init failed — falling back to direct Kafka", zap.Error(err))
		} else {
			outboxPublisher = outboxpkg.NewPublisher(sdkClient, logger)
		}
	}

	// Kafka producer — always initialised when brokers are configured.
	// Acts as primary path when outbox is disabled, and as fallback when
	// outbox publish fails at runtime (outbox-then-Kafka pattern).
	var kafkaProducer *kafkapkg.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaProducer = kafkapkg.NewProducer(cfg.Kafka.Brokers, logger)
	}

	dlqReplay := dlq.NewReplayHandler(dlqPub, logger)

	// Phase 4: Lab adapter handler with all registered lab adapters
	labHandler := labs.NewHandler(logger,
		labs.NewThyrocareAdapter(cfg.GetLabAPIKey("thyrocare"), nil, logger),
		labs.NewRedcliffeAdapter(cfg.GetLabAPIKey("redcliffe"), nil, logger),
		labs.NewSRLAgilusAdapter(cfg.GetLabAPIKey("srl_agilus"), nil, logger),
		labs.NewDrLalAdapter(cfg.GetLabAPIKey("dr_lal"), nil, logger),
		labs.NewMetropolisAdapter(cfg.GetLabAPIKey("metropolis"), nil, logger),
		labs.NewOrangeHealthAdapter(cfg.GetLabAPIKey("orange_health"), nil, logger),
		labs.NewGenericCSVAdapter("generic_csv", nil, logger),
	)

	// Phase 4: EHR adapter handler
	fhirRestAdapter := ehr.NewFHIRRestAdapter(logger)
	ehrHandler := ehr.NewHandler(fhirRestAdapter, nil, logger) // SFTPAdapter wired when SFTP configs are loaded

	// Phase 4: ABDM HIU handler — nil until X25519 crypto keys are configured
	// abdmHandler will be initialized when ABDM_PRIVATE_KEY env is set

	// Phase 5: Wearable ingest handler (Health Connect, Ultrahuman, Apple HealthKit)
	wearableHandler := wearables.NewHandler(logger)

	// Phase 5: DLQ resolver for admin query/discard operations
	dlqResolver := dlq.NewResolver(db, logger)

	s := &Server{
		Router:          router,
		config:          cfg,
		db:              db,
		redis:           redisClient,
		fhirClient:      fhirClient,
		logger:          logger,
		orchestrator:    orchestrator,
		outboxPublisher: outboxPublisher,
		kafkaProducer:   kafkaProducer,
		topicRouter:     topicRouter,
		dlqPublisher:  dlqPub,
		dlqReplay:     dlqReplay,
		labHandler:    labHandler,
		ehrHandler:      ehrHandler,
		wearableHandler: wearableHandler,
		dlqResolver:     dlqResolver,
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
	router.Use(ingestmetrics.TracingMiddleware(ingestmetrics.IngestionServiceName))
	s.setupRoutes()

	return s
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		_ = duration
		_ = strconv.Itoa(c.Writer.Status())
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Close releases resources held by the server's publishers. The caller (main)
// should invoke this during graceful shutdown to drain outbox SDK connections
// and flush any pending Kafka writes.
func (s *Server) Close() {
	if s.outboxPublisher != nil {
		if err := s.outboxPublisher.Close(); err != nil {
			s.logger.Error("outbox publisher close failed", zap.Error(err))
		} else {
			s.logger.Info("outbox publisher closed")
		}
	}
	if s.kafkaProducer != nil {
		if err := s.kafkaProducer.Close(); err != nil {
			s.logger.Error("kafka producer close failed", zap.Error(err))
		} else {
			s.logger.Info("kafka producer closed")
		}
	}
}

// prometheusHandler returns a gin handler for Prometheus metrics.
func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
