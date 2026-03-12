package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/config"
	"kb-7-terminology/internal/database"
	"kb-7-terminology/internal/etl"
	"kb-7-terminology/internal/metrics"
	"go.uber.org/zap"
	"github.com/sirupsen/logrus"
)

func main() {
	// Command line flags
	var (
		configPath    = flag.String("config", "", "Path to configuration file")
		dataDir       = flag.String("data", "./data", "Directory containing terminology data")
		system        = flag.String("system", "all", "Terminology system to load (all, rxnorm, loinc, snomed, icd10)")
		batchSize     = flag.Int("batch", 1000, "Batch size for database operations")
		maxWorkers    = flag.Int("workers", 2, "Maximum number of worker threads")
		validateOnly  = flag.Bool("validate", false, "Only validate data, don't load")
		enableDebug   = flag.Bool("debug", false, "Enable debug logging")
		skipExisting  = flag.Bool("skip-existing", false, "Skip loading if system already exists")
		// GraphDB flags
		enableGraphDB = flag.Bool("enable-graphdb", false, "Enable GraphDB triple loading (default: false)")
		graphDBURL    = flag.String("graphdb-url", "http://localhost:7200", "GraphDB server URL")
		graphDBRepo   = flag.String("graphdb-repo", "kb7-terminology", "GraphDB repository ID")
		graphDBBatch  = flag.Int("graphdb-batch-size", 1000, "Concepts per batch for GraphDB upload")
	)
	flag.Parse()

	// Initialize logger
	var logger *zap.Logger
	var err error
	
	if *enableDebug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting KB-7 Terminology ETL Process",
		zap.String("data_directory", *dataDir),
		zap.String("system", *system),
		zap.Int("batch_size", *batchSize),
		zap.Int("max_workers", *maxWorkers),
		zap.Bool("validate_only", *validateOnly))

	// Load configuration
	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath)
	} else {
		cfg, err = config.LoadConfig()
	}
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Skip database connection in validation-only mode
	var db *database.Connection
	if !*validateOnly {
		// Connect to database
		db, err = database.NewConnection(cfg.DatabaseURL)
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
		}
		defer db.Close()
		logger.Info("Connected to terminology database successfully")
	} else {
		logger.Info("Validation-only mode - skipping database connection")
	}

	// Validate data directory
	if _, err := os.Stat(*dataDir); os.IsNotExist(err) {
		logger.Fatal("Data directory does not exist", zap.String("path", *dataDir))
	}

	// Configure ETL coordinator
	coordinatorConfig := etl.CoordinatorConfig{
		BatchSize:         *batchSize,
		MaxWorkers:        *maxWorkers,
		ValidationEnabled: true,
		BackupEnabled:     false,
		ParallelLoading:   true,
		RetryAttempts:     3,
	}

	// Handle validation-only mode separately
	if *validateOnly {
		logger.Info("Running in validation-only mode")

		if *system == "all" {
			// Validate all systems
			systems := []string{"rxnorm", "loinc", "snomed", "icd10"}
			allValid := true

			for _, sys := range systems {
				if err := validateSystem(sys, *dataDir, logger); err != nil {
					logger.Error("Validation failed", zap.String("system", sys), zap.Error(err))
					allValid = false
				} else {
					logger.Info("Validation passed", zap.String("system", sys))
				}
			}

			if allValid {
				logger.Info("All systems passed validation")
				os.Exit(0)
			} else {
				logger.Error("Some systems failed validation")
				os.Exit(1)
			}
		} else {
			if err := validateSystem(*system, *dataDir, logger); err != nil {
				logger.Fatal("Validation failed", zap.String("system", *system), zap.Error(err))
			}
			logger.Info("Validation passed", zap.String("system", *system))
		}
		return
	}

	// Create cache configuration for ETL operations
	cacheConfig := cache.CacheConfig{
		L1Config: cache.L1Config{
			MaxSizeMB:     100,
			TTL:           30 * time.Minute,
			NumCounters:   1e7,
			MaxCost:       1 << 30,
			BufferItems:   64,
			HitRateTarget: 0.8,
		},
		L2Config: cache.L2Config{
			TTL:           2 * time.Hour,
			MaxSizeMB:     500,
			HitRateTarget: 0.9,
		},
		L3Config: cache.L3Config{
			TTL:           24 * time.Hour,
			MaxSizeMB:     1000,
			HitRateTarget: 0.95,
		},
	}

	// Create multi-layer cache (simplified for ETL - no Redis required)
	mlCache, err := cache.NewMultiLayerCache(cacheConfig, nil, nil)
	if err != nil {
		logger.Warn("Failed to create multi-layer cache, using basic cache", zap.Error(err))
		mlCache = nil
	}

	// Create metrics collector
	metricsCollector := metrics.NewEnhancedCollector("kb7_etl", db.DB, mlCache, logrus.New())

	// Create base coordinator
	enhancedCoordinator := etl.NewEnhancedCoordinator(db.DB, mlCache, logger, metricsCollector, coordinatorConfig)

	// Create dual-store configuration for Elasticsearch integration
	dualStoreConfig := &etl.DualStoreConfig{
		EnableElasticsearch:     true,
		ElasticsearchURLs:       []string{"http://localhost:9200"},
		IndexName:               "kb7-terminology",
		DualWriteMode:           etl.DualWriteParallel,
		ConsistencyCheckEnabled: true,
		ConsistencyThreshold:    0.95,
		ElasticsearchBatchSize:  1000,
		MaxRetries:              3,
		RetryDelay:              time.Second * 2,
		EnableRollback:          true,
		TransactionTimeout:      time.Minute * 5,
	}

	// Create dual-store coordinator for PostgreSQL + Elasticsearch
	dualCoordinator, err := etl.NewDualStoreCoordinator(enhancedCoordinator, dualStoreConfig, logger)
	if err != nil {
		logger.Fatal("Failed to create dual-store coordinator", zap.Error(err))
	}

	// Create GraphDB configuration if enabled
	var coordinator interface {
		LoadAllTerminologiesTripleStore(context.Context, map[string]string) error
		GetStatus() *etl.ETLStatus
	}

	if *enableGraphDB {
		logger.Info("GraphDB triple loading enabled",
			zap.String("server_url", *graphDBURL),
			zap.String("repository", *graphDBRepo))

		graphDBConfig := &etl.GraphDBConfig{
			Enabled:            true,
			ServerURL:          *graphDBURL,
			RepositoryID:       *graphDBRepo,
			BatchSize:          10000,
			MaxRetries:         3,
			RetryDelay:         5 * time.Second,
			EnableInference:    true,
			NamedGraph:         "http://cardiofit.ai/kb7/graph/default",
			TransactionTimeout: 10 * time.Minute,
			ValidateTriples:    true,
			ConceptBatchSize:   *graphDBBatch,
		}

		// Create triple-store coordinator
		tripleCoordinator, err := etl.NewTripleStoreCoordinator(dualCoordinator, graphDBConfig, logger)
		if err != nil {
			logger.Warn("Failed to create triple-store coordinator, falling back to dual-store only", zap.Error(err))
			coordinator = dualCoordinator
		} else {
			coordinator = tripleCoordinator
			logger.Info("Triple-store coordinator created successfully")
		}
	} else {
		logger.Info("GraphDB triple loading disabled (use --enable-graphdb to enable)")
		coordinator = dualCoordinator
	}

	// Check if systems already exist (if skip-existing is enabled)
	if *skipExisting {
		status := coordinator.GetStatus()
		if status.OverallStatus != "idle" {
			logger.Info("ETL system already active", zap.String("status", status.OverallStatus))
			fmt.Println("Systems already loaded or in progress. Use --skip-existing=false to reload.")
			return
		}
	}

	// Validation mode
	if *validateOnly {
		logger.Info("Running in validation-only mode")
		
		if *system == "all" {
			// Validate all systems
			systems := []string{"rxnorm", "loinc", "snomed", "icd10"}
			allValid := true
			
			for _, sys := range systems {
				if err := validateSystem(sys, *dataDir, logger); err != nil {
					logger.Error("Validation failed", zap.String("system", sys), zap.Error(err))
					allValid = false
				} else {
					logger.Info("Validation passed", zap.String("system", sys))
				}
			}
			
			if allValid {
				logger.Info("All systems passed validation")
				os.Exit(0)
			} else {
				logger.Error("Some systems failed validation")
				os.Exit(1)
			}
		} else {
			if err := validateSystem(*system, *dataDir, logger); err != nil {
				logger.Fatal("Validation failed", zap.String("system", *system), zap.Error(err))
			}
			logger.Info("Validation passed", zap.String("system", *system))
		}
		return
	}

	// Loading mode
	if *system == "all" {
		// Load all terminology systems
		logger.Info("Loading all terminology systems")

		dataSources := map[string]string{
			"SNOMED": *dataDir + "/snomed",
			"RxNorm": *dataDir + "/rxnorm",
			"LOINC":  *dataDir + "/loinc",
			"ICD10":  *dataDir + "/icd10",
		}

		// Use triple-store loading (backward compatible with dual-store)
		ctx := context.Background()
		err := coordinator.LoadAllTerminologiesTripleStore(ctx, dataSources)

		if err != nil {
			logger.Fatal("Failed to load terminologies", zap.Error(err))
		}

	} else {
		// Load specific system
		logger.Info("Loading specific terminology system", zap.String("system", *system))

		if err := loadSpecificSystem(*system, *dataDir, db, logger, coordinatorConfig); err != nil {
			logger.Fatal("Failed to load terminology system",
				zap.String("system", *system),
				zap.Error(err))
		}
	}

	// Generate final status report
	finalStatus := coordinator.GetStatus()
	logger.Info("Final ETL status",
		zap.String("overall_status", finalStatus.OverallStatus),
		zap.String("current_operation", finalStatus.CurrentOperation),
		zap.Duration("duration", time.Since(finalStatus.StartTime)))

	logger.Info("KB-7 Terminology ETL Process completed successfully")
}

