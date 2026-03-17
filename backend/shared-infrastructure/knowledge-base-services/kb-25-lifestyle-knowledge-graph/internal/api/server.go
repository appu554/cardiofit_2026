package api

import (
	"context"
	"net/http"
	"time"

	"kb-25-lifestyle-knowledge-graph/internal/cache"
	"kb-25-lifestyle-knowledge-graph/internal/config"
	"kb-25-lifestyle-knowledge-graph/internal/graph"
	"kb-25-lifestyle-knowledge-graph/internal/metrics"
	"kb-25-lifestyle-knowledge-graph/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	Router         *gin.Engine
	config         *config.Config
	graph          graph.GraphClient
	cache          *cache.RedisClient
	metrics        *metrics.Collector
	logger         *zap.Logger
	chainTraversal *services.ChainTraversalService
}

func NewServer(
	cfg *config.Config,
	graphClient graph.GraphClient,
	cacheClient *cache.RedisClient,
	metricsCollector *metrics.Collector,
	logger *zap.Logger,
	chainSvc *services.ChainTraversalService,
) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		Router:         router,
		config:         cfg,
		graph:          graphClient,
		cache:          cacheClient,
		metrics:        metricsCollector,
		logger:         logger,
		chainTraversal: chainSvc,
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

func (s *Server) setupRoutes() {
	s.Router.GET("/health", s.healthCheck)
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := s.Router.Group("/api/v1/kb25")
	v1.GET("/causal-chain/:target", s.getCausalChain)
}

func (s *Server) healthCheck(c *gin.Context) {
	checks := map[string]interface{}{}

	graphStatus := "healthy"
	if s.graph != nil {
		if err := s.graph.HealthCheck(context.Background()); err != nil {
			graphStatus = "unhealthy"
		}
	} else {
		graphStatus = "not_connected"
	}
	checks["neo4j"] = map[string]string{"status": graphStatus}

	if s.cache != nil {
		cacheStatus := "healthy"
		if err := s.cache.Ping(); err != nil {
			cacheStatus = "unhealthy"
		}
		checks["cache"] = map[string]string{"status": cacheStatus}
	}

	overallStatus := "healthy"
	if graphStatus == "unhealthy" {
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
		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start).Seconds())
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

func sendSuccess(c *gin.Context, data interface{}, metadata map[string]interface{}) {
	resp := gin.H{"success": true, "data": data}
	if metadata != nil {
		resp["metadata"] = metadata
	}
	c.JSON(http.StatusOK, resp)
}

func sendError(c *gin.Context, statusCode int, message, code string, details map[string]interface{}) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   gin.H{"code": code, "message": message, "details": details},
	})
}
