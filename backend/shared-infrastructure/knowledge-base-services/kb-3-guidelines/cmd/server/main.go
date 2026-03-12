// Package main provides the entry point for KB-3 Guidelines Service
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-3-guidelines/pkg/api"
	"github.com/cardiofit/kb-3-guidelines/pkg/config"
	"github.com/cardiofit/kb-3-guidelines/pkg/database"
	"github.com/cardiofit/kb-3-guidelines/pkg/graph"
	"github.com/cardiofit/kb-3-guidelines/pkg/protocols"
	"github.com/cardiofit/kb-3-guidelines/pkg/temporal"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("Failed to validate configuration")
	}

	// Set log level
	switch cfg.LogLevel {
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

	logger.WithFields(logrus.Fields{
		"service":     cfg.ServiceName,
		"environment": cfg.Environment,
		"port":        cfg.Port,
	}).Info("Starting KB-3 Guidelines Service")

	// Initialize services
	var dbService *database.PostgresService
	var neo4jService *graph.Neo4jService

	// Connect to PostgreSQL if configured
	if cfg.DatabaseURL != "" {
		var dbErr error
		dbService, dbErr = database.NewPostgresService(cfg.DatabaseURL)
		if dbErr != nil {
			logger.WithError(dbErr).Warn("Failed to connect to PostgreSQL - continuing without database")
		} else {
			logger.Info("Connected to PostgreSQL")
			defer dbService.Close()
		}
	}

	// Connect to Neo4j if configured
	if cfg.Neo4jURL != "" {
		var neo4jErr error
		neo4jService, neo4jErr = graph.NewNeo4jService(cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPassword)
		if neo4jErr != nil {
			logger.WithError(neo4jErr).Warn("Failed to connect to Neo4j - continuing without graph database")
		} else {
			logger.Info("Connected to Neo4j")
			defer neo4jService.Close(context.Background())
		}
	}

	// Initialize protocol registry
	registry := protocols.GetRegistry()
	summary := registry.GetProtocolSummary()
	logger.WithFields(logrus.Fields{
		"acute_protocols":      summary.AcuteProtocolCount,
		"chronic_schedules":    summary.ChronicScheduleCount,
		"preventive_schedules": summary.PreventiveScheduleCount,
	}).Info("Protocol registry initialized")

	// Initialize temporal engines
	pathwayEngine := temporal.GetPathwayEngine()
	schedulingEngine := temporal.GetSchedulingEngine()
	_ = pathwayEngine     // Used by API handlers via singleton
	_ = schedulingEngine  // Used by API handlers via singleton

	logger.Info("Temporal engines initialized")

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.New()

	// Add middleware
	middleware := api.NewMiddleware()
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	// Initialize API handler
	handler := api.NewHandler()

	// Register routes
	api.RegisterRoutes(router, handler)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.WithField("port", cfg.Port).Info("HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Print startup banner
	printBanner(cfg, summary)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server stopped")
}

func printBanner(cfg *config.Config, summary protocols.ProtocolSummary) {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║    KB-3 Guidelines & Temporal Logic Service                   ║
║    ─────────────────────────────────────────                  ║
║                                                               ║
║    Port:        %d                                          ║
║    Environment: %-10s                                     ║
║                                                               ║
║    Protocols Loaded:                                          ║
║      • Acute:      %d (Sepsis, Stroke, STEMI, DKA, Trauma, PE) ║
║      • Chronic:    %d (Diabetes, HF, CKD, Anticoag, COPD, HTN) ║
║      • Preventive: %d (Prenatal, WellChild, Adult, Cancer)    ║
║                                                               ║
║    Features:                                                  ║
║      ✓ Allen's Interval Algebra (13 operators)               ║
║      ✓ Clinical Pathway State Machine                        ║
║      ✓ Time-Bound Constraint Tracking                        ║
║      ✓ Recurrence Pattern Scheduling                         ║
║      ✓ Overdue Alert Management                              ║
║                                                               ║
║    API Docs: http://localhost:%d/health                      ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Printf(banner,
		cfg.Port,
		cfg.Environment,
		summary.AcuteProtocolCount,
		summary.ChronicScheduleCount,
		summary.PreventiveScheduleCount,
		cfg.Port,
	)
}
