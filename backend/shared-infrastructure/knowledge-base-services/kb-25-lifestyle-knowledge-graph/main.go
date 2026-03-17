package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kb-25-lifestyle-knowledge-graph/internal/api"
	"kb-25-lifestyle-knowledge-graph/internal/cache"
	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/config"
	"kb-25-lifestyle-knowledge-graph/internal/graph"
	"kb-25-lifestyle-knowledge-graph/internal/metrics"
	"kb-25-lifestyle-knowledge-graph/internal/services"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting KB-25 Lifestyle Knowledge Graph Service")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	if cfg.IsDevelopment() {
		logger, _ = zap.NewDevelopment()
		defer logger.Sync()
	}

	graphClient, err := graph.NewClient(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Neo4j", zap.Error(err))
	}
	defer graphClient.Close(context.Background())

	cacheClient, err := cache.NewRedisClient(cfg, logger)
	if err != nil {
		logger.Warn("Redis cache unavailable — operating without cache", zap.Error(err))
	}
	if cacheClient != nil {
		defer cacheClient.Close()
	}

	metricsCollector := metrics.NewCollector()

	chainSvc := services.NewChainTraversalService(graphClient, logger)

	kb4Client := clients.NewKB4Client(cfg.KB4PatientSafetyURL, logger)
	safetyEng := services.NewSafetyEngine(graphClient, kb4Client, logger)

	kb20Client := clients.NewKB20Client(cfg.KB20PatientProfileURL, logger)

	comparatorEng := services.NewComparatorEngine(chainSvc, logger)

	server := api.NewServer(cfg, graphClient, cacheClient, metricsCollector, logger, chainSvc, safetyEng, kb20Client, comparatorEng)

	go func() {
		addr := ":" + cfg.Server.Port
		logger.Info("HTTP server starting", zap.String("address", addr))
		if err := server.Router.Run(addr); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  KB-25 Lifestyle Knowledge Graph Service                ║")
	fmt.Println("║  Vaidshala Clinical Runtime                             ║")
	fmt.Printf("║  HTTP: http://localhost:%s                           ║\n", cfg.Server.Port)
	fmt.Printf("║  Health: http://localhost:%s/health                  ║\n", cfg.Server.Port)
	fmt.Printf("║  Environment: %-41s ║\n", cfg.Environment)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KB-25 Lifestyle Knowledge Graph Service")
}
