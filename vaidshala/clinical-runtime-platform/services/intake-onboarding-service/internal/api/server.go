package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Server holds the HTTP router and all service dependencies.
type Server struct {
	Router     *gin.Engine
	config     *config.Config
	db         *pgxpool.Pool
	redis      *redis.Client
	fhirClient *fhirclient.Client
	logger     *zap.Logger
}

// NewServer constructs a Server with Gin router, middleware, and routes.
func NewServer(cfg *config.Config, db *pgxpool.Pool, redisClient *redis.Client, fc *fhirclient.Client, logger *zap.Logger) *Server {
	if !cfg.IsDevelopment() {
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
		fhirClient: fc,
		logger:     logger,
	}

	s.setupRoutes()
	return s
}

// prometheusHandler returns a Gin handler that serves Prometheus metrics.
func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// corsMiddleware adds permissive CORS headers for development.
func corsMiddleware() gin.HandlerFunc {
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
