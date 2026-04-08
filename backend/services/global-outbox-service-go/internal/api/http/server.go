package http

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/database"
	"global-outbox-service-go/internal/circuitbreaker"
)

// Server implements the HTTP REST API
type Server struct {
	app            *fiber.App
	repo           *database.Repository
	circuitBreaker *circuitbreaker.MedicalCircuitBreaker
	config         *config.Config
	logger         *logrus.Logger
	
	// Prometheus metrics
	requestDuration prometheus.HistogramVec
	requestCounter  prometheus.CounterVec
}

// NewServer creates a new HTTP server
func NewServer(
	repo *database.Repository,
	circuitBreaker *circuitbreaker.MedicalCircuitBreaker,
	config *config.Config,
	logger *logrus.Logger,
) *Server {
	app := fiber.New(fiber.Config{
		AppName:      config.ProjectName,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			
			logger.Errorf("HTTP error: %v", err)
			
			return c.Status(code).JSON(fiber.Map{
				"service": config.ProjectName,
				"error":   "Internal server error",
				"detail":  err.Error(),
			})
		},
	})

	server := &Server{
		app:            app,
		repo:           repo,
		circuitBreaker: circuitBreaker,
		config:         config,
		logger:         logger,
		requestDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "Duration of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestCounter: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// CORS
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Logger
	s.app.Use(logger.New(logger.Config{
		Format: "${time} ${method} ${path} ${status} ${latency} ${bytesSent}\n",
		Output: s.logger.Writer(),
	}))

	// Recovery
	s.app.Use(recover.New())

	// Metrics middleware
	s.app.Use(s.metricsMiddleware)
}

// metricsMiddleware records request metrics
func (s *Server) metricsMiddleware(c *fiber.Ctx) error {
	start := time.Now()
	
	err := c.Next()
	
	duration := time.Since(start).Seconds()
	status := strconv.Itoa(c.Response().StatusCode())
	
	s.requestDuration.WithLabelValues(c.Method(), c.Path(), status).Observe(duration)
	s.requestCounter.WithLabelValues(c.Method(), c.Path(), status).Inc()
	
	return err
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	// Root endpoint
	s.app.Get("/", s.handleRoot)

	// Health check
	s.app.Get("/health", s.handleHealthCheck)

	// Statistics
	s.app.Get("/stats", s.handleStats)

	// Metrics (Prometheus)
	s.app.Get("/metrics", s.handleMetrics)

	// Circuit breaker status
	s.app.Get("/circuit-breaker", s.handleCircuitBreakerStatus)

	// Debug endpoints (development only)
	if s.config.Debug {
		s.app.Get("/debug/config", s.handleDebugConfig)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Infof("Starting HTTP server on port %d", s.config.Port)
	
	go func() {
		if err := s.app.Listen(fmt.Sprintf(":%d", s.config.Port)); err != nil {
			s.logger.Errorf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping HTTP server...")
	return s.app.Shutdown()
}

// Route handlers

func (s *Server) handleRoot(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"service":     s.config.ProjectName,
		"version":     s.config.Version,
		"status":      "running",
		"environment": s.config.Environment,
		"endpoints": fiber.Map{
			"health":         "/health",
			"metrics":        "/metrics",
			"stats":          "/stats",
			"circuit-breaker": "/circuit-breaker",
			"grpc":           fmt.Sprintf("localhost:%d", s.config.GRPCPort),
		},
	})
}

func (s *Server) handleHealthCheck(c *fiber.Ctx) error {
	ctx := context.Background()
	
	// Check database health
	dbHealth := s.repo.HealthCheck(ctx)
	
	// Determine overall health
	dbHealthy := false
	if status, ok := dbHealth["status"].(string); ok && status == "healthy" {
		dbHealthy = true
	}
	
	overallHealthy := dbHealthy
	
	response := fiber.Map{
		"service":     s.config.ProjectName,
		"version":     s.config.Version,
		"status":      "healthy",
		"timestamp":   time.Now().Unix(),
		"environment": s.config.Environment,
		"components": fiber.Map{
			"database": dbHealth,
			"grpc_server": fiber.Map{
				"status": "healthy",
				"port":   s.config.GRPCPort,
			},
		},
	}
	
	if !overallHealthy {
		response["status"] = "unhealthy"
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}
	
	return c.JSON(response)
}

func (s *Server) handleStats(c *fiber.Ctx) error {
	ctx := context.Background()
	
	stats, err := s.repo.GetOutboxStats(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get outbox stats: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"service": s.config.ProjectName,
			"error":   "Failed to get statistics",
			"detail":  err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"service":    s.config.ProjectName,
		"timestamp":  time.Now().Unix(),
		"statistics": stats,
	})
}

