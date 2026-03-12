package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"medication-service-v2/internal/config"
	"medication-service-v2/internal/infrastructure/google_fhir"
	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/graphql/resolvers"
	httpserver "medication-service-v2/internal/interfaces/http"

	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting medication-service-v2 (minimal version) with Google FHIR integration")

	// Load minimal configuration from environment
	cfg := loadConfigFromEnv()

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Google FHIR client if enabled
	var googleFHIRClient *google_fhir.GoogleFHIRClient
	if cfg.GoogleFHIR.Enabled {
		googleFHIRConfig := &google_fhir.Config{
			ProjectID:       cfg.GoogleFHIR.ProjectID,
			Location:        cfg.GoogleFHIR.Location,
			DatasetID:       cfg.GoogleFHIR.DatasetID,
			FHIRStoreID:     cfg.GoogleFHIR.FHIRStoreID,
			CredentialsPath: cfg.GoogleFHIR.CredentialsPath,
		}

		googleFHIRClient = google_fhir.NewGoogleFHIRClient(googleFHIRConfig)

		// Initialize the client
		if err := googleFHIRClient.Initialize(ctx); err != nil {
			logger.Error("Failed to initialize Google FHIR client", zap.Error(err))
			logger.Warn("Continuing without Google FHIR - service will use fallback behavior")
			googleFHIRClient = nil
		} else {
			logger.Info("Successfully initialized Google FHIR client")
		}
	}

	// Initialize FHIR medication service (only the service we actually need)
	fhirMedicationService := services.NewFHIRMedicationService(logger, googleFHIRClient)

	// Initialize GraphQL resolver
	medicationResolver := resolvers.NewMedicationResolver(fhirMedicationService, logger)

	// Initialize GraphQL server
	graphqlServer, err := httpserver.NewGraphQLServer(medicationResolver, logger)
	if err != nil {
		logger.Fatal("Failed to create GraphQL server", zap.Error(err))
	}

	// Start GraphQL server for Apollo Federation
	go func() {
		router := gin.New()
		router.Use(gin.Logger())
		router.Use(gin.Recovery())

		// Enable CORS
		router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
		})

		// Register GraphQL routes
		graphqlServer.RegisterRoutes(router)

		// Add health check endpoint
		router.GET("/health", func(c *gin.Context) {
			status := map[string]interface{}{
				"service":     "medication-service-v2",
				"status":      "healthy",
				"google_fhir": googleFHIRClient != nil,
				"timestamp":   time.Now().Format(time.RFC3339),
				"version":     "2.0.0-minimal",
			}
			c.JSON(200, status)
		})

		// Start server
		port := cfg.GraphQL.Port
		logger.Info("Starting GraphQL Federation server",
			zap.String("port", fmt.Sprintf("%d", port)),
			zap.String("federation_url", fmt.Sprintf("http://localhost:%d/federation", port)),
			zap.String("playground_url", fmt.Sprintf("http://localhost:%d/graphql", port)))

		srv := &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("GraphQL server failed", zap.Error(err))
		}
	}()

	logger.Info("medication-service-v2 started successfully!")
	logger.Info("🚀 GraphQL Federation endpoint: http://localhost:" + fmt.Sprintf("%d", cfg.GraphQL.Port) + "/federation")
	logger.Info("🎮 GraphQL Playground: http://localhost:" + fmt.Sprintf("%d", cfg.GraphQL.Port) + "/graphql")
	logger.Info("💚 Health check: http://localhost:" + fmt.Sprintf("%d", cfg.GraphQL.Port) + "/health")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down medication-service-v2...")
	logger.Info("Shutdown complete")
}

// Simplified config loading from environment
func loadConfigFromEnv() *config.Config {
	cfg := &config.Config{}

	// Google FHIR Config
	cfg.GoogleFHIR.Enabled = getEnvBool("USE_GOOGLE_HEALTHCARE_API", true)
	cfg.GoogleFHIR.ProjectID = getEnv("GOOGLE_CLOUD_PROJECT_ID", "cardiofit-905a8")
	cfg.GoogleFHIR.Location = getEnv("GOOGLE_CLOUD_LOCATION", "asia-south1")
	cfg.GoogleFHIR.DatasetID = getEnv("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub")
	cfg.GoogleFHIR.FHIRStoreID = getEnv("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store")
	cfg.GoogleFHIR.CredentialsPath = getEnv("GOOGLE_CLOUD_CREDENTIALS_PATH", "credentials/google-credentials.json")

	// GraphQL Config
	cfg.GraphQL.Port = getEnvInt("GRAPHQL_PORT", 8005)
	cfg.GraphQL.EnablePlayground = getEnvBool("GRAPHQL_ENABLE_PLAYGROUND", true)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}