// validateSystem validates a specific terminology system
func validateSystem(system, dataDir string, logger *zap.Logger) error {
	// For now, just validate that the data directory exists
	systemPath := dataDir + "/" + system
	if _, err := os.Stat(systemPath); os.IsNotExist(err) {
		return fmt.Errorf("data directory does not exist: %s", systemPath)
	}

	logger.Info("Data directory validation passed", zap.String("system", system), zap.String("path", systemPath))
	return nil
}

// loadSpecificSystem loads a specific terminology system using coordinator
func loadSpecificSystem(system, dataDir string, db *database.Connection, logger *zap.Logger, config etl.CoordinatorConfig) error {
	// Create a simple cache for single system loading
	cacheConfig := cache.CacheConfig{
		L1Config: cache.L1Config{
			MaxSizeMB:     50,
			TTL:           15 * time.Minute,
			NumCounters:   1e6,
			MaxCost:       1 << 29,
			BufferItems:   64,
			HitRateTarget: 0.8,
		},
	}

	mlCache, err := cache.NewMultiLayerCache(cacheConfig, nil, nil)
	if err != nil {
		logger.Warn("Failed to create cache for single system loading", zap.Error(err))
		mlCache = nil
	}

	metricsCollector := metrics.NewEnhancedCollector("kb7_single", db.DB, mlCache, logrus.New())

	// Create base coordinator
	enhancedCoordinator := etl.NewEnhancedCoordinator(db.DB, mlCache, logger, metricsCollector, config)

	// Create dual-store configuration for Elasticsearch integration
	dualStoreConfig := &etl.DualStoreConfig{
		EnableElasticsearch:     true,
		ElasticsearchURLs:       []string{"http://localhost:9200"},
		IndexName:               "kb7-terminology",
		DualWriteMode:           etl.DualWriteParallel,
		ConsistencyCheckEnabled: true,
		ConsistencyThreshold:    0.95,
		ElasticsearchBatchSize:  1000,
		MaxRetries:              3,
		RetryDelay:              time.Second * 2,
		EnableRollback:          true,
		TransactionTimeout:      time.Minute * 5,
	}

	// Create dual-store coordinator for PostgreSQL + Elasticsearch
	coordinator, err := etl.NewDualStoreCoordinator(enhancedCoordinator, dualStoreConfig, logger)
	if err != nil {
		logger.Error("Failed to create dual-store coordinator, falling back to PostgreSQL-only", zap.Error(err))
		// Fall back to PostgreSQL-only if Elasticsearch is not available
		coordinator = &etl.DualStoreCoordinator{
			EnhancedCoordinator: enhancedCoordinator,
		}
	}

	// Create data source map for single system with proper case mapping
	var systemKey string
	switch strings.ToLower(system) {
	case "snomed":
		systemKey = "SNOMED"
	case "rxnorm":
		systemKey = "RxNorm"
	case "loinc":
		systemKey = "LOINC"
	case "icd10":
		systemKey = "ICD10"
	default:
		systemKey = strings.ToUpper(system)
	}

	dataSources := map[string]string{
		systemKey: dataDir,
	}

	// Use dual-store loading
	ctx := context.Background()
	return coordinator.LoadAllTerminologiesDualStore(ctx, dataSources)
}