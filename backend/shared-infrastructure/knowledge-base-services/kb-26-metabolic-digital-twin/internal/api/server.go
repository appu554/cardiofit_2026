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
	Router       *gin.Engine
	config       *config.Config
	db           *database.Database
	cache        *cache.RedisClient
	metrics      *metrics.Collector
	logger       *zap.Logger
	twinUpdater  *services.TwinUpdater
	calibrator   *services.BayesianCalibrator
}

// NewServer creates and configures the HTTP server with all dependencies.
func NewServer(
	cfg *config.Config,
	db *database.Database,
	cacheClient *cache.RedisClient,
	metricsCollector *metrics.Collector,
	logger *zap.Logger,
	twinUpdater *services.TwinUpdater,
	calibrator *services.BayesianCalibrator,
) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		Router:      router,
		config:      cfg,
		db:          db,
		cache:       cacheClient,
		metrics:     metricsCollector,
		logger:      logger,
		twinUpdater: twinUpdater,
		calibrator:  calibrator,
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
