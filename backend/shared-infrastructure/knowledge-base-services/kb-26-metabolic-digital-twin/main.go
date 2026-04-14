package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	"kb-26-metabolic-digital-twin/pkg/stability"

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

	// Top-level cancellable context for graceful shutdown — propagated to
	// scheduler, Kafka consumer, and any other long-running goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	// Phase 4 P9: KB-23 client triggers composite card synthesis after each
	// successful BP context classification. Best-effort — orchestrator
	// swallows failures so composite outages never block classification.
	kb23Client := clients.NewKB23Client(cfg.KB23DecisionCardsURL, time.Duration(cfg.KB22SignalTimeoutMS)*time.Millisecond, logger)
	bpContextRepo := services.NewBPContextRepository(db.DB)

	// BP context phenotype stability engine (Phase 4 P2 + Phase 5 P5-1)
	// MinDwell 14 days: phenotype must be held 2 weeks before transition
	// FlapWindow 30 days: oscillation lookback
	// MaxFlapsBeforeLock 3: after 3 state changes in 30d, lock transitions
	// MaxDwellOverrideRate 0.7: if the raw classifier output has agreed
	// with the proposed transition on >=70% of in-window snapshots, the
	// dwell yields. This prevents the dwell from indefinitely suppressing
	// genuine phenotype changes that occur without a discrete override
	// event (e.g. gradual physiological shifts, sustained measurement
	// improvements that aren't tied to a medication change).
	bpStabilityPolicy := stability.Policy{
		MinDwell:             14 * 24 * time.Hour,
		FlapWindow:           30 * 24 * time.Hour,
		MaxFlapsBeforeLock:   3,
		MaxDwellOverrideRate: 0.7,
	}
	bpStabilityEngine := stability.NewEngine(bpStabilityPolicy)

	bpContextOrch := services.NewBPContextOrchestrator(kb20Client, kb21Client, bpContextRepo, bpThresholds, logger, metricsCollector, kb19Client, bpStabilityEngine, kb23Client)

	// 7c. BP context daily batch scheduler (Phase 3, hour-gated in Phase 5 P5-3)
	bpBatchJob := services.NewBPContextDailyBatch(
		bpContextRepo,
		bpContextOrch,
		time.Duration(cfg.BPActiveWindowDays)*24*time.Hour,
		cfg.BPBatchConcurrency,
		logger,
		metricsCollector,
	)
	// Phase 5 P5-3: scheduler now ticks hourly and each job self-filters
	// via ShouldRun. The BP job fires only when the current UTC hour
	// matches the configured BatchHourUTC — replaces the Phase 3
	// computeNextScheduleInterval / 24h-tick dance with a single gate.
	bpBatchJob.BatchHourUTC = cfg.BPBatchHourUTC

	batchScheduler := services.NewBatchScheduler(logger)
	batchScheduler.Register(bpBatchJob)

	// Phase 6 P6-1: the inertia weekly batch moved from KB-26 to KB-23.
	// The Phase 5 P5-3 heartbeat that used to live here was a placeholder
	// because the detector lives in KB-23 — KB-26 cannot import
	// KB-23 internals. The canonical InertiaWeeklyBatch now lives at
	// kb-23-decision-cards/internal/services/inertia_weekly_batch.go
	// and is registered in KB-23's BatchScheduler (built in Phase 6 P6-5).
	// This makes the detector + orchestrator + batch all colocated.

	// 8. Create HTTP server
	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, bpContextOrch, twinUpdater, calibrator, eventProcessor, mriScorer, preventScorer, relapseDetector)

	// 9. Start HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      server.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 9a. Start BP context + inertia batch scheduler (Phase 3, hourly tick in Phase 5 P5-3)
	if cfg.BPBatchEnabled {
		go func() {
			// Per-market batch hour recommendations — log a warning if the
			// configured hour overlaps local clinic hours for the market.
			recommendedHour := recommendedBatchHourForMarket(cfg.MarketCode)
			if recommendedHour != -1 && cfg.BPBatchHourUTC != recommendedHour {
				logger.Warn("BP batch hour may overlap local clinic hours",
					zap.String("market", cfg.MarketCode),
					zap.Int("configured_hour_utc", cfg.BPBatchHourUTC),
					zap.Int("recommended_hour_utc", recommendedHour))
			}

			// Phase 5 P5-3: scheduler ticks hourly; each registered job
			// self-filters via ShouldRun. BPContextDailyBatch.ShouldRun
			// returns true only when the current UTC hour matches
			// BatchHourUTC, InertiaWeeklyBatch.ShouldRun returns true only
			// on Mondays, and any future consumer brings its own cadence
			// predicate without touching the scheduler.
			logger.Info("batch scheduler starting (hourly tick, per-job cadence gate)",
				zap.Int("bp_hour_utc", cfg.BPBatchHourUTC),
				zap.Int("bp_concurrency", cfg.BPBatchConcurrency),
				zap.Int("active_window_days", cfg.BPActiveWindowDays))
			batchScheduler.StartLoop(ctx, 1*time.Hour)
		}()
	} else {
		logger.Info("batch scheduler disabled (BP_BATCH_ENABLED=false)")
	}

	// 9b. Start Kafka signal consumer (feature-flagged)
	if os.Getenv("KB26_KAFKA_ENABLED") == "true" {
		brokerEnv := os.Getenv("KAFKA_BROKERS")
		if brokerEnv == "" {
			logger.Fatal("KB26_KAFKA_ENABLED is set but KAFKA_BROKERS is not configured")
		}
		brokers := strings.Split(brokerEnv, ",")
		signalConsumer := services.NewSignalConsumer(brokers, logger)
		consumerCtx, consumerCancel := context.WithCancel(ctx)
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
	sig := <-quit

	logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	logger.Info("Shutting down KB-26 Metabolic Digital Twin Service")

	// Cancel top-level context — propagates to scheduler, Kafka consumer, etc.
	cancel()

	// Drain in-flight batch work (waits for any currently-executing RunOnce)
	batchScheduler.Drain()

	// Graceful HTTP shutdown — waits for in-flight requests with a 15s deadline
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info("HTTP server shutdown completed")
	}
}

// recommendedBatchHourForMarket returns the UTC hour that minimises
// overlap with local clinic hours in each supported market.
//
//	india     -> 22:00 UTC (03:30 IST — pre-dawn)
//	australia -> 14:00 UTC (00:00 AEST — midnight)
//	shared    -> 02:00 UTC (default — assumes US/EU clinic flow)
//
// Returns -1 for unknown markets so the warning is suppressed.
func recommendedBatchHourForMarket(market string) int {
	switch market {
	case "india":
		return 22
	case "australia":
		return 14
	case "shared", "":
		return 2
	default:
		return -1
	}
}
