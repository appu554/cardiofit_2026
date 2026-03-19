package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kb-21-behavioral-intelligence/internal/api"
	"kb-21-behavioral-intelligence/internal/cache"
	"kb-21-behavioral-intelligence/internal/config"
	"kb-21-behavioral-intelligence/internal/database"
	"kb-21-behavioral-intelligence/internal/events"
	"kb-21-behavioral-intelligence/internal/metrics"
	"kb-21-behavioral-intelligence/internal/services"

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

	logger.Info("Starting KB-21 Behavioral Intelligence Service")

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

	// 4. Initialize cache (optional — service runs without it)
	cacheClient, err := cache.NewRedisClient(cfg, logger)
	if err != nil {
		logger.Warn("Redis cache unavailable — operating without cache", zap.Error(err))
	}
	if cacheClient != nil {
		defer cacheClient.Close()
	}

	// 5. Initialize metrics
	metricsCollector := metrics.NewCollector()

	// 6. Initialize event publisher
	publisher := events.NewPublisher(logger, cfg.EventBusEnabled, cfg.KafkaTopic)

	// 7. Initialize domain services
	safetyClient := services.NewSafetyClient(cfg.KB23SafetyURL, logger) // G-01/G-03: KB-23 fast-path
	kb1Client := services.NewKB1Client(cfg.KB1DrugRulesURL, logger)     // FDC decomposition for adherence
	adherenceSvc := services.NewAdherenceServiceWithKB1(db.DB, logger, kb1Client)
	engagementSvc := services.NewEngagementService(db.DB, logger, cfg.PreGatewayDefaultAdherence) // G-04
	correlationSvc := services.NewCorrelationService(db.DB, logger, cfg.OutcomeCorrelationMinEvents, safetyClient, publisher) // G-01 + Gap #23
	hypoRiskSvc := services.NewHypoRiskService(db.DB, logger, publisher, safetyClient) // G-03

	// BCE v1.0 Nudge Engine
	bayesianEngine := services.NewBayesianEngine(db.DB, logger)
	phaseEngine := services.NewPhaseEngine(db.DB, logger)
	barrierDiag := services.NewBarrierDiagnostic(db.DB, logger)

	// BCE v2.0 E1: Cold-Start Profiling
	var coldStartEngine *services.ColdStartEngine
	if cfg.ColdStartEnabled {
		coldStartEngine = services.NewColdStartEngine(db.DB, logger)
		logger.Info("BCE v2.0 E1: Cold-start profiling enabled")
	}

	nudgeEngine := services.NewNudgeEngine(
		db.DB, logger, bayesianEngine, phaseEngine, barrierDiag,
		coldStartEngine,    // E1
		nil,                // E2: gamificationEngine — wired in Task 11
		nil,                // E4: timingBandit — wired in Task 11
		cfg.NudgeMaxPerDay, cfg.NudgeCooldownHours,
	)

	// 8. Load festival calendar (optional — graceful nil if file missing)
	var festivalCal *services.FestivalCalendar
	if cal, err := services.NewFestivalCalendar(cfg.FestivalCalendarPath); err != nil {
		logger.Warn("Festival calendar not loaded — P4 perturbation data unavailable",
			zap.String("path", cfg.FestivalCalendarPath),
			zap.Error(err))
	} else {
		festivalCal = cal
		logger.Info("Festival calendar loaded", zap.String("path", cfg.FestivalCalendarPath))
	}

	// 9. Initialize event subscriber
	subscriber := events.NewSubscriber(logger, correlationSvc, adherenceSvc, cfg.EventBusEnabled)

	// 10. Create HTTP server
	server := api.NewServer(
		cfg, db, cacheClient, metricsCollector, logger,
		adherenceSvc, engagementSvc, correlationSvc, hypoRiskSvc,
		festivalCal, nudgeEngine, coldStartEngine, subscriber,
	)

	// 11. Start event subscriber
	if err := subscriber.Start(); err != nil {
		logger.Error("Event subscriber start failed", zap.Error(err))
	}

	// 12. Start HTTP server
	go func() {
		addr := ":" + cfg.Server.Port
		logger.Info("HTTP server starting", zap.String("address", addr))
		if err := server.Router.Run(addr); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 13. Print service info
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  KB-21 Behavioral Intelligence Service                  ║")
	fmt.Println("║  Vaidshala Clinical Synthesis Hub                       ║")
	fmt.Printf("║  HTTP: http://localhost:%s                           ║\n", cfg.Server.Port)
	fmt.Printf("║  Health: http://localhost:%s/health                  ║\n", cfg.Server.Port)
	fmt.Printf("║  Metrics: http://localhost:%s/metrics                ║\n", cfg.Server.Port)
	fmt.Printf("║  Environment: %-41s ║\n", cfg.Environment)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// 14. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KB-21 Behavioral Intelligence Service")
}
