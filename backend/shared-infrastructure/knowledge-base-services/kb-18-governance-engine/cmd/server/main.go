// Package main provides the entry point for KB-18 Governance Engine
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-18-governance-engine/internal/api"
	"kb-18-governance-engine/internal/config"
)

const (
	serviceName    = "kb-18-governance-engine"
	serviceVersion = "1.0.0"
)

func main() {
	// Initialize logger
	log := setupLogger()
	log.WithFields(logrus.Fields{
		"service": serviceName,
		"version": serviceVersion,
	}).Info("Starting KB-18 Governance Engine")

	// Load configuration
	cfg := config.Load()
	log.WithFields(logrus.Fields{
		"port":        cfg.Server.Port,
		"environment": cfg.Server.Environment,
	}).Info("Configuration loaded")

	// Create and start server
	server, err := api.NewServer(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to create server")
	}

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.WithError(err).Fatal("Server failed")
		}
	}()

	log.WithField("port", cfg.Server.Port).Info("KB-18 Governance Engine is running")

	// Print banner
	printBanner(log)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Received shutdown signal")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("KB-18 Governance Engine stopped")
}

// setupLogger initializes the logger with appropriate settings
func setupLogger() *logrus.Entry {
	logger := logrus.New()

	// Set log format
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set log level from environment
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	return logger.WithField("service", serviceName)
}

// printBanner prints the service banner
func printBanner(log *logrus.Entry) {
	banner := `
╔═══════════════════════════════════════════════════════════════════════════╗
║                     KB-18 GOVERNANCE ENGINE                               ║
║                 Clinical Governance Enforcement Platform                  ║
╠═══════════════════════════════════════════════════════════════════════════╣
║  Version: %s                                                           ║
║                                                                           ║
║  Features:                                                                ║
║  • Deterministic clinical governance decisions                            ║
║  • Reproducible rule evaluation (same input → same output)                ║
║  • Immutable evidence trails with SHA-256 hashing                         ║
║  • Pre-configured programs: Maternal Safety, Opioid Stewardship,         ║
║    Anticoagulation Management                                             ║
║  • 6 enforcement levels from IGNORE to MANDATORY_ESCALATION               ║
║                                                                           ║
║  Endpoints:                                                               ║
║  • POST /api/v1/evaluate           - General governance evaluation        ║
║  • POST /api/v1/evaluate/medication - Medication order evaluation         ║
║  • GET  /api/v1/programs           - List governance programs             ║
║  • POST /api/v1/overrides/request  - Request override                     ║
║  • GET  /api/v1/stats              - Engine statistics                    ║
║  • GET  /health                    - Health check                         ║
╚═══════════════════════════════════════════════════════════════════════════╝
`
	fmt.Printf(banner, serviceVersion)
	log.Info("Service banner displayed")
}
