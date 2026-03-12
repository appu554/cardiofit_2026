// Package main is the entry point for KB-13 Quality Measures Engine.
//
// KB-13 provides enterprise-wide quality measure calculation and reporting:
//   - HEDIS, CMS, NQF, MIPS, ACO measure definitions
//   - Population-level batch calculations (not per-patient)
//   - Quality dashboards and performance trends
//   - Derived care gaps (source: QUALITY_MEASURE)
//   - Benchmark comparisons with versioned data
//   - FHIR R4 MeasureReport compliance
//
// Port: 8113 (default)
//
// Critical Architecture Notes (CTO/CMO Gate Approved):
//
//	🔴 Batch CQL Evaluation ONLY - No per-patient CQL calls
//	🔴 All date logic through internal/period module
//	🔴 Care gaps marked as DERIVED, not authoritative
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"kb-13-quality-measures/internal/api"
	"kb-13-quality-measures/internal/config"
	"kb-13-quality-measures/internal/cql"
	"kb-13-quality-measures/internal/database"
)

func main() {
	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("Starting KB-13 Quality Measures Engine",
		zap.String("service", config.ServiceName),
		zap.String("version", config.Version),
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("Configuration loaded",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
		zap.String("measures_path", cfg.Server.MeasuresPath),
		zap.String("benchmarks_path", cfg.Server.BenchmarksPath),
	)

	// Initialize dependencies
	deps := initDependencies(cfg, logger)

	// Create HTTP server with dependencies
	server, err := api.NewServer(cfg, logger, deps)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Log available endpoints
	logEndpoints(logger, cfg, deps)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KB-13 Quality Measures Engine...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close database connection
	if deps != nil && deps.DB != nil {
		if err := deps.DB.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}

	logger.Info("KB-13 Quality Measures Engine shutdown complete")
}

// initDependencies initializes database and external service connections.
func initDependencies(cfg *config.Config, logger *zap.Logger) *api.ServerDependencies {
	deps := &api.ServerDependencies{
		// Pass KB integration URLs from config
		KB7URL:  cfg.Integrations.KB7URL,
		KB18URL: cfg.Integrations.KB18URL,
		KB19URL: cfg.Integrations.KB19URL,
	}

	// Connect to PostgreSQL
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		logger.Warn("Failed to connect to database - Phase 2/3 features disabled",
			zap.Error(err),
		)
	} else {
		deps.DB = db.DB // Extract the embedded *sql.DB for repositories
		logger.Info("Database connection established",
			zap.String("host", cfg.Database.Host),
			zap.Int("port", cfg.Database.Port),
			zap.String("database", cfg.Database.Name),
		)
	}

	// Create CQL client for Vaidshala integration
	cqlClient := cql.NewClient(cql.ClientConfig{
		BaseURL: cfg.Integrations.VaidshalaURL,
		Timeout: cfg.Calculator.Timeout,
	}, logger)

	// Test CQL client connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cqlClient.Health(ctx); err != nil {
		logger.Warn("CQL engine not available - calculation features disabled",
			zap.String("url", cfg.Integrations.VaidshalaURL),
			zap.Error(err),
		)
	} else {
		deps.CQLClient = cqlClient
		logger.Info("CQL engine connected",
			zap.String("url", cfg.Integrations.VaidshalaURL),
		)
	}

	// Log KB integration URLs
	logger.Info("KB integration URLs configured",
		zap.String("kb7_url", cfg.Integrations.KB7URL),
		zap.String("kb18_url", cfg.Integrations.KB18URL),
		zap.String("kb19_url", cfg.Integrations.KB19URL),
	)

	return deps
}

