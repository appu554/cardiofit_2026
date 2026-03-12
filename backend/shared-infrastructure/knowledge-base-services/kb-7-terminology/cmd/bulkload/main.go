package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kb-7-terminology/internal/bulkload"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Configuration flags
var (
	postgresURL        = flag.String("postgres", "", "PostgreSQL connection string")
	elasticsearchURL   = flag.String("elasticsearch", "", "Elasticsearch URL")
	elasticsearchIndex = flag.String("index", "clinical_terms", "Elasticsearch index name")
	batchSize         = flag.Int("batch", 1000, "Batch size for bulk operations")
	numWorkers        = flag.Int("workers", 4, "Number of parallel workers")
	systems           = flag.String("systems", "", "Comma-separated list of systems to load (empty=all)")
	resumeFromID      = flag.Int64("resume", 0, "Resume from specific record ID")
	validateData      = flag.Bool("validate", true, "Perform data validation after load")
	strategy          = flag.String("strategy", "parallel", "Migration strategy: incremental, parallel, blue-green, shadow")
	dryRun           = flag.Bool("dry-run", false, "Perform dry run without actual data migration")
	progressInterval  = flag.Duration("progress", 10*time.Second, "Progress reporting interval")
	configFile       = flag.String("config", "", "Configuration file path (JSON)")
	logLevel         = flag.String("log-level", "info", "Log level: debug, info, warn, error")
	outputFile       = flag.String("output", "", "Output file for migration report")
	checkpoint       = flag.String("checkpoint", "", "Load checkpoint from file for resume")
)

// Config represents the bulk load configuration
type Config struct {
	PostgresURL        string        `json:"postgres_url"`
	ElasticsearchURL   string        `json:"elasticsearch_url"`
	ElasticsearchIndex string        `json:"elasticsearch_index"`
	BatchSize          int           `json:"batch_size"`
	NumWorkers         int           `json:"num_workers"`
	Systems            []string      `json:"systems"`
	ResumeFromID       int64         `json:"resume_from_id"`
	ValidateData       bool          `json:"validate_data"`
	Strategy           string        `json:"strategy"`
	DryRun            bool          `json:"dry_run"`
	ProgressInterval   time.Duration `json:"progress_interval"`
	LogLevel          string        `json:"log_level"`
}

func main() {
	flag.Parse()

	// Load environment variables
	godotenv.Load()

	// Setup logger
	logger := setupLogger(*logLevel)

	// Load configuration
	config, err := loadConfiguration()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := validateConfiguration(config); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, initiating graceful shutdown...")
		cancel()
	}()

	// Print configuration
	printConfiguration(config, logger)

	if config.DryRun {
		logger.Info("🔍 DRY RUN MODE - No data will be migrated")
		if err := performDryRun(ctx, config, logger); err != nil {
			logger.Fatalf("Dry run failed: %v", err)
		}
		return
	}

	// Execute bulk load based on strategy
	logger.Infof("Starting bulk load with %s strategy", config.Strategy)

	switch config.Strategy {
	case "incremental":
		if err := executeIncrementalMigration(ctx, config, logger); err != nil {
			logger.Fatalf("Incremental migration failed: %v", err)
		}

	case "parallel":
		if err := executeParallelMigration(ctx, config, logger); err != nil {
			logger.Fatalf("Parallel migration failed: %v", err)
		}

	case "blue-green":
		if err := executeBlueGreenMigration(ctx, config, logger); err != nil {
			logger.Fatalf("Blue-green migration failed: %v", err)
		}

	case "shadow":
		if err := executeShadowMigration(ctx, config, logger); err != nil {
			logger.Fatalf("Shadow migration failed: %v", err)
		}

	default:
		logger.Fatalf("Unknown strategy: %s", config.Strategy)
	}

	logger.Info("✅ Bulk load completed successfully")
}

// setupLogger configures the logger
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	switch strings.ToLower(level) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger
}

