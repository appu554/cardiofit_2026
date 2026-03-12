// Package main is the entry point for KB-17 Population Registry Service
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/api"
	"kb-17-population-registry/internal/config"
)

const (
	serviceName    = "kb-17-population-registry"
	serviceVersion = "1.0.0"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetLevel(logrus.InfoLevel)

	log := logger.WithFields(logrus.Fields{
		"service": serviceName,
		"version": serviceVersion,
	})

	log.Info("Starting KB-17 Population Registry Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level from config
	level, err := logrus.ParseLevel(cfg.Server.LogLevel)
	if err == nil {
		logger.SetLevel(level)
	}

	log.WithFields(logrus.Fields{
		"port":        cfg.Server.Port,
		"environment": cfg.Server.Environment,
	}).Info("Configuration loaded")

	// Create and initialize server
	server, err := api.NewServer(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create server")
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.WithField("address", addr).Info("HTTP server starting")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.WithError(err).Error("HTTP server shutdown error")
	}

	// Close server resources
	if err := server.Close(); err != nil {
		log.WithError(err).Error("Server close error")
	}

	log.Info("Server exited properly")
}
