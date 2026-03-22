package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/api"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	intakekafka "github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting Intake-Onboarding Service...")

	cfg := config.Load()

	// Connect PostgreSQL
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logger.Fatal("invalid database URL", zap.Error(err))
	}
	poolCfg.MaxConns = cfg.Database.MaxConnections
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer dbPool.Close()
	logger.Info("Connected to PostgreSQL")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}
	opt.Password = cfg.Redis.Password
	opt.DB = cfg.Redis.DB
	redisClient := redis.NewClient(opt)
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// Create FHIR client (optional -- disabled in dev if no credentials)
	var fhirClient *fhirclient.Client
	if cfg.FHIR.Enabled {
		fhirClient, err = fhirclient.New(cfg.FHIR, logger)
		if err != nil {
			logger.Warn("FHIR Store client disabled — no credentials", zap.Error(err))
		} else {
			logger.Info("FHIR Store client initialized")
		}
	}

	// Initialize safety engine (deterministic, no external deps)
	safetyEngine := safety.NewEngine()
	logger.Info("Safety engine initialized", zap.Int("hard_stops", 11), zap.Int("soft_flags", 8))

	// Load flow graph
	var flowEngine *flow.Engine
	flowPath := "configs/flows/intake_full.yaml"
	if graph, err := flow.LoadGraph(flowPath); err != nil {
		logger.Warn("Flow graph not loaded — using stub mode", zap.Error(err))
	} else {
		flowEngine = flow.NewEngine(graph)
		logger.Info("Flow graph loaded", zap.String("id", graph.ID), zap.Int("nodes", len(graph.Nodes)))
	}

	// Initialize Kafka producer
	var producer *intakekafka.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer = intakekafka.NewProducer(cfg.Kafka.Brokers, logger)
		defer producer.Close()
		logger.Info("Kafka producer initialized", zap.Int("topics", len(intakekafka.AllTopics())))
	}

	// Create HTTP server with all dependencies
	server := api.NewServer(cfg, dbPool, redisClient, fhirClient, logger,
		safetyEngine, flowEngine, producer)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: server.Router,
	}

	go func() {
		logger.Info("Intake-Onboarding Service listening", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Intake-Onboarding Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Intake-Onboarding Service stopped")
}
