// Package main is the entry point for KB-1 Drug Dosing Rules Service.
//
// KB-1 provides comprehensive drug dosing calculations:
//   - General dose calculation with auto-adjustments
//   - Weight-based dosing (mg/kg)
//   - BSA-based dosing (mg/m²)
//   - Pediatric dosing with age categories
//   - Renal dose adjustments (eGFR/CrCl-based)
//   - Hepatic dose adjustments (Child-Pugh based)
//   - Geriatric dosing with Beers Criteria
//   - Dose validation against max limits
//   - Patient parameter calculations (BSA, IBW, CrCl, eGFR)
//   - 24 built-in drug rules across 5 categories
//
// Port: 8081 (default)
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-1-drug-rules/internal/api"
	"kb-1-drug-rules/internal/config"
)

const (
	serviceName = "kb-1-drug-rules"
	version     = "1.0.0"
)

func main() {
	// Initialize logger
	log := initLogger()
	log.Info("Starting KB-1 Drug Dosing Rules Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}
	log.WithField("environment", cfg.Server.Environment).Info("Configuration loaded")

	// Create and initialize the API server
	server, err := api.NewServer(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to create server")
	}

	// Start server in goroutine
	go func() {
		log.WithField("port", cfg.Server.Port).Info("HTTP server starting...")
		if err := server.Start(); err != nil {
			log.WithError(err).Fatal("HTTP server error")
		}
	}()

	log.WithFields(logrus.Fields{
		"port":        cfg.Server.Port,
		"environment": cfg.Server.Environment,
		"version":     version,
		"drugs":       24,
		"categories":  5,
	}).Info("KB-1 Drug Dosing Rules Service started successfully")

	// Log available endpoints
	log.Info("Available endpoints:")
	log.Info("  Health:")
	log.Info("    GET  /health")
	log.Info("    GET  /ready")
	log.Info("  Dose Calculation:")
	log.Info("    POST /v1/calculate")
	log.Info("    POST /v1/calculate/weight-based")
	log.Info("    POST /v1/calculate/bsa-based")
	log.Info("    POST /v1/calculate/pediatric")
	log.Info("    POST /v1/calculate/renal")
	log.Info("    POST /v1/calculate/hepatic")
	log.Info("    POST /v1/calculate/geriatric")
	log.Info("  Patient Parameters:")
	log.Info("    POST /v1/patient/bsa")
	log.Info("    POST /v1/patient/ibw")
	log.Info("    POST /v1/patient/crcl")
	log.Info("    POST /v1/patient/egfr")
	log.Info("  Dose Validation:")
	log.Info("    POST /v1/validate/dose")
	log.Info("    GET  /v1/validate/max-dose")
	log.Info("  Dosing Rules:")
	log.Info("    GET  /v1/rules")
	log.Info("    GET  /v1/rules/search")
	log.Info("    GET  /v1/rules/:rxnorm")
	log.Info("  Adjustments:")
	log.Info("    GET  /v1/adjustments/renal")
	log.Info("    GET  /v1/adjustments/hepatic")
	log.Info("    GET  /v1/adjustments/age")
	log.Info("  High-Alert:")
	log.Info("    GET  /v1/high-alert/check")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
	}

	log.Info("KB-1 Drug Dosing Rules Service shutdown complete")
}

// initLogger initializes the logrus logger with JSON formatting.
func initLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	logger.SetOutput(os.Stdout)

	// Set log level from environment
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
	})
}
