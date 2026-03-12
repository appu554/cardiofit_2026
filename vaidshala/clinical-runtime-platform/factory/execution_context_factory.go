// Package factory provides the ExecutionContextFactory that assembles
// the complete ClinicalExecutionContext from all components.
//
// ARCHITECTURE:
//   ExecutionContextFactory
//       │
//       ├── KB2Adapter (KB-2A: raw assembly)
//       │       └── strips intelligence fields
//       │
//       ├── KB2IntelligenceAdapter (KB-2B: enrichment)
//       │       └── adds phenotypes, risks, care gaps
//       │
//       └── KnowledgeSnapshotBuilder (KB-7, KB-4, KB-5, KB-6, KB-8)
//               └── pre-answers all KB queries
//
// CRITICAL FLOW:
// 1. KB-2A assembles base PatientContext (data-only)
// 2. KnowledgeSnapshot builds using base PatientContext (NOT enriched)
// 3. KB-2B enriches PatientContext with intelligence
// 4. Factory returns frozen ClinicalExecutionContext
//
// This order matters for audit and replay.
package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"vaidshala/clinical-runtime-platform/adapters"
	"vaidshala/clinical-runtime-platform/builders"
	"vaidshala/clinical-runtime-platform/contracts"
)

// ExecutionContextFactory assembles ClinicalExecutionContext.
// This is the ONLY way engines should obtain their context.
type ExecutionContextFactory struct {
	// KB-2A: Raw patient assembly
	kb2Adapter *adapters.KB2Adapter

	// KB-2B: Intelligence enrichment
	kb2Intelligence adapters.KB2Intelligence

	// Knowledge snapshot builder
	snapshotBuilder *builders.KnowledgeSnapshotBuilder

	// Factory configuration
	config FactoryConfig
}

// FactoryConfig configures the factory behavior.
type FactoryConfig struct {
	// Region for regional rules (IN, AU)
	Region string

	// DefaultMeasurementPeriod for quality measures
	DefaultMeasurementPeriodDays int

	// EnableParallelBuilds allows concurrent KB queries
	EnableParallelBuilds bool

	// BuildTimeout maximum time for context building
	BuildTimeout time.Duration

	// TenantID for multi-tenant deployments
	TenantID string
}

// DefaultFactoryConfig returns sensible defaults.
func DefaultFactoryConfig() FactoryConfig {
	return FactoryConfig{
		Region:                       "AU",
		DefaultMeasurementPeriodDays: 365,
		EnableParallelBuilds:         true,
		BuildTimeout:                 30 * time.Second,
		TenantID:                     "default",
	}
}

// NewExecutionContextFactory creates a new factory with all dependencies.
func NewExecutionContextFactory(
	kb2Adapter *adapters.KB2Adapter,
	kb2Intelligence adapters.KB2Intelligence,
	snapshotBuilder *builders.KnowledgeSnapshotBuilder,
	config FactoryConfig,
) *ExecutionContextFactory {
	return &ExecutionContextFactory{
		kb2Adapter:      kb2Adapter,
		kb2Intelligence: kb2Intelligence,
		snapshotBuilder: snapshotBuilder,
		config:          config,
	}
}

// BuildRequest contains parameters for building execution context.
type BuildRequest struct {
	// PatientID the FHIR patient identifier
	PatientID string

	// RawFHIRInput patient data (from FHIR store or bundle)
	RawFHIRInput map[string]interface{}

	// RequestedBy user/system making the request
	RequestedBy string

	// RequestedEngines specific engines to run (empty = all)
	RequestedEngines []string

	// MeasurementPeriod override for quality measures
	MeasurementPeriod *contracts.Period

	// ExecutionMode (sync, async, batch)
	ExecutionMode string
}

// BuildResult contains the built context and metadata.
type BuildResult struct {
	// Context the frozen execution context
	Context *contracts.ClinicalExecutionContext

	// BuildTimeMs how long the build took
	BuildTimeMs int64

	// Warnings non-fatal issues during build
	Warnings []string
}

