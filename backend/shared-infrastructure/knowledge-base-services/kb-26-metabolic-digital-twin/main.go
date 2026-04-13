package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kb-26-metabolic-digital-twin/internal/api"
	"kb-26-metabolic-digital-twin/internal/cache"
	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/database"
	"kb-26-metabolic-digital-twin/internal/metrics"
	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"

	"go.uber.org/zap"
)

func main() {
	// 1. Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting KB-26 Metabolic Digital Twin Service")

	// 2. Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	if cfg.IsDevelopment() {
		logger, _ = zap.NewDevelopment()
		defer logger.Sync()
	}

	// 3. Connect to database
	db, err := database.NewConnection(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// 4. AutoMigrate domain models
	if err := db.DB.AutoMigrate(
		&models.TwinState{},
		&models.CalibratedEffect{},
		&models.SimulationRun{},
		&models.MRIScore{},
		&models.MRINadir{},
		&models.RelapseEvent{},
		&models.QuarterlySummary{},
		&models.PREVENTScore{},
		&models.BPContextHistory{},
	); err != nil {
		logger.Fatal("Failed to auto-migrate models", zap.Error(err))
	}
	logger.Info("Database migration complete")

	// 5. Initialize cache (optional — service runs without it)
	cacheClient, err := cache.NewRedisClient(cfg, logger)
	if err != nil {
		logger.Warn("Redis cache unavailable — operating without cache", zap.Error(err))
	}
	if cacheClient != nil {
		defer cacheClient.Close()
	}

	// 6. Initialize metrics
	metricsCollector := metrics.NewCollector()

	// 7. Initialize domain services
	twinUpdater := services.NewTwinUpdater(db.DB, logger)
	calibrator := services.NewBayesianCalibratorWithConfig(db.DB, logger, cfg.BurnInWeeks, cfg.ObservationWindowDays)
	mriScorer := services.NewMRIScorer(db.DB, logger)
	preventScorer := services.NewPREVENTScorer(db.DB, logger)
	kb22Client := clients.NewKB22Client(
		cfg.KB22HPIURL,
		time.Duration(cfg.KB22SignalTimeoutMS)*time.Millisecond,
		logger,
	)
	mriPublisher := services.NewMRIEventPublisher(cfg.KB22HPIURL, cfg.KB23DecisionCardsURL, logger)
	eventProcessor := services.NewEventProcessor(twinUpdater, mriScorer, preventScorer, kb22Client, mriPublisher, logger)
	relapseDetector := services.NewRelapseDetector(db.DB, logger)
	mriScorer.SetRelapseDetector(relapseDetector) // auto-update nadir on every MRI persist
	quarterlyAggregator := services.NewQuarterlyAggregator(db.DB, logger)
	_ = quarterlyAggregator // available for scheduled jobs; not wired into request pipeline

	// 7b. Initialize BP context orchestrator (Phase 2)
	bpThresholds, err := config.LoadBPContextThresholds(cfg.MarketConfigDir, cfg.MarketCode)
	if err != nil {
		logger.Warn("BP context thresholds load failed; using defaults",
			zap.String("market", cfg.MarketCode), zap.Error(err))
		// bpThresholds is nil; orchestrator will fall back to defaultBPContextThresholds()
	}
	kb19Client := clients.NewKB19Client(cfg.KB19ProtocolURL, time.Duration(cfg.KB22SignalTimeoutMS)*time.Millisecond, logger, metricsCollector)
	kb20Client := clients.NewKB20Client(cfg.KB20PatientProfileURL, time.Duration(cfg.KB22SignalTimeoutMS)*time.Millisecond, logger)
	kb21Client := clients.NewKB21Client(cfg.KB21BehavioralURL, time.Duration(cfg.KB22SignalTimeoutMS)*time.Millisecond, logger)
	bpContextRepo := services.NewBPContextRepository(db.DB)
	bpContextOrch := services.NewBPContextOrchestrator(kb20Client, kb21Client, bpContextRepo, bpThresholds, logger, metricsCollector, kb19Client)

	// 8. Create HTTP server
	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, bpContextOrch, twinUpdater, calibrator, eventProcessor, mriScorer, preventScorer, relapseDetector)

	// 9. Start HTTP server
	go func() {
		addr := ":" + cfg.Server.Port
		logger.Info("HTTP server starting", zap.String("address", addr))
		if err := server.Router.Run(addr); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 9b. Start Kafka signal consumer (feature-flagged)
	if os.Getenv("KB26_KAFKA_ENABLED") == "true" {
		brokerEnv := os.Getenv("KAFKA_BROKERS")
		if brokerEnv == "" {
			logger.Fatal("KB26_KAFKA_ENABLED is set but KAFKA_BROKERS is not configured")
		}
		brokers := strings.Split(brokerEnv, ",")
		signalConsumer := services.NewSignalConsumer(brokers, logger)
		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		defer consumerCancel()
		// NOTE: No-op handler — routing validates the pipeline end-to-end but
		// does not process signals yet. Wire eventProcessor.HandleSignal here
		// when Phase 2 signal processing is implemented. Do NOT enable
		// KB26_KAFKA_ENABLED in staging/production until this is wired.
		signalConsumer.Start(consumerCtx, func(ctx context.Context, action services.RouteAction, patientID string, payload json.RawMessage) error {
			logger.Debug("KB-26 signal received (not yet wired)",
				zap.String("action", string(action)),
				zap.String("patient_id", patientID))
			return nil
		})
		defer signalConsumer.Stop()
		logger.Info("KB-26 Kafka signal consumer started")
	}

	// 10. Print service info
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  KB-26 Metabolic Digital Twin Service                   ║")
	fmt.Println("║  Vaidshala Clinical Runtime                             ║")
	fmt.Printf("║  HTTP: http://localhost:%s                           ║\n", cfg.Server.Port)
	fmt.Printf("║  Health: http://localhost:%s/health                  ║\n", cfg.Server.Port)
	fmt.Printf("║  Metrics: http://localhost:%s/metrics                ║\n", cfg.Server.Port)
	fmt.Printf("║  Environment: %-41s ║\n", cfg.Environment)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// 11. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KB-26 Metabolic Digital Twin Service")
}
