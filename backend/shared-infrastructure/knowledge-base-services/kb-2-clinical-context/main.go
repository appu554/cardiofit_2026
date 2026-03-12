package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kb-clinical-context/internal/api"
	"kb-clinical-context/internal/cache"
	"kb-clinical-context/internal/config"
	"kb-clinical-context/internal/database"
	"kb-clinical-context/internal/metrics"
	"kb-clinical-context/internal/services"
)

func main() {
	log.Println("Starting KB-2 Clinical Context Service...")

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database (MongoDB)
	log.Println("Connecting to MongoDB...")
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Close()

	// Initialize L2 Redis cache
	log.Println("Connecting to Redis cache...")
	l2Cache, err := cache.NewCacheClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to cache: %v", err)
	}
	defer l2Cache.Close()

	// Initialize multi-tier cache (L1 + L2 + L3)
	log.Println("Initializing multi-tier cache...")
	cacheConfig := &cache.CacheConfig{
		L1TTL:          cfg.Cache.L1TTL,
		L2TTL:          1 * time.Hour,
		L3TTL:          24 * time.Hour,
		L1MaxSize:      cfg.Cache.L1MaxSize,
		L2MaxSize:      100000,
		EnableL3:       cfg.Cache.CDNEnabled,
		CDNBaseURL:     cfg.Cache.CDNBaseURL,
	}
	multiCache := cache.NewMultiTierCache(l2Cache, logger, cacheConfig)
	defer multiCache.Close()

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Initialize context service
	contextService, err := services.NewContextService(
		db,
		multiCache,
		metricsCollector,
		cfg,
		"./phenotypes", // phenotype directory
	)
	if err != nil {
		log.Fatalf("Failed to initialize context service: %v", err)
	}

	// Initialize HTTP server
	server := api.NewServer(
		cfg,
		db,
		multiCache,
		metricsCollector,
		contextService,
		logger,
	)

	// Start server in a goroutine
	go func() {
		port := cfg.Server.Port
		if port == "" {
			port = "8082"
		}

		log.Printf("Server starting on port %s", port)
		if err := server.Router.Run(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Health check
	log.Println("Performing initial health checks...")

	// Check database health
	if err := db.HealthCheck(); err != nil {
		log.Printf("Warning: MongoDB health check failed: %v", err)
	} else {
		log.Println("MongoDB health check passed")
	}

	// Check multi-tier cache health
	if err := multiCache.HealthCheck(); err != nil {
		log.Printf("Warning: Multi-tier cache health check failed: %v", err)
	} else {
		log.Println("Multi-tier cache health check passed")
	}

	log.Println("KB-2 Clinical Context Service started successfully")

	// Service information
	fmt.Printf(`
========================================
KB-2 Clinical Context Service
========================================
Service: kb-2-clinical-context
Port: %s
Version: 1.0.0
Environment: %s
========================================

Available Endpoints:
- Health: GET /health
- Metrics: GET /metrics

Context Endpoints:
- Build Context: POST /api/v1/context/build
- Context History: GET /api/v1/context/{patient_id}/history
- Context Statistics: GET /api/v1/context/statistics

Phenotype Endpoints:
- Detect Phenotypes: POST /api/v1/phenotypes/detect
- Phenotype Definitions: GET /api/v1/phenotypes/definitions

Risk Assessment Endpoints:
- Assess Risk: POST /api/v1/risk/assess

Care Gaps Endpoints:
- Identify Care Gaps: GET /api/v1/care-gaps/{patient_id}

Admin Endpoints:
- System Health: GET /api/v1/admin/health
- Clear Cache: POST /api/v1/admin/cache/clear

========================================
Database: MongoDB
Cache: Redis (DB %d)
Metrics: Prometheus
========================================
`, cfg.Server.Port, cfg.Server.Environment, cfg.Redis.Database)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down KB-2 Clinical Context Service...")
	log.Println("Service stopped")
}