// Build assembles a complete ClinicalExecutionContext.
//
// CRITICAL ORDER:
// 1. KB-2A: Assemble base PatientContext (data-only, NO intelligence)
// 2. KnowledgeSnapshot: Build using BASE context (audit requirement)
// 3. KB-2B: Enrich PatientContext with intelligence
// 4. Assemble: Combine into frozen contract
func (f *ExecutionContextFactory) Build(
	ctx context.Context,
	request BuildRequest,
) (*BuildResult, error) {

	startTime := time.Now()
	warnings := make([]string, 0)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, f.config.BuildTimeout)
	defer cancel()

	// ========================================================================
	// STEP 1: KB-2A Assembly (data-only PatientContext)
	// ========================================================================
	basePatient, err := f.kb2Adapter.AssemblePatientContext(
		ctx,
		request.PatientID,
		request.RawFHIRInput,
	)
	if err != nil {
		return nil, fmt.Errorf("KB-2A assembly failed: %w", err)
	}

	// Set patient ID in demographics (wasn't available to adapter)
	basePatient.Demographics.PatientID = request.PatientID

	// ========================================================================
	// STEP 2: Build KnowledgeSnapshot (using BASE context, not enriched)
	// This is CRITICAL for audit - snapshot must be based on raw data
	// ========================================================================
	snapshot, err := f.snapshotBuilder.Build(ctx, basePatient)
	if err != nil {
		// Non-fatal: continue with empty snapshot
		warnings = append(warnings, fmt.Sprintf("KnowledgeSnapshot partial: %v", err))
		snapshot = &contracts.KnowledgeSnapshot{
			SnapshotTimestamp: time.Now(),
			KBVersions:        make(map[string]string),
		}
	}

	// ========================================================================
	// STEP 3: KB-2B Intelligence Enrichment
	// ========================================================================
	enrichedPatient, err := f.kb2Intelligence.Enrich(ctx, basePatient)
	if err != nil {
		// Non-fatal: use base patient without intelligence
		warnings = append(warnings, fmt.Sprintf("KB-2B enrichment partial: %v", err))
		enrichedPatient = basePatient
	}

	// ========================================================================
	// STEP 4: Assemble Frozen ClinicalExecutionContext
	// ========================================================================
	executionContext := &contracts.ClinicalExecutionContext{
		Patient:   *enrichedPatient,
		Knowledge: *snapshot,
		Runtime:   f.buildExecutionMetadata(request),
	}

	// Build result
	result := &BuildResult{
		Context:     executionContext,
		BuildTimeMs: time.Since(startTime).Milliseconds(),
		Warnings:    warnings,
	}

	return result, nil
}

// buildExecutionMetadata creates runtime metadata.
func (f *ExecutionContextFactory) buildExecutionMetadata(request BuildRequest) contracts.ExecutionMetadata {
	metadata := contracts.ExecutionMetadata{
		RequestID:        uuid.New().String(),
		RequestedBy:      request.RequestedBy,
		RequestedAt:      time.Now(),
		Region:           f.config.Region,
		TenantID:         f.config.TenantID,
		RequestedEngines: request.RequestedEngines,
		ExecutionMode:    request.ExecutionMode,
	}

	// Set measurement period
	if request.MeasurementPeriod != nil {
		metadata.MeasurementPeriod = request.MeasurementPeriod
	} else {
		// Default: last year
		now := time.Now()
		start := now.AddDate(-1, 0, 0) // 1 year ago
		metadata.MeasurementPeriod = &contracts.Period{
			Start: &start,
			End:   &now,
		}
	}

	if metadata.ExecutionMode == "" {
		metadata.ExecutionMode = "sync"
	}

	return metadata
}

// ============================================================================
// ENGINE INTERFACES (CTO/CMO Architecture)
// ============================================================================

// Engine interface that all clinical engines must implement.
// Engines receive ClinicalExecutionContext and return EngineResult.
type Engine interface {
	// Name returns the engine identifier
	Name() string

	// Evaluate processes the context and returns results
	Evaluate(ctx context.Context, execCtx *contracts.ClinicalExecutionContext) (*contracts.EngineResult, error)
}

