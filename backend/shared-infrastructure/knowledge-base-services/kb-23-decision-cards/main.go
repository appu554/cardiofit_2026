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

	"kb-23-decision-cards/internal/api"
	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/services"
)

func main() {
	// 1. Initialize structured logging
	var logger *zap.Logger
	var logErr error
	if os.Getenv("ENVIRONMENT") == "production" {
		logger, logErr = zap.NewProduction()
	} else {
		logger, logErr = zap.NewDevelopment()
	}
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", logErr)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting KB-23 Decision Cards Engine...")

	// 2. Load configuration
	cfg := config.Load()

	// 3. Initialize database connection
	logger.Info("Connecting to database...")
	db, err := database.NewConnection(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Database connected")

	// 4. Run AutoMigrate for all 8 KB-23 tables
	logger.Info("Running auto-migration...")
	if err := db.AutoMigrate(); err != nil {
		logger.Fatal("Failed to run auto-migration", zap.Error(err))
	}
	logger.Info("Database migration completed")

	// 5. Initialize Redis cache
	logger.Info("Connecting to Redis cache...")
	cacheClient, err := cache.NewCacheClient(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer cacheClient.Close()
	logger.Info("Redis cache connected")

	// 6. Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// 7. Initialize TemplateLoader and load all decision card YAML definitions
	logger.Info("Loading decision card templates...", zap.String("dir", cfg.TemplatesDir))
	templateLoader := services.NewTemplateLoader(cfg.TemplatesDir, logger)
	if err := templateLoader.Load(); err != nil {
		logger.Fatal("Failed to load decision card templates", zap.Error(err))
	}
	templatesLoaded := len(templateLoader.List())
	logger.Info("Decision card templates loaded", zap.Int("count", templatesLoaded))

	// 8. Create server
	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, templateLoader)

	// 9. Initialize all service dependencies
	logger.Info("Initializing services...")
	server.InitServices()

	// 10. Register HTTP routes
	server.RegisterRoutes()

	// 11. Log startup banner
	fmt.Printf(`
========================================
KB-23 Decision Cards Engine
========================================
Service:     kb-23-decision-cards
Port:        %s
Environment: %s
Templates:   %d loaded
Database:    connected
Redis:       connected
========================================

API Endpoints:
  Card Management:
    POST   /api/v1/cards                       Create decision card
    GET    /api/v1/cards                       List decision cards
    GET    /api/v1/cards/:id                   Get decision card
    PUT    /api/v1/cards/:id                   Update decision card
    DELETE /api/v1/cards/:id                   Delete decision card

  Card Rendering:
    POST   /api/v1/cards/:id/render            Render card for context
    POST   /api/v1/cards/batch-render          Batch render cards

  Templates:
    GET    /api/v1/templates                   List all templates
    GET    /api/v1/templates/:template_id      Get template detail

  Calibration:
    POST   /api/v1/calibration/feedback        Adjudication feedback
    GET    /api/v1/calibration/status/:id      Concordance metrics

  Infrastructure:
    GET    /health                             Health check
    GET    /readiness                          Readiness probe
    GET    /metrics                            Prometheus metrics
    POST   /internal/templates/reload          Hot-reload templates
========================================
`, cfg.Port, cfg.Environment, templatesLoaded)

	// 12. Start HTTP server
	httpServer := &http.Server{
		Addr:         cfg.GetAddr(),
		Handler:      server.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("KB-23 Decision Cards Engine started successfully",
		zap.String("port", cfg.Port),
		zap.Int("templates_loaded", templatesLoaded),
		zap.String("environment", cfg.Environment),
	)

	// 13. Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	logger.Info("Shutting down KB-23 Decision Cards Engine...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info("HTTP server shutdown completed")
	}

	logger.Info("KB-23 Decision Cards Engine stopped")
}
