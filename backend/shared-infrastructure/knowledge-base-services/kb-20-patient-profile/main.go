package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-patient-profile/internal/api"
	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/clients"
	"kb-patient-profile/internal/config"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/fhir"
	"kb-patient-profile/internal/metrics"
	"kb-patient-profile/internal/models"
	"kb-patient-profile/internal/services"
	"kb-patient-profile/internal/storage"
	"kb-patient-profile/pkg/resilience"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

func main() {
	// Initialize structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting KB-20 Patient Profile & Contextual State Engine...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database (auto-migrates all KB-20 models)
	logger.Info("Connecting to database...")
	db, err := database.NewConnection(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("Database connected and migrations completed")

	// Initialize cache
	logger.Info("Connecting to Redis cache...")
	cacheClient, err := cache.NewClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to cache", zap.Error(err))
	}
	defer cacheClient.Close()

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Initialize event bus with transactional outbox (G-03 remediation)
	eventBus := services.NewEventBus(db.DB, logger, metricsCollector)
	eventBus.StartPoller(context.Background())
	defer eventBus.Stop()

	// Kafka outbox relay (feature-flagged) — created here, started after projectionService
	// is available so A1 personalised targets can be enriched onto lab events.
	var kafkaRelay *services.KafkaOutboxRelay
	var kafkaWriter *services.KafkaGoWriter
	if os.Getenv("KAFKA_RELAY_ENABLED") == "true" {
		brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
		kafkaWriter = services.NewKafkaGoWriter(brokers, "kb20-outbox-relay")
		kafkaRelay = services.NewKafkaOutboxRelay(db.DB, kafkaWriter, metricsCollector, logger)
		defer func() {
			if kafkaRelay != nil { kafkaRelay.Stop() }
			if kafkaWriter != nil { kafkaWriter.Close() }
		}()
	}

	// Initialize services
	logger.Info("Initializing KB-20 services...")

	patientService := services.NewPatientService(db, cacheClient, logger)
	patientService.SetEventBus(eventBus)

	// Wire KB-21 adherence client into PatientService (Task 19: V-MCU adherence wiring)
	kb21AdherenceClient := clients.NewKB21Client(cfg.KB21, logger)
	patientService.SetKB21Client(kb21AdherenceClient)
	labService := services.NewLabService(db, cacheClient, logger, metricsCollector, eventBus)
	// Phase 8 P8-6: wire the safety event recorder into the lab
	// service so every safety alert publish path also persists the
	// event to the safety_events audit table. The summary-context
	// endpoint's ConfounderFlags query reads from this table to
	// derive IsAcuteIll / HasRecentTransfusion / HasRecentHypoglycaemia.
	labService.SetSafetyRecorder(services.NewSafetyEventRecorder(db.DB, logger))

	medicationService := services.NewMedicationService(db, cacheClient, logger, eventBus)
	adrService := services.NewADRService(db, logger)
	pipelineService := services.NewPipelineService(db, logger, adrService)
	cmRegistry := services.NewCMRegistry(db, logger)
	stratumEngine := services.NewStratumEngine(db, cacheClient, logger, metricsCollector, cmRegistry, eventBus)

	// Initialize LOINC registry (validates codes against KB-7 Terminology Service)
	kb7Client := fhir.NewKB7Client(cfg.KB7, logger)
	kb7Adapter := &kb7ConceptAdapter{client: kb7Client}
	loincRegistry := services.NewLOINCRegistry(kb7Adapter, logger)
	loincRegistry.Initialize(context.Background())
	logger.Info("LOINC registry status", zap.String("summary", loincRegistry.VerificationSummary()))

	// Initialize KB-21 client for festival calendar (P4 perturbation)
	kb21Client := fhir.NewKB21Client(cfg.KB21, logger)
	kb21Adapter := &kb21FestivalAdapter{client: kb21Client}

	projectionService := services.NewProjectionService(db, cacheClient, logger, loincRegistry, cfg.PREVENT, kb21Adapter)

	// A1: Wire personalised target enrichment into Kafka outbox relay and start it
	if kafkaRelay != nil {
		kafkaRelay.SetTargetProvider(projectionService)
		kafkaRelay.Start(context.Background())
		logger.Info("Kafka outbox relay started with A1 personalised target enrichment")
	}

	protocolRegistry := services.NewProtocolRegistry()
	protocolService := services.NewProtocolService(db, protocolRegistry, eventBus, logger)

	// G-6: Attach KB-25 Lifestyle Knowledge Graph client for protocol safety checks
	// and post-transition projections.
	kb25Client := clients.NewKB25Client(cfg.KB25, logger)
	protocolService.SetKB25Client(kb25Client)

	// G-5 (MRI): Create KB-26 Metabolic Digital Twin client.
	// The caller that populates TrajectoryInput uses this client to fetch MRI data
	// (score and 14-day delta) for the MRI forcing rules (Spec §7).
	// The TrajectoryEngine itself remains a pure computation engine.
	//
	// Phase 8 P8-3: the same KB-26 client also backs the CGM status
	// fetch used by the summary-context handler. A small adapter
	// translates the clients.CGMPeriodReportSnapshot return into
	// the services.CGMStatusSnapshot the summary-context service
	// consumes — keeps the services package free of clients imports
	// and the clients package free of services imports.
	kb26Client := clients.NewKB26Client(cfg.KB26.BaseURL, logger)
	// Phase 10 P10-D: wire Prometheus metrics into KB-26 circuit
	// breaker so state transitions are observable in Grafana.
	if metricsCollector.CircuitBreakerTransitions != nil {
		kb26Client.SetOnStateChange(func(name string, from, to resilience.State) {
			metricsCollector.CircuitBreakerTransitions.WithLabelValues(name, from.String(), to.String()).Inc()
		})
	}

	// Initialize HTTP server with all services
	logger.Info("Initializing HTTP server...")
	httpServer := api.NewServer(
		cfg,
		db,
		cacheClient,
		metricsCollector,
		patientService,
		labService,
		medicationService,
		stratumEngine,
		cmRegistry,
		adrService,
		pipelineService,
		projectionService,
		loincRegistry,
		protocolService,
		protocolRegistry,
		eventBus,
		logger,
	)

	// Phase 8 P8-3: wire the KB-26 CGM status fetcher into the
	// summary-context handler. The adapter wraps the clients-layer
	// KB26Client into the services-layer CGMStatusFetcher interface,
	// keeping the two packages decoupled. Every call to
	// GET /patient/:id/summary-context now produces real CGM fields
	// when the patient has a recent cgm_period_reports row, or
	// HasCGM=false + clean fallback when they don't.
	httpServer.SetKB26CGMFetcher(newKB26CGMFetcherAdapter(kb26Client, logger))

	// V2 substrate routes (milestone 1B-β.1). Mounted at /v2 alongside the
	// existing /api/v1 routes; opt-in and non-breaking.
	if sqlDB, err := db.DB.DB(); err != nil {
		logger.Warn("v2 substrate routes not registered: failed to acquire *sql.DB",
			zap.Error(err))
	} else {
		v2Store := storage.NewV2SubstrateStoreWithDB(sqlDB)
		// Wire delta-on-write BaselineProvider via the persistent baseline
		// store (Layer 2 substrate plan, Wave 2.1 — replaces the in-memory
		// stub). The BaselineStore reads/writes the baseline_state table
		// (migration 013); PersistentBaselineProvider serves the read side
		// of delta.BaselineProvider; UpsertObservation writes the row in
		// the same transaction as the observation INSERT so persisted
		// state is always consistent.
		//
		// TODO(kb-26-acute-repository): once kb-26's AcuteRepository goes
		// live with a network API, replace this local PersistentBaseline-
		// Provider with a kb-26 client. The transactional recompute path
		// (BaselineStore.RecomputeAndUpsertTx) stays here because it must
		// be co-located with the observations table.
		// Wave 2.2: per-observation-type baseline configuration. The
		// BaselineConfigStore reads/writes the baseline_configs table
		// (migration 014) and parameterises the recompute (window days,
		// morning-only filter, velocity flagging, etc.) per Layer 2 §2.2.
		// When no row matches a vital type, the recompute falls back to
		// delta.DefaultConfig (14-day window, no filters) — i.e. exactly
		// the Wave 2.1 behaviour.
		baselineConfigStore := storage.NewBaselineConfigStore(sqlDB)
		baselineStore := storage.NewBaselineStore(sqlDB).WithConfigStore(baselineConfigStore)
		v2Store.SetBaselineStore(baselineStore)
		v2Store.SetBaselineProvider(
			delta.NewPersistentBaselineProvider(baselineStore).WithConfigStore(baselineConfigStore),
		)
		v2Handlers := api.NewV2SubstrateHandlers(v2Store)
		v2Handlers.RegisterRoutes(httpServer.Router.Group("/v2"))
		logger.Info("v2 substrate routes registered at /v2 (residents, persons, roles, medicine_uses, observations)")

		// Wave 1R.3: identity matching service (Layer 2 §3.3). Mounted at
		// /v2/identity so the IdentityMatcher ships non-breakingly. The
		// IdentityStore shares sqlDB with V2SubstrateStore so EvidenceTrace
		// audit nodes are written through the same connection pool.
		identityStore := storage.NewIdentityStore(sqlDB, v2Store)
		identityHandlers := api.NewIdentityHandlers(identityStore)
		identityHandlers.RegisterRoutes(httpServer.Router.Group("/v2/identity"))
		logger.Info("v2 identity matching routes registered at /v2/identity (match, review-queue, review/:id/resolve)")

		// Wave 2.3: active-concern lifecycle. Mounts CRUD + lifecycle
		// endpoints under /v2 (POST /residents/:id/active-concerns,
		// GET /residents/:id/active-concerns, PATCH /active-concerns/:id,
		// GET /active-concerns/expiring). The same store also provides
		// the ConcernTriggerLookup the engine consumes; that wiring is
		// done in the cron loop / event-write path once Layer 3 calls
		// it. See shared/v2_substrate/clinical_state/active_concerns.go.
		activeConcernStore := storage.NewActiveConcernStore(sqlDB)
		activeConcernHandlers := api.NewActiveConcernHandlers(activeConcernStore)
		activeConcernHandlers.RegisterRoutes(httpServer.Router.Group("/v2"))

		// Wave 2.4: care-intensity transitions. Mounts CRUD endpoints
		// under /v2 (POST /residents/:id/care-intensity,
		// GET /residents/:id/care-intensity/current,
		// GET /residents/:id/care-intensity/history). The store wraps
		// the pure clinical_state.CareIntensityEngine and persists the
		// transition Event + one EvidenceTrace node per cascade hint
		// alongside the new care_intensity_history row in a single
		// transaction. See migration 016 for the schema +
		// care_intensity_current view, and
		// shared/v2_substrate/clinical_state/care_intensity_engine.go
		// for the cascade rule table.
		careIntensityStore := storage.NewCareIntensityStore(sqlDB, v2Store)
		careIntensityHandlers := api.NewCareIntensityHandlers(careIntensityStore)
		careIntensityHandlers.RegisterRoutes(httpServer.Router.Group("/v2"))
		logger.Info("v2 care-intensity routes registered at /v2 (residents/:id/care-intensity, /current, /history)")

		// Wave 2.5: per-domain capacity assessments (Layer 2 §2.5).
		// Mounts CRUD endpoints under /v2 (POST /residents/:id/capacity,
		// GET /residents/:id/capacity/current,
		// GET /residents/:id/capacity/current/:domain,
		// GET /residents/:id/capacity/history/:domain). The store writes
		// the assessment + an EvidenceTrace node for every call; when
		// Outcome=impaired AND Domain=medical_decisions it additionally
		// emits a capacity_change Event and tags the EvidenceTrace node
		// with state_machine=Consent. Layer 3's Consent state machine
		// consumes that Event to re-evaluate consent paths. See
		// migration 017 for the schema + capacity_current view.
		capacityStore := storage.NewCapacityAssessmentStore(sqlDB, v2Store)
		capacityHandlers := api.NewCapacityHandlers(capacityStore)
		capacityHandlers.RegisterRoutes(httpServer.Router.Group("/v2"))
		logger.Info("v2 capacity routes registered at /v2 (residents/:id/capacity, /current[/:domain], /history/:domain)")

		// Wave 2.6: CFS / AKPS / DBI / ACB scoring instruments
		// (Layer 2 §2.4 / §2.6). CFS and AKPS are clinician-entered
		// capture endpoints; DBI and ACB are computed from the
		// resident's active MedicineUse list using the dbi_drug_weights
		// + acb_drug_weights seed tables (migration 018).
		//
		// CFS>=7 or AKPS<=40 surfaces a CareIntensityReviewHint via the
		// EvidenceTrace graph (state_machine=ClinicalState,
		// state_change_type=care_intensity_review_suggested). The
		// substrate NEVER auto-transitions care intensity from a score.
		//
		// The DBI/ACB recompute is wired through the v2 MedicineUse
		// upsert path via SetOnMedicineUseChanged below: every
		// MedicineUse insert/update/end synchronously triggers
		// RecomputeDrugBurden. Recompute is best-effort — errors are
		// logged and swallowed so the underlying MedicineUse write
		// always commits. TODO(production): move to outbox-driven
		// async with per-resident coalescing.
		scoringStore := storage.NewScoringStore(sqlDB, v2Store)
		scoringHandlers := api.NewScoringHandlers(scoringStore)
		scoringHandlers.RegisterRoutes(httpServer.Router.Group("/v2"))
		v2Handlers.SetOnMedicineUseChanged(func(ctx context.Context, residentRef uuid.UUID) {
			if _, err := scoringStore.RecomputeDrugBurden(ctx, residentRef); err != nil {
				logger.Warn("DBI/ACB recompute failed (best-effort; underlying MedicineUse write succeeded)",
					zap.String("resident_ref", residentRef.String()),
					zap.Error(err))
			}
		})
		logger.Info("v2 scoring routes registered at /v2 (residents/:id/{cfs,akps,scores/current,{cfs,akps,dbi,acb}/history}); DBI/ACB recompute wired to MedicineUse changes")
		// Baseline-exclusion wiring: BaselineStore.buildObsQuery now
		// joins against active_concerns directly via SQL when a config's
		// ExcludeDuringActiveConcerns list is non-empty. No additional
		// wiring needed here — the join uses the same DB connection.
		// Closes the wave-2.2 TODO in baseline_store.go.
		logger.Info("v2 active-concern routes registered at /v2 (residents/:id/active-concerns, active-concerns/expiring)")
	}

	// Start HTTP server
	go func() {
		port := cfg.Server.Port
		if port == "" {
			port = "8131"
		}
		logger.Info("Starting HTTP server", zap.String("port", port))
		if err := httpServer.Router.Run(":" + port); err != nil {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// FHIR Store integration (conditional)
	if cfg.FHIR.Enabled {
		logger.Info("FHIR sync enabled — initializing Google Healthcare FHIR client...")
		fhirClient, err := fhir.NewFHIRClient(cfg.FHIR, logger)
		if err != nil {
			logger.Error("Failed to initialize FHIR client — service continues without FHIR sync",
				zap.Error(err))
		} else {
			kb7Client := fhir.NewKB7Client(cfg.KB7, logger)

			// Phase 7 P7-B: wire CKMRecomputationService so the sync
			// worker can re-run CKM staging on every new LVEF / NT-proBNP
			// / CAC observation. Reuses the existing eventBus — the
			// publisher writes both the profile state change and the
			// CKM_STAGE_TRANSITION outbox event inside the same flow.
			ckmPublisher := services.NewCKMTransitionPublisher(db.DB, eventBus, logger)
			ckmRecompute := services.NewCKMRecomputationService(db.DB, ckmPublisher, logger)

			// Start FHIR→KB-20 sync worker
			syncWorker := fhir.NewSyncWorker(fhirClient, kb7Client, db.DB, logger, eventBus, ckmRecompute)
			syncWorker.Start(context.Background())
			defer syncWorker.Stop()
			logger.Info("FHIR sync worker started")

			// Start KB-20→FHIR write-back publisher
			if cfg.FHIR.WriteBack {
				publisher := fhir.NewPublisher(fhirClient, db.DB, logger)
				eventBus.Subscribe(models.EventMedicationThresholdCrossed, publisher.HandleThresholdCrossed)
				eventBus.Subscribe(models.EventStratumChange, publisher.HandleStratumChange)
				// Phase 10 Gap 9: KB-23 decision cards → FHIR CommunicationRequest
				// write-back for MHR (Australia) and ABDM (India) compliance.
				eventBus.Subscribe(models.EventDecisionCardGenerated, publisher.HandleDecisionCard)
				logger.Info("FHIR write-back publisher registered for MEDICATION_THRESHOLD_CROSSED, STRATUM_CHANGE, and DECISION_CARD_GENERATED events")
			}
		}
	} else {
		logger.Info("FHIR sync disabled (set FHIR_SYNC_ENABLED=true to enable)")
	}

	// Health checks
	logger.Info("Performing health checks...")
	if err := db.HealthCheck(); err != nil {
		logger.Warn("Database health check failed", zap.Error(err))
	} else {
		logger.Info("Database health check passed")
	}
	if err := cacheClient.HealthCheck(); err != nil {
		logger.Warn("Cache health check failed", zap.Error(err))
	} else {
		logger.Info("Cache health check passed")
	}

	logger.Info("KB-20 Patient Profile & Contextual State Engine started successfully")

	httpPort := cfg.Server.Port
	if httpPort == "" {
		httpPort = "8131"
	}

	fmt.Printf(`
========================================
KB-20 Patient Profile & Contextual State Engine
========================================
Service: kb-20-patient-profile
HTTP Port: %s
Version: 1.0.0
Environment: %s
========================================

REST Endpoints:
- Health:            GET  /health
- Metrics:           GET  /metrics
- Create Patient:    POST /api/v1/patient
- Get Profile:       GET  /api/v1/patient/:id/profile
- Add Lab:           POST /api/v1/patient/:id/labs
- Get Labs:          GET  /api/v1/patient/:id/labs
- eGFR Trajectory:   GET  /api/v1/patient/:id/labs/egfr
- Add Medication:    POST /api/v1/patient/:id/medications
- Update Medication: PUT  /api/v1/patient/:id/medications/:med_id
- Get Medications:   GET  /api/v1/patient/:id/medications
- Get Stratum:       GET  /api/v1/patient/:id/stratum/:node_id
- CM Registry:       GET  /api/v1/modifiers/registry/:node_id
- ADR Profiles:      GET  /api/v1/adr/profiles/:drug_class
- Batch Modifiers:   POST /api/v1/pipeline/modifiers
- Batch ADR:         POST /api/v1/pipeline/adr-profiles
- Channel B Inputs:  GET  /api/v1/patient/:id/channel-b-inputs
- Channel C Inputs:  GET  /api/v1/patient/:id/channel-c-inputs
- Bust Proj Cache:   DEL  /api/v1/patient/:id/projections/cache
- LOINC Registry:    GET  /api/v1/loinc/registry

RED Findings Incorporated:
  F-01: FDC decomposition (fdc_components field)
  F-03: MEDICATION_THRESHOLD_CROSSED events + ckd_substage
  F-05: Lab plausibility validation (ACCEPTED/FLAGGED/REJECTED)
========================================
`, httpPort, cfg.Environment)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KB-20 Patient Profile service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	select {
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded")
	case <-time.After(2 * time.Second):
		logger.Info("HTTP server shutdown completed")
	}

	logger.Info("KB-20 service stopped successfully")
}

// kb7ConceptAdapter adapts fhir.KB7Client to the services.KB7ConceptLookup interface,
// breaking the import cycle between services ↔ fhir packages.
type kb7ConceptAdapter struct {
	client *fhir.KB7Client
}

func (a *kb7ConceptAdapter) LookupConcept(loincCode string) (*services.KB7ConceptResult, error) {
	concept, err := a.client.LookupConcept(loincCode)
	if err != nil {
		return nil, err
	}
	if concept == nil {
		return nil, nil
	}
	return &services.KB7ConceptResult{
		Code:    concept.Code,
		Display: concept.Display,
	}, nil
}

// kb21FestivalAdapter adapts fhir.KB21Client to the services.KB21FestivalLookup interface,
// breaking the import cycle between services ↔ fhir packages.
type kb21FestivalAdapter struct {
	client *fhir.KB21Client
}

func (a *kb21FestivalAdapter) GetFestivalStatus(region string) *services.FestivalStatusResult {
	status := a.client.GetFestivalStatus(region)
	if status == nil {
		return nil
	}
	return &services.FestivalStatusResult{
		Active:      status.Active,
		FastingType: fhir.MapFestivalToPerturbationFastingType(status.FastingType),
		End:         status.End,
	}
}

// kb26CGMFetcherAdapter adapts clients.KB26Client to the
// services.CGMStatusFetcher interface, keeping the services package
// free of imports from the clients package and vice versa. Phase 8 P8-3.
//
// The adapter performs two jobs:
//   1. Translates the HTTP shape (clients.CGMPeriodReportSnapshot)
//      into the service-layer DTO (services.CGMStatusSnapshot).
//   2. Converts (nil, nil) from the client (patient has no CGM data,
//      a 404 from KB-26) into a services-layer "no CGM status" return
//      with HasCGM=false.
type kb26CGMFetcherAdapter struct {
	client *clients.KB26Client
	logger *zap.Logger
}

// newKB26CGMFetcherAdapter constructs the adapter. Returning a pointer
// via this helper matches the convention used by kb21FestivalAdapter.
func newKB26CGMFetcherAdapter(client *clients.KB26Client, logger *zap.Logger) *kb26CGMFetcherAdapter {
	return &kb26CGMFetcherAdapter{client: client, logger: logger}
}

// FetchLatestCGMStatus implements services.CGMStatusFetcher.
func (a *kb26CGMFetcherAdapter) FetchLatestCGMStatus(ctx context.Context, patientID string) (*services.CGMStatusSnapshot, error) {
	snap, err := a.client.GetLatestCGMStatus(ctx, patientID)
	if err != nil {
		return nil, err
	}
	// nil response from the client means 404 from KB-26 — the patient
	// has no CGM data. Return a non-nil snapshot with HasCGM=false so
	// the summary-context service populates the CGM fields cleanly
	// rather than leaving them nil and tripping downstream nil checks.
	if snap == nil {
		return &services.CGMStatusSnapshot{HasCGM: false}, nil
	}
	return &services.CGMStatusSnapshot{
		HasCGM:     true,
		TIRPct:     snap.TIRPct,
		GRIZone:    snap.GRIZone,
		ReportedAt: snap.PeriodEnd,
	}, nil
}
