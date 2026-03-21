package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/api"
	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/services"
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

	logger.Info("Starting KB-22 HPI Engine...")

	// 2. Load configuration
	cfg := config.Load()

	// 3. Initialize database connection
	logger.Info("Connecting to database...")
	db, err := database.NewConnection(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Database connected")

	// 4. Run AutoMigrate for all 6 KB-22 tables
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

	// 7. Initialize NodeLoader and load all P1-P26 YAML definitions
	logger.Info("Loading HPI node definitions...", zap.String("dir", cfg.NodesDir))
	nodeLoader := services.NewNodeLoader(cfg.NodesDir, logger)
	if err := nodeLoader.Load(); err != nil {
		logger.Fatal("Failed to load node definitions", zap.Error(err))
	}
	nodesLoaded := len(nodeLoader.List())
	logger.Info("Node definitions loaded", zap.Int("count", nodesLoaded))

	// 8. Create server
	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, nodeLoader)

	// 9. Initialize all service dependencies
	logger.Info("Initializing services...")
	server.InitServices()

	// 10. Register HTTP routes
	server.RegisterRoutes()

	// 11. Log startup banner
	fmt.Printf(`
========================================
KB-22 HPI Engine
========================================
Service:     kb-22-hpi-engine
Port:        %s
Environment: %s
Nodes:       %d loaded
Database:    connected
Redis:       connected
========================================

API Endpoints:
  Session Management:
    POST   /api/v1/sessions                  Create session
    GET    /api/v1/sessions/:id              Get session state
    POST   /api/v1/sessions/:id/answers      Submit answer
    POST   /api/v1/sessions/:id/suspend      Suspend session
    POST   /api/v1/sessions/:id/resume       Resume session
    POST   /api/v1/sessions/:id/complete     Complete session

  Differential & Safety:
    GET    /api/v1/sessions/:id/differential Ranked differentials
    GET    /api/v1/sessions/:id/safety       Safety flags
    GET    /api/v1/snapshots/:session_id     Completion snapshot

  Node Definitions:
    GET    /api/v1/nodes                     List all nodes
    GET    /api/v1/nodes/:node_id            Get node detail

  Calibration:
    POST   /api/v1/calibration/feedback      Adjudication feedback
    GET    /api/v1/calibration/status/:nid   Concordance metrics
    POST   /api/v1/calibration/import-golden Import golden dataset

  Infrastructure:
    GET    /health                           Health check
    GET    /readiness                        Readiness probe
    GET    /metrics                          Prometheus metrics
    POST   /internal/nodes/reload            Hot-reload nodes
========================================
`, cfg.Port, cfg.Environment, nodesLoaded)

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

	// 12b. Start Kafka signal consumer (feature-flagged)
	if os.Getenv("KB22_KAFKA_ENABLED") == "true" {
		brokerEnv := os.Getenv("KAFKA_BROKERS")
		if brokerEnv == "" {
			logger.Fatal("KB22_KAFKA_ENABLED is set but KAFKA_BROKERS is not configured")
		}
		brokers := strings.Split(brokerEnv, ",")
		signalConsumer := services.NewKB22SignalConsumer(brokers, logger)
		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		defer consumerCancel()
		// NOTE: No-op handler — routing validates the pipeline end-to-end but
		// does not process signals yet. Wire the HPI session intake here
		// when Phase 2 signal processing is implemented. Do NOT enable
		// KB22_KAFKA_ENABLED in staging/production until this is wired.
		signalConsumer.Start(consumerCtx, func(ctx context.Context, action services.KB22RouteAction, data []byte) error {
			logger.Debug("KB-22 signal received (not yet wired)",
				zap.String("action", string(action)))
			return nil
		})
		defer signalConsumer.Stop()
		logger.Info("KB-22 Kafka signal consumer started")
	}

	logger.Info("KB-22 HPI Engine started successfully",
		zap.String("port", cfg.Port),
		zap.Int("nodes_loaded", nodesLoaded),
		zap.String("environment", cfg.Environment),
	)

	// 13. Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	logger.Info("Shutting down KB-22 HPI Engine...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info("HTTP server shutdown completed")
	}

	logger.Info("KB-22 HPI Engine stopped")
}