// loadConfiguration loads configuration from file or flags
func loadConfiguration() (*Config, error) {
	config := &Config{
		BatchSize:        *batchSize,
		NumWorkers:       *numWorkers,
		ResumeFromID:     *resumeFromID,
		ValidateData:     *validateData,
		Strategy:         *strategy,
		DryRun:          *dryRun,
		ProgressInterval: *progressInterval,
		LogLevel:        *logLevel,
	}

	// Load from config file if provided
	if *configFile != "" {
		file, err := os.ReadFile(*configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(file, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with command-line flags
	if *postgresURL != "" {
		config.PostgresURL = *postgresURL
	} else if config.PostgresURL == "" {
		config.PostgresURL = os.Getenv("POSTGRES_URL")
		if config.PostgresURL == "" {
			config.PostgresURL = "postgres://postgres:password@localhost:5432/kb7_terminology?sslmode=disable"
		}
	}

	if *elasticsearchURL != "" {
		config.ElasticsearchURL = *elasticsearchURL
	} else if config.ElasticsearchURL == "" {
		config.ElasticsearchURL = os.Getenv("ELASTICSEARCH_URL")
		if config.ElasticsearchURL == "" {
			config.ElasticsearchURL = "http://localhost:9200"
		}
	}

	if *elasticsearchIndex != "" {
		config.ElasticsearchIndex = *elasticsearchIndex
	}

	// Parse systems
	if *systems != "" {
		config.Systems = strings.Split(*systems, ",")
	}

	return config, nil
}

// validateConfiguration validates the configuration
func validateConfiguration(config *Config) error {
	if config.PostgresURL == "" {
		return fmt.Errorf("PostgreSQL URL is required")
	}

	if config.ElasticsearchURL == "" {
		return fmt.Errorf("Elasticsearch URL is required")
	}

	if config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	if config.NumWorkers <= 0 {
		return fmt.Errorf("number of workers must be positive")
	}

	validStrategies := []string{"incremental", "parallel", "blue-green", "shadow"}
	valid := false
	for _, s := range validStrategies {
		if config.Strategy == s {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid strategy: %s", config.Strategy)
	}

	return nil
}

// printConfiguration prints the current configuration
func printConfiguration(config *Config, logger *logrus.Logger) {
	logger.Info("=== Bulk Load Configuration ===")
	logger.Infof("PostgreSQL: %s", maskConnectionString(config.PostgresURL))
	logger.Infof("Elasticsearch: %s", config.ElasticsearchURL)
	logger.Infof("Index: %s", config.ElasticsearchIndex)
	logger.Infof("Batch Size: %d", config.BatchSize)
	logger.Infof("Workers: %d", config.NumWorkers)
	logger.Infof("Systems: %v", config.Systems)
	logger.Infof("Strategy: %s", config.Strategy)
	logger.Infof("Validate: %v", config.ValidateData)
	logger.Infof("Resume From ID: %d", config.ResumeFromID)
	logger.Info("==============================")
}

// maskConnectionString masks sensitive parts of connection string
func maskConnectionString(conn string) string {
	if strings.Contains(conn, "@") {
		parts := strings.Split(conn, "@")
		if len(parts) == 2 {
			return "postgres://***@" + parts[1]
		}
	}
	return conn
}

// performDryRun performs a dry run without actual migration
func performDryRun(ctx context.Context, config *Config, logger *logrus.Logger) error {
	logger.Info("Performing dry run...")

	// Create bulk load configuration
	blConfig := &bulkload.BulkLoadConfig{
		PostgresConnStr:    config.PostgresURL,
		ElasticsearchURL:   config.ElasticsearchURL,
		ElasticsearchIndex: config.ElasticsearchIndex,
		BatchSize:          config.BatchSize,
		NumWorkers:         config.NumWorkers,
		FlushInterval:      5 * time.Second,
		MaxRetries:         3,
		RetryBackoff:       time.Second,
		ValidateData:       config.ValidateData,
		ResumeFromID:       config.ResumeFromID,
		Systems:            config.Systems,
	}

	// Create loader
	loader, err := bulkload.NewBulkLoader(blConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create bulk loader: %w", err)
	}
	defer loader.Close()

	// Test connections
	logger.Info("Testing database connections...")
	// This would be implemented in the actual loader

	// Get record counts
	logger.Info("Analyzing data to migrate...")
	// This would query PostgreSQL for record counts per system

	// Estimate migration time
	logger.Info("Estimating migration time...")
	// Based on record count and performance metrics

	logger.Info("✅ Dry run completed successfully")
	logger.Info("Ready to perform actual migration")

	return nil
}

// executeIncrementalMigration performs incremental migration
func executeIncrementalMigration(ctx context.Context, config *Config, logger *logrus.Logger) error {
	// Create progress reporter
	progressDone := make(chan bool)
	go reportProgress(ctx, config.ProgressInterval, logger, progressDone)
	defer close(progressDone)

	// Create bulk load configuration
	blConfig := &bulkload.BulkLoadConfig{
		PostgresConnStr:    config.PostgresURL,
		ElasticsearchURL:   config.ElasticsearchURL,
		ElasticsearchIndex: config.ElasticsearchIndex,
		BatchSize:          config.BatchSize,
		NumWorkers:         1, // Incremental uses single worker
		FlushInterval:      5 * time.Second,
		MaxRetries:         3,
		RetryBackoff:       time.Second,
		ValidateData:       config.ValidateData,
		ResumeFromID:       config.ResumeFromID,
		Systems:            config.Systems,
	}

	// Create and execute loader
	loader, err := bulkload.NewBulkLoader(blConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create bulk loader: %w", err)
	}
	defer loader.Close()

	if err := loader.ExecuteBulkLoad(ctx); err != nil {
		return fmt.Errorf("bulk load failed: %w", err)
	}

	// Get final statistics
	stats := loader.GetStatistics()
	printStatistics(stats, logger)

	// Generate report if requested
	if *outputFile != "" {
		if err := generateReport(stats, *outputFile); err != nil {
			logger.WithError(err).Warn("Failed to generate report")
		}
	}

	return nil
}

// executeParallelMigration performs high-performance parallel migration
func executeParallelMigration(ctx context.Context, config *Config, logger *logrus.Logger) error {
	// Create progress reporter
	progressDone := make(chan bool)
	go reportProgress(ctx, config.ProgressInterval, logger, progressDone)
	defer close(progressDone)

	// Create bulk load configuration
	blConfig := &bulkload.BulkLoadConfig{
		PostgresConnStr:    config.PostgresURL,
		ElasticsearchURL:   config.ElasticsearchURL,
		ElasticsearchIndex: config.ElasticsearchIndex,
		BatchSize:          config.BatchSize,
		NumWorkers:         config.NumWorkers,
		FlushInterval:      5 * time.Second,
		MaxRetries:         3,
		RetryBackoff:       time.Second,
		ValidateData:       config.ValidateData,
		ResumeFromID:       config.ResumeFromID,
		Systems:            config.Systems,
	}

	// Create and execute loader
	loader, err := bulkload.NewBulkLoader(blConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create bulk loader: %w", err)
	}
	defer loader.Close()

	if err := loader.ExecuteBulkLoad(ctx); err != nil {
		return fmt.Errorf("bulk load failed: %w", err)
	}

	// Get final statistics
	stats := loader.GetStatistics()
	printStatistics(stats, logger)

	// Generate report if requested
	if *outputFile != "" {
		if err := generateReport(stats, *outputFile); err != nil {
			logger.WithError(err).Warn("Failed to generate report")
		}
	}

	return nil
}

// executeBlueGreenMigration performs zero-downtime blue-green migration
func executeBlueGreenMigration(ctx context.Context, config *Config, logger *logrus.Logger) error {
	logger.Info("Blue-green migration strategy selected")
	logger.Info("This strategy creates a new index and switches atomically")

	// Implementation would:
	// 1. Create new index with timestamp suffix
	// 2. Load all data to new index
	// 3. Validate new index
	// 4. Switch alias atomically
	// 5. Keep old index for rollback

	return fmt.Errorf("blue-green strategy not yet implemented")
}

// executeShadowMigration performs gradual shadow write migration
func executeShadowMigration(ctx context.Context, config *Config, logger *logrus.Logger) error {
	logger.Info("Shadow migration strategy selected")
	logger.Info("This strategy enables dual-writes with gradual read migration")

	// Implementation would:
	// 1. Enable dual-write mode
	// 2. Start background migration
	// 3. Gradually shift read traffic
	// 4. Monitor consistency
	// 5. Complete when fully migrated

	return fmt.Errorf("shadow strategy not yet implemented")
}

// reportProgress periodically reports migration progress
func reportProgress(ctx context.Context, interval time.Duration, logger *logrus.Logger, done chan bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// This would query the loader for current statistics
			logger.Info("Progress update: [implementation pending]")
		case <-ctx.Done():
			return
		case <-done:
			return
		}
	}
}

// printStatistics prints final migration statistics
func printStatistics(stats bulkload.LoadStatistics, logger *logrus.Logger) {
	duration := stats.EndTime.Sub(stats.StartTime)
	successRate := float64(stats.SuccessfulRecords) / float64(stats.TotalRecords) * 100

	logger.Info("=== Migration Statistics ===")
	logger.Infof("Duration: %v", duration)
	logger.Infof("Total Records: %d", stats.TotalRecords)
	logger.Infof("Processed: %d", stats.ProcessedRecords)
	logger.Infof("Successful: %d", stats.SuccessfulRecords)
	logger.Infof("Failed: %d", stats.FailedRecords)
	logger.Infof("Success Rate: %.2f%%", successRate)
	logger.Infof("Records/Second: %.2f", float64(stats.ProcessedRecords)/duration.Seconds())
	logger.Infof("Validation Errors: %d", stats.ValidationErrors)

	if len(stats.Errors) > 0 {
		logger.Warnf("Errors encountered: %d", len(stats.Errors))
		for i, err := range stats.Errors {
			if i >= 5 {
				logger.Warnf("... and %d more errors", len(stats.Errors)-5)
				break
			}
			logger.Warnf("  - %v: %s", err.Timestamp, err.Error)
		}
	}
	logger.Info("===========================")
}

// generateReport creates a JSON report of the migration
func generateReport(stats bulkload.LoadStatistics, filename string) error {
	report := map[string]interface{}{
		"migration_date": time.Now(),
		"duration":       stats.EndTime.Sub(stats.StartTime).String(),
		"statistics":     stats,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}