// initLogger creates a production-ready zap logger.
func initLogger() *zap.Logger {
	// Get log level from environment
	level := zapcore.InfoLevel
	if levelStr := os.Getenv("KB13_LOG_LEVEL"); levelStr != "" {
		switch levelStr {
		case "debug":
			level = zapcore.DebugLevel
		case "warn":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		}
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Use JSON format in production, console in development
	var encoder zapcore.Encoder
	if os.Getenv("KB13_ENVIRONMENT") == "production" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

// logEndpoints logs all available API endpoints.
func logEndpoints(logger *zap.Logger, cfg *config.Config, deps *api.ServerDependencies) {
	logger.Info("KB-13 Quality Measures Engine started successfully",
		zap.Int("port", cfg.Server.Port),
		zap.String("environment", cfg.Server.Environment),
		zap.String("version", config.Version),
	)

	// Log feature availability
	dbConnected := deps != nil && deps.DB != nil
	cqlConnected := deps != nil && deps.CQLClient != nil

	logger.Info("Feature Status:",
		zap.Bool("database_connected", dbConnected),
		zap.Bool("cql_engine_connected", cqlConnected),
	)

	logger.Info("Available endpoints:")
	logger.Info("  Health:")
	logger.Info("    GET  /health                          - Liveness probe")
	logger.Info("    GET  /ready                           - Readiness probe")
	if cfg.Metrics.Enabled {
		logger.Info("    GET  " + cfg.Metrics.Path + "                        - Prometheus metrics")
	}

	logger.Info("  Measure Definitions:")
	logger.Info("    GET  /v1/measures                     - List all measures")
	logger.Info("    GET  /v1/measures/:id                 - Get measure by ID")
	logger.Info("    GET  /v1/measures/search              - Search measures")
	logger.Info("    GET  /v1/measures/by-program/:program - Filter by program")
	logger.Info("    GET  /v1/measures/by-domain/:domain   - Filter by domain")

	logger.Info("  Benchmarks:")
	logger.Info("    GET  /v1/benchmarks/:measureId        - Get benchmarks for measure")
	logger.Info("    GET  /v1/benchmarks/:measureId/:year  - Get benchmark by year")

	if cqlConnected {
		logger.Info("  Calculations (✅ Available):")
		logger.Info("    POST /v1/calculations/measure/:id       - Calculate single measure")
		logger.Info("    POST /v1/calculations/measure/:id/async - Start async calculation")
		logger.Info("    GET  /v1/calculations/jobs/:jobId       - Get job status")
		logger.Info("    POST /v1/calculations/batch             - Batch calculate measures")
	} else {
		logger.Info("  Calculations (❌ CQL engine required):")
	}

	if dbConnected {
		logger.Info("  Care Gaps (✅ Available):")
		logger.Info("    GET  /v1/care-gaps                      - List care gaps")
		logger.Info("    GET  /v1/care-gaps/:id                  - Get care gap")
		logger.Info("    GET  /v1/care-gaps/by-measure/:id       - Gaps by measure")
		logger.Info("    GET  /v1/care-gaps/by-patient/:id       - Gaps by patient")
		logger.Info("    GET  /v1/care-gaps/summary/:id          - Gap summary")

		logger.Info("  Dashboard (✅ Available):")
		logger.Info("    GET  /v1/dashboard/overview             - Dashboard overview")
		logger.Info("    GET  /v1/dashboard/measures             - Measure performance")
		logger.Info("    GET  /v1/dashboard/programs             - Program summaries")
		logger.Info("    GET  /v1/dashboard/domains              - Domain summaries")
		logger.Info("    GET  /v1/dashboard/trends/:measureId    - Trend data")
		logger.Info("    GET  /v1/dashboard/care-gaps            - Care gap dashboard")
	} else {
		logger.Info("  Care Gaps (❌ Database required):")
		logger.Info("  Dashboard (❌ Database required):")
	}

	logger.Info("")
	logger.Info("Architecture Notes (CTO/CMO Gate):")
	logger.Info("  🔴 Batch CQL Evaluation ONLY - No per-patient CQL calls")
	logger.Info("  🔴 All date logic through internal/period module")
	logger.Info("  🔴 Care gaps marked as DERIVED (Source: QUALITY_MEASURE)")
	logger.Info("  🟡 Versioned benchmarks stored separately by year")
	logger.Info("  🟡 ExecutionContextVersion required for all calculations")
}
