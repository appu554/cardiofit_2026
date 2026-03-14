// Package api provides the HTTP server and route registration for the
// KB-24 Safety Constraint Engine REST API.
package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-24-safety-constraint-engine/internal/config"
	"kb-24-safety-constraint-engine/internal/services"
)

// Server holds the Gin router and all service dependencies.
type Server struct {
	Router    *gin.Engine
	cfg       *config.Config
	evaluator *services.SafetyTriggerEvaluator
	publisher services.KafkaPublisher
	log       *zap.Logger
}

// NewServer creates a new API server with the given dependencies.
func NewServer(
	cfg *config.Config,
	evaluator *services.SafetyTriggerEvaluator,
	publisher services.KafkaPublisher,
	log *zap.Logger,
) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	return &Server{
		Router:    router,
		cfg:       cfg,
		evaluator: evaluator,
		publisher: publisher,
		log:       log,
	}
}

// RegisterRoutes wires up all HTTP endpoints.
func (s *Server) RegisterRoutes() {
	h := NewHandlers(s.evaluator, s.publisher, s.log)

	// Health check
	s.Router.GET("/health", h.HandleHealth)

	// API v1
	v1 := s.Router.Group("/api/v1")
	{
		v1.POST("/evaluate", h.HandleEvaluate)
		v1.POST("/sessions/:id/clear", h.HandleClearSession)
	}
}
