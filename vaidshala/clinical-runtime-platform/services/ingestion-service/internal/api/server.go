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

	"github.com/cardiofit/ingestion-service/internal/config"
	"github.com/cardiofit/ingestion-service/internal/dlq"
	fhirmapper "github.com/cardiofit/ingestion-service/internal/fhir"
	kafkapkg "github.com/cardiofit/ingestion-service/internal/kafka"
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
	kafkaProducer *kafkapkg.Producer
	topicRouter   *kafkapkg.TopicRouter
	dlqPublisher  dlq.Publisher
	dlqReplay     *dlq.ReplayHandler
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

	// Kafka producer
	var kafkaProducer *kafkapkg.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaProducer = kafkapkg.NewProducer(cfg.Kafka.Brokers, logger)
	}

	dlqReplay := dlq.NewReplayHandler(dlqPub, logger)

	s := &Server{
		Router:        router,
		config:        cfg,
		db:            db,
		redis:         redisClient,
		fhirClient:    fhirClient,
		logger:        logger,
		orchestrator:  orchestrator,
		kafkaProducer: kafkaProducer,
		topicRouter:   topicRouter,
		dlqPublisher:  dlqPub,
		dlqReplay:     dlqReplay,
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
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

// prometheusHandler returns a gin handler for Prometheus metrics.
func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
