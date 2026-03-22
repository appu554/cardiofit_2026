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

	"github.com/cardiofit/intake-onboarding-service/internal/app"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	intakekafka "github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Server holds the HTTP router and all service dependencies.
type Server struct {
	Router       *gin.Engine
	config       *config.Config
	db           *pgxpool.Pool
	redis        *redis.Client
	fhirClient   *fhirclient.Client
	logger       *zap.Logger
	appHandler   *app.Handler
	safetyEngine *safety.Engine
	flowEngine   *flow.Engine
	producer     *intakekafka.Producer
	eventStore   slots.EventStore
}

// NewServer constructs a Server with Gin router, middleware, and routes.
func NewServer(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	fhirClient *fhirclient.Client,
	logger *zap.Logger,
	safetyEngine *safety.Engine,
	flowEngine *flow.Engine,
	producer *intakekafka.Producer,
) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	eventStore := slots.NewPgEventStore(db)

	s := &Server{
		Router:       router,
		config:       cfg,
		db:           db,
		redis:        redisClient,
		fhirClient:   fhirClient,
		logger:       logger,
		safetyEngine: safetyEngine,
		flowEngine:   flowEngine,
		producer:     producer,
		eventStore:   eventStore,
		appHandler:   app.NewHandler(eventStore, safetyEngine, flowEngine, fhirClient, producer, db, logger),
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
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role, X-Patient-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
