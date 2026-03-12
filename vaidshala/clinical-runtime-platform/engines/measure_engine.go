// Package engines provides the MeasureEngine for clinical care accountability.
//
// MEASURE ENGINE ARCHITECTURE (per CTO/CMO spec):
//
// PURPOSE: Clinical Accountability & Care Gap Detection
// QUESTION IT ANSWERS: "Given the facts, are we meeting standards of care?"
//
// The Measure Engine is a CARE ACCOUNTABILITY LAYER, not a truth evaluator.
// It interprets clinical truths IN THE CONTEXT OF standards, policy, and performance.
//
// INPUTS:
//   - ClinicalExecutionContext (FROZEN) - all patient data pre-assembled
//   - ClinicalFacts (from CQL Engine) - pre-evaluated clinical truths
//   - Patient demographics, time windows, exclusion criteria
//   - CMS/payer definitions
//
// OUTPUTS:
//   - MeasureResults: Population membership judgments (InDenominator, InNumerator)
//   - CareGaps: Identified gaps in recommended care
//   - Recommendations: Actions to close care gaps
//
// CRITICAL DISTINCTION:
//   - CQL Engine tells you: "The value is abnormal" (truth)
//   - Measure Engine tells you: "Are we doing what we're supposed to do?" (judgment)
//
// ANALOGY: Measure Engine = Quality officer + guideline auditor
//          It answers: "Are we meeting standards of care?"
package engines

import (
	"context"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines/measures"
)

// ============================================================================
// MEASURE ENGINE: CLINICAL CARE ACCOUNTABILITY
// ============================================================================

// MeasureEngine evaluates care standards compliance from patient context and CQL facts.
// It implements the Engine interface and produces MeasureResults (care judgments).
//
// Per CTO/CMO Architecture:
//   - Measure Engine produces JUDGMENTS, not clinical truths
//   - It consumes facts from CQL Engine for truth determination
//   - It determines population membership and care gaps based on standards
//   - No external calls - works entirely from frozen context and pre-computed facts
type MeasureEngine struct {
	// evaluators is the registry of all CMS measure evaluators
	evaluators map[string]measures.MeasureEvaluator

	// config for engine behavior
	config MeasureEngineConfig

	// clinicalFacts from CQL Engine (populated by orchestrator)
	clinicalFacts map[string]contracts.ClinicalFact
}

// MeasureEngineConfig configures the measure engine.
type MeasureEngineConfig struct {
	// DefaultMeasures to evaluate if none specified
	DefaultMeasures []string

	// EnableCareGapDetection auto-detect care gaps
	EnableCareGapDetection bool

	// Region for regional measure selection (AU, IN, US)
	Region string
}

// DefaultMeasureEngineConfig returns sensible defaults.
func DefaultMeasureEngineConfig() MeasureEngineConfig {
	return MeasureEngineConfig{
		DefaultMeasures: []string{
			"CMS122", // Diabetes HbA1c
			"CMS165", // Blood Pressure Control
			"CMS134", // Diabetes Kidney Health
			"CMS2",   // Depression Screening
		},
		EnableCareGapDetection: true,
		Region:                 "AU",
	}
}

// NewMeasureEngine creates a new measure engine with all evaluators registered.
func NewMeasureEngine(config MeasureEngineConfig) *MeasureEngine {
	engine := &MeasureEngine{
		evaluators:    make(map[string]measures.MeasureEvaluator),
		config:        config,
		clinicalFacts: make(map[string]contracts.ClinicalFact),
	}

	// Register all CMS evaluators
	engine.registerEvaluators()

	return engine
}

// registerEvaluators registers all available CMS measure evaluators.
func (e *MeasureEngine) registerEvaluators() {
	// CMS122: Diabetes HbA1c Poor Control (>9%)
	cms122 := measures.NewCMS122Evaluator()
	e.evaluators[cms122.MeasureID()] = cms122

	// CMS165: Controlling High Blood Pressure
	cms165 := measures.NewCMS165Evaluator()
	e.evaluators[cms165.MeasureID()] = cms165

	// CMS134: Diabetes: Medical Attention for Nephropathy
	cms134 := measures.NewCMS134Evaluator()
	e.evaluators[cms134.MeasureID()] = cms134

	// CMS2: Preventive Care and Screening: Screening for Depression
	cms2 := measures.NewCMS2Evaluator()
	e.evaluators[cms2.MeasureID()] = cms2
}

