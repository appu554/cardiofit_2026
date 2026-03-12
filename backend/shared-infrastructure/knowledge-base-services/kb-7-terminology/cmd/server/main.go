package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kb-7-terminology/internal/api"
	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/cdss"
	"kb-7-terminology/internal/config"
	"kb-7-terminology/internal/database"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/semantic"
	"kb-7-terminology/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logging
	logger := logrus.New()
	logger.SetLevel(logrus.Level(cfg.LogLevel))
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.WithFields(logrus.Fields{
		"service": "kb-7-terminology",
		"version": cfg.Version,
		"port":    cfg.Port,
	}).Info("Starting KB-7 Terminology Service")

	// Initialize database connection
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(db, cfg.MigrationsPath); err != nil {
		logger.Fatalf("Failed to run database migrations: %v", err)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// CRITICAL STARTUP CHECK: Verify precomputed ValueSet codes exist
	// CTO/CMO DIRECTIVE: "CQL does not need a terminology ENGINE — it needs a terminology ANSWER."
	// KB-7 MUST refuse to start if no precomputed codes exist (silent data bug prevention)
	// ═══════════════════════════════════════════════════════════════════════════
	var precomputedCodeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM precomputed_valueset_codes").Scan(&precomputedCodeCount)
	if err != nil {
		logger.Warnf("Could not check precomputed_valueset_codes table: %v (table may not exist yet)", err)
		precomputedCodeCount = -1 // Signal that check failed
	}

	if precomputedCodeCount == 0 {
		logger.Error("╔══════════════════════════════════════════════════════════════════════════════╗")
		logger.Error("║  CRITICAL: KB-7 CANNOT START - NO PRECOMPUTED VALUESET CODES!                ║")
		logger.Error("╠══════════════════════════════════════════════════════════════════════════════╣")
		logger.Error("║  The precomputed_valueset_codes table is EMPTY.                              ║")
		logger.Error("║  This means $validate-code and $expand will NOT work correctly.              ║")
		logger.Error("║                                                                              ║")
		logger.Error("║  TO FIX: Run the materialization script BEFORE starting KB-7:                ║")
		logger.Error("║                                                                              ║")
		logger.Error("║    python3 scripts/kb7-materialize-all.py                                    ║")
		logger.Error("║                                                                              ║")
		logger.Error("║  This is a BUILD-TIME job that precomputes ValueSet expansions.              ║")
		logger.Error("║  It must run ONCE after SNOMED data load or ValueSet definition changes.     ║")
		logger.Error("║                                                                              ║")
		logger.Error("║  Architecture: CQL needs ANSWERS, not an ENGINE at runtime (CTO/CMO)         ║")
		logger.Error("╚══════════════════════════════════════════════════════════════════════════════╝")
		logger.Fatalf("KB-7 startup blocked: precomputed_valueset_codes is empty. Run kb7-materialize-all.py first.")
	} else if precomputedCodeCount > 0 {
		// Log success with count of precomputed codes
		var distinctValueSets int
		db.QueryRow("SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes").Scan(&distinctValueSets)
		logger.WithFields(logrus.Fields{
			"precomputed_codes":    precomputedCodeCount,
			"distinct_value_sets":  distinctValueSets,
		}).Info("✅ Startup check passed: precomputed ValueSet codes available")
	}
	// If precomputedCodeCount == -1, we couldn't check (table may not exist), so continue with warning

	// Initialize Redis cache
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize GraphDB client (Semantic Layer)
	var graphDBClient *semantic.GraphDBClient
	if cfg.GraphDBEnabled {
		graphDBClient = semantic.NewGraphDBClient(cfg.GraphDBURL, cfg.GraphDBRepository, logger)
		if cfg.GraphDBUsername != "" {
			graphDBClient.SetAuthentication(cfg.GraphDBUsername, cfg.GraphDBPassword)
		}

		// Verify GraphDB connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := graphDBClient.HealthCheck(ctx); err != nil {
			logger.WithError(err).Warn("GraphDB health check failed - semantic features will be unavailable")
			graphDBClient = nil
		} else {
			logger.WithFields(logrus.Fields{
				"url":        cfg.GraphDBURL,
				"repository": cfg.GraphDBRepository,
			}).Info("Connected to GraphDB semantic layer")
		}
		cancel()
	}

	// Initialize metrics
	metricsCollector := metrics.NewCollector("kb_7_terminology")

	// Initialize services
	terminologyService := services.NewTerminologyService(db, redisClient, logger, metricsCollector)

	// Initialize ValueSetService (database-driven mode - no hardcoded value sets)
	// Value sets are now managed via RuleManager and stored in PostgreSQL
	valueSetService := services.NewValueSetService(db, redisClient, logger, metricsCollector)
	logger.Info("ValueSetService initialized (database-driven mode - hardcoded value sets removed)")

	// Initialize SubsumptionService (OWL reasoning for concept hierarchy)
	var subsumptionService *services.SubsumptionService
	if graphDBClient != nil {
		subsumptionService = services.NewSubsumptionService(graphDBClient, redisClient, logger)
		logger.Info("SubsumptionService initialized with GraphDB backend for OWL reasoning")
	} else {
		logger.Warn("SubsumptionService disabled - GraphDB not available")
	}

	// Initialize Neo4jBridge for fast subsumption testing (Phase 7 - Multi-region)
	// Neo4j has pre-computed ELK hierarchy materialized via rdfs__subClassOf relationships
	var neo4jBridge *services.Neo4jBridge
	if cfg.Neo4jMultiRegionEnabled {
		// Build regional Neo4j config from .env
		regionalConfig := &semantic.RegionalNeo4jConfig{
			DefaultRegion: semantic.Region(cfg.Neo4jDefaultRegion),
			Regions:       make(map[semantic.Region]*semantic.Neo4jConfig),
		}

		// Add enabled regions to the config
		for regionName, regionCfg := range cfg.Neo4jRegions {
			if regionCfg.Enabled {
				region := semantic.Region(strings.ToLower(regionName))
				regionalConfig.Regions[region] = &semantic.Neo4jConfig{
					URL:            regionCfg.URL,
					Username:       regionCfg.Username,
					Password:       regionCfg.Password,
					Database:       regionCfg.Database,
					MaxConnections: 50,
					ConnTimeout:    10 * time.Second,
					ReadTimeout:    30 * time.Second,
				}
				logger.WithFields(logrus.Fields{
					"region":   regionName,
					"url":      regionCfg.URL,
					"database": regionCfg.Database,
				}).Info("Configured Neo4j region")
			}
		}

		// Create Neo4jBridge with regional manager
		if len(regionalConfig.Regions) > 0 {
			bridgeConfig := &services.Neo4jBridgeConfig{
				Neo4jURL:            cfg.Neo4jURL,
				Neo4jUsername:       cfg.Neo4jUsername,
				Neo4jPassword:       cfg.Neo4jPassword,
				Neo4jDatabase:       cfg.Neo4jDatabase,
				FallbackEnabled:     true,
				PreferNeo4j:         true,
				CacheEnabled:        true,
				CacheTTL:            30 * time.Minute,
				ConceptCacheTTL:     1 * time.Hour,
				HierarchyCacheTTL:   30 * time.Minute,
			}

			// Use AU region for AU testing (explicit preference over random map iteration)
			// Priority: AU > default region > any enabled region
			selectedRegion := ""
			regionOrder := []string{"AU", strings.ToUpper(cfg.Neo4jDefaultRegion)}

			// Try preferred regions first
			for _, regionName := range regionOrder {
				if regionCfg, ok := cfg.Neo4jRegions[regionName]; ok && regionCfg.Enabled {
					bridgeConfig.Neo4jURL = regionCfg.URL
					bridgeConfig.Neo4jUsername = regionCfg.Username
					bridgeConfig.Neo4jPassword = regionCfg.Password
					bridgeConfig.Neo4jDatabase = regionCfg.Database
					selectedRegion = regionName
					break
				}
			}

			// Fallback to any enabled region if AU not found
			if selectedRegion == "" {
				for regionName, regionCfg := range cfg.Neo4jRegions {
					if regionCfg.Enabled {
						bridgeConfig.Neo4jURL = regionCfg.URL
						bridgeConfig.Neo4jUsername = regionCfg.Username
						bridgeConfig.Neo4jPassword = regionCfg.Password
						bridgeConfig.Neo4jDatabase = regionCfg.Database
						selectedRegion = regionName
						break
					}
				}
			}

			if selectedRegion != "" {
				logger.WithField("region", selectedRegion).Info("Using Neo4j region for subsumption")
			}

			var err error
			neo4jBridge, err = services.NewNeo4jBridge(bridgeConfig, graphDBClient, redisClient, logger)
			if err != nil {
				logger.WithError(err).Warn("Failed to initialize Neo4jBridge - subsumption will use GraphDB fallback")
			} else {
				logger.Info("Neo4jBridge initialized for fast subsumption testing (ELK materialized hierarchy)")
			}
		}
	}

	// Initialize RuleManager (database-driven value set rule engine)
	// This replaces hardcoded builtin_valuesets.go with dynamic PostgreSQL-backed rules
	// NOW WITH THREE-CHECK PIPELINE: Expansion → Exact Match → Subsumption
	// Priority: Neo4jBridge (fast) > SubsumptionService (GraphDB fallback)
	ruleManager := services.NewRuleManager(db, redisClient, graphDBClient, subsumptionService, neo4jBridge, logger, metricsCollector)
	if neo4jBridge != nil {
		logger.Info("RuleManager initialized with THREE-CHECK PIPELINE via Neo4jBridge (fast ELK hierarchy)")
	} else if subsumptionService != nil {
		logger.Info("RuleManager initialized with THREE-CHECK PIPELINE via GraphDB (OWL reasoning)")
	} else {
		logger.Warn("RuleManager initialized with TWO-CHECK PIPELINE only (Expansion + Membership) - Subsumption disabled")
	}

	// Optionally seed builtin value sets to database on first run
	// This migrates the 18 FHIR R4 value sets from hardcoded Go to PostgreSQL
	if cfg.SeedBuiltinValueSets {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		if err := ruleManager.SeedBuiltinValueSets(ctx); err != nil {
			logger.WithError(err).Warn("Failed to seed builtin value sets - they may already exist")
		} else {
			logger.Info("Successfully seeded builtin value sets to database")
		}
		cancel()
	}

	// Initialize TerminologyBridge - Multi-Layer Caching (L0 Bloom → L1 Hot → L2 Local → L2.5 Redis → L3 Neo4j)
	// This provides sub-millisecond clinical terminology validation for high-throughput scenarios
	var terminologyBridge *services.TerminologyBridge
	if neo4jBridge != nil || graphDBClient != nil {
		bridgeConfig := services.DefaultTerminologyBridgeConfig()
		// Configure hot value sets for pre-loading (clinical protocols most commonly validated)
		bridgeConfig.HotValueSets = []string{
			"SepsisDiagnosis",
			"AcuteRenalFailure",
			"AUAKIConditions",
			"AUSepsisConditions",
			"DiabetesMellitus",
			"Hypertension",
		}
		bridgeConfig.RedisEnabled = redisClient != nil

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		var err error
		terminologyBridge, err = services.NewTerminologyBridge(
			ctx,
			bridgeConfig,
			nil, // Direct Neo4j client not needed - uses RuleManager for THREE-CHECK PIPELINE
			redisClient,
			ruleManager,
			logger,
		)
		cancel()

		if err != nil {
			logger.WithError(err).Warn("Failed to initialize TerminologyBridge - multi-layer caching disabled")
		} else {
			logger.WithFields(logrus.Fields{
				"hot_value_sets":   len(bridgeConfig.HotValueSets),
				"redis_enabled":    bridgeConfig.RedisEnabled,
				"bloom_filter":     "enabled",
				"subsumption":      bridgeConfig.EnableSubsumption,
			}).Info("TerminologyBridge initialized with multi-layer caching (L0→L1→L2→L2.5→L3)")
		}
	} else {
		logger.Warn("TerminologyBridge disabled - Neo4j or GraphDB required for caching layer")
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize API server with GraphDB client for semantic operations and RuleManager
	// Pass neo4jBridge for fast subsumption testing via ELK materialized hierarchy
	// Pass terminologyBridge for multi-layer caching (L0 Bloom → L1 Hot → L2 Local → L2.5 Redis → L3 Neo4j)
	apiServer := api.NewServer(cfg, terminologyService, valueSetService, subsumptionService, neo4jBridge, terminologyBridge, ruleManager, graphDBClient, logger, metricsCollector)

	// Initialize CDSS (Clinical Decision Support System) services
	// Enables patient-level evaluation: FHIR Resources → Facts → Evaluation → Alerts
	factBuilder := cdss.NewFactBuilder(logger)
	alertGenerator := cdss.NewAlertGenerator(logger)

	// Initialize Rule Repository for persistent rule storage
	// Rules are stored in PostgreSQL clinical_rules table with fallback to in-memory defaults
	ruleRepository := cdss.NewPostgresRuleRepository(db, logger)

	// Initialize Rule Engine for compound conditions and threshold evaluation
	// Supports: value set matches + lab thresholds (e.g., "Sepsis AND Lactate > 2.0")
	// Uses database as primary source with fallback to in-memory defaults
	ruleEngine := cdss.NewRuleEngineWithRepository(ruleManager, ruleRepository, logger)
	logger.WithField("rules_loaded", len(ruleEngine.GetRules())).Info("RuleEngine initialized with clinical rules (DB-backed with fallback)")

	cdssEvaluator := cdss.NewCDSSEvaluator(
		factBuilder,
		ruleManager,
		ruleEngine,
		alertGenerator,
		logger,
		cdss.DefaultCDSSEvaluatorConfig(),
	)

	// Create CDSS handlers and attach to API server
	// Use the WithRepository constructor to enable database persistence for clinical rules
	cdssHandlers := api.NewCDSSHandlersWithRepository(factBuilder, cdssEvaluator, alertGenerator, ruleRepository, logger)
	apiServer.SetCDSSHandlers(cdssHandlers)
	logger.Info("CDSS services initialized (FactBuilder, RuleEngine, CDSSEvaluator, AlertGenerator, RuleRepository)")

	// Initialize NCTS Refset Service for reference set queries
	// Uses Neo4j for fast IN_REFSET relationship lookups
	if neo4jBridge != nil && neo4jBridge.IsNeo4jAvailable() {
		// Get the Neo4j client from the bridge for refset operations
		neo4jClient := neo4jBridge.GetNeo4jClient()
		if neo4jClient != nil {
			refsetService := services.NewRefsetService(neo4jClient, redisClient, logger)
			refsetHandlers := api.NewRefsetHandlers(refsetService, logger)
			apiServer.SetRefsetHandlers(refsetHandlers)
			logger.Info("RefsetService initialized for NCTS reference set queries via Neo4j")
		} else {
			logger.Warn("RefsetService disabled - Neo4jClient not available from bridge")
		}
	} else {
		logger.Warn("RefsetService disabled - Neo4jBridge not available")
	}

	// Initialize FHIR R4 Handlers for CQL Integration (Ontoserver ValueSets)
	// CRITICAL ARCHITECTURE: These handlers use ONLY precomputed PostgreSQL expansions - NO Neo4j at runtime!
	// - BUILD TIME: materialize_expansions.py runs Neo4j traversal → precomputed_valueset_codes table
	// - RUNTIME: Pure SELECT from PostgreSQL (target <50ms)
	fhirHandlers := api.NewFHIRHandlers(db, logger)
	apiServer.SetFHIRHandlers(fhirHandlers)
	logger.Info("FHIR R4 handlers initialized for CQL integration (pure PostgreSQL read, no runtime Neo4j)")

	// Setup routes
	router := apiServer.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", cfg.Port).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}