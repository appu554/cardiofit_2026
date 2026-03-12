// Package factstore provides the SPL FactStore Pipeline Runner.
//
// This implements the 9-phase pipeline from SPL_FactStore_Execution_Runbook.docx:
//
//	Phase A: Verify Spine (drug_master, source_documents, source_sections)
//	Phase B: Select Scope (targeted drugs)
//	Phase C: SPL Acquisition (fetch from DailyMed)
//	Phase D: LOINC Section Routing (authority routing)
//	Phase E-F: Table Extraction & Rule Generation
//	Phase G: DraftFact Creation (6 fact types)
//	Phase H: Governance Handoff (KB-0)
//	Phase I: KB Projection (to downstream KBs)
//
// DESIGN PRINCIPLE: "Parse once, extract to multiple KBs"
package factstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/datasources/kdigo"
	"github.com/cardiofit/shared/datasources/rxnav"
	"github.com/cardiofit/shared/governance/routing"
	"github.com/cardiofit/shared/terminology"
)

// =============================================================================
// PIPELINE RUNNER CONFIGURATION
// =============================================================================

// RunnerConfig configures the SPL Pipeline Runner
type RunnerConfig struct {
	// Phase A: Spine verification
	VerifySpine bool `json:"verifySpine"`

	// Phase B: Scope selection
	TargetDrugs []TargetDrug `json:"targetDrugs"`

	// Phase C: SPL Acquisition
	FDABaseURL    string        `json:"fdaBaseUrl"`
	RxNavURL      string        `json:"rxnavUrl"`
	FetchTimeout  time.Duration `json:"fetchTimeout"`
	RateLimitMs   int           `json:"rateLimitMs"`

	// Phase D: LOINC Routing
	EnableAuthorityRouting bool `json:"enableAuthorityRouting"`

	// Phase E-F: Table Extraction
	MinTableConfidence float64 `json:"minTableConfidence"`

	// Phase G: Fact Creation
	FactTypes []FactType `json:"factTypes"` // Which fact types to create

	// Phase H: Governance
	AutoApproveThreshold float64 `json:"autoApproveThreshold"`
	ReviewThreshold      float64 `json:"reviewThreshold"`
	EnableLLM            bool    `json:"enableLlm"` // Default: false for deterministic runs

	// Phase I: KB Projection
	EnableKBProjection bool     `json:"enableKbProjection"`
	TargetKBs          []string `json:"targetKbs"` // ["KB-1", "KB-4", "KB-5", "KB-6", "KB-16"]

	// Concurrency
	MaxConcurrent int `json:"maxConcurrent"`
	BatchSize     int `json:"batchSize"`

	// Phase 3 Issues 2+3: MedDRA adverse event normalization
	MRCONSOPath  string `json:"mrconsoPath"`  // Path to UMLS MRCONSO.RRF file for MedDRA loading
	MRHIERPath   string `json:"mrhierPath"`   // Path to UMLS MRHIER.RRF file for PT→SOC hierarchy enrichment
	MedDRADBPath      string `json:"meddraDbPath"`      // Path to persist MedDRA SQLite DB (optional, in-memory if empty)
	MedDRAValueSetPath string `json:"meddraValueSetPath"` // Path to KB7 MedDRA ValueSet JSON (alternative to MRCONSO)

	// Phase 3c: LLM Fallback
	AnthropicAPIKey string  `json:"anthropicApiKey"` // Claude API key (or ANTHROPIC_API_KEY env)
	LLMBudgetUSD    float64 `json:"llmBudgetUsd"`    // per-run budget cap (default $50)

	// Phase 3d: Organ impairment enrichment (KDIGO only — CPIC removed, it's pharmacogenomics not OI)
	IncludeOrganImpairment bool `json:"includeOrganImpairment"` // Enable organ impairment pass

	// KDIGO rules from MCP-RAG atomiser (V2: JSON file, not PDF directory)
	KDIGORulesPath string `json:"kdigoRulesPath"` // Path to KDIGO rules JSON file
}

// TargetDrug represents a drug to process
type TargetDrug struct {
	RxCUI    string `json:"rxcui"`
	DrugName string `json:"drugName"`
	Reason   string `json:"reason"` // Why this drug was selected
}

// DefaultRunnerConfig returns sensible defaults matching the runbook
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		VerifySpine:            true,
		FDABaseURL:             "https://dailymed.nlm.nih.gov/dailymed", // Base URL (search adds /services/v2)
		RxNavURL:               "http://localhost:4000/REST", // rxnav-in-a-box
		FetchTimeout:           60 * time.Second,
		RateLimitMs:            350, // ~3 req/sec to respect DailyMed limits
		EnableAuthorityRouting: true,
		MinTableConfidence:     0.70,
		FactTypes: []FactType{
			FactTypeOrganImpairment,
			FactTypeSafetySignal,
			FactTypeReproductiveSafety,
			FactTypeInteraction,
			FactTypeFormulary,
			FactTypeLabReference,
		},
		AutoApproveThreshold: 2.0, // DISABLED — all facts route to PENDING_REVIEW for pharmacist review. No auto-approval.
		ReviewThreshold:      0.65,
		EnableLLM:            false, // Deterministic run by default
		EnableKBProjection:   true,
		TargetKBs:            []string{"KB-1", "KB-4", "KB-5", "KB-6", "KB-16"},
		MaxConcurrent:        5,
		BatchSize:            10,
	}
}

