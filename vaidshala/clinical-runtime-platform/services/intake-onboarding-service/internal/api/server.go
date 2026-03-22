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

	intakemetrics "github.com/cardiofit/intake-onboarding-service/internal/metrics"

	"github.com/cardiofit/intake-onboarding-service/internal/abdm"
	"github.com/cardiofit/intake-onboarding-service/internal/app"
	"github.com/cardiofit/intake-onboarding-service/internal/asha"
	"github.com/cardiofit/intake-onboarding-service/internal/checkin"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	intakekafka "github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/review"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
	"github.com/cardiofit/intake-onboarding-service/internal/whatsapp"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Server holds the HTTP router and all service dependencies.
type Server struct {
	Router           *gin.Engine
	config           *config.Config
	db               *pgxpool.Pool
	redis            *redis.Client
	fhirClient       *fhirclient.Client
	logger           *zap.Logger
	appHandler       *app.Handler
	safetyEngine     *safety.Engine
	flowEngine       *flow.Engine
	producer         *intakekafka.Producer
	eventStore       slots.EventStore
	whatsappHandler  *whatsapp.WebhookHandler
	ashaHandler      *asha.Handler
	abhaClient       *abdm.ABHAClient
	consentCollector *abdm.ConsentCollector
	checkinHandler   *checkin.Handler
	reviewHandler    *review.ReviewHandler
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

	// Phase 4: WhatsApp webhook handler
	// FlowDispatcher and MessageDeduplicator are nil-safe — wired in Phase 5
	waHandler := whatsapp.NewWebhookHandler(
		cfg.WhatsApp.AppSecret,
		cfg.WhatsApp.VerifyToken,
		nil, // FlowDispatcher — wired when flow engine supports WhatsApp channel
		nil, // MessageDeduplicator — wired when Redis dedup is implemented
		logger,
	)

	// Phase 4: ASHA tablet handler
	ashaQueue := asha.NewOfflineQueue(db, logger)
	ashaSyncSvc := asha.NewSyncService(ashaQueue, logger)
	ashaHandler := asha.NewHandler(ashaSyncSvc, ashaQueue, logger)

	// Phase 4: ABDM ABHA client + consent collector
	abhaClient := abdm.NewABHAClient(abdm.ABHAConfig{
		BaseURL:      cfg.ABDM.BaseURL,
		ClientID:     cfg.ABDM.ClientID,
		ClientSecret: cfg.ABDM.ClientSecret,
		IsSandbox:    cfg.ABDM.IsSandbox,
	}, logger)
	consentCollector := abdm.NewConsentCollector(abhaClient, logger)

	// Phase 5: Check-in and review handlers
	checkinHandler := checkin.NewHandler(db, logger)
	reviewHandler := review.NewReviewHandler(db, logger)

	s := &Server{
		Router:           router,
		config:           cfg,
		db:               db,
		redis:            redisClient,
		fhirClient:       fhirClient,
		logger:           logger,
		safetyEngine:     safetyEngine,
		flowEngine:       flowEngine,
		producer:         producer,
		eventStore:       eventStore,
		appHandler:       app.NewHandler(eventStore, safetyEngine, flowEngine, fhirClient, producer, db, logger),
		whatsappHandler:  waHandler,
		ashaHandler:      ashaHandler,
		abhaClient:       abhaClient,
		consentCollector: consentCollector,
		checkinHandler:   checkinHandler,
		reviewHandler:    reviewHandler,
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
	router.Use(intakemetrics.TracingMiddleware())
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
