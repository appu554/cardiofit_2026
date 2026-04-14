package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/api"
	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/services"
)

func main() {
	// 1. Initialize structured logging
	var logger *zap.Logger
	var logErr error
	if os.Getenv("ENVIRONMENT") == "production" {
		logger, logErr = zap.NewProduction()
	} else {
		logger, logErr = zap.NewDevelopment()
	}
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", logErr)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting KB-23 Decision Cards Engine...")

	// 2. Load configuration
	cfg := config.Load()

	// 3. Initialize database connection
	logger.Info("Connecting to database...")
	db, err := database.NewConnection(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Database connected")

	// 4. Run AutoMigrate for all 8 KB-23 tables
	logger.Info("Running auto-migration...")
	if err := db.AutoMigrate(); err != nil {
		logger.Fatal("Failed to run auto-migration", zap.Error(err))
	}
	logger.Info("Database migration completed")

	// 5. Initialize Redis cache
	logger.Info("Connecting to Redis cache...")
	cacheClient, err := cache.NewCacheClient(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer cacheClient.Close()
	logger.Info("Redis cache connected")

	// 6. Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// 7. Initialize TemplateLoader and load all decision card YAML definitions
	logger.Info("Loading decision card templates...", zap.String("dir", cfg.TemplatesDir))
	templateLoader := services.NewTemplateLoader(cfg.TemplatesDir, logger)
	if err := templateLoader.Load(); err != nil {
		logger.Fatal("Failed to load decision card templates", zap.Error(err))
	}
	templatesLoaded := len(templateLoader.List())
	logger.Info("Decision card templates loaded", zap.Int("count", templatesLoaded))

	// 8. Create server
	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, templateLoader)

	// 9. Initialize all service dependencies
	logger.Info("Initializing services...")
	server.InitServices()

	// 10. Register HTTP routes
	server.RegisterRoutes()

	// 10b. Feature-flagged Kafka priority signal consumer
	var priorityConsumer *services.PrioritySignalConsumer
	if os.Getenv("KB23_KAFKA_ENABLED") == "true" {
		brokerEnv := os.Getenv("KAFKA_BROKERS")
		if brokerEnv == "" {
			logger.Fatal("KB23_KAFKA_ENABLED is set but KAFKA_BROKERS is not configured")
		}
		brokers := strings.Split(brokerEnv, ",")

		// Phase 6 P6-6: MandatoryMedChecker + KB20Client wired into the
		// priority signal handler so the new CKM_STAGE_TRANSITION
		// dispatch can fetch patient context and detect GDMT gaps on
		// 4c transitions. MandatoryMedChecker is stateless — fresh
		// instance per startup is fine.
		//
		// Phase 6 P6-2: RenalDoseGate (built from the renal formulary)
		// also wired so the new EGFR_LAB dispatch can run reactive
		// renal dose gating. Formulary load failure logs a warning
		// and leaves the gate nil — the handler defensively no-ops.
		var renalGate *services.RenalDoseGate
		if formulary, ferr := services.LoadRenalFormulary(cfg.TemplatesDir, cfg.Market); ferr == nil {
			renalGate = services.NewRenalDoseGate(formulary)
		} else {
			logger.Warn("renal formulary load failed; reactive renal gate disabled",
				zap.Error(ferr))
		}

		priorityHandler := services.NewPrioritySignalHandler(
			server.Database(),
			server.MCUGateCache(),
			server.KB19Publisher(),
			server.HypoHandler(),
			services.NewMandatoryMedChecker(),
			server.KB20Client(),
			renalGate,
			server.MetricsCollector(),
			logger,
		)
		priorityConsumer = services.NewPrioritySignalConsumer(brokers, logger)

		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		defer consumerCancel()

		priorityConsumer.Start(consumerCtx, priorityHandler.Handle)
		logger.Info("KB-23 priority signal consumer started")
	}

	// 10c. Phase 6 P6-5: KB-23 BatchScheduler + RenalAnticipatoryBatch
	// (first KB-23 batch consumer, proves the Phase 5 P5-3 scheduler
	// abstraction extracts cleanly to a second host service). Currently
	// ships as a heartbeat — per-patient FindApproachingThresholds /
	// DetectStaleEGFR invocation is a Phase 6 follow-up that needs a
	// KB-20 active-renal-patient endpoint and a small orchestrator.
	batchScheduler := services.NewBatchScheduler(logger)
	renalAnticipatoryJob := services.NewRenalAnticipatoryBatch(nil, logger)
	batchScheduler.Register(renalAnticipatoryJob)

	// Phase 6 P6-1: InertiaWeeklyBatch registered as the second KB-23
	// batch consumer. Proves the Phase 6 P6-5 scheduler can host two
	// jobs with different cadences (monthly + weekly). Heartbeat mode
	// (assembler + orchestrator nil) — per-patient assembly lands in
	// a Phase 6 follow-up when KB-20 exposes intervention timeline
	// and KB-26 exposes target status over HTTP.
	inertiaWeeklyJob := services.NewInertiaWeeklyBatch(nil, nil, nil, logger)
	batchScheduler.Register(inertiaWeeklyJob)

	batchCtx, batchCancel := context.WithCancel(context.Background())
	defer batchCancel()
	go batchScheduler.StartLoop(batchCtx, 1*time.Hour)
	logger.Info("KB-23 batch scheduler started",
		zap.String("registered_jobs", "renal_anticipatory_monthly + inertia_weekly"))

	// 11. Log startup banner
	fmt.Printf(`
========================================
KB-23 Decision Cards Engine
========================================
Service:     kb-23-decision-cards
Port:        %s
Environment: %s
Templates:   %d loaded
Database:    connected
Redis:       connected
========================================

API Endpoints:
  Card Management:
    POST   /api/v1/cards                       Create decision card
    GET    /api/v1/cards                       List decision cards
    GET    /api/v1/cards/:id                   Get decision card
    PUT    /api/v1/cards/:id                   Update decision card
    DELETE /api/v1/cards/:id                   Delete decision card

  Card Rendering:
    POST   /api/v1/cards/:id/render            Render card for context
    POST   /api/v1/cards/batch-render          Batch render cards

  Templates:
    GET    /api/v1/templates                   List all templates
    GET    /api/v1/templates/:template_id      Get template detail

  Calibration:
    POST   /api/v1/calibration/feedback        Adjudication feedback
    GET    /api/v1/calibration/status/:id      Concordance metrics

  Infrastructure:
    GET    /health                             Health check
    GET    /readiness                          Readiness probe
    GET    /metrics                            Prometheus metrics
    POST   /internal/templates/reload          Hot-reload templates
========================================
`, cfg.Port, cfg.Environment, templatesLoaded)

	// 12. Start HTTP server
	httpServer := &http.Server{
		Addr:         cfg.GetAddr(),
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

	logger.Info("KB-23 Decision Cards Engine started successfully",
		zap.String("port", cfg.Port),
		zap.Int("templates_loaded", templatesLoaded),
		zap.String("environment", cfg.Environment),
	)

	// 13. Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	logger.Info("Shutting down KB-23 Decision Cards Engine...")

	if priorityConsumer != nil {
		priorityConsumer.Stop()
		logger.Info("Priority signal consumer stopped")
	}

	// Phase 6 P6-5: stop the batch scheduler ticker and wait for any
	// in-flight RunOnce to finish before the process exits.
	batchCancel()
	batchScheduler.Drain()
	logger.Info("Batch scheduler drained")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info("HTTP server shutdown completed")
	}

	logger.Info("KB-23 Decision Cards Engine stopped")
}
