// Package main is the entry point for KB-16 Lab Interpretation & Trending Service
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/internal/api"
	"kb-16-lab-interpretation/internal/config"
	"kb-16-lab-interpretation/internal/database"
)

const (
	serviceName = "kb-16-lab-interpretation"
	version     = "1.0.0"
)

func main() {
	// Initialize logger
	log := initLogger()

	log.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
	}).Info("Starting KB-16 Lab Interpretation & Trending Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	log.WithFields(logrus.Fields{
		"port":        cfg.Server.Port,
		"environment": cfg.Server.Environment,
		"db_host":     cfg.Database.Host,
		"redis_host":  cfg.Redis.Host,
	}).Info("Configuration loaded")

	// Initialize database connections
	db, err := database.New(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.WithError(err).Error("Error closing database connections")
		}
	}()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.WithError(err).Fatal("Failed to run database migrations")
	}

	// Create and start server
	server, err := api.NewServer(cfg, db, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create server")
	}

	// Start server in goroutine
	go func() {
		log.WithField("port", cfg.Server.Port).Info("HTTP server starting...")
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("HTTP server error")
		}
	}()

	// Log successful startup
	log.WithFields(logrus.Fields{
		"port":        cfg.Server.Port,
		"environment": cfg.Server.Environment,
		"version":     version,
		"endpoints": map[string]string{
			"health":  "/health",
			"ready":   "/ready",
			"metrics": cfg.Metrics.Path,
			"api":     "/api/v1",
			"fhir":    "/fhir",
		},
	}).Info("KB-16 Lab Interpretation Service started successfully")

	// Print available endpoints
	printEndpoints(cfg.Server.Port)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.WithField("signal", sig.String()).Info("Shutdown signal received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	log.Info("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
	}

	log.Info("KB-16 Lab Interpretation Service shutdown complete")
}

// initLogger initializes the structured logger
func initLogger() *logrus.Entry {
	logger := logrus.New()

	// Set log format
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "text" {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	logger.SetOutput(os.Stdout)

	// Set log level
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

// printEndpoints prints available API endpoints
func printEndpoints(port int) {
	endpoints := []struct {
		method string
		path   string
		desc   string
	}{
		// Health
		{"GET", "/health", "Health check"},
		{"GET", "/ready", "Readiness check"},
		{"GET", "/metrics", "Prometheus metrics"},

		// Results
		{"POST", "/api/v1/results", "Store a lab result"},
		{"GET", "/api/v1/results/:id", "Get result by ID"},
		{"POST", "/api/v1/results/batch", "Store multiple results"},
		{"GET", "/api/v1/patients/:patientId/results", "Get patient results"},
		{"GET", "/api/v1/patients/:patientId/results/:code", "Get patient results by test code"},

		// Interpretation
		{"POST", "/api/v1/interpret", "Interpret a lab result"},
		{"POST", "/api/v1/interpret/batch", "Interpret multiple results"},

		// Trending
		{"GET", "/api/v1/trending/:patientId/:code", "Get trend for a test"},
		{"GET", "/api/v1/trending/:patientId/:code/multi", "Get multi-window trend"},
		{"GET", "/api/v1/trending/:patientId", "Get all trends for patient"},

		// Baselines
		{"GET", "/api/v1/baselines/:patientId", "Get patient baselines"},
		{"GET", "/api/v1/baselines/:patientId/:code", "Get baseline for test"},
		{"POST", "/api/v1/baselines/:patientId/:code", "Set manual baseline"},
		{"POST", "/api/v1/baselines/:patientId/:code/calculate", "Calculate baseline"},

		// Panels
		{"GET", "/api/v1/panels", "List panel definitions"},
		{"GET", "/api/v1/panels/:type", "Get panel definition"},
		{"POST", "/api/v1/panels/:patientId/assemble/:type", "Assemble panel from results"},
		{"GET", "/api/v1/panels/:patientId/detect", "Detect available panels"},

		// Review
		{"GET", "/api/v1/review/pending", "Get pending reviews"},
		{"GET", "/api/v1/review/critical", "Get critical value queue"},
		{"POST", "/api/v1/review/acknowledge", "Acknowledge result"},
		{"POST", "/api/v1/review/complete", "Complete review"},
		{"GET", "/api/v1/review/stats", "Review statistics"},

		// Visualization
		{"GET", "/api/v1/charts/:patientId/:code", "Get chart data"},
		{"GET", "/api/v1/sparklines/:patientId/:code", "Get sparkline data"},
		{"GET", "/api/v1/dashboard/:patientId", "Get dashboard data"},

		// Reference
		{"GET", "/api/v1/reference/tests", "List test definitions"},
		{"GET", "/api/v1/reference/tests/:code", "Get test definition"},

		// FHIR
		{"GET", "/fhir/Observation", "Search FHIR Observations"},
		{"GET", "/fhir/Observation/:id", "Get FHIR Observation"},
		{"GET", "/fhir/DiagnosticReport/:patientId/:panelType", "Get FHIR DiagnosticReport"},
	}

	logrus.Info("Available endpoints:")
	for _, e := range endpoints {
		logrus.WithFields(logrus.Fields{
			"method": e.method,
			"path":   e.path,
		}).Debug(e.desc)
	}
}
