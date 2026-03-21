package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Server is the HTTP server for the Ingestion Service.
type Server struct {
	Router     *gin.Engine
	config     *config.Config
	db         *pgxpool.Pool
	redis      *redis.Client
	fhirClient *fhirclient.Client
	logger     *zap.Logger
}

// NewServer creates and configures the Ingestion Service HTTP server.
func NewServer(cfg *config.Config, db *pgxpool.Pool, redisClient *redis.Client, fhirClient *fhirclient.Client, logger *zap.Logger) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	s := &Server{
		Router:     router,
		config:     cfg,
		db:         db,
		redis:      redisClient,
		fhirClient: fhirClient,
		logger:     logger,
	}

	s.setupRoutes()
	return s
}

// prometheusHandler returns a gin handler for Prometheus metrics.
func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// corsMiddleware adds CORS headers for development convenience.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
