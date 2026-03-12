// Package api provides the HTTP API server
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/clients"
	"kb-17-population-registry/internal/config"
	"kb-17-population-registry/internal/consumer"
	"kb-17-population-registry/internal/criteria"
	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/producer"
	"kb-17-population-registry/internal/registry"
)

// Server represents the HTTP API server
type Server struct {
	config     *config.Config
	logger     *logrus.Entry
	router     *gin.Engine
	httpServer *http.Server
	startTime  time.Time

	// Components
	db         *database.Connection
	repo       *database.Repository
	engine     *criteria.Engine
	consumer   *consumer.Consumer
	producer   *producer.EventProducer

	// Clients
	kb2Client  *clients.KB2Client
	kb8Client  *clients.KB8Client
	kb9Client  *clients.KB9Client
	kb14Client *clients.KB14Client
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, log *logrus.Entry) (*Server, error) {

	// Initialize database connection
	db, err := database.NewConnection(&cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := db.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize repository
	repo := database.NewRepository(db, log)

	// Seed default registries
	if err := seedRegistries(repo, log); err != nil {
		log.WithError(err).Warn("Failed to seed registries")
	}

	// Initialize criteria engine
	engine := criteria.NewEngine(log)

	// Initialize KB clients
	kb2Client := clients.NewKB2Client(&cfg.KBServices.KB2, log)
	kb8Client := clients.NewKB8Client(&cfg.KBServices.KB8, log)
	kb9Client := clients.NewKB9Client(&cfg.KBServices.KB9, log)
	kb14Client := clients.NewKB14Client(&cfg.KBServices.KB14, log)

	// Initialize Kafka producer
	eventProducer, err := producer.NewEventProducer(&cfg.Kafka, log)
	if err != nil {
		log.WithError(err).Warn("Failed to create Kafka producer")
	}

	// Initialize Kafka consumer
	kafkaConsumer, err := consumer.NewConsumer(
		&cfg.Kafka,
		log,
		repo,
		engine,
		eventProducer,
		kb2Client,
	)
	if err != nil {
		log.WithError(err).Warn("Failed to create Kafka consumer")
	}

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(LoggingMiddleware(log))
	router.Use(CORSMiddleware())

	server := &Server{
		config:     cfg,
		logger:     log,
		router:     router,
		startTime:  time.Now(),
		db:         db,
		repo:       repo,
		engine:     engine,
		consumer:   kafkaConsumer,
		producer:   eventProducer,
		kb2Client:  kb2Client,
		kb8Client:  kb8Client,
		kb9Client:  kb9Client,
		kb14Client: kb14Client,
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health and metrics
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Registries
		registries := v1.Group("/registries")
		{
			registries.GET("", s.listRegistriesHandler)
			registries.GET("/:code", s.getRegistryHandler)
			registries.POST("", s.createRegistryHandler)
			registries.GET("/:code/patients", s.getRegistryPatientsHandler)
		}

		// Enrollments
		enrollments := v1.Group("/enrollments")
		{
			enrollments.GET("", s.listEnrollmentsHandler)
			enrollments.POST("", s.createEnrollmentHandler)
			enrollments.GET("/:id", s.getEnrollmentHandler)
			enrollments.PUT("/:id", s.updateEnrollmentHandler)
			enrollments.DELETE("/:id", s.deleteEnrollmentHandler)
			enrollments.POST("/bulk", s.bulkEnrollHandler)
		}

		// Patient-centric endpoints
		patients := v1.Group("/patients")
		{
			patients.GET("/:id/registries", s.getPatientRegistriesHandler)
			patients.GET("/:id/enrollment/:code", s.getPatientEnrollmentHandler)
		}

		// Evaluation
		v1.POST("/evaluate", s.evaluateHandler)

		// Analytics
		v1.GET("/stats", s.getAllStatsHandler)
		v1.GET("/stats/:code", s.getRegistryStatsHandler)
		v1.GET("/high-risk", s.getHighRiskPatientsHandler)
		v1.GET("/care-gaps", s.getCareGapPatientsHandler)

		// Events
		v1.POST("/events", s.processEventHandler)
	}
}

// Router returns the Gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Close closes server resources
func (s *Server) Close() error {
	s.logger.Info("Closing server resources...")

	// Stop Kafka consumer
	if s.consumer != nil {
		s.consumer.Stop()
	}

	// Close Kafka producer
	if s.producer != nil {
		s.producer.Close()
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close database")
			return err
		}
	}

	return nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.WithField("port", s.config.Server.Port).Info("Starting HTTP server")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Stop Kafka consumer
	if s.consumer != nil {
		s.consumer.Stop()
	}

	// Close Kafka producer
	if s.producer != nil {
		s.producer.Close()
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close database")
		}
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// StartConsumer starts the Kafka consumer
func (s *Server) StartConsumer(ctx context.Context) error {
	if s.consumer == nil {
		return nil
	}
	return s.consumer.Start(ctx)
}

// GetRepo returns the repository
func (s *Server) GetRepo() *database.Repository {
	return s.repo
}

// GetEngine returns the criteria engine
func (s *Server) GetEngine() *criteria.Engine {
	return s.engine
}

// GetConsumer returns the Kafka consumer
func (s *Server) GetConsumer() *consumer.Consumer {
	return s.consumer
}

// seedRegistries seeds default registry definitions
func seedRegistries(repo *database.Repository, log *logrus.Entry) error {
	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		existing, err := repo.GetRegistry(reg.Code)
		if err != nil {
			return err
		}

		if existing == nil {
			if err := repo.CreateRegistry(&reg); err != nil {
				log.WithError(err).WithField("registry", reg.Code).Warn("Failed to seed registry")
			} else {
				log.WithField("registry", reg.Code).Info("Seeded registry")
			}
		}
	}

	return nil
}

// LoggingMiddleware provides request logging
func LoggingMiddleware(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.WithFields(logrus.Fields{
			"status":  status,
			"method":  c.Request.Method,
			"path":    path,
			"latency": latency,
			"ip":      c.ClientIP(),
		}).Info("Request completed")
	}
}

// CORSMiddleware provides CORS support
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
