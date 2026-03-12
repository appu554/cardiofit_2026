// Package main provides the entry point for KB-9 Care Gaps Service.
//
// KB-9 is a Care Gaps Detection and Quality Measure Evaluation Service
// that uses the Query-Based CQL pattern to evaluate clinical quality
// measures against FHIR patient data.
//
// Supported Measures:
//   - CMS122: Diabetes HbA1c Poor Control (>9%)
//   - CMS165: Controlling High Blood Pressure
//   - CMS130: Colorectal Cancer Screening
//   - CMS2: Depression Screening
//   - India Diabetes Care (Custom)
//   - India Hypertension Care (Custom)
//
// Integration:
//   - Uses vaidshala CQL infrastructure for measure evaluation
//   - Queries FHIR server for patient data
//   - Integrates with KB-7 for terminology services
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"kb-9-care-gaps/internal/api"
	"kb-9-care-gaps/internal/cache"
	"kb-9-care-gaps/internal/caregaps"
	"kb-9-care-gaps/internal/config"
)

const (
	serviceName    = "kb-9-care-gaps"
	serviceVersion = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting KB-9 Care Gaps Service",
		zap.String("service", serviceName),
		zap.String("version", serviceVersion),
		zap.String("environment", cfg.Environment),
		zap.Int("port", cfg.Port),
	)

	// Initialize Redis cache for CQL libraries and measure definitions
	cacheConfig := cache.Config{
		RedisURL: cfg.RedisURL,
		TTL:      cfg.CacheTTL,
		Enabled:  cfg.RedisURL != "",
		Prefix:   "kb9:",
	}
	redisCache, err := cache.NewCache(cacheConfig, logger)
	if err != nil {
		logger.Warn("Failed to initialize Redis cache, continuing without caching",
			zap.Error(err),
		)
	} else {
		defer redisCache.Close()
		logger.Info("Redis cache initialized",
			zap.Bool("enabled", cacheConfig.Enabled),
			zap.Duration("ttl", cacheConfig.TTL),
		)
	}

	// Initialize care gaps service
	careGapsService := caregaps.NewService(cfg, logger)
	logger.Info("Care Gaps service initialized",
		zap.String("fhir_server", cfg.FHIRServerURL),
		zap.String("terminology_url", cfg.TerminologyURL),
	)

	// Initialize API server with cache
	apiServer := api.NewServer(cfg, careGapsService, redisCache, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      apiServer.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server starting",
			zap.String("address", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// Print startup banner
	printBanner(cfg, logger)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// initLogger creates a structured logger based on environment.
func initLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.IsProduction() {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level from config
	switch cfg.LogLevel {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return zapCfg.Build()
}

// printBanner prints service startup information.
func printBanner(cfg *config.Config, logger *zap.Logger) {
	banner := `
╔═══════════════════════════════════════════════════════════════════╗
║              KB-9 CARE GAPS SERVICE                               ║
║         Quality Measure Evaluation & Care Gap Detection           ║
╠═══════════════════════════════════════════════════════════════════╣
║  Version:     %s                                              ║
║  Environment: %-10s                                           ║
║  Port:        %-5d                                                ║
╠═══════════════════════════════════════════════════════════════════╣
║  REST Endpoints:                                                  ║
║    POST /api/v1/care-gaps              - Get patient care gaps    ║
║    POST /api/v1/measure/evaluate       - Evaluate single measure  ║
║    POST /api/v1/measure/evaluate-population - Population eval     ║
║    GET  /api/v1/measures               - List available measures  ║
║    GET  /api/v1/measures/{type}        - Get measure details      ║
╠═══════════════════════════════════════════════════════════════════╣
║  FHIR Operations:                                                 ║
║    POST /fhir/Measure/$care-gaps       - Da Vinci DEQM            ║
║    POST /fhir/Measure/{id}/$evaluate-measure                      ║
╠═══════════════════════════════════════════════════════════════════╣
║  Health & Metrics:                                                ║
║    GET  /health                        - Health Check             ║
║    GET  /ready                         - Readiness Probe          ║
║    GET  /live                          - Liveness Probe           ║
║    GET  /metrics                       - Prometheus Metrics       ║
╚═══════════════════════════════════════════════════════════════════╝
`
	fmt.Printf(banner, serviceVersion, cfg.Environment, cfg.Port)

	logger.Info("Service ready",
		zap.String("health_endpoint", fmt.Sprintf("http://localhost:%d/health", cfg.Port)),
		zap.String("metrics_endpoint", fmt.Sprintf("http://localhost:%d/metrics", cfg.Port)),
		zap.String("care_gaps_endpoint", fmt.Sprintf("http://localhost:%d/api/v1/care-gaps", cfg.Port)),
	)
}
