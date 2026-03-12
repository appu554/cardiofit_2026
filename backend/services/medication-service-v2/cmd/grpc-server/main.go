package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/config"
	"medication-service-v2/internal/infrastructure/database"
	"medication-service-v2/internal/infrastructure/redis"
	"medication-service-v2/internal/infrastructure/clients"
	"medication-service-v2/internal/infrastructure/monitoring"
	"medication-service-v2/internal/interfaces/grpc"

	"go.uber.org/zap"
	"github.com/spf13/cobra"
)

var (
	configPath string
	grpcCmd = &cobra.Command{
		Use:   "grpc-server",
		Short: "Start the gRPC API server",
		Long:  `Starts the gRPC API server with high-performance binary protocol, authentication, and monitoring.`,
		RunE:  runGRPCServer,
	}
)

func init() {
	grpcCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/config.yaml", "path to configuration file")
}

func main() {
	if err := grpcCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runGRPCServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logger.Level, cfg.Logger.Environment)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting Medication Service V2 gRPC Server",
		zap.String("version", "2.0.0"),
		zap.String("config", configPath))

	// Initialize database
	db, err := database.NewPostgreSQL(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := redis.NewClient(cfg.Redis, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize external clients
	contextGatewayClient, err := clients.NewContextGatewayClient(cfg.ContextGateway, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Context Gateway client", zap.Error(err))
	}

	apolloFederationClient, err := clients.NewApolloFederationClient(cfg.ApolloFederation, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Apollo Federation client", zap.Error(err))
	}

	rustEngineClient, err := clients.NewRustEngineClient(cfg.RustEngine, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Rust Engine client", zap.Error(err))
	}

	// Initialize monitoring
	metrics, err := monitoring.NewMetrics("medication_service_v2", logger)
	if err != nil {
		logger.Fatal("Failed to initialize metrics", zap.Error(err))
	}

	// Initialize application services
	appServices := services.NewServices(
		db,
		redisClient,
		contextGatewayClient,
		apolloFederationClient,
		rustEngineClient,
		logger,
		metrics,
		services.ContextGatewayConfig{
			BaseURL:        cfg.ContextGateway.BaseURL,
			Timeout:        cfg.ContextGateway.Timeout,
			RetryAttempts:  cfg.ContextGateway.RetryAttempts,
			CircuitBreaker: cfg.ContextGateway.CircuitBreaker,
		},
		services.ContextIntegrationConfig{
			CacheEnabled: cfg.Cache.Enabled,
			CacheTTL:     cfg.Cache.DefaultTTL,
			BatchSize:    cfg.RecipeResolver.BatchSize,
		},
		services.WorkflowOrchestratorConfig{
			MaxConcurrentWorkflows: cfg.Workflow.MaxConcurrentWorkflows,
			WorkflowTimeout:        cfg.Workflow.DefaultTimeout,
			StateRetentionDays:     cfg.Workflow.StateRetentionDays,
			EnableMetrics:          cfg.Monitoring.Enabled,
		},
		services.ClinicalIntelligenceConfig{
			MaxConcurrentAnalysis: cfg.ClinicalEngine.MaxConcurrentAnalysis,
			AnalysisTimeout:       cfg.ClinicalEngine.DefaultTimeout,
			CacheResults:          cfg.Cache.Enabled,
			CacheTTL:              cfg.Cache.DefaultTTL,
		},
		services.ProposalGenerationConfig{
			MaxProposalsPerRequest: cfg.Proposals.MaxPerRequest,
			ValidationTimeout:      cfg.Proposals.ValidationTimeout,
			RequireValidation:      cfg.Proposals.RequireValidation,
			AutoExpireHours:        cfg.Proposals.AutoExpireHours,
		},
		services.WorkflowStateServiceConfig{
			StateStoreTTL:          cfg.Workflow.StateStoreTTL,
			CleanupIntervalMinutes: cfg.Workflow.CleanupIntervalMinutes,
			MaxStatesPerWorkflow:   cfg.Workflow.MaxStatesPerWorkflow,
			EnableCompression:      cfg.Workflow.EnableCompression,
		},
		services.MetricsServiceConfig{
			CollectionInterval: cfg.Monitoring.CollectionInterval,
			RetentionDays:      cfg.Monitoring.RetentionDays,
			EnableDetailed:     cfg.Monitoring.EnableDetailed,
			BufferSize:         cfg.Monitoring.BufferSize,
		},
	)

	// Initialize gRPC server
	serverConfig := grpc.ServerConfig{
		Port:                 cfg.GRPC.Port,
		MaxConnectionIdle:    cfg.GRPC.MaxConnectionIdle,
		MaxConnectionAge:     cfg.GRPC.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.GRPC.MaxConnectionAgeGrace,
		Time:                 cfg.GRPC.KeepAlive.Time,
		Timeout:              cfg.GRPC.KeepAlive.Timeout,
		EnableReflection:     cfg.GRPC.EnableReflection,
		EnableAuth:           cfg.Auth.Enabled,
		AuthSecret:           cfg.Auth.JWTSecret,
	}

	grpcServer := grpc.NewServer(logger, appServices, serverConfig)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("gRPC server starting",
			zap.Int("port", serverConfig.Port))
		serverErrors <- grpcServer.Start()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-shutdown:
		logger.Info("Shutdown signal received")
		grpcServer.Stop()
		logger.Info("gRPC server shutdown complete")
	}

	return nil
}

func initLogger(level, environment string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zap.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zap.ISO8601TimeEncoder
	}

	// Set log level
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Add fields for HIPAA compliance
	config.InitialFields = map[string]interface{}{
		"service":   "medication-service-v2",
		"component": "grpc-server",
		"version":   "2.0.0",
	}

	return config.Build()
}