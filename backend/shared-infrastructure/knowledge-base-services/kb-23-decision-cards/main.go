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
	"kb-23-decision-cards/internal/models"
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
	// Phase 9 P9-C: persistent inertia verdict history table.
	// AutoMigrate'd here rather than in database.AutoMigrate()
	// because InertiaVerdictRow lives in the services package
	// and importing services from database would create a cycle.
	if err := db.DB.AutoMigrate(&services.InertiaVerdictRow{}); err != nil {
		logger.Fatal("Failed to auto-migrate InertiaVerdictRow", zap.Error(err))
	}
	// Phase 10 Gap 11 follow-up: clinical audit log table.
	if err := db.DB.AutoMigrate(&services.ClinicalAuditEntry{}); err != nil {
		logger.Fatal("Failed to auto-migrate ClinicalAuditEntry", zap.Error(err))
	}
	// Gap 21: outcome ingestion + consolidated alert record.
	if err := db.DB.AutoMigrate(&models.OutcomeRecord{}, &models.ConsolidatedAlertRecord{}); err != nil {
		logger.Fatal("Failed to auto-migrate Gap 21 models", zap.Error(err))
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
			server.TemplateLoader(), // Phase 7 P7-A: renal card templates
			server.MetricsCollector(),
			logger,
		)
		priorityConsumer = services.NewPrioritySignalConsumer(brokers, logger)

		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		defer consumerCancel()

		priorityConsumer.Start(consumerCtx, priorityHandler.Handle)
		logger.Info("KB-23 priority signal consumer started")
	}

	// 10c. KB-23 BatchScheduler + RenalAnticipatoryBatch.
	//
	// Phase 6 P6-5 shipped this as a heartbeat — the scheduler abstraction
	// proof. Phase 7 P7-C now wires the real orchestrator + real KB-20
	// client so the monthly job persists DecisionCards for every detected
	// approaching-threshold alert and stale-eGFR surveillance gap.
	//
	// The formulary is re-loaded here (cheap YAML parse) rather than
	// threaded through from the Kafka branch above, so the batch works
	// even when KB23_KAFKA_ENABLED is false.
	batchScheduler := services.NewBatchScheduler(logger)

	var renalAnticipatoryJob *services.RenalAnticipatoryBatch
	if formulary, ferr := services.LoadRenalFormulary(cfg.TemplatesDir, cfg.Market); ferr == nil {
		orchestrator := services.NewRenalAnticipatoryOrchestrator(
			server.KB20Client(),
			formulary,
			logger,
		)
		renalLister := services.NewKB20RenalActivePatientLister(server.KB20Client())
		renalAnticipatoryJob = services.NewRenalAnticipatoryBatch(
			renalLister,
			orchestrator,
			server.TemplateLoader(),
			server.Database(),
			server.MCUGateCache(),
			server.KB19Publisher(),
			server.MetricsCollector(),
			logger,
		)
	} else {
		logger.Warn("renal formulary load failed; renal anticipatory batch in heartbeat mode",
			zap.Error(ferr))
		renalAnticipatoryJob = services.NewRenalAnticipatoryBatch(nil, nil, nil, nil, nil, nil, nil, logger)
	}
	batchScheduler.Register(renalAnticipatoryJob)

	// Phase 7 P7-D: InertiaWeeklyBatch wired with real orchestrator +
	// assembler + verdict history. KB-20 intervention-timeline endpoint
	// and KB-26 target-status endpoint are now live, so the assembler
	// can fetch everything it needs. In-memory verdict history ships
	// in this phase; a PostgreSQL-backed store is a Phase 8 follow-up.
	//
	// The active-patient lister reuses the same renal-active wrapper
	// as P7-C: patients on at least one renal-sensitive medication
	// are the initial inertia population. A broader
	// "clinically-active" lister is a future refinement.
	// Phase 9 P9-C: persistent Postgres store replaces the Phase 7
	// in-memory store. Dampening now survives service restart — no
	// more "first post-deployment run skips dampening" gap.
	inertiaHistory := services.NewPostgresInertiaHistory(db.DB, logger)
	kb26Client := services.NewKB26Client(cfg, server.MetricsCollector(), logger)
	// Phase 7 P7-E Milestone 2: kb26Client now doubles as the
	// InertiaCGMLatestFetcher — the assembler prefers CGM TIR over
	// HbA1c when a recent CGM period report is available.
	inertiaAssembler := services.NewInertiaInputAssembler(
		server.KB20Client(),
		server.KB20Client(),
		kb26Client,
		kb26Client,
		logger,
	)
	inertiaOrchestrator := services.NewInertiaOrchestrator(
		inertiaHistory,
		server.TemplateLoader(),
		server.Database(),
		server.MCUGateCache(),
		server.KB19Publisher(),
		server.MetricsCollector(),
		logger,
	)
	inertiaActiveLister := services.NewKB20RenalActivePatientLister(server.KB20Client())
	inertiaWeeklyJob := services.NewInertiaWeeklyBatch(
		services.NewRenalListerAsInertiaLister(inertiaActiveLister),
		inertiaAssembler,
		inertiaOrchestrator,
		logger,
	)
	batchScheduler.Register(inertiaWeeklyJob)

	// Phase 9 P9-B: monitoring engagement weekly batch. Fires on
	// Wednesdays at 04:00 UTC (offset from the Sunday 03:00 inertia
	// batch to spread load). Calls KB-20's monitoring-lapsed
	// endpoint and generates MONITORING_LAPSED cards for patients
	// who stopped home BP monitoring.
	monitoringBatch := services.NewMonitoringEngagementBatch(
		cfg,
		server.TemplateLoader(),
		server.Database(),
		server.MCUGateCache(),
		server.KB19Publisher(),
		server.MetricsCollector(),
		logger,
	)
	batchScheduler.Register(monitoringBatch)

	batchCtx, batchCancel := context.WithCancel(context.Background())
	defer batchCancel()
	go batchScheduler.StartLoop(batchCtx, 1*time.Hour)
	logger.Info("KB-23 batch scheduler started",
		zap.String("registered_jobs", "renal_anticipatory_monthly + inertia_weekly + monitoring_engagement_weekly"))

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
