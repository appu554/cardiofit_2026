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

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/config"
	"flow2-go-engine/internal/flow2"
	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/services"
	"flow2-go-engine/internal/middleware"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting Flow 2 Go Enhanced Orchestrator")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	cacheService, err := services.NewCacheService(cfg.Redis)
	if err != nil {
		logger.Fatalf("Failed to initialize cache service: %v", err)
	}

	metricsService := services.NewMetricsService()
	healthService := services.NewHealthService()

	// Initialize clients
	rustRecipeClient, err := clients.NewRustRecipeClient(cfg.RustEngine)
	if err != nil {
		logger.Fatalf("Failed to initialize Rust recipe client: %v", err)
	}

	contextServiceClient, err := clients.NewContextServiceClient(cfg.ContextService)
	if err != nil {
		logger.Fatalf("Failed to initialize context service client: %v", err)
	}

	// Initialize Context Gateway client for snapshot operations
	contextGatewayClient, err := clients.NewContextGatewayClient(cfg.ContextService)
	if err != nil {
		logger.Fatalf("Failed to initialize context gateway client: %v", err)
	}

	medicationAPIClient, err := clients.NewMedicationAPIClient(cfg.MedicationAPI)
	if err != nil {
		logger.Fatalf("Failed to initialize medication API client: %v", err)
	}

	// Initialize Flow 2 orchestrator
	flow2Orchestrator, err := flow2.NewOrchestrator(&flow2.Config{
		RustRecipeClient:     rustRecipeClient,
		ContextServiceClient: contextServiceClient,
		ContextGatewayClient: contextGatewayClient,
		MedicationAPIClient:  medicationAPIClient,
		CacheService:        cacheService,
		MetricsService:      metricsService,
		HealthService:       healthService,
		Logger:              logger,
	})
	if err != nil {
		logger.Fatalf("Failed to initialize Flow 2 orchestrator: %v", err)
	}

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logging(logger))
	router.Use(middleware.Metrics(metricsService))
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit(cfg.RateLimit))

	// Health endpoints
	router.GET("/health", healthService.HealthCheck)
	router.GET("/health/ready", healthService.ReadinessCheck)
	router.GET("/health/live", healthService.LivenessCheck)
	router.GET("/metrics", metricsService.PrometheusHandler)

	// Flow 2 API endpoints
	v1 := router.Group("/api/v1/flow2")
	{
		v1.POST("/execute", flow2Orchestrator.ExecuteFlow2)
		v1.POST("/medication-intelligence", flow2Orchestrator.MedicationIntelligence)
		v1.POST("/dose-optimization", flow2Orchestrator.DoseOptimization)
		v1.POST("/safety-validation", flow2Orchestrator.SafetyValidation)
		v1.POST("/clinical-intelligence", flow2Orchestrator.ClinicalIntelligence)
		v1.POST("/analytics/collect", flow2Orchestrator.CollectAnalytics)
		v1.GET("/analytics/{patient_id}", flow2Orchestrator.GetPatientAnalytics)
		v1.GET("/recommendations/{patient_id}", flow2Orchestrator.GetRecommendations)
	}

	// Snapshot-based Flow2 endpoints (Recipe Snapshot Architecture)
	snapshots := router.Group("/api/v1/snapshots")
	{
		snapshots.POST("/execute", flow2Orchestrator.ExecuteWithSnapshots)
		snapshots.POST("/execute-advanced", flow2Orchestrator.AdvancedSnapshotWorkflow)
		snapshots.POST("/execute-batch", flow2Orchestrator.BatchSnapshotExecution)
		snapshots.GET("/health", flow2Orchestrator.SnapshotHealthCheck)
		snapshots.GET("/metrics", flow2Orchestrator.GetSnapshotMetrics)
	}

	// GraphQL endpoint (for compatibility with existing medication service)
	router.POST("/graphql", flow2Orchestrator.GraphQLHandler)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Starting Flow 2 Go Engine HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Flow 2 Go Engine...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	// Close clients and services
	rustRecipeClient.Close()
	contextServiceClient.Close()
	medicationAPIClient.Close()
	cacheService.Close()

	logger.Info("Flow 2 Go Engine shutdown complete")
}
