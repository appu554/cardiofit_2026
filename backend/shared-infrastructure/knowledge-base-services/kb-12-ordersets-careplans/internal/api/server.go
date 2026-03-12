// Package api provides the HTTP API server for KB-12
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"kb-12-ordersets-careplans/internal/cache"
	"kb-12-ordersets-careplans/internal/clients"
	"kb-12-ordersets-careplans/internal/config"
	"kb-12-ordersets-careplans/internal/database"
)

// Server represents the HTTP API server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	db         *database.Connection
	cache      *cache.Cache
	kb1Client  *clients.KB1DosingClient
	kb3Client  *clients.KB3TemporalClient
	kb6Client  *clients.KB6FormularyClient
	kb7Client  *clients.KB7TerminologyClient
	log        *logrus.Entry
}

// Dependencies holds all service dependencies
type Dependencies struct {
	DB        *database.Connection
	Cache     *cache.Cache
	KB1Client *clients.KB1DosingClient
	KB3Client *clients.KB3TemporalClient
	KB6Client *clients.KB6FormularyClient
	KB7Client *clients.KB7TerminologyClient
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, deps *Dependencies) *Server {
	// Set Gin mode based on environment
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add request logging middleware
	router.Use(requestLogger())

	// Add CORS middleware
	router.Use(corsMiddleware())

	// Set trusted proxies
	if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		logrus.WithError(err).Warn("Failed to set trusted proxies")
	}

	server := &Server{
		config:    cfg,
		router:    router,
		db:        deps.DB,
		cache:     deps.Cache,
		kb1Client: deps.KB1Client,
		kb3Client: deps.KB3Client,
		kb6Client: deps.KB6Client,
		kb7Client: deps.KB7Client,
		log:       logrus.WithField("component", "api-server"),
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	s.log.WithField("addr", addr).Info("Starting HTTP server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down HTTP server")

	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

// GetRouter returns the Gin router for testing
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// requestLogger returns a Gin middleware for request logging
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		entry := logrus.WithFields(logrus.Fields{
			"status":     statusCode,
			"method":     method,
			"path":       path,
			"ip":         clientIP,
			"latency":    latency,
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		})

		if len(c.Errors) > 0 {
			entry.WithField("errors", c.Errors.String()).Error("Request completed with errors")
		} else if statusCode >= 500 {
			entry.Error("Request completed with server error")
		} else if statusCode >= 400 {
			entry.Warn("Request completed with client error")
		} else {
			entry.Info("Request completed")
		}
	}
}

// corsMiddleware returns a Gin middleware for CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Client-Service")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// respondError sends an error response
func respondError(c *gin.Context, statusCode int, message string, code string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	})
}

// respondSuccess sends a success response
func respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// respondCreated sends a created response
func respondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}