// Name returns the engine identifier.
func (e *MeasureEngine) Name() string {
	return "measure-engine"
}

// SetClinicalFacts allows the orchestrator to inject CQL Engine facts.
// This enables the CQL → Measure Engine flow per CTO/CMO architecture.
//
// Per CTO/CMO Architecture:
//   - CQL Engine produces facts: "HbA1cPoorControl = true"
//   - Orchestrator passes facts to Measure Engine
//   - Measure Engine uses facts to determine care judgments
func (e *MeasureEngine) SetClinicalFacts(facts []contracts.ClinicalFact) {
	e.clinicalFacts = make(map[string]contracts.ClinicalFact)
	for _, fact := range facts {
		e.clinicalFacts[fact.FactID] = fact
	}
}

// GetFact retrieves a clinical fact by ID.
// Returns the fact and true if found, empty fact and false if not found.
func (e *MeasureEngine) GetFact(factID string) (contracts.ClinicalFact, bool) {
	fact, exists := e.clinicalFacts[factID]
	return fact, exists
}

// HasFact checks if a specific clinical fact is true.
// This is the primary interface for measure evaluators to check CQL truths.
func (e *MeasureEngine) HasFact(factID string) bool {
	if fact, exists := e.clinicalFacts[factID]; exists {
		return fact.Value
	}
	return false
}

