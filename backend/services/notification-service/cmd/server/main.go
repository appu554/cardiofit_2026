package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/database"
	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/escalation"
	"github.com/cardiofit/notification-service/internal/fatigue"
	"github.com/cardiofit/notification-service/internal/kafka"
	"github.com/cardiofit/notification-service/internal/routing"
	"github.com/cardiofit/notification-service/internal/server"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database connection
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis client
	redisClient := database.NewRedisClient(cfg.Redis)
	defer redisClient.Close()

	// Initialize delivery providers
	deliveryManager := delivery.NewManager(cfg.Delivery, logger)

	// Initialize fatigue management
	fatigueManager := fatigue.NewManager(redisClient, cfg.Fatigue, logger)

	// Initialize escalation engine
	escalationEngine := escalation.NewEngine(cfg.Escalation, logger)

	// Initialize routing engine
	routingEngine := routing.NewEngine(cfg.Routing, fatigueManager, logger)

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(
		cfg.Kafka,
		routingEngine,
		deliveryManager,
		escalationEngine,
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create Kafka consumer", zap.Error(err))
	}

	// Initialize HTTP server
	httpServer := server.NewHTTPServer(
		deliveryManager,
		escalationEngine,
		db,
		redisClient,
		logger,
		cfg.Server.HTTPPort,
	)

	// Initialize gRPC server
	grpcServer := server.NewGRPCServer(
		deliveryManager,
		escalationEngine,
		db,
		redisClient,
		logger,
		cfg.Server.GRPCPort,
	)

	// Start Kafka consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			logger.Error("Kafka consumer error", zap.Error(err))
		}
	}()

	// Start HTTP server
	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		if err := grpcServer.Start(); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop Kafka consumer
	if err := kafkaConsumer.Stop(); err != nil {
		logger.Error("Error stopping Kafka consumer", zap.Error(err))
	}

	// Stop gRPC server
	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down gRPC server", zap.Error(err))
	}

	// Stop HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down HTTP server", zap.Error(err))
	}

	logger.Info("Service stopped")
}
