package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/server"
	"safety-gateway-platform/pkg/logger"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	port       = flag.Int("port", 8030, "Server port")
	version    = flag.Bool("version", false, "Show version information")
)

const (
	serviceName    = "safety-gateway-platform"
	serviceVersion = "1.0.0"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", serviceName, serviceVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override port if specified via flag
	if *port != 8030 {
		cfg.Service.Port = *port
	}

	// Initialize logger
	logger, err := logger.New(cfg.Observability.Logging)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Safety Gateway Platform",
		zap.String("service", serviceName),
		zap.String("version", serviceVersion),
		zap.Int("port", cfg.Service.Port),
		zap.String("environment", cfg.Service.Environment),
	)

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Start the server
	if err := srv.Start(ctx); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}

	logger.Info("Safety Gateway Platform started successfully")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	logger.Info("Shutting down Safety Gateway Platform...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
	} else {
		logger.Info("Safety Gateway Platform shut down gracefully")
	}
}

// healthCheck performs a basic health check
func healthCheck() error {
	// This would be called by Docker health check
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", *port), 3*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