// Evaluate runs all configured measure evaluators against the context.
//
// CRITICAL: This engine:
// 1. Uses ONLY data from ClinicalExecutionContext (frozen contract)
// 2. Optionally uses ClinicalFacts from CQL Engine for truth lookups
// 3. Makes NO external KB calls, NO database calls, NO HTTP calls
// 4. Returns deterministic results (same input → same output)
// 5. Produces audit-ready MeasureResults with version traceability
//
// Per CTO/CMO Architecture:
//   - CQL Engine = "What is true about this patient?"
//   - Measure Engine = "Given what's true, are we meeting standards of care?"
func (e *MeasureEngine) Evaluate(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
) (*contracts.EngineResult, error) {

	startTime := time.Now()

	result := &contracts.EngineResult{
		EngineName:      e.Name(),
		Success:         true,
		ClinicalFacts:   make([]contracts.ClinicalFact, 0), // Measure Engine produces NO new facts
		Recommendations: make([]contracts.Recommendation, 0),
		Alerts:          make([]contracts.Alert, 0),
		MeasureResults:  make([]contracts.MeasureResult, 0),
		EvidenceLinks:   make([]string, 0),
	}

	// Determine which measures to evaluate
	measureIDs := e.config.DefaultMeasures
	if len(execCtx.Runtime.RequestedEngines) > 0 {
		measureIDs = e.filterMeasureIDs(execCtx.Runtime.RequestedEngines)
	}

	// Evaluate each measure
	for _, measureID := range measureIDs {
		evaluator, exists := e.evaluators[measureID]
		if !exists {
			continue // Skip unknown measures
		}

		// Execute the pure function evaluator
		// Note: Evaluators can access CQL facts via e.HasFact() or direct context
		measureResult := evaluator.Evaluate(execCtx)

		// Enrich rationale with CQL fact evidence if available
		measureResult = e.enrichWithFactEvidence(measureResult)

		result.MeasureResults = append(result.MeasureResults, measureResult)

		// Generate care gap recommendation if applicable
		if e.config.EnableCareGapDetection && measureResult.CareGapIdentified {
			recommendation := e.generateCareGapRecommendation(measureResult)
			result.Recommendations = append(result.Recommendations, recommendation)
		}
	}

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// EvaluateWithFacts runs measure evaluation with pre-computed CQL facts.
// This is the preferred method when CQL Engine has already run.
//
// Per CTO/CMO Architecture:
//   - This method makes the CQL → Measure Engine flow explicit
//   - Facts are injected, then evaluation proceeds
//   - Clear separation between truth determination and care accountability
func (e *MeasureEngine) EvaluateWithFacts(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
	cqlFacts []contracts.ClinicalFact,
) (*contracts.EngineResult, error) {
	// Inject CQL facts first
	e.SetClinicalFacts(cqlFacts)

	// Then run standard evaluation
	return e.Evaluate(ctx, execCtx)
}

// enrichWithFactEvidence adds CQL fact evidence to measure rationale.
func (e *MeasureEngine) enrichWithFactEvidence(mr contracts.MeasureResult) contracts.MeasureResult {
	if len(e.clinicalFacts) == 0 {
		return mr // No CQL facts available
	}

	// Map measures to relevant facts
	relevantFacts := e.getRelevantFacts(mr.MeasureID)
	if len(relevantFacts) == 0 {
		return mr
	}

	// Append fact evidence to rationale
	factEvidence := " [CQL Facts: "
	for i, factID := range relevantFacts {
		if fact, exists := e.clinicalFacts[factID]; exists {
			if i > 0 {
				factEvidence += ", "
			}
			factEvidence += factID + "="
			if fact.Value {
				factEvidence += "true"
			} else {
				factEvidence += "false"
			}
		}
	}
	factEvidence += "]"

	mr.Rationale += factEvidence
	return mr
}

// getRelevantFacts returns fact IDs relevant to a specific measure.
func (e *MeasureEngine) getRelevantFacts(measureID string) []string {
	// Map measures to their relevant CQL facts
	measureToFacts := map[string][]string{
		"CMS122": {
			contracts.FactHasDiabetes,
			contracts.FactHbA1cPoorControl,
			contracts.FactHbA1cGoodControl,
			contracts.FactHasOutpatientEncounter,
			contracts.FactIsEligibleAge,
		},
		"CMS165": {
			contracts.FactHasHypertension,
			contracts.FactBloodPressureControlled,
			contracts.FactBloodPressureUncontrolled,
			contracts.FactHasOutpatientEncounter,
			contracts.FactIsEligibleAge,
		},
		"CMS134": {
			contracts.FactHasDiabetes,
			contracts.FactKidneyScreeningComplete,
			contracts.FactHasACEorARB,
			contracts.FactHasCKD,
			contracts.FactHasOutpatientEncounter,
		},
		"CMS2": {
			contracts.FactDepressionScreeningComplete,
			contracts.FactPositiveDepressionScreen,
			contracts.FactFollowUpPlanDocumented,
			contracts.FactHasOutpatientEncounter,
			contracts.FactIsAdult,
		},
	}

	if facts, ok := measureToFacts[measureID]; ok {
		return facts
	}
	return nil
}

// filterMeasureIDs extracts measure IDs from requested engines.
func (e *MeasureEngine) filterMeasureIDs(requested []string) []string {
	measureList := make([]string, 0)
	for _, r := range requested {
		// Check if this is a registered measure
		if _, exists := e.evaluators[r]; exists {
			measureList = append(measureList, r)
		}
	}
	if len(measureList) == 0 {
		return e.config.DefaultMeasures
	}
	return measureList
}

// generateCareGapRecommendation creates a recommendation for care gap.
func (e *MeasureEngine) generateCareGapRecommendation(measureResult contracts.MeasureResult) contracts.Recommendation {
	// Build description with evidence trail
	description := measureResult.Rationale
	if measureResult.MeasureVersion != "" {
		description += " [Measure: " + measureResult.MeasureID + " v" + measureResult.MeasureVersion + "]"
	}

	return contracts.Recommendation{
		ID:          "REC-" + measureResult.MeasureID + "-CARE-GAP",
		Type:        "care-gap",
		Title:       "Care Gap: " + measureResult.MeasureName,
		Description: description,
		Priority:    e.determinePriority(measureResult),
		Source:      "measure-engine/" + measureResult.MeasureID,
	}
}

// determinePriority assigns priority based on measure type.
func (e *MeasureEngine) determinePriority(measureResult contracts.MeasureResult) string {
	// Inverse measures (high value = bad) are more urgent
	switch measureResult.MeasureID {
	case "CMS122": // Diabetes HbA1c >9% = poor control
		return "high"
	case "CMS165": // Uncontrolled BP
		return "high"
	case "CMS134": // Missing kidney screening
		return "medium"
	case "CMS2": // Missing depression screening
		return "medium"
	default:
		return "medium"
	}
}

// AvailableMeasures returns the list of registered measures.
func (e *MeasureEngine) AvailableMeasures() []string {
	measureList := make([]string, 0, len(e.evaluators))
	for id := range e.evaluators {
		measureList = append(measureList, id)
	}
	return measureList
}

// GetEvaluator returns a specific evaluator by ID.
func (e *MeasureEngine) GetEvaluator(measureID string) (measures.MeasureEvaluator, bool) {
	evaluator, exists := e.evaluators[measureID]
	return evaluator, exists
}
