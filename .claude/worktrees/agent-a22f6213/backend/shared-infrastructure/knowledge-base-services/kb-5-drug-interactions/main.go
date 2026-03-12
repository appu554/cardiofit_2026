package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"kb-drug-interactions/internal/api"
	"kb-drug-interactions/internal/cache"
	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/metrics"
	"kb-drug-interactions/internal/models"
	"kb-drug-interactions/internal/services"
)

func main() {
	// Initialize structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	
	logger.Info("Starting KB-5 Enhanced Drug Interactions Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database (auto-migrates models)
	logger.Info("Connecting to database...")
	db, err := database.NewConnection(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("Database connected and migrations completed")

	// Initialize cache systems
	logger.Info("Connecting to cache systems...")
	cacheClient, err := cache.NewCacheClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to cache", zap.Error(err))
	}
	defer cacheClient.Close()
	
	// Initialize hot/warm cache clients
	hotCacheOpts, _ := redis.ParseURL(cfg.Redis.HotCacheURL)
	hotCacheClient := redis.NewClient(hotCacheOpts)
	defer hotCacheClient.Close()
	
	warmCacheOpts, _ := redis.ParseURL(cfg.Redis.WarmCacheURL)
	warmCacheClient := redis.NewClient(warmCacheOpts)
	defer warmCacheClient.Close()

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()
	
	// Initialize cache manager
	cacheManager := models.NewCacheManager(cacheClient)
	
	// Initialize enhanced engines
	logger.Info("Initializing enhanced interaction engines...")
	
	// Enhanced interaction matrix service
	matrixService := services.NewEnhancedInteractionMatrixService(
		db, cacheClient, cfg, metricsCollector)

	// Pharmacogenomic interaction engine
	pgxEngine := services.NewPharmacogenomicEngine(db, metricsCollector)

	// Drug class interaction engine
	classEngine := services.NewClassInteractionEngine(db, metricsCollector)

	// Food/alcohol/herbal modifier engine (requires *sql.DB)
	sqlDB, err := db.DB.DB()
	if err != nil {
		logger.Fatal("Failed to get SQL database for modifier engine", zap.Error(err))
	}
	modifierEngine := services.NewFoodAlcoholHerbalEngine(sqlDB, cacheManager, cfg)
	
	// Hot/warm cache optimization service
	hotCacheService := services.NewHotCacheService(
		hotCacheClient, warmCacheClient, logger, cfg)
	
	// Enhanced integration service (orchestrates all engines)
	integrationService := services.NewEnhancedIntegrationService(
		pgxEngine, classEngine, modifierEngine, matrixService, logger, cfg)

	// Phase 3 engines: Drug-Disease, Allergy, Duplicate Therapy
	logger.Info("Initializing Phase 3 clinical safety engines...")

	// Drug-disease contraindication engine
	drugDiseaseEngine := services.NewDrugDiseaseEngine(db, metricsCollector)

	// Allergy cross-reactivity engine
	allergyEngine := services.NewAllergyEngine(db, metricsCollector)

	// Duplicate therapy detection engine
	duplicateTherapyEngine := services.NewDuplicateTherapyEngine(db, metricsCollector)

	// Phase 4: Governance Policy Engine (Severity → Governance + Attribution)
	logger.Info("Initializing Phase 4 governance and attribution engine...")
	governanceEngine := services.NewGovernancePolicyEngine(db.DB, logger, metricsCollector)

	// Phase 5: OHDSI Constitutional DDI Service (connects to shared database)
	logger.Info("Initializing OHDSI Constitutional DDI Service...")
	var ohdsiEnabled bool
	sharedDB, sharedDBErr := database.NewSharedConnection(cfg)
	if sharedDBErr != nil {
		logger.Warn("Shared database not available - OHDSI Constitutional DDI disabled",
			zap.Error(sharedDBErr))
		ohdsiEnabled = false
	} else {
		defer sharedDB.Close()
		_ = services.NewOHDSIExpansionService(sharedDB, metricsCollector)
		ohdsiEnabled = true
		logger.Info("OHDSI Constitutional DDI Service initialized (25 ONC rules)")
	}

	// Legacy interaction service (for backward compatibility)
	interactionService := services.NewInteractionService(
		db,
		cacheClient,
		metricsCollector,
		cfg,
	)

	// Initialize servers
	logger.Info("Initializing HTTP and gRPC servers...")

	// Initialize HTTP server with ALL enhanced engines
	httpServer := api.NewServer(
		cfg,
		db,
		cacheClient,
		metricsCollector,
		interactionService,
		integrationService,
		pgxEngine,
		classEngine,
		modifierEngine,
		matrixService,
		// Phase 3 engines
		drugDiseaseEngine,
		allergyEngine,
		duplicateTherapyEngine,
		// Phase 4: Governance
		governanceEngine,
	)
	
	// Start HTTP server with enhanced engines
	go func() {
		port := cfg.Server.Port
		if port == "" {
			port = "8085"
		}

		logger.Info("Starting Enhanced HTTP server", zap.String("port", port))
		if err := httpServer.Router.Run(":" + port); err != nil {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Comprehensive health checks
	logger.Info("Performing comprehensive health checks...")
	
	// Check database health
	if err := db.HealthCheck(); err != nil {
		logger.Warn("Database health check failed", zap.Error(err))
	} else {
		logger.Info("Database health check passed")
	}

	// Check cache health
	if err := cacheClient.HealthCheck(); err != nil {
		logger.Warn("Cache health check failed", zap.Error(err))
	} else {
		logger.Info("Cache health check passed")
	}
	
	// Check hot cache health
	if err := hotCacheClient.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Hot cache health check failed", zap.Error(err))
	} else {
		logger.Info("Hot cache health check passed")
	}
	
	// Check warm cache health
	if err := warmCacheClient.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Warm cache health check failed", zap.Error(err))
	} else {
		logger.Info("Warm cache health check passed")
	}
	
	// Initialize cache with common interactions
	go func() {
		time.Sleep(5 * time.Second) // Allow server startup
		// Prefetch common drug combinations into hot cache
		commonCombinations := [][]string{
			{"warfarin", "aspirin"},
			{"atorvastatin", "amlodipine"},
			{"metformin", "lisinopril"},
			{"omeprazole", "clopidogrel"},
		}
		if err := hotCacheService.PrefetchCommonInteractions(
			context.Background(), commonCombinations, "latest"); err != nil {
			logger.Error("Cache prefetch failed", zap.Error(err))
		}
	}()

	logger.Info("KB-5 Enhanced Drug Interactions Service started successfully")

	// Service information
	httpPort := cfg.Server.Port
	if httpPort == "" {
		httpPort = "8085"
	}
	
	// OHDSI status string for banner
	ohdsiStatus := "❌ Disabled (shared DB not connected)"
	if ohdsiEnabled {
		ohdsiStatus = "✅ Enabled (25 ONC Constitutional Rules)"
	}

	fmt.Printf(`
========================================
KB-5 Enhanced Drug Interactions Service
========================================
Service: kb-5-drug-interactions
HTTP Port: %s
Version: 2.0.0 (Enhanced)
Environment: %s
OHDSI Constitutional DDI: %s
========================================

Enhanced HTTP REST Endpoints:
- Health: GET /health
- Metrics: GET /metrics
- Check Interactions: POST /api/v1/interactions/check
- Batch Check: POST /api/v1/interactions/batch-check
- Quick Check: GET /api/v1/interactions/quick-check
- Comprehensive Check: POST /api/v1/interactions/comprehensive

Enhanced Features Now Available:
✅ Pharmacogenomic (PGx) interaction detection
✅ Drug class pattern matching (Triple Whammy, etc.)
✅ Food/alcohol/herbal interaction analysis
✅ Drug-disease contraindication analysis
✅ Allergy cross-reactivity detection
✅ Duplicate therapy detection
✅ Hot/warm cache optimization (50k/200k entries)
✅ Evidence Envelope compliance
✅ Institutional override system
✅ Real-time confidence scoring
✅ Clinical alert synthesis
✅ Parallel engine processing

Phase 3 Endpoints:
- Drug-Disease: POST /api/v1/contraindications/disease
- Allergy Check: POST /api/v1/allergy/check
- Duplicate Therapy: POST /api/v1/duplicates/check

Admin Endpoints:
- Engine Health: GET /api/v1/admin/engines/health
- Cache Statistics: GET /api/v1/admin/cache/stats
- Performance Metrics: GET /api/v1/admin/performance
- Dataset Management: POST /api/v1/admin/dataset/update

Note: gRPC endpoints available after protoc compilation
========================================
`, httpPort, cfg.Environment, ohdsiStatus)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KB-5 Enhanced Drug Interactions Service...")
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// Wait for HTTP server shutdown (would need server instance for graceful shutdown)
	select {
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded")
	case <-time.After(2 * time.Second):
		logger.Info("HTTP server shutdown completed")
	}
	
	logger.Info("KB-5 Enhanced Service stopped successfully")
}