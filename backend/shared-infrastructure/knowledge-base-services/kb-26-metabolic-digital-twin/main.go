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
		// Phase 7 P7-E Milestone 2: persistence target for the
		// CGM analytics consumer.
		&models.CGMPeriodReport{},
		// PAI (Patient Acuity Index) persistence models
		&models.PAIScore{},
		&models.PAIHistory{},
		// Acute-on-chronic detection (Gap 16)
		&models.AcuteEvent{},
		&models.PatientBaselineSnapshot{},
		// Predictive risk layer (Gap 20)
		&models.PredictedRisk{},
		// Attribution + governance ledger (Gap 21)
		&models.AttributionVerdict{},
		&models.LedgerEntry{},
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

	// Phase 6 P6-4: CGM daily batch — checks every CGM-active patient
	// once per day at 01:00 UTC and computes a fresh 14-day period
	// report for any patient whose last report is ≥14 days old.
	// Currently heartbeat mode (nil repo + nil fetcher) — the
	// ComputePeriodReport function is fully built and tested, but the
	// raw reading store + CGM-active patient query are Phase 6 follow-ups.
	cgmDailyJob := services.NewCGMDailyBatch(nil, nil, logger)
	batchScheduler.Register(cgmDailyJob)

	// Phase 7 P7-F: Construct KafkaTrajectoryPublisher up front so both
	// the HTTP server (TrajectoryEngine) and the SignalConsumer share the
	// same feature-flag gate. When KB26_KAFKA_ENABLED is unset, we pass nil
	// and NewServer defaults to NoopTrajectoryPublisher.
	var trajectoryPublisher services.TrajectoryPublisher
	if os.Getenv("KB26_KAFKA_ENABLED") == "true" {
		brokerEnv := os.Getenv("KAFKA_BROKERS")
		if brokerEnv == "" {
			logger.Fatal("KB26_KAFKA_ENABLED is set but KAFKA_BROKERS is not configured")
		}
		brokers := strings.Split(brokerEnv, ",")
		trajectoryPublisher = services.NewKafkaTrajectoryPublisher(brokers, "kb26.domain_trajectory.v1", logger)
		logger.Info("KafkaTrajectoryPublisher wired", zap.String("topic", "kb26.domain_trajectory.v1"))
	}

	// 7d. Initialize PAI services (Patient Acuity Index)
	paiRepo := services.NewPAIRepository(db.DB)
	paiTrigger := services.NewPAIEventTrigger(15, 10.0, paiRepo) // 15-min rate limit, delta 10, DB fallback

	// Load PAI config from YAML — try deployment path, then relative dev path.
	var paiCfg *services.PAIConfig
	for _, p := range []string{
		"/app/market-configs/shared/pai_dimensions.yaml",
		"../../market-configs/shared/pai_dimensions.yaml",
	} {
		if loaded, err := services.LoadPAIConfig(p); err == nil {
			paiCfg = loaded
			logger.Info("loaded PAI config from YAML", zap.String("path", p))
			break
		}
	}
	if paiCfg == nil {
		logger.Warn("pai_dimensions.yaml not found, using default PAI config")
		paiCfg = services.DefaultPAIConfig()
	}

	// 7e. Initialize acute-on-chronic detection services (Gap 16)
	acuteRepo := services.NewAcuteRepository(db.DB)
	acuteHandler := services.NewAcuteEventHandler(nil, acuteRepo, logger) // nil config → defaults

	// 8. Create HTTP server
	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, bpContextOrch, twinUpdater, calibrator, eventProcessor, mriScorer, preventScorer, relapseDetector, trajectoryPublisher)
	server.SetPAIServices(paiRepo, paiTrigger, paiCfg)
	server.SetAcuteServices(acuteRepo, acuteHandler)

	// Gap 21: governance ledger for attribution runs + model promotions.
	// HMAC key from env for Sprint 1; Sprint 2 adds Ed25519 per-entry signatures.
	hmacKey := os.Getenv("GAP21_LEDGER_HMAC_KEY")
	if hmacKey == "" {
		logger.Warn("GAP21_LEDGER_HMAC_KEY is not set — ledger will use an insecure default key (not suitable for production)")
	}
	gap21Ledger := services.NewInMemoryLedger([]byte(hmacKey))

	// Seed the in-memory sequence counter from the DB so restarts don't
	// collide with the Sequence uniqueIndex. If the governance_ledger_entries
	// table is empty or unreachable, default to 0.
	var maxLedgerSeq int64 = -1
	if db != nil && db.DB != nil {
		row := db.DB.Table("governance_ledger_entries").Select("COALESCE(MAX(sequence), -1)").Row()
		if err := row.Scan(&maxLedgerSeq); err != nil {
			logger.Warn("failed to seed ledger sequence from DB; starting fresh at 0", zap.Error(err))
			maxLedgerSeq = -1
		}
	}
	if maxLedgerSeq >= 0 {
		gap21Ledger.SeedSequence(maxLedgerSeq + 1)
		logger.Info("governance ledger sequence seeded from DB", zap.Int64("next_sequence", maxLedgerSeq+1))
	}
	server.SetGap21Services(gap21Ledger)

	// Gap 21 Sprint 2a Task 5: load attribution config from YAML. Degrades to
	// rule-based defaults if file is missing; logs error if file exists
	// but parses invalidly.
	//
	// Path resolution matches the PAI pattern: env override first, then
	// Docker mount (/app/market-configs/), then repo-relative path from
	// the service directory. Without this fallback chain the YAML never
	// resolves in Docker (market-configs/ is not a child of /app/).
	attributionCfgPath := os.Getenv("GAP21_ATTRIBUTION_CONFIG_PATH")
	if attributionCfgPath == "" {
		for _, p := range []string{
			"/app/market-configs/shared/attribution_parameters.yaml",
			"../../market-configs/shared/attribution_parameters.yaml",
			"market-configs/shared/attribution_parameters.yaml",
		} {
			if _, statErr := os.Stat(p); statErr == nil {
				attributionCfgPath = p
				break
			}
		}
		if attributionCfgPath == "" {
			// No file found on any candidate path — pass the Docker path
			// so LoadAttributionConfig's missing-file fallback produces
			// defaults and the error log names the expected location.
			attributionCfgPath = "/app/market-configs/shared/attribution_parameters.yaml"
		}
	}
	attrCfg, attrCfgErr := config.LoadAttributionConfig(attributionCfgPath)
	if attrCfgErr != nil {
		logger.Error("failed to load attribution config; using defaults",
			zap.Error(attrCfgErr),
			zap.String("path", attributionCfgPath))
		logger.Info("attribution config effective (fallback default after load error)",
			zap.String("method", attrCfg.Method),
			zap.String("version", attrCfg.MethodVersion))
	} else {
		logger.Info("attribution config loaded from YAML",
			zap.String("path", attributionCfgPath),
			zap.String("method", attrCfg.Method),
			zap.String("version", attrCfg.MethodVersion))
	}
	server.SetAttributionConfig(attrCfg)

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

		// Phase 7 P7-E Milestone 2: CGM analytics consumer subscribes
		// to clinical.cgm-analytics.v1 produced by Flink's
		// Module3_CGMStreamJob and persists each event into
		// cgm_period_reports. KB-23's inertia input assembler reads
		// these rows to populate the CGM_TIR branch of the glycaemic
		// inertia detector.
		cgmPeriodReportRepo := services.NewCGMPeriodReportRepository(db.DB, logger)
		cgmAnalyticsConsumer := services.NewCGMAnalyticsConsumer(brokers, logger)
		cgmAnalyticsConsumer.Start(
			consumerCtx,
			services.PersistingCGMAnalyticsHandler(cgmPeriodReportRepo, logger),
		)
		defer cgmAnalyticsConsumer.Stop()
		logger.Info("KB-26 CGM analytics consumer started (persisting to cgm_period_reports)")
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
