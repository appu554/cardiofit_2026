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
	bpContextRepo := services.NewBPContextRepository(db.DB)
	bpContextOrch := services.NewBPContextOrchestrator(kb20Client, kb21Client, bpContextRepo, bpThresholds, logger, metricsCollector, kb19Client)

	// 7c. BP context daily batch scheduler (Phase 3)
	bpBatchJob := services.NewBPContextDailyBatch(
		bpContextRepo,
		bpContextOrch,
		time.Duration(cfg.BPActiveWindowDays)*24*time.Hour,
		cfg.BPBatchConcurrency,
		logger,
		metricsCollector,
	)
	batchScheduler := services.NewBatchScheduler(logger)
	batchScheduler.Register(bpBatchJob)

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

	// 9a. Start BP context daily batch scheduler (Phase 3)
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

			// Align first run to BP_BATCH_HOUR_UTC, then loop every 24h.
			delay := computeNextScheduleInterval(time.Now().UTC(), cfg.BPBatchHourUTC)
			logger.Info("BP context batch scheduler waiting for first run",
				zap.Duration("delay", delay),
				zap.Int("hour_utc", cfg.BPBatchHourUTC),
				zap.Int("concurrency", cfg.BPBatchConcurrency),
				zap.Int("active_window_days", cfg.BPActiveWindowDays))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
			batchScheduler.StartLoop(ctx, 24*time.Hour)
		}()
	} else {
		logger.Info("BP context batch scheduler disabled (BP_BATCH_ENABLED=false)")
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

// computeNextScheduleInterval returns the duration until the next
// occurrence of the given UTC hour (0-23). If the current time is already
// past that hour today, it returns the duration until tomorrow's occurrence.
func computeNextScheduleInterval(now time.Time, hourUTC int) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
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