// FactAwareEngine is implemented by engines that can consume ClinicalFacts.
// This enables the CQL → Measure Engine flow per CTO/CMO architecture.
//
// Per CTO/CMO Architecture:
//   - CQL Engine produces ClinicalFacts (truths)
//   - Measure Engine consumes those facts to produce MeasureResults (judgments)
//   - This interface makes that dependency explicit
type FactAwareEngine interface {
	Engine

	// EvaluateWithFacts processes context with pre-computed CQL facts injected.
	// This is the preferred method when CQL Engine has already run.
	EvaluateWithFacts(ctx context.Context, execCtx *contracts.ClinicalExecutionContext, facts []contracts.ClinicalFact) (*contracts.EngineResult, error)
}

// ============================================================================
// ENGINE ORCHESTRATOR (CTO/CMO Compliant)
// ============================================================================

// EngineOrchestrator runs multiple engines against a context.
// It implements the CTO/CMO architecture by ensuring proper data flow:
//
// CRITICAL FLOW:
//   1. CQL Engine runs FIRST (produces ClinicalFacts - truths)
//   2. ClinicalFacts flow to Measure Engine (produces MeasureResults - judgments)
//   3. Other engines run independently (can be parallel)
//
// This ensures:
//   - Clean separation between truth determination and care accountability
//   - Auditability (facts are logged before judgments)
//   - Deterministic execution order
type EngineOrchestrator struct {
	// cqlEngine is the CQL Engine (produces clinical truths)
	cqlEngine Engine

	// measureEngine is the Measure Engine (consumes truths, produces care gaps)
	measureEngine FactAwareEngine

	// otherEngines are engines that don't participate in CQL → Measure flow
	otherEngines []Engine

	// factory builds the frozen ClinicalExecutionContext
	factory *ExecutionContextFactory

	// config for orchestration behavior
	config OrchestratorConfig
}

// OrchestratorConfig configures orchestrator behavior.
type OrchestratorConfig struct {
	// EnableCQLToMeasureFlow enables the CQL → Measure Engine fact passing
	// When true: CQL runs first, facts flow to Measure Engine
	// When false: All engines run independently (legacy behavior)
	EnableCQLToMeasureFlow bool

	// ParallelOtherEngines runs non-CQL/Measure engines in parallel
	ParallelOtherEngines bool
}

// DefaultOrchestratorConfig returns CTO/CMO compliant defaults.
func DefaultOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		EnableCQLToMeasureFlow: true, // CTO/CMO architecture
		ParallelOtherEngines:   true,
	}
}

// NewEngineOrchestrator creates a new CTO/CMO compliant orchestrator.
//
// Per CTO/CMO Architecture:
//   - cqlEngine produces ClinicalFacts (truths)
//   - measureEngine consumes facts and produces MeasureResults (judgments)
//   - otherEngines run independently
func NewEngineOrchestrator(
	factory *ExecutionContextFactory,
	cqlEngine Engine,
	measureEngine FactAwareEngine,
	otherEngines ...Engine,
) *EngineOrchestrator {
	return &EngineOrchestrator{
		cqlEngine:     cqlEngine,
		measureEngine: measureEngine,
		otherEngines:  otherEngines,
		factory:       factory,
		config:        DefaultOrchestratorConfig(),
	}
}

// NewEngineOrchestratorWithConfig creates orchestrator with custom config.
func NewEngineOrchestratorWithConfig(
	factory *ExecutionContextFactory,
	config OrchestratorConfig,
	cqlEngine Engine,
	measureEngine FactAwareEngine,
	otherEngines ...Engine,
) *EngineOrchestrator {
	return &EngineOrchestrator{
		cqlEngine:     cqlEngine,
		measureEngine: measureEngine,
		otherEngines:  otherEngines,
		factory:       factory,
		config:        config,
	}
}

