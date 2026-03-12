// Package main is the entry point for KB-14 Care Navigator & Tasking Engine
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/api"
	"kb-14-care-navigator/internal/config"
	"kb-14-care-navigator/internal/workers"
)

func main() {
	// Initialize logger
	log := initLogger()
	log.Info("Starting KB-14 Care Navigator & Tasking Engine...")

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

	// Start background workers if enabled
	var workerManager *workers.WorkerManager
	if cfg.Workers.Enabled {
		workerManager = workers.NewWorkerManager(
			cfg,
			server.GetTaskRepo(),
			server.GetEscalationRepo(),
			server.GetEscalationEngine(),
			server.GetTaskFactory(),
			server.GetKB3Client(),
			server.GetKB9Client(),
			server.GetKB12Client(),
			server.GetRedisCache(),
		)
		if err := workerManager.Start(context.Background()); err != nil {
			log.WithError(err).Fatal("Failed to start background workers")
		}
		log.Info("Background workers started")
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
		"version":     "1.0.0",
	}).Info("KB-14 Care Navigator started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Stop background workers
	if workerManager != nil {
		workerManager.Stop()
		log.Info("Background workers stopped")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
	}

	log.Info("KB-14 Care Navigator shutdown complete")
}

// initLogger initializes the logrus logger
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
		"service": "kb-14-care-navigator",
		"version": "1.0.0",
	})
}
