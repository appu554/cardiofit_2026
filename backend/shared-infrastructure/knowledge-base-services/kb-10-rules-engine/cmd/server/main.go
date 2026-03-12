// Package main is the entry point for the KB-10 Clinical Rules Engine
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/api"
	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/database"
	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/loader"
	"github.com/cardiofit/kb-10-rules-engine/internal/metrics"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
)

const (
	serviceName    = "kb-10-rules-engine"
	serviceVersion = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := config.SetupLogger(&cfg.Logging)
	logger.WithFields(logrus.Fields{
		"service": serviceName,
		"version": serviceVersion,
		"port":    cfg.Server.Port,
	}).Info("Starting Clinical Rules Engine")

	// Initialize metrics
	metricsCollector := metrics.NewCollector(&cfg.Metrics)
	metricsCollector.Register()

	// Initialize database connection
	db, err := database.NewPostgresDB(&cfg.Database, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		logger.WithError(err).Warn("Failed to run migrations (may already exist)")
	}

	// Initialize rule store
	store := models.NewRuleStore()

	// Initialize YAML loader
	yamlLoader := loader.NewYAMLLoader(cfg.Rules.Path, store, logger)

	// Load rules from disk
	startLoad := time.Now()
	if err := yamlLoader.LoadRules(); err != nil {
		logger.WithError(err).Fatal("Failed to load rules")
	}
	loadDuration := time.Since(startLoad)
	store.SetReloadMetadata(time.Now(), loadDuration)

	logger.WithFields(logrus.Fields{
		"rules_loaded":     store.Count(),
		"load_duration_ms": loadDuration.Milliseconds(),
	}).Info("Rules loaded successfully")

	// Initialize evaluation cache
	cache := engine.NewCache(cfg.Rules.EnableCaching, cfg.Rules.CacheTTL, logger)

	// Initialize rules engine
	rulesEngine := engine.NewRulesEngine(store, db, cache, &cfg.Vaidshala, logger, metricsCollector)

	// Initialize API server
	server := api.NewServer(cfg, rulesEngine, store, db, yamlLoader, logger, metricsCollector)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      server.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Setup SIGHUP handler for hot reload
	setupHotReload(yamlLoader, store, logger)

	// Start server in goroutine
	go func() {
		logger.WithFields(logrus.Fields{
			"address": httpServer.Addr,
		}).Info("HTTP server starting")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server error")
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	// Close cache
	cache.Close()

	logger.Info("Server exited gracefully")
}

// setupHotReload sets up SIGHUP signal handler for hot-reloading rules
func setupHotReload(yamlLoader *loader.YAMLLoader, store *models.RuleStore, logger *logrus.Logger) {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	go func() {
		for range sighup {
			logger.Info("Received SIGHUP, reloading rules...")

			startLoad := time.Now()
			if err := yamlLoader.Reload(); err != nil {
				logger.WithError(err).Error("Failed to reload rules")
				continue
			}
			loadDuration := time.Since(startLoad)
			store.SetReloadMetadata(time.Now(), loadDuration)

			logger.WithFields(logrus.Fields{
				"rules_loaded":     store.Count(),
				"load_duration_ms": loadDuration.Milliseconds(),
			}).Info("Rules reloaded successfully")
		}
	}()
}
