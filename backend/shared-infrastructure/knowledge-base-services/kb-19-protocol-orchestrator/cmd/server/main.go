// Package main is the entry point for KB-19 Protocol Orchestrator Service.
//
// KB-19 is the Clinical Protocol Orchestrator - the "decision synthesis brain" that:
//   - Consumes clinical truth from Vaidshala CQL Engine
//   - Orchestrates KB-3 (temporal), KB-8 (calculators), KB-12 (ordersets), KB-14 (governance)
//   - Performs arbitration when multiple protocols conflict
//   - Produces evidence-backed recommendations with ACC/AHA Class grading
//   - Generates audit trails for regulatory compliance
//
// Key Design Principles:
//   1. KB-19 is STATELESS - No protocol logic lives here
//   2. KB-19 DELEGATES truth - CQL Engine owns clinical truth
//   3. KB-19 OWNS synthesis - Arbitration is the unique value
//   4. KB-19 EXPLAINS itself - Every decision has narrative + evidence
//   5. KB-19 BINDS execution - But never executes directly
//
// Port: 8099 (default)
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/api"
	"kb-19-protocol-orchestrator/internal/config"
)

const (
	serviceName = "kb-19-protocol-orchestrator"
	version     = "1.0.0"
)

func main() {
	// Initialize logger
	log := initLogger()
	log.Info("Starting KB-19 Protocol Orchestrator Service...")
	log.Info("The Decision Synthesis Brain - Where Protocols Meet Reality")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}
	log.WithFields(logrus.Fields{
		"environment": cfg.Server.Environment,
		"port":        cfg.Server.Port,
	}).Info("Configuration loaded")

	// Log KB service connections
	log.WithFields(logrus.Fields{
		"vaidshala_cql":   cfg.Vaidshala.CQLEngineURL,
		"kb3_temporal":    cfg.KBServices.KB3URL,
		"kb8_calculator":  cfg.KBServices.KB8URL,
		"kb12_orderset":   cfg.KBServices.KB12URL,
		"kb14_governance": cfg.KBServices.KB14URL,
	}).Info("KB service endpoints configured")

	// Create and initialize the API server
	server, err := api.NewServer(cfg, log)
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
	}).Info("KB-19 Protocol Orchestrator Service started successfully")

	// Log available endpoints
	printEndpoints(log)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down KB-19 Protocol Orchestrator...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
	}

	log.Info("KB-19 Protocol Orchestrator shutdown complete")
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

// printEndpoints logs all available API endpoints.
func printEndpoints(log *logrus.Entry) {
	log.Info("Available endpoints:")
	log.Info("")
	log.Info("  Health & Status:")
	log.Info("    GET  /health              - Health check")
	log.Info("    GET  /ready               - Readiness check")
	log.Info("    GET  /metrics             - Prometheus metrics")
	log.Info("")
	log.Info("  Protocol Orchestration:")
	log.Info("    POST /api/v1/execute      - Execute full protocol arbitration")
	log.Info("    POST /api/v1/evaluate     - Evaluate single protocol")
	log.Info("")
	log.Info("  Protocol Management:")
	log.Info("    GET  /api/v1/protocols    - List available protocols")
	log.Info("    GET  /api/v1/protocols/:id - Get protocol details")
	log.Info("")
	log.Info("  Decision History:")
	log.Info("    GET  /api/v1/decisions/:patientId - Get decisions for patient")
	log.Info("    GET  /api/v1/bundle/:id   - Get recommendation bundle by ID")
	log.Info("")
	log.Info("  Conflict Matrix:")
	log.Info("    GET  /api/v1/conflicts    - List known conflict rules")
	log.Info("    GET  /api/v1/conflicts/:protocolId - Get conflicts for protocol")
	log.Info("")
}
