package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/config"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/database"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/handlers"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/middleware"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/monitoring"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/orchestration"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/repositories"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/services"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/pkg/clients"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize logger
	logger := initLogger(cfg.Debug, cfg.LogLevel)
	defer func() { _ = logger.Sync() }()

	logger.Info("Starting Workflow Engine Service",
		zap.String("service", cfg.ServiceName),
		zap.Int("port", cfg.ServicePort),
		zap.Bool("debug", cfg.Debug))

	// Initialize monitoring
	metrics := monitoring.NewMetrics()
	tracer, err := monitoring.InitTracing(cfg.ServiceName, cfg.Monitoring.JaegerEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize tracing", zap.Error(err))
	}

	// Initialize database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// Initialize repositories
	workflowRepo := repository.NewWorkflowRepository(db)
	snapshotRepo := repository.NewSnapshotRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	eventRepo := repository.NewEventRepository(db)

	// Initialize external service clients
	flow2GoClient := clients.NewFlow2GoClient(cfg.ExternalServices.Flow2GoURL, logger)
	flow2RustClient := clients.NewFlow2RustClient(cfg.ExternalServices.Flow2RustURL, logger)
	safetyClient := clients.NewSafetyGatewayClient(cfg.ExternalServices.SafetyGatewayURL, logger)
	medicationClient := clients.NewMedicationServiceClient(cfg.ExternalServices.MedicationServiceURL, logger)

	// Initialize services
	snapshotService := service.NewSnapshotService(snapshotRepo, logger)
	workflowService := service.NewWorkflowService(workflowRepo, taskRepo, eventRepo, logger)

	// Initialize strategic orchestrator
	orchestrator := orchestration.NewStrategicOrchestrator(
		flow2GoClient,
		flow2RustClient,
		safetyClient,
		medicationClient,
		snapshotService,
		cfg,
		metrics,
		tracer,
		logger,
	)

	// Initialize HTTP handlers
	workflowHandler := handler.NewWorkflowHandler(orchestrator, workflowService, logger)
	healthHandler := handler.NewHealthHandler(orchestrator, db, logger)

	// Setup HTTP router
	router := setupRouter(cfg, workflowHandler, healthHandler, logger)

	// Setup metrics endpoint
	if cfg.Monitoring.PrometheusEnabled {
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
		logger.Info("Metrics endpoint enabled at /metrics")
	}

	// Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServicePort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", cfg.ServicePort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workflow monitoring
	go orchestrator.StartMonitoring(ctx)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Cancel background services
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}

func initLogger(debug bool, level string) *zap.Logger {
	var config zap.Config
	if debug {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Parse log level
	if parsedLevel, err := zap.ParseAtomicLevel(level); err == nil {
		config.Level = parsedLevel
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

func setupRouter(
	cfg *config.Config,
	workflowHandler *handler.WorkflowHandler,
	healthHandler *handler.HealthHandler,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin mode
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(middleware.LoggerMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestIDMiddleware())

	// Add authentication middleware for protected routes
	authMiddleware := middleware.NewAuthMiddleware(cfg.ExternalServices.AuthServiceURL, logger)

	// Health check endpoints (no auth required)
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/health/ready", healthHandler.ReadinessCheck)
	router.GET("/health/live", healthHandler.LivenessCheck)

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to the Workflow Engine Service",
			"service": cfg.ServiceName,
			"version": "1.0.0",
		})
	})

	// API v1 routes (with authentication)
	v1 := router.Group("/api/v1")
	v1.Use(authMiddleware.ValidateToken())
	{
		// Orchestration endpoints
		orchestration := v1.Group("/orchestration")
		{
			orchestration.POST("/medication", workflowHandler.OrchestrateMedicationRequest)
			orchestration.POST("/override", workflowHandler.ProcessProviderOverride)
			orchestration.GET("/status/:correlation_id", workflowHandler.GetOrchestrationStatus)
		}

		// Workflow management endpoints
		workflows := v1.Group("/workflows")
		{
			workflows.GET("/", workflowHandler.ListWorkflows)
			workflows.POST("/", workflowHandler.CreateWorkflow)
			workflows.GET("/:id", workflowHandler.GetWorkflow)
			workflows.PUT("/:id", workflowHandler.UpdateWorkflow)
			workflows.DELETE("/:id", workflowHandler.DeleteWorkflow)
			workflows.POST("/:id/start", workflowHandler.StartWorkflow)
			workflows.POST("/:id/cancel", workflowHandler.CancelWorkflow)
		}

		// Workflow instances
		instances := v1.Group("/instances")
		{
			instances.GET("/", workflowHandler.ListInstances)
			instances.GET("/:id", workflowHandler.GetInstance)
			instances.GET("/patient/:patient_id", workflowHandler.GetInstancesByPatient)
			instances.POST("/:id/signal", workflowHandler.SignalInstance)
		}

		// Tasks
		tasks := v1.Group("/tasks")
		{
			tasks.GET("/", workflowHandler.ListTasks)
			tasks.GET("/:id", workflowHandler.GetTask)
			tasks.POST("/:id/claim", workflowHandler.ClaimTask)
			tasks.POST("/:id/complete", workflowHandler.CompleteTask)
			tasks.POST("/:id/delegate", workflowHandler.DelegateTask)
		}

		// Snapshots
		snapshots := v1.Group("/snapshots")
		{
			snapshots.GET("/:id", workflowHandler.GetSnapshot)
			snapshots.GET("/patient/:patient_id", workflowHandler.GetSnapshotsByPatient)
		}
	}

	// GraphQL Federation endpoint (no auth for schema introspection)
	router.POST("/graphql", workflowHandler.GraphQLHandler)
	router.GET("/graphql", workflowHandler.GraphQLPlaygroundHandler)

	return router
}