// InitialScopeDrugs returns the 10 high-value drugs from the runbook.
// RxCUIs are intentionally omitted — Phase B resolves them at runtime via RxNav
// to prevent stale/wrong hardcoded RxCUI values from causing FK constraint failures.
func InitialScopeDrugs() []TargetDrug {
	return []TargetDrug{
		{DrugName: "Metformin", Reason: "Renal dosing tables"},
		{DrugName: "Warfarin", Reason: "DDI, genetic dosing"},
		{DrugName: "Simvastatin", Reason: "CPIC, muscle toxicity"},
		{DrugName: "Apixaban", Reason: "Renal + hepatic tables"},
		{DrugName: "Dapagliflozin", Reason: "eGFR thresholds"},
		{DrugName: "Digoxin", Reason: "Narrow therapeutic index"},
		{DrugName: "Spironolactone", Reason: "K+ monitoring"},
		{DrugName: "Vancomycin", Reason: "Renal dosing, TDM"},
		{DrugName: "Lithium Carbonate", Reason: "Renal, pregnancy"},
		{DrugName: "Amiodarone", Reason: "QT, DDIs"},
	}
}

// =============================================================================
// PIPELINE RUNNER
// =============================================================================

// PipelineRunner orchestrates the 9-phase SPL to FactStore pipeline
type PipelineRunner struct {
	mu     sync.RWMutex
	config RunnerConfig
	log    *logrus.Entry

	// Dependencies
	repo            *Repository
	splFetcher      *dailymed.SPLFetcher
	sectionRouter   *dailymed.SectionRouter
	tableClassifier *dailymed.TableClassifier
	authorityRouter *routing.AuthorityRouter
	rxnavClient     *rxnav.Client
	drugNormalizer  terminology.DrugNormalizer
	pipeline        *Pipeline

	// State
	currentRun *PipelineRun
}

// PipelineRun tracks a single pipeline execution
type PipelineRun struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Status    string    `json:"status"` // RUNNING, COMPLETED, FAILED

	// Phase tracking
	Phases map[string]*PhaseResult `json:"phases"`

	// Aggregated metrics
	Metrics RunMetrics `json:"metrics"`

	// Errors
	Errors []string `json:"errors,omitempty"`
}

