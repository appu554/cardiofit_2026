package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/api/grpc"
	"global-outbox-service-go/internal/api/http"
	"global-outbox-service-go/internal/circuitbreaker"
	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/database"
	"global-outbox-service-go/internal/publisher"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger := setupLogger(cfg)
	logger.Infof("Starting %s v%s", cfg.ProjectName, cfg.Version)
	logger.Infof("Environment: %s", cfg.Environment)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Info("Received shutdown signal, initiating graceful shutdown...")
		cancel()
	}()

	// Initialize components
	if err := run(ctx, cfg, logger); err != nil {
		logger.Fatalf("Application failed: %v", err)
	}

	logger.Info("Application shutdown complete")
}

func run(ctx context.Context, cfg *config.Config, logger *logrus.Logger) error {
	// Initialize database repository
	logger.Info("Initializing database connection...")
	repo, err := database.NewRepository(cfg, logger)
	if err != nil {
		return err
	}
	defer repo.Close()

	// Initialize medical circuit breaker
	logger.Info("Initializing medical circuit breaker...")
	circuitBreaker := circuitbreaker.NewMedicalCircuitBreaker(cfg, logger)

	// Initialize Kafka publisher
	var kafkaPublisher *publisher.KafkaPublisher
	if cfg.PublisherEnabled {
		logger.Info("Initializing Kafka publisher...")
		kafkaPublisher, err = publisher.NewKafkaPublisher(cfg, repo, circuitBreaker, logger)
		if err != nil {
			return err
		}
		defer kafkaPublisher.Stop()

		// Start publisher
		if err := kafkaPublisher.Start(ctx); err != nil {
			return err
		}
	}

	// Initialize and start gRPC server
	logger.Info("Initializing gRPC server...")
	grpcServer := grpc.NewServer(repo, circuitBreaker, cfg, logger)
	if err := grpcServer.Start(); err != nil {
		return err
	}
	defer grpcServer.Stop()

	// Initialize and start HTTP server
	logger.Info("Initializing HTTP server...")
	httpServer := http.NewServer(repo, circuitBreaker, cfg, logger)
	if err := httpServer.Start(); err != nil {
		return err
	}
	defer httpServer.Stop()

	logger.Info("All services started successfully!")

	// Wait for shutdown signal
	<-ctx.Done()

	logger.Info("Shutting down services...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop HTTP server
	if err := httpServer.Stop(); err != nil {
		logger.Errorf("Error stopping HTTP server: %v", err)
	}

	// Stop gRPC server
	grpcServer.Stop()

	// Stop Kafka publisher
	if kafkaPublisher != nil {
		if err := kafkaPublisher.Stop(); err != nil {
			logger.Errorf("Error stopping Kafka publisher: %v", err)
		}
	}

	// Wait for shutdown or timeout
	select {
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded")
	default:
		logger.Info("Graceful shutdown completed")
	}

	return nil
}

func setupLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if cfg.IsProduction() {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
			ForceColors:     true,
		})
	}

	// Set output
	logger.SetOutput(os.Stdout)

	return logger
}