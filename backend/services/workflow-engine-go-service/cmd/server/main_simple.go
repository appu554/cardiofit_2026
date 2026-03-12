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

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/graph"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/graph/generated"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/config"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/database"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/handlers"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/orchestration"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/repositories"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/services"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/pkg/clients"
)

func main() {
	// Initialize configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Debug)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Workflow Engine Service",
		zap.String("version", "1.0.0"),
		zap.String("environment", cfg.Environment),
		zap.Int("port", cfg.Port))

	// Initialize database
	db, err := database.NewConnection(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(db.DB, logger); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// Initialize repositories
	workflowRepo := repositories.NewWorkflowRepository(db, logger)
	snapshotRepo := repositories.NewSnapshotRepository(db, logger)

	// Initialize external service clients
	flow2GoClient := clients.NewFlow2GoClient(cfg.Flow2GoURL, logger)
	safetyGatewayClient := clients.NewSafetyGatewayClient(cfg.SafetyGatewayURL, logger)
	medicationClient := clients.NewMedicationServiceClient(cfg.MedicationServiceURL, logger)

	// Initialize strategic orchestrator
	strategicOrchestrator := orchestration.NewStrategicOrchestrator(
		flow2GoClient,
		safetyGatewayClient,
		medicationClient,
		snapshotRepo,
		workflowRepo,
		logger,
	)

	// Initialize services
	orchestrationService := services.NewOrchestrationService(
		strategicOrchestrator,
		workflowRepo,
		logger,
	)

	// Initialize handlers
	orchestrationHandler := handlers.NewOrchestrationHandler(
		orchestrationService,
		logger,
	)

	// Initialize GraphQL resolver
	resolver := graph.NewResolver(orchestrationService, logger)

	// Initialize Gin router
	router := setupRouter(cfg, logger, orchestrationHandler, resolver)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server", 
			zap.Int("port", cfg.Port),
			zap.String("graphql_endpoint", "/graphql"),
			zap.String("playground_endpoint", "/playground"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}

func initLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func setupRouter(cfg *config.Config, logger *zap.Logger, orchestrationHandler *handlers.OrchestrationHandler, resolver *graph.Resolver) *gin.Engine {
	// Set Gin mode based on environment
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add global middleware
	router.Use(gin.Recovery())
	router.Use(orchestrationHandler.ErrorHandlingMiddleware())
	router.Use(orchestrationHandler.RequestLoggingMiddleware())
	router.Use(orchestrationHandler.CORSMiddleware())

	// GraphQL endpoints
	gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	
	router.POST("/graphql", gin.WrapH(gqlHandler))
	router.GET("/graphql", gin.WrapH(gqlHandler))
	
	// GraphQL Playground (only in development)
	if cfg.Debug {
		playgroundHandler := playground.Handler("GraphQL Playground", "/graphql")
		router.GET("/playground", gin.WrapH(playgroundHandler))
	}

	// REST API routes
	v1 := router.Group("/api/v1")
	orchestrationHandler.RegisterRoutes(v1)

	// Additional API endpoints
	v1.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "operational",
			"version":   "1.0.0",
			"timestamp": time.Now().UTC(),
			"service":   "workflow-engine-service",
		})
	})

	// Metrics endpoint for monitoring
	if cfg.MonitoringEnabled {
		router.GET("/metrics", gin.WrapH(getMetricsHandler()))
	}

	return router
}

func getMetricsHandler() http.Handler {
	// This would return the Prometheus metrics handler
	// For now, return a simple handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Workflow Engine Metrics\n"))
	})
}