// PhaseResult tracks a single phase's execution
type PhaseResult struct {
	Phase     string        `json:"phase"`
	Name      string        `json:"name"`
	StartedAt time.Time     `json:"startedAt"`
	EndedAt   *time.Time    `json:"endedAt,omitempty"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED, SKIPPED
	Message   string        `json:"message,omitempty"`
	Details   interface{}   `json:"details,omitempty"`
}

// RunMetrics aggregates metrics across all phases
type RunMetrics struct {
	// Phase A
	SpineVerified bool `json:"spineVerified"`

	// Phase B
	DrugsInScope int `json:"drugsInScope"`

	// Phase C
	SPLsFetched int `json:"splsFetched"`
	SPLsFailed  int `json:"splsFailed"`

	// Phase D
	SectionsRouted    int `json:"sectionsRouted"`
	AuthorityLookups  int `json:"authorityLookups"`
	TablesClassified  int `json:"tablesClassified"`

	// Phase E-F
	RulesGenerated int `json:"rulesGenerated"`

	// Phase G
	FactsCreated       int `json:"factsCreated"`
	FactsByType        map[string]int `json:"factsByType"`

	// Phase H
	FactsAutoApproved  int `json:"factsAutoApproved"`
	FactsPendingReview int `json:"factsPendingReview"`
	FactsRejected      int `json:"factsRejected"`

	// Phase I
	KBProjections map[string]int `json:"kbProjections"` // KB name -> count

	// Timing
	TotalDuration time.Duration `json:"totalDuration"`
}

// NewPipelineRunner creates a new pipeline runner with all dependencies
func NewPipelineRunner(
	config RunnerConfig,
	repo *Repository,
	log *logrus.Logger,
) *PipelineRunner {
	entry := log.WithField("component", "pipeline-runner")

	// Initialize SPL fetcher with cache
	splCache := dailymed.NewMemorySPLCache(24 * time.Hour)
	splFetcherConfig := dailymed.Config{
		BaseURL:            config.FDABaseURL,
		Timeout:            config.FetchTimeout,
		RateLimitPerSecond: 3, // ~3 req/sec to respect DailyMed limits
		CacheTTL:           24 * time.Hour,
	}
	splFetcher := dailymed.NewSPLFetcher(splFetcherConfig, splCache)

	// Initialize section router (includes table classifier)
	sectionRouter := dailymed.NewSectionRouter()

	// Initialize table classifier
	tableClassifier := dailymed.NewTableClassifier()

	// Initialize authority router
	authorityRouter := routing.NewAuthorityRouter()

	// Initialize RxNav client
	rxnavConfig := rxnav.LocalConfig()
	rxnavConfig.BaseURL = config.RxNavURL
	rxnavConfig.Logger = entry
	rxnavClient := rxnav.NewClient(rxnavConfig)

	// Phase 3 Issue 1 Fix: Create RxNorm normalizer for RxCUI validation/correction.
	// This ensures SPL documents with wrong/outdated RxCUIs are corrected before storage,
	// preventing FK constraint failures when projecting to clinical_facts.
	var drugNormalizer terminology.DrugNormalizer
	rxNormConfig := terminology.RxNormNormalizerConfig{
		RxNavConfig: rxnavConfig,
		Logger:      entry,
	}
	normalizer, err := terminology.NewRxNormNormalizer(rxNormConfig)
	if err != nil {
		entry.WithError(err).Warn("Failed to create drug normalizer, RxCUI validation disabled")
	} else {
		drugNormalizer = normalizer
		entry.Info("Drug normalizer initialized - RxCUI validation enabled (Issue 1 FK fix)")
	}

	// Phase 3 Issues 2+3: Create MedDRA normalizer for adverse event validation.
	// Three loading channels (in priority order):
	//   1. --mrconso: UMLS MRCONSO.RRF (full hierarchy, requires UMLS license)
	//   2. --meddra-valueset: KB7 FHIR ValueSet JSON (79K terms, zero-dependency)
	// Both populate the same SQLite schema, so the normalizer works identically.
	var aeNormalizer terminology.AdverseEventNormalizer
	var meddraDB *sql.DB // Hoisted: captures MedDRA SQLite DB for prose scanner creation
	if config.MRCONSOPath != "" || config.MedDRAValueSetPath != "" {
		meddraLoader, meddraErr := terminology.NewMedDRALoader(terminology.MedDRALoaderConfig{
			DBPath: config.MedDRADBPath, // Persistent SQLite or in-memory if empty
			Logger: entry,
		})
		if meddraErr != nil {
			entry.WithError(meddraErr).Warn("Failed to create MedDRA loader, adverse event normalization disabled")
		} else {
			var loadSuccess bool

			// Channel 1: UMLS MRCONSO.RRF (preferred — gives full hierarchy)
			if config.MRCONSOPath != "" {
				loadCtx, loadCancel := context.WithTimeout(context.Background(), 120*time.Second)
				defer loadCancel()
				if loadErr := meddraLoader.LoadFromMRCONSO(loadCtx, config.MRCONSOPath); loadErr != nil {
					entry.WithError(loadErr).Warn("Failed to load MRCONSO.RRF, trying ValueSet fallback")
				} else {
					loadSuccess = true
					entry.Info("MedDRA loaded from MRCONSO.RRF")

					// Enrich PT→SOC hierarchy from MRHIER.RRF (needed for biomarker filtering)
					if config.MRHIERPath != "" {
						hierCtx, hierCancel := context.WithTimeout(context.Background(), 600*time.Second)
						defer hierCancel()
						if hierErr := meddraLoader.LoadFromMRHIER(hierCtx, config.MRHIERPath, config.MRCONSOPath); hierErr != nil {
							entry.WithError(hierErr).Warn("Failed to load MRHIER.RRF, PT→SOC enrichment skipped (biomarker filter will not activate)")
						} else {
							entry.Info("MRHIER.RRF loaded — PT→SOC hierarchy enriched for biomarker filtering")
						}
					} else {
						entry.Warn("No MRHIER path configured — PT→SOC enrichment skipped (biomarker SOC filter inactive)")
					}
				}
			}

			// Channel 2: KB7 FHIR ValueSet JSON (fallback — term validation only, no hierarchy)
			if !loadSuccess && config.MedDRAValueSetPath != "" {
				loadCtx, loadCancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer loadCancel()
				if loadErr := meddraLoader.LoadFromValueSetJSON(loadCtx, config.MedDRAValueSetPath); loadErr != nil {
					entry.WithError(loadErr).Warn("Failed to load MedDRA ValueSet JSON, adverse event normalization disabled")
				} else {
					loadSuccess = true
					entry.WithField("source", config.MedDRAValueSetPath).Info("MedDRA loaded from KB7 FHIR ValueSet JSON (no SOC hierarchy)")
				}
			}

			if loadSuccess {
				meddraDB = meddraLoader.DB() // Capture for prose scanner creation below
				normalizer, normErr := terminology.NewMedDRANormalizer(terminology.MedDRANormalizerConfig{
					DB:     meddraDB,
					Logger: entry,
				})
				if normErr != nil {
					entry.WithError(normErr).Warn("Failed to create MedDRA normalizer")
				} else {
					aeNormalizer = normalizer
					stats := normalizer.Stats()
					entry.WithFields(logrus.Fields{
						"llt_count": stats.LLTCount,
						"pt_count":  stats.PTCount,
						"soc_count": stats.SOCCount,
					}).Info("MedDRA normalizer initialized - adverse event normalization enabled (Issues 2+3 fix)")
				}
			}
		}
	}

	// Fix 3: Create MedDRA prose scanner from the same SQLite DB as the normalizer.
	// The scanner FINDS all MedDRA terms in prose text (vs the normalizer which VALIDATES known terms).
	var proseScanner *MedDRAProseScanner
	if aeNormalizer != nil && meddraDB != nil {
		scanner, scanErr := NewMedDRAProseScanner(meddraDB, entry)
		if scanErr != nil {
			entry.WithError(scanErr).Warn("Failed to create MedDRA prose scanner, prose scanning disabled")
		} else {
			proseScanner = scanner
			entry.WithField("terms", scanner.TermCount()).Info("MedDRA prose scanner created — deterministic prose AE extraction enabled")
		}
	}

	// Initialize the underlying pipeline
	pipelineConfig := PipelineConfig{
		BatchSize:            config.BatchSize,
		MaxConcurrentDocs:    config.MaxConcurrent,
		AutoApproveThreshold: config.AutoApproveThreshold,
		ReviewThreshold:      config.ReviewThreshold,
		LLMConsensusRequired: 2,
		LLMMinConfidence:     0.70,
	}
	pipeline := NewPipeline(pipelineConfig, repo, sectionRouter, authorityRouter, entry, drugNormalizer, aeNormalizer, proseScanner)

	// Phase 3c: Initialize LLM fallback provider if API key provided and LLM enabled
	if config.EnableLLM && config.AnthropicAPIKey != "" {
		budget := config.LLMBudgetUSD
		if budget <= 0 {
			budget = 50.0
		}
		pipeline.llmProvider = newLLMFallbackProvider(config.AnthropicAPIKey, "")
		pipeline.llmBudget = newBudgetTracker(budget)
		entry.WithField("budget_usd", budget).Info("LLM fallback enabled (Claude Sonnet)")
	} else {
		entry.Info("LLM fallback disabled (no API key or --skip-llm)")
	}

	// Phase 3d: KDIGO organ impairment extraction
	// NOTE: CPIC removed from OI path — CPIC provides pharmacogenomics (gene-drug), not organ impairment (eGFR thresholds)
	if config.IncludeOrganImpairment {
		pipeline.includeOrganImpairment = true
		entry.Info("Organ impairment enrichment enabled (KDIGO only)")
	}
	if config.KDIGORulesPath != "" {
		kdigoClient, kdigoErr := kdigo.NewClient(config.KDIGORulesPath)
		if kdigoErr != nil {
			entry.WithError(kdigoErr).Error("KDIGO client init failed — continuing without KDIGO")
		} else {
			pipeline.kdigoClient = kdigoClient
			entry.WithField("rules_path", config.KDIGORulesPath).Info("KDIGO MCP-RAG extraction enabled (all rules PENDING_REVIEW)")
		}
	}

	// Pre-flight degradation summary: log which extraction paths are active vs inactive.
	// This prevents silent degradation where components are nil-guarded at call sites
	// (pipeline.go:788, 1034) and the pipeline runs without warning about missing channels.
	activePaths := 0
	totalPaths := 5
	pathStatus := make(map[string]string)

	if aeNormalizer != nil {
		pathStatus["AE Normalizer (MedDRA validation)"] = "ACTIVE"
		activePaths++
	} else {
		pathStatus["AE Normalizer (MedDRA validation)"] = "INACTIVE — pass --meddra-valueset or --mrconso to activate"
	}
	if proseScanner != nil {
		pathStatus["Prose Scanner (MedDRA term discovery)"] = "ACTIVE"
		activePaths++
	} else {
		pathStatus["Prose Scanner (MedDRA term discovery)"] = "INACTIVE — requires MedDRA to be loaded"
	}
	// DDI grammar is always initialized (pipeline.go:245) — unconditional
	pathStatus["DDI Grammar (interaction patterns)"] = "ACTIVE"
	activePaths++
	if pipeline.kdigoClient != nil {
		pathStatus["KDIGO Enrichment (organ impairment)"] = "ACTIVE"
		activePaths++
	} else {
		pathStatus["KDIGO Enrichment (organ impairment)"] = "INACTIVE — pass --kdigo-rules to activate"
	}
	if pipeline.llmProvider != nil {
		pathStatus["LLM Fallback (Claude extraction)"] = "ACTIVE"
		activePaths++
	} else {
		pathStatus["LLM Fallback (Claude extraction)"] = "INACTIVE — pass --llm-budget and API key to activate"
	}

	for path, status := range pathStatus {
		entry.WithField("status", status).Info("Extraction path: " + path)
	}

	if activePaths < totalPaths {
		entry.WithFields(logrus.Fields{
			"active": activePaths,
			"total":  totalPaths,
		}).Warn("PRE-FLIGHT: Running in DEGRADED mode — not all extraction paths are active")
	} else {
		entry.WithField("paths", activePaths).Info("PRE-FLIGHT: All extraction paths active")
	}

	return &PipelineRunner{
		config:          config,
		log:             entry,
		repo:            repo,
		splFetcher:      splFetcher,
		sectionRouter:   sectionRouter,
		tableClassifier: tableClassifier,
		authorityRouter: authorityRouter,
		rxnavClient:     rxnavClient,
		drugNormalizer:  drugNormalizer,
		pipeline:        pipeline,
	}
}

// =============================================================================
// MAIN EXECUTION
// =============================================================================

// Run executes the full 9-phase pipeline
func (r *PipelineRunner) Run(ctx context.Context) (*PipelineRun, error) {
	run := r.initializeRun()
	r.currentRun = run

	r.log.WithField("runId", run.ID).Info("Starting SPL FactStore Pipeline")

	// Execute phases in sequence
	phases := []struct {
		id     string
		name   string
		fn     func(ctx context.Context, run *PipelineRun) error
		skip   bool
	}{
		{"A", "Verify Spine", r.phaseA_VerifySpine, !r.config.VerifySpine},
		{"B", "Select Scope", r.phaseB_SelectScope, false},
		{"C", "SPL Acquisition", r.phaseC_SPLAcquisition, false},
		{"D", "LOINC Section Routing", r.phaseD_LOINCSectionRouting, false},
		{"E-F", "Table Extraction & Rule Generation", r.phaseEF_TableExtraction, false},
		{"G", "DraftFact Creation", r.phaseG_DraftFactCreation, false},
		{"H", "Governance Handoff", r.phaseH_GovernanceHandoff, false},
		{"I", "KB Projection", r.phaseI_KBProjection, !r.config.EnableKBProjection},
	}

	for _, phase := range phases {
		if phase.skip {
			r.skipPhase(run, phase.id, phase.name)
			continue
		}

		if err := r.executePhase(ctx, run, phase.id, phase.name, phase.fn); err != nil {
			r.failRun(run, fmt.Sprintf("Phase %s failed: %v", phase.id, err))
			return run, err
		}

		// Check context cancellation between phases
		if ctx.Err() != nil {
			r.failRun(run, "Pipeline cancelled")
			return run, ctx.Err()
		}
	}

	r.completeRun(run)
	return run, nil
}

// RunSingleDrug runs the pipeline for a single drug (useful for testing)
func (r *PipelineRunner) RunSingleDrug(ctx context.Context, rxcui string, drugName string) (*PipelineRun, error) {
	// Override config with single drug
	r.config.TargetDrugs = []TargetDrug{
		{RxCUI: rxcui, DrugName: drugName, Reason: "Single drug run"},
	}
	return r.Run(ctx)
}

// =============================================================================
// PHASE IMPLEMENTATIONS
// =============================================================================

// Phase A: Verify Spine
func (r *PipelineRunner) phaseA_VerifySpine(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase A: Verifying spine tables...")

	// Check drug_master table
	drugCount, err := r.repo.CountDrugMaster(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify drug_master: %w", err)
	}

	// Check source_documents table exists
	if err := r.repo.VerifyTableExists(ctx, "source_documents"); err != nil {
		return fmt.Errorf("source_documents table not found: %w", err)
	}

	// Check source_sections table exists
	if err := r.repo.VerifyTableExists(ctx, "source_sections"); err != nil {
		return fmt.Errorf("source_sections table not found: %w", err)
	}

	// Check derived_facts table exists
	if err := r.repo.VerifyTableExists(ctx, "derived_facts"); err != nil {
		return fmt.Errorf("derived_facts table not found: %w", err)
	}

	run.Metrics.SpineVerified = true
	run.Phases["A"].Details = map[string]interface{}{
		"drugMasterCount":       drugCount,
		"sourceDocumentsExists": true,
		"sourceSectionsExists":  true,
		"derivedFactsExists":    true,
	}

	r.log.WithField("drugMasterCount", drugCount).Info("Spine verified successfully")
	return nil
}

// Phase B: Select Scope
func (r *PipelineRunner) phaseB_SelectScope(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase B: Selecting ingestion scope...")

	// Use configured drugs or default to initial 10
	if len(r.config.TargetDrugs) == 0 {
		r.config.TargetDrugs = InitialScopeDrugs()
	}

	// Future-proof RxCUI resolution: validate every drug's RxCUI via RxNav before processing.
	// This catches wrong/stale hardcoded RxCUIs and resolves missing ones from drug names.
	// Without this, wrong RxCUIs silently propagate and cause FK failures at projection time.
	var validated int
	for i := range r.config.TargetDrugs {
		drug := &r.config.TargetDrugs[i]

		// Strategy 1: Use drug normalizer (RxNav) to validate/resolve — most reliable
		if r.drugNormalizer != nil && drug.DrugName != "" {
			normalized, err := r.drugNormalizer.ValidateAndNormalize(ctx, drug.RxCUI, drug.DrugName)
			if err != nil {
				r.log.WithError(err).WithField("drugName", drug.DrugName).Warn("RxNav validation failed, falling back to drug_master")
			} else {
				if normalized.WasCorrected {
					r.log.WithFields(logrus.Fields{
						"drugName":     drug.DrugName,
						"old_rxcui":    drug.RxCUI,
						"new_rxcui":    normalized.CanonicalRxCUI,
						"canonical":    normalized.CanonicalName,
					}).Warn("Phase B: RxCUI corrected via RxNav")
				}
				drug.RxCUI = normalized.CanonicalRxCUI
				drug.DrugName = normalized.CanonicalName
				validated++
				continue
			}
		}

		// Strategy 2: Fallback to drug_master lookup if normalizer unavailable
		if drug.RxCUI == "" && drug.DrugName != "" {
			rxcui, err := r.repo.LookupRxCUIByName(ctx, drug.DrugName)
			if err != nil {
				r.log.WithError(err).WithField("drugName", drug.DrugName).Warn("Failed to lookup RxCUI")
			} else if rxcui != "" {
				drug.RxCUI = rxcui
				validated++
				r.log.WithFields(logrus.Fields{
					"drugName": drug.DrugName,
					"rxcui":    rxcui,
				}).Debug("Enriched drug with RxCUI from drug_master")
			}
		}
	}

	run.Metrics.DrugsInScope = len(r.config.TargetDrugs)
	run.Phases["B"].Details = map[string]interface{}{
		"targetDrugs":    r.config.TargetDrugs,
		"rxcuiValidated": validated,
	}

	r.log.WithFields(logrus.Fields{
		"drugCount":      len(r.config.TargetDrugs),
		"rxcuiValidated": validated,
	}).Info("Scope selected — all RxCUIs validated via RxNav")
	return nil
}

// Phase C: SPL Acquisition
func (r *PipelineRunner) phaseC_SPLAcquisition(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase C: Acquiring SPL documents from DailyMed...")

	var fetched, failed int
	results := make([]map[string]interface{}, 0)

	for _, drug := range r.config.TargetDrugs {
		r.log.WithFields(logrus.Fields{
			"rxcui":    drug.RxCUI,
			"drugName": drug.DrugName,
		}).Debug("Fetching SPL")

		// Try multiple strategies to get SPL SetID
		var setID string
		var searchErr error

		// Strategy 1: Try RxNav (drug name → SPL SetID)
		if drug.DrugName != "" {
			setID, _ = r.rxnavClient.GetSPLSetIDFromDrugName(ctx, drug.DrugName)
		}

		// Strategy 2: Try RxNav (RxCUI → SPL SetID)
		if setID == "" && drug.RxCUI != "" {
			setID, _ = r.rxnavClient.GetSPLSetID(ctx, drug.RxCUI)
		}

		// Strategy 3: Search DailyMed directly by drug name (fallback)
		if setID == "" && drug.DrugName != "" {
			r.log.WithField("drug", drug.DrugName).Debug("RxNav lookup failed, trying DailyMed search")
			searchResults, err := r.splFetcher.SearchByDrugName(ctx, drug.DrugName)
			if err == nil && len(searchResults) > 0 {
				setID = searchResults[0].SetID
				r.log.WithFields(logrus.Fields{
					"drug":  drug.DrugName,
					"setId": setID,
					"title": searchResults[0].Title,
				}).Debug("Found SPL via DailyMed search")
			} else {
				searchErr = err
			}
		}

		if setID == "" {
			errMsg := "SetID not found via RxNav or DailyMed"
			if searchErr != nil {
				errMsg = searchErr.Error()
			}
			r.log.WithField("drug", drug.DrugName).Warn("Could not find SPL SetID")
			failed++
			results = append(results, map[string]interface{}{
				"rxcui":  drug.RxCUI,
				"drug":   drug.DrugName,
				"status": "FAILED",
				"error":  errMsg,
			})
			continue
		}

		// Fetch and parse SPL document
		splDoc, err := r.splFetcher.FetchBySetID(ctx, setID)
		if err != nil {
			r.log.WithError(err).WithField("setId", setID).Warn("Failed to fetch SPL")
			failed++
			results = append(results, map[string]interface{}{
				"rxcui":  drug.RxCUI,
				"drug":   drug.DrugName,
				"setId":  setID,
				"status": "FAILED",
				"error":  err.Error(),
			})
			continue
		}

		sourceDoc := &SourceDocument{
			ID:               uuid.New().String(),
			SourceType:       "FDA_SPL",
			DocumentID:       setID,
			VersionNumber:    fmt.Sprintf("%d", splDoc.VersionNumber.Value),
			RawContentHash:   splDoc.ContentHash,
			FetchedAt:        time.Now(),
			DrugName:         drug.DrugName,
			RxCUI:            drug.RxCUI,
			ProcessingStatus: "FETCHED",
		}

		if err := r.repo.CreateSourceDocument(ctx, sourceDoc); err != nil {
			r.log.WithError(err).Warn("Failed to store source document")
			// Continue anyway - may be duplicate
		}

		fetched++
		results = append(results, map[string]interface{}{
			"rxcui":     drug.RxCUI,
			"drug":      drug.DrugName,
			"setId":     setID,
			"version":   splDoc.VersionNumber.Value,
			"status":    "FETCHED",
			"sections":  len(splDoc.Sections),
		})

		// Rate limiting
		time.Sleep(time.Duration(r.config.RateLimitMs) * time.Millisecond)
	}

	run.Metrics.SPLsFetched = fetched
	run.Metrics.SPLsFailed = failed
	run.Phases["C"].Details = map[string]interface{}{
		"fetched": fetched,
		"failed":  failed,
		"results": results,
	}

	r.log.WithFields(logrus.Fields{
		"fetched": fetched,
		"failed":  failed,
	}).Info("SPL acquisition complete")

	return nil
}

// Phase D: LOINC Section Routing
func (r *PipelineRunner) phaseD_LOINCSectionRouting(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase D: Routing LOINC sections...")

	// Get all source documents that need processing
	docs, err := r.repo.GetSourceDocumentsByStatus(ctx, "FETCHED")
	if err != nil {
		return fmt.Errorf("failed to get source documents: %w", err)
	}

	var sectionsRouted, authorityLookups, tablesClassified int

	for _, doc := range docs {
		// Fetch the SPL (uses cache if available)
		splDoc, err := r.splFetcher.FetchBySetID(ctx, doc.DocumentID)
		if err != nil {
			continue
		}

		// Route all sections
		routedSections := r.sectionRouter.RouteDocument(splDoc)

		for _, routed := range routedSections {
			if len(routed.TargetKBs) == 0 {
				continue
			}

			sectionsRouted++

			// Check if authority routing applies
			if routed.RequiresAuthority != "" {
				authorityLookups++
			}

			// Count classified tables
			tablesClassified += len(routed.ExtractedTables)

			// Create source section record (sanitize text for Postgres compatibility)
			section := &SourceSection{
				ID:                   uuid.New().String(),
				SourceDocumentID:     doc.ID,
				SectionCode:          routed.Section.Code.Code,
				SectionName:          sanitizeForJSON(routed.Section.Code.DisplayName),
				TargetKBs:            routed.TargetKBs,
				RawText:              sanitizeForJSON(routed.PlainText),
				HasStructuredTables:  routed.HasTables,
				TableCount:           routed.TableCount,
				WordCount:            len(strings.Fields(routed.PlainText)),
			}

			if routed.RequiresAuthority != "" {
				section.ExtractionMethod = "AUTHORITY"
			} else if routed.HasTables {
				section.ExtractionMethod = "TABLE_PARSE"
			} else {
				section.ExtractionMethod = "NARRATIVE_PARSE"
			}

			_ = r.repo.CreateSourceSection(ctx, section)
		}

		// Update document status
		_ = r.repo.UpdateSourceDocumentStatus(ctx, doc.ID, "ROUTED", "")
	}

	run.Metrics.SectionsRouted = sectionsRouted
	run.Metrics.AuthorityLookups = authorityLookups
	run.Metrics.TablesClassified = tablesClassified
	run.Phases["D"].Details = map[string]interface{}{
		"sectionsRouted":   sectionsRouted,
		"authorityLookups": authorityLookups,
		"tablesClassified": tablesClassified,
	}

	r.log.WithFields(logrus.Fields{
		"sectionsRouted":   sectionsRouted,
		"authorityLookups": authorityLookups,
		"tablesClassified": tablesClassified,
	}).Info("LOINC section routing complete")

	return nil
}

// Phase E-F: Table Extraction & Rule Generation
func (r *PipelineRunner) phaseEF_TableExtraction(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase E-F: Extracting tables and generating rules...")

	// Get all source documents that have been routed
	docs, err := r.repo.GetSourceDocumentsByStatus(ctx, "ROUTED")
	if err != nil {
		return fmt.Errorf("failed to get source documents: %w", err)
	}

	var rulesGenerated int

	for _, doc := range docs {
		// Fetch the SPL (uses cache if available)
		splDoc, err := r.splFetcher.FetchBySetID(ctx, doc.DocumentID)
		if err != nil {
			continue
		}

		// The pipeline handles fact extraction from tables
		result, err := r.pipeline.ProcessSPLDocument(ctx, splDoc, doc.RxCUI, doc.DrugName)
		if err != nil {
			r.log.WithError(err).WithField("doc", doc.DocumentID).Warn("Pipeline processing failed")
			continue
		}

		rulesGenerated += result.FactsExtracted

		// Update document status
		_ = r.repo.UpdateSourceDocumentStatus(ctx, doc.ID, "EXTRACTED", "")
	}

	run.Metrics.RulesGenerated = rulesGenerated
	run.Phases["E-F"].Details = map[string]interface{}{
		"rulesGenerated": rulesGenerated,
	}

	r.log.WithField("rulesGenerated", rulesGenerated).Info("Table extraction complete")
	return nil
}

// Phase G: DraftFact Creation
func (r *PipelineRunner) phaseG_DraftFactCreation(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase G: Creating draft facts...")

	// Count facts by type
	factsByType, err := r.repo.CountFactsByType(ctx)
	if err != nil {
		return fmt.Errorf("failed to count facts: %w", err)
	}

	totalFacts := 0
	for _, count := range factsByType {
		totalFacts += count
	}

	run.Metrics.FactsCreated = totalFacts
	run.Metrics.FactsByType = factsByType
	run.Phases["G"].Details = map[string]interface{}{
		"totalFacts": totalFacts,
		"byType":     factsByType,
	}

	r.log.WithFields(logrus.Fields{
		"totalFacts": totalFacts,
		"byType":     factsByType,
	}).Info("Draft facts created")

	return nil
}

// Phase H: Governance Handoff
func (r *PipelineRunner) phaseH_GovernanceHandoff(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase H: Processing governance decisions...")

	// Get governance statistics
	stats, err := r.repo.GetGovernanceStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get governance stats: %w", err)
	}

	run.Metrics.FactsAutoApproved = stats.AutoApproved
	run.Metrics.FactsPendingReview = stats.PendingReview
	run.Metrics.FactsRejected = stats.Rejected
	run.Phases["H"].Details = map[string]interface{}{
		"autoApproved":  stats.AutoApproved,
		"pendingReview": stats.PendingReview,
		"rejected":      stats.Rejected,
	}

	r.log.WithFields(logrus.Fields{
		"autoApproved":  stats.AutoApproved,
		"pendingReview": stats.PendingReview,
		"rejected":      stats.Rejected,
	}).Info("Governance handoff complete")

	return nil
}

// Phase I: KB Projection
func (r *PipelineRunner) phaseI_KBProjection(ctx context.Context, run *PipelineRun) error {
	r.log.Info("Phase I: Projecting facts to downstream KBs...")

	// Step 1: Project approved derived_facts → clinical_facts
	// This is the key operation that bridges ingestion layer → consumption layer
	projected, err := r.repo.ProjectApprovedFactsToClinical(ctx)
	if err != nil {
		r.log.WithError(err).Warn("Projection to clinical_facts encountered errors")
	}

	r.log.WithField("projected", projected).Info("Projected approved facts to clinical_facts")

	// Step 2: Get statistics from clinical_facts (the KB views read from this)
	projectionStats, err := r.repo.GetProjectionStats(ctx)
	if err != nil {
		r.log.WithError(err).Warn("Failed to get projection stats")
		projectionStats = make(map[string]int)
	}

	// Step 3: Also count in derived_facts by target KB for metrics
	projections := make(map[string]int)
	for _, kb := range r.config.TargetKBs {
		count, err := r.repo.CountFactsByTargetKB(ctx, kb)
		if err != nil {
			r.log.WithError(err).WithField("kb", kb).Warn("Failed to count projections")
			continue
		}
		projections[kb] = count
	}

	run.Metrics.KBProjections = projections
	run.Phases["I"].Details = map[string]interface{}{
		"derivedFactsByKB":  projections,
		"clinicalFactStats": projectionStats,
		"newlyProjected":    projected,
	}

	r.log.WithFields(logrus.Fields{
		"derivedFactsByKB":  projections,
		"clinicalFactStats": projectionStats,
		"newlyProjected":    projected,
	}).Info("KB projection complete")

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func (r *PipelineRunner) initializeRun() *PipelineRun {
	return &PipelineRun{
		ID:        uuid.New().String(),
		StartedAt: time.Now(),
		Status:    "RUNNING",
		Phases:    make(map[string]*PhaseResult),
		Metrics: RunMetrics{
			FactsByType:   make(map[string]int),
			KBProjections: make(map[string]int),
		},
	}
}

func (r *PipelineRunner) executePhase(ctx context.Context, run *PipelineRun, id, name string, fn func(context.Context, *PipelineRun) error) error {
	phase := &PhaseResult{
		Phase:     id,
		Name:      name,
		StartedAt: time.Now(),
		Status:    "RUNNING",
	}
	run.Phases[id] = phase

	r.log.WithFields(logrus.Fields{
		"phase": id,
		"name":  name,
	}).Info("Starting phase")

	err := fn(ctx, run)
	now := time.Now()
	phase.EndedAt = &now
	phase.Duration = now.Sub(phase.StartedAt)

	if err != nil {
		phase.Status = "FAILED"
		phase.Message = err.Error()
		run.Errors = append(run.Errors, fmt.Sprintf("Phase %s: %v", id, err))
		return err
	}

	phase.Status = "COMPLETED"
	r.log.WithFields(logrus.Fields{
		"phase":    id,
		"duration": phase.Duration,
	}).Info("Phase completed")

	return nil
}

func (r *PipelineRunner) skipPhase(run *PipelineRun, id, name string) {
	run.Phases[id] = &PhaseResult{
		Phase:     id,
		Name:      name,
		StartedAt: time.Now(),
		Status:    "SKIPPED",
		Message:   "Skipped by configuration",
	}
	r.log.WithField("phase", id).Debug("Phase skipped")
}

func (r *PipelineRunner) completeRun(run *PipelineRun) {
	now := time.Now()
	run.EndedAt = &now
	run.Status = "COMPLETED"
	run.Metrics.TotalDuration = now.Sub(run.StartedAt)

	r.log.WithFields(logrus.Fields{
		"runId":    run.ID,
		"duration": run.Metrics.TotalDuration,
		"facts":    run.Metrics.FactsCreated,
	}).Info("Pipeline run completed successfully")
}

func (r *PipelineRunner) failRun(run *PipelineRun, message string) {
	now := time.Now()
	run.EndedAt = &now
	run.Status = "FAILED"
	run.Metrics.TotalDuration = now.Sub(run.StartedAt)
	run.Errors = append(run.Errors, message)

	r.log.WithFields(logrus.Fields{
		"runId":  run.ID,
		"errors": run.Errors,
	}).Error("Pipeline run failed")
}

// GetCurrentRun returns the currently executing or last completed run
func (r *PipelineRunner) GetCurrentRun() *PipelineRun {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.currentRun
}

// VerifySpine checks that all required database tables exist
func (r *PipelineRunner) VerifySpine(ctx context.Context) error {
	r.log.Info("Verifying spine tables...")

	// Check drug_master table
	drugCount, err := r.repo.CountDrugMaster(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify drug_master: %w", err)
	}
	r.log.WithField("drugMasterCount", drugCount).Debug("drug_master verified")

	// Check source_documents table exists
	if err := r.repo.VerifyTableExists(ctx, "source_documents"); err != nil {
		return fmt.Errorf("source_documents table not found: %w", err)
	}

	// Check source_sections table exists
	if err := r.repo.VerifyTableExists(ctx, "source_sections"); err != nil {
		return fmt.Errorf("source_sections table not found: %w", err)
	}

	// Check derived_facts table exists
	if err := r.repo.VerifyTableExists(ctx, "derived_facts"); err != nil {
		return fmt.Errorf("derived_facts table not found: %w", err)
	}

	// Check loinc_section_routing table
	if err := r.repo.VerifyTableExists(ctx, "loinc_section_routing"); err != nil {
		return fmt.Errorf("loinc_section_routing table not found: %w", err)
	}

	// Check authority_sources table
	if err := r.repo.VerifyTableExists(ctx, "authority_sources"); err != nil {
		return fmt.Errorf("authority_sources table not found: %w", err)
	}

	r.log.Info("All spine tables verified successfully")
	return nil
}

// AllFactTypes returns all 6 canonical fact types
func AllFactTypes() []FactType {
	return []FactType{
		FactTypeOrganImpairment,
		FactTypeSafetySignal,
		FactTypeReproductiveSafety,
		FactTypeInteraction,
		FactTypeFormulary,
		FactTypeLabReference,
	}
}

// Health checks all dependent services
func (r *PipelineRunner) Health(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// Check database
	results["database"] = r.repo.Health(ctx)

	// Check RxNav
	results["rxnav"] = r.rxnavClient.HealthCheck(ctx)

	return results
}