// NewSimpleOrchestrator creates orchestrator without CQL → Measure flow.
// Use this for legacy compatibility or when engines are truly independent.
func NewSimpleOrchestrator(factory *ExecutionContextFactory, engines ...Engine) *EngineOrchestrator {
	return &EngineOrchestrator{
		otherEngines: engines,
		factory:      factory,
		config: OrchestratorConfig{
			EnableCQLToMeasureFlow: false,
			ParallelOtherEngines:   true,
		},
	}
}

// Execute builds context and runs all engines per CTO/CMO architecture.
//
// EXECUTION ORDER (when EnableCQLToMeasureFlow = true):
//   1. Build frozen ClinicalExecutionContext
//   2. Run CQL Engine → extract ClinicalFacts
//   3. Run Measure Engine with CQL facts injected
//   4. Run other engines (can be parallel)
//   5. Combine all results
//
// Per CTO/CMO Architecture:
//   - CQL Engine answers: "What is true about this patient?"
//   - Measure Engine answers: "Given what's true, are we meeting standards of care?"
func (o *EngineOrchestrator) Execute(
	ctx context.Context,
	request BuildRequest,
) (*OrchestratorResult, error) {

	startTime := time.Now()
	warnings := make([]string, 0)

	// ========================================================================
	// STEP 1: Build frozen ClinicalExecutionContext
	// ========================================================================
	buildResult, err := o.factory.Build(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("context build failed: %w", err)
	}
	warnings = append(warnings, buildResult.Warnings...)

	results := make([]*contracts.EngineResult, 0)
	var cqlFacts []contracts.ClinicalFact

	// ========================================================================
	// STEP 2: CQL → Measure Engine Flow (if enabled)
	// ========================================================================
	if o.config.EnableCQLToMeasureFlow && o.cqlEngine != nil {
		// 2a. Run CQL Engine FIRST (produces truths)
		cqlResult, err := o.cqlEngine.Evaluate(ctx, buildResult.Context)
		if err != nil {
			cqlResult = &contracts.EngineResult{
				EngineName: o.cqlEngine.Name(),
				Success:    false,
				Error:      err.Error(),
			}
			warnings = append(warnings, fmt.Sprintf("CQL Engine failed: %v", err))
		}
		results = append(results, cqlResult)

		// Extract facts for Measure Engine
		if cqlResult.Success {
			cqlFacts = cqlResult.ClinicalFacts
		}

		// 2b. Run Measure Engine WITH facts (produces judgments)
		if o.measureEngine != nil {
			var measureResult *contracts.EngineResult
			if len(cqlFacts) > 0 {
				// CTO/CMO flow: inject CQL facts into Measure Engine
				measureResult, err = o.measureEngine.EvaluateWithFacts(ctx, buildResult.Context, cqlFacts)
			} else {
				// Fallback: run Measure Engine without facts
				measureResult, err = o.measureEngine.Evaluate(ctx, buildResult.Context)
				warnings = append(warnings, "Measure Engine ran without CQL facts (CQL produced no facts)")
			}

			if err != nil {
				measureResult = &contracts.EngineResult{
					EngineName: o.measureEngine.Name(),
					Success:    false,
					Error:      err.Error(),
				}
				warnings = append(warnings, fmt.Sprintf("Measure Engine failed: %v", err))
			}
			results = append(results, measureResult)
		}
	}

	// ========================================================================
	// STEP 3: Run Other Engines (independent of CQL → Measure flow)
	// ========================================================================
	for _, engine := range o.otherEngines {
		// Skip if already handled
		if o.cqlEngine != nil && engine.Name() == o.cqlEngine.Name() {
			continue
		}
		if o.measureEngine != nil && engine.Name() == o.measureEngine.Name() {
			continue
		}

		result, err := engine.Evaluate(ctx, buildResult.Context)
		if err != nil {
			result = &contracts.EngineResult{
				EngineName: engine.Name(),
				Success:    false,
				Error:      err.Error(),
			}
		}
		results = append(results, result)
	}

	// ========================================================================
	// STEP 4: Assemble Result
	// ========================================================================
	return &OrchestratorResult{
		Context:          buildResult.Context,
		EngineResults:    results,
		ClinicalFacts:    cqlFacts, // Expose CQL facts at orchestrator level
		BuildTimeMs:      buildResult.BuildTimeMs,
		TotalExecutionMs: time.Since(startTime).Milliseconds(),
		Warnings:         warnings,
	}, nil
}

