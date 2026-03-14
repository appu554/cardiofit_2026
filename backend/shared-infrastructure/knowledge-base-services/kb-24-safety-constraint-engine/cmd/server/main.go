// Package main is the entry point for the KB-24 Safety Constraint Engine.
// The SCE evaluates safety triggers independently of the KB-22 Bayesian inference
// loop. It runs as a sidecar on the same host as KB-22 (~1-2ms latency) and can
// veto the Bayesian engine's output via escalation to KB-19.
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

	"kb-24-safety-constraint-engine/internal/api"
	"kb-24-safety-constraint-engine/internal/config"
	"kb-24-safety-constraint-engine/internal/services"
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

	logger.Info("Starting KB-24 Safety Constraint Engine...")

	// 2. Load configuration
	cfg := config.Load()

	// 3. Load node definitions (safety triggers only)
	logger.Info("Loading node definitions for safety triggers...",
		zap.String("dir", cfg.NodeDefinitionPath),
	)
	nodeLoader := services.NewNodeLoader(cfg.NodeDefinitionPath, logger)
	if err := nodeLoader.Load(); err != nil {
		logger.Fatal("Failed to load node definitions", zap.Error(err))
	}
	nodesLoaded := nodeLoader.Count()
	logger.Info("Node definitions loaded", zap.Int("count", nodesLoaded))

	// 4. Initialize Kafka publisher (or log-only fallback)
	var publisher services.KafkaPublisher
	if cfg.KafkaEnabled {
		logger.Info("Initializing Kafka publisher...",
			zap.String("bootstrap", cfg.KafkaBootstrapServers),
			zap.String("client_id", cfg.KafkaClientID),
		)
		kafkaPub, err := services.NewKafkaGoPublisher(
			cfg.KafkaBootstrapServers,
			cfg.KafkaClientID,
			logger,
		)
		if err != nil {
			logger.Fatal("Failed to initialize Kafka publisher", zap.Error(err))
		}
		publisher = kafkaPub
		defer kafkaPub.Close()
	} else {
		logger.Info("Kafka disabled, using log-only publisher")
		publisher = services.NewLogOnlyPublisher(logger)
	}

	// 5. Create the core evaluator
	evaluator := services.NewSafetyTriggerEvaluator(nodeLoader, logger)

	// 6. Create and configure HTTP server
	server := api.NewServer(cfg, evaluator, publisher, logger)
	server.RegisterRoutes()

	// 7. Log startup banner
	fmt.Printf(`
========================================
KB-24 Safety Constraint Engine
========================================
Service:     kb-24-safety-constraint-engine
Port:        %s
Environment: %s
Nodes:       %d loaded
Kafka:       %v
========================================

API Endpoints:
  POST   /api/v1/evaluate              Evaluate safety triggers
  POST   /api/v1/sessions/:id/clear    Clear session state
  GET    /health                        Health check
========================================
`, cfg.Port, cfg.Environment, nodesLoaded, cfg.KafkaEnabled)

	// 8. Start HTTP server
	httpServer := &http.Server{
		Addr:         cfg.GetAddr(),
		Handler:      server.Router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("KB-24 Safety Constraint Engine started successfully",
		zap.String("port", cfg.Port),
		zap.Int("nodes_loaded", nodesLoaded),
		zap.String("environment", cfg.Environment),
	)

	// 9. Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	logger.Info("Shutting down KB-24 Safety Constraint Engine...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info("HTTP server shutdown completed")
	}

	logger.Info("KB-24 Safety Constraint Engine stopped")
}
