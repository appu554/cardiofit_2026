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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kb-2-clinical-context-go/internal/api"
	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/metrics"
	"kb-2-clinical-context-go/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize metrics
	metricsCollector := metrics.NewPrometheusMetrics()

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(context.TODO())

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Test connections
	if err := mongoClient.Ping(context.TODO(), nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	if err := redisClient.Ping(context.TODO()).Err(); err != nil {
		log.Fatal("Redis ping failed:", err)
	}

	// Initialize services
	contextService := services.NewContextService(mongoClient, redisClient, cfg)
	phenotypeEngine := services.NewPhenotypeEngine(cfg)
	riskService := services.NewRiskAssessmentService(mongoClient, redisClient)
	treatmentService := services.NewTreatmentPreferenceService(mongoClient, redisClient)

	// Initialize API server
	apiServer := api.NewServer(api.ServerConfig{
		Config:              cfg,
		MetricsCollector:    metricsCollector,
		ContextService:      contextService,
		PhenotypeEngine:     phenotypeEngine,
		RiskService:         riskService,
		TreatmentService:    treatmentService,
	})

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Register routes
	apiServer.RegisterRoutes(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("KB-2 Clinical Context service starting on port %d", cfg.Port)
		log.Printf("Environment: %s", cfg.Environment)
		log.Printf("Health check: http://localhost:%d/health", cfg.Port)
		log.Printf("Metrics: http://localhost:%d/metrics", cfg.Port)
		log.Printf("API Documentation: http://localhost:%d/v1/docs", cfg.Port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 15 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}