// filterEngines returns only the requested engines from otherEngines.
func (o *EngineOrchestrator) filterEngines(names []string) []Engine {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	filtered := make([]Engine, 0)
	for _, e := range o.otherEngines {
		if nameSet[e.Name()] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// GetAllEngines returns all registered engines.
func (o *EngineOrchestrator) GetAllEngines() []Engine {
	engines := make([]Engine, 0)
	if o.cqlEngine != nil {
		engines = append(engines, o.cqlEngine)
	}
	if o.measureEngine != nil {
		engines = append(engines, o.measureEngine)
	}
	engines = append(engines, o.otherEngines...)
	return engines
}

// OrchestratorResult contains results from all engines.
type OrchestratorResult struct {
	// Context the execution context used
	Context *contracts.ClinicalExecutionContext

	// EngineResults from each engine
	EngineResults []*contracts.EngineResult

	// ClinicalFacts from CQL Engine (exposed at orchestrator level)
	// Per CTO/CMO Architecture: These are the clinical truths that
	// Measure Engine consumed to produce care gap judgments
	ClinicalFacts []contracts.ClinicalFact

	// BuildTimeMs time to build context
	BuildTimeMs int64

	// TotalExecutionMs total time including engines
	TotalExecutionMs int64

	// Warnings any non-fatal issues
	Warnings []string
}

// AllSucceeded returns true if all engines succeeded.
func (r *OrchestratorResult) AllSucceeded() bool {
	for _, result := range r.EngineResults {
		if !result.Success {
			return false
		}
	}
	return true
}

// GetRecommendations returns all recommendations from all engines.
func (r *OrchestratorResult) GetRecommendations() []contracts.Recommendation {
	all := make([]contracts.Recommendation, 0)
	for _, result := range r.EngineResults {
		all = append(all, result.Recommendations...)
	}
	return all
}

// GetAlerts returns all alerts from all engines.
func (r *OrchestratorResult) GetAlerts() []contracts.Alert {
	all := make([]contracts.Alert, 0)
	for _, result := range r.EngineResults {
		all = append(all, result.Alerts...)
	}
	return all
}

// GetMeasureResults returns all measure results from all engines.
func (r *OrchestratorResult) GetMeasureResults() []contracts.MeasureResult {
	all := make([]contracts.MeasureResult, 0)
	for _, result := range r.EngineResults {
		all = append(all, result.MeasureResults...)
	}
	return all
}

// GetClinicalFacts returns clinical facts from CQL Engine.
// Per CTO/CMO Architecture: These are the truths that Measure Engine consumed.
func (r *OrchestratorResult) GetClinicalFacts() []contracts.ClinicalFact {
	return r.ClinicalFacts
}

// GetFactByID returns a specific clinical fact by ID.
func (r *OrchestratorResult) GetFactByID(factID string) (contracts.ClinicalFact, bool) {
	for _, fact := range r.ClinicalFacts {
		if fact.FactID == factID {
			return fact, true
		}
	}
	return contracts.ClinicalFact{}, false
}

// HasCareGaps returns true if any measure identified a care gap.
func (r *OrchestratorResult) HasCareGaps() bool {
	for _, mr := range r.GetMeasureResults() {
		if mr.CareGapIdentified {
			return true
		}
	}
	return false
}

// GetCareGaps returns all measure results where care gaps were identified.
func (r *OrchestratorResult) GetCareGaps() []contracts.MeasureResult {
	gaps := make([]contracts.MeasureResult, 0)
	for _, mr := range r.GetMeasureResults() {
		if mr.CareGapIdentified {
			gaps = append(gaps, mr)
		}
	}
	return gaps
}