func (s *Server) handleMetrics(c *fiber.Ctx) error {
	ctx := context.Background()
	
	stats, err := s.repo.GetOutboxStats(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get stats for metrics: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to generate metrics")
	}
	
	var metrics []string
	
	// Queue depth metrics
	for service, depth := range stats.QueueDepths {
		metrics = append(metrics, fmt.Sprintf(`outbox_queue_depth{service="%s"} %d`, service, depth))
	}
	
	// Total metrics
	metrics = append(metrics, fmt.Sprintf("outbox_total_processed_24h %d", stats.TotalProcessed24h))
	metrics = append(metrics, fmt.Sprintf("outbox_dead_letter_count %d", stats.DeadLetterCount))
	
	// Circuit breaker metrics
	cbStatus := s.circuitBreaker.GetStatus()
	metrics = append(metrics, fmt.Sprintf(`outbox_circuit_breaker_enabled %d`, boolToInt(cbStatus.Enabled)))
	metrics = append(metrics, fmt.Sprintf(`outbox_circuit_breaker_load %f`, cbStatus.CurrentLoad))
	metrics = append(metrics, fmt.Sprintf(`outbox_critical_events_processed %d`, cbStatus.CriticalEventsProcessed))
	metrics = append(metrics, fmt.Sprintf(`outbox_non_critical_events_dropped %d`, cbStatus.NonCriticalEventsDropped))
	
	// Service health metrics
	dbHealth := s.repo.HealthCheck(context.Background())
	dbHealthy := 0
	if status, ok := dbHealth["status"].(string); ok && status == "healthy" {
		dbHealthy = 1
	}
	metrics = append(metrics, fmt.Sprintf(`outbox_service_healthy{component="database"} %d`, dbHealthy))
	metrics = append(metrics, fmt.Sprintf(`outbox_service_healthy{component="grpc"} %d`, 1))
	
	c.Set("Content-Type", "text/plain")
	return c.SendString(fmt.Sprintf("%s\n", joinStrings(metrics, "\n")))
}

func (s *Server) handleCircuitBreakerStatus(c *fiber.Ctx) error {
	if !s.config.MedicalCircuitBreakerEnabled {
		return c.JSON(fiber.Map{
			"enabled": false,
			"message": "Medical circuit breaker is disabled",
		})
	}
	
	status := s.circuitBreaker.GetStatus()
	
	return c.JSON(fiber.Map{
		"service":   s.config.ProjectName,
		"timestamp": time.Now().Unix(),
		"medical_circuit_breaker": fiber.Map{
			"enabled": true,
			"state":   status.State,
			"current_load": status.CurrentLoad,
			"total_requests": status.TotalRequests,
			"failed_requests": status.FailedRequests,
			"critical_events_processed": status.CriticalEventsProcessed,
			"non_critical_events_dropped": status.NonCriticalEventsDropped,
			"next_retry_at": status.NextRetryAt,
		},
	})
}

func (s *Server) handleDebugConfig(c *fiber.Ctx) error {
	if !s.config.Debug {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	}
	
	dbHealth := s.repo.HealthCheck(context.Background())
	
	return c.JSON(fiber.Map{
		"project_name": s.config.ProjectName,
		"version":      s.config.Version,
		"environment":  s.config.Environment,
		"ports": fiber.Map{
			"http":    s.config.Port,
			"grpc":    s.config.GRPCPort,
			"metrics": s.config.MetricsPort,
		},
		"database": fiber.Map{
			"connected": dbHealth["status"] == "healthy",
			"pool_size": s.config.DatabasePoolSize,
		},
		"publisher": fiber.Map{
			"enabled":       s.config.PublisherEnabled,
			"poll_interval": s.config.PublisherPollInterval,
			"batch_size":    s.config.PublisherBatchSize,
		},
	})
}

// Helper functions

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := strs[0]
	for _, str := range strs[1:] {
		result += sep + str
	}
	return result
}