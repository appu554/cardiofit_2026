// Package main provides the entry point for KB-8 Calculator Service.
//
// KB-8 is a high-performance clinical calculator microservice implementing
// the ATOMIC pattern - pure mathematical calculations with pre-fetched
// parameters, targeting ~5ms latency.
//
// Supported calculators:
//   - eGFR (CKD-EPI 2021, race-free)
//   - CrCl (Cockcroft-Gault)
//   - BMI (with Asian/India cutoffs)
//   - SOFA (ICU mortality)
//   - qSOFA (sepsis screening)
//   - CHA₂DS₂-VASc (stroke risk)
//   - HAS-BLED (bleeding risk)
//   - ASCVD 10-Year (cardiovascular risk)
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

	"kb-8-calculator-service/internal/api"
	"kb-8-calculator-service/internal/calculator"
	"kb-8-calculator-service/internal/config"
)

const (
	serviceName    = "kb-8-calculator-service"
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

	logger.Info("Starting KB-8 Calculator Service",
		zap.String("service", serviceName),
		zap.String("version", serviceVersion),
		zap.String("environment", cfg.Environment),
		zap.Int("port", cfg.Port),
	)

	// Initialize calculator service
	calcService := calculator.NewService(logger)
	availableCalcs := calcService.GetAvailableCalculators()
	calcNames := make([]string, len(availableCalcs))
	for i, c := range availableCalcs {
		calcNames[i] = string(c.Type)
	}
	logger.Info("Calculator service initialized",
		zap.Strings("available_calculators", calcNames),
	)

	// Initialize API server (handles Gin mode internally)
	apiServer := api.NewServer(cfg, calcService, logger)

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

	if cfg.Environment == "production" {
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
╔═══════════════════════════════════════════════════════════════╗
║           KB-8 CALCULATOR SERVICE                             ║
║           Clinical Score Calculations                         ║
╠═══════════════════════════════════════════════════════════════╣
║  Version:     %s                                          ║
║  Environment: %-10s                                       ║
║  Port:        %-5d                                            ║
╠═══════════════════════════════════════════════════════════════╣
║  Endpoints:                                                   ║
║    POST /api/v1/calculate/egfr     - eGFR (CKD-EPI 2021)     ║
║    POST /api/v1/calculate/crcl     - CrCl (Cockcroft-Gault)  ║
║    POST /api/v1/calculate/bmi      - BMI (Asian cutoffs)     ║
║    POST /api/v1/calculate/sofa     - SOFA Score              ║
║    POST /api/v1/calculate/qsofa    - qSOFA Score             ║
║    POST /api/v1/calculate/cha2ds2vasc - Stroke Risk          ║
║    POST /api/v1/calculate/hasbled  - Bleeding Risk           ║
║    POST /api/v1/calculate/ascvd    - ASCVD 10-Year           ║
║    POST /api/v1/calculate/batch    - Batch Calculation       ║
║    GET  /health                    - Health Check            ║
║    GET  /metrics                   - Prometheus Metrics      ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Printf(banner, serviceVersion, cfg.Environment, cfg.Port)

	logger.Info("Service ready",
		zap.String("health_endpoint", fmt.Sprintf("http://localhost:%d/health", cfg.Port)),
		zap.String("metrics_endpoint", fmt.Sprintf("http://localhost:%d/metrics", cfg.Port)),
	)
}
