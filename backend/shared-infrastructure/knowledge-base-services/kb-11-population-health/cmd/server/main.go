// Package main is the entry point for KB-11 Population Health Engine.
//
// KB-11 Architecture Overview:
// ═══════════════════════════════════════════════════════════════════════════
//
// North Star: "KB-11 answers population-level questions, NOT patient-level decisions."
//
// KB-11 is a "Population Intelligence Layer" that:
// 1. CONSUMES patient data from FHIR Store and KB-17 Registry (READ-ONLY)
// 2. CALCULATES risk scores with governance via KB-18
// 3. PROVIDES population-level analytics and insights
//
// Data Ownership:
// - Patient demographics: CONSUMED from upstream (NOT authoritative)
// - Risk assessments: OWNED by KB-11, GOVERNED by KB-18
// - Attribution data: OWNED by KB-11 (PCP assignments)
// - Care gaps: CONSUMED from KB-13 (aggregated counts only)
//
// ═══════════════════════════════════════════════════════════════════════════
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cardiofit/kb-11-population-health/internal/analytics"
	"github.com/cardiofit/kb-11-population-health/internal/api"
	"github.com/cardiofit/kb-11-population-health/internal/clients"
	"github.com/cardiofit/kb-11-population-health/internal/cohort"
	"github.com/cardiofit/kb-11-population-health/internal/config"
	"github.com/cardiofit/kb-11-population-health/internal/database"
	"github.com/cardiofit/kb-11-population-health/internal/projection"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	logger := cfg.Logging.InitLogger()
	logger.Info("Starting KB-11 Population Health Engine")
	logger.WithField("north_star", "KB-11 answers population-level questions, NOT patient-level decisions").Info("Architecture principle")

	// Connect to database
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Create projection repository
	repo := database.NewProjectionRepository(db, logger)

	// Create external clients (READ-ONLY)
	var fhirClient *clients.FHIRClient
	if cfg.External.FHIRStoreURL != "" {
		fhirClient = clients.NewFHIRClient(cfg.External.FHIRStoreURL, logger)
		logger.WithField("url", cfg.External.FHIRStoreURL).Info("FHIR client initialized (READ-ONLY)")
	}

	var kb17Client *clients.KB17Client
	if cfg.External.KB17URL != "" {
		kb17Client = clients.NewKB17Client(cfg.External.KB17URL, logger)
		logger.WithField("url", cfg.External.KB17URL).Info("KB-17 client initialized (READ-ONLY)")
	}

	// Create KB-18 governance client (for risk calculation governance)
	var kb18Client *clients.KB18Client
	if cfg.External.KB18URL != "" {
		kb18Client = clients.NewKB18Client(cfg.External.KB18URL, logger)
		logger.WithField("url", cfg.External.KB18URL).Info("KB-18 governance client initialized")
	}

	// Create KB-13 care gap client (for care gap aggregation - READ-ONLY)
	var kb13Client *clients.KB13Client
	if cfg.External.KB13URL != "" {
		kb13Client = clients.NewKB13Client(cfg.External.KB13URL, logger)
		logger.WithField("url", cfg.External.KB13URL).Info("KB-13 care gap client initialized (READ-ONLY)")
	}

	// Create projection service
	projService := projection.NewService(repo, fhirClient, kb17Client, cfg, logger)

	// Create risk engine (GOVERNED by KB-18)
	riskEngine := risk.NewEngine(repo, kb18Client, cfg.Risk.MaxConcurrent, logger)
	logger.Info("Risk engine initialized with KB-18 governance")

	// Create cohort service (OWNED by KB-11)
	cohortRepo := cohort.NewRepository(db.SQLX(), logger)
	cohortService := cohort.NewService(cohortRepo, cfg, logger)
	logger.Info("Cohort service initialized")

	// Create analytics engine (CORE PURPOSE of KB-11)
	analyticsEngine := analytics.NewEngine(repo, kb13Client, logger)
	logger.Info("Analytics engine initialized")

	// Create and start API server
	server := api.NewServer(cfg, db, projService, riskEngine, cohortService, analyticsEngine, logger)

	// Graceful shutdown handling
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Server failed to start")
		}
	}()

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

	logger.Info("KB-11 Population Health Engine stopped")
}
