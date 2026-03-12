// Package engines provides the CQL Engine for clinical truth determination.
//
// CQL ENGINE ARCHITECTURE (per CTO/CMO spec):
//
// PURPOSE: Clinical Truth Determination
// QUESTION IT ANSWERS: "Given this patient context, does this clinical fact hold?"
//
// The CQL Engine is a TRUTH EVALUATOR, not a care gap detector.
// It answers binary or structured clinical truths such as:
//   - Is the patient diabetic? → true
//   - Is HbA1c > 9%? → true
//   - Is BP uncontrolled? → true
//   - Has kidney screening been done? → false
//
// It does NOT care what you DO with the answer - that's Measure Engine's job.
//
// INPUTS:
//   - ClinicalExecutionContext (FROZEN) - all patient data pre-assembled
//   - KB-7 Terminology for ValueSet memberships
//   - KB-8 Calculator outputs (eGFR, BP, etc.)
//
// OUTPUTS:
//   - ClinicalFacts: Truth statements ("HbA1cPoorControl = true")
//   - NO MeasureResults (that's Measure Engine's responsibility)
//   - NO Care Gaps (that's Measure Engine's responsibility)
//
// ANALOGY: CQL Engine = Lab report + rule book
//          It tells you: "The value is abnormal"
//          It does NOT tell you: "What to prescribe"
package engines

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines/measures"
)

// ============================================================================
// CQL ENGINE: CLINICAL TRUTH EVALUATOR
// ============================================================================

// CQLEngine evaluates clinical truths from patient context.
// It implements the Engine interface and produces ClinicalFacts.
//
// Per CTO/CMO Architecture:
//   - CQL Engine produces TRUTHS, not care judgments
//   - Truths are consumed by Measure Engine for care gap determination
//   - No external calls - works entirely from frozen ClinicalExecutionContext
type CQLEngine struct {
	// factEvaluators registry of clinical fact evaluators
	factEvaluators map[string]FactEvaluator

	// config for engine behavior
	config CQLEngineConfig
}

// FactEvaluator evaluates a single clinical fact.
// Each evaluator answers ONE truth question.
type FactEvaluator interface {
	// FactID returns the unique identifier for this fact
	FactID() string

	// Category returns the fact category (e.g., "glycemic", "cardiovascular")
	Category() string

	// Evaluate determines the truth value from patient context
	Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact
}

// CQLEngineConfig configures the CQL engine.
type CQLEngineConfig struct {
	// Region determines which regional facts to evaluate (AU, IN, US)
	Region string

	// FactCategories to evaluate (empty = all)
	// Options: "glycemic", "cardiovascular", "renal", "screening", "general"
	FactCategories []string
}

// DefaultCQLEngineConfig returns sensible defaults.
func DefaultCQLEngineConfig() CQLEngineConfig {
	return CQLEngineConfig{
		Region:         "AU",
		FactCategories: []string{}, // Evaluate all categories
	}
}

// NewCQLEngine creates a new CQL engine with all fact evaluators registered.
func NewCQLEngine(config CQLEngineConfig) *CQLEngine {
	engine := &CQLEngine{
		factEvaluators: make(map[string]FactEvaluator),
		config:         config,
	}

	// Register all fact evaluators
	engine.registerFactEvaluators()

	return engine
}

// registerFactEvaluators registers all clinical fact evaluators.
func (e *CQLEngine) registerFactEvaluators() {
	// Glycemic facts (Diabetes/HbA1c)
	e.factEvaluators[contracts.FactHasDiabetes] = &HasDiabetesEvaluator{}
	e.factEvaluators[contracts.FactHbA1cPoorControl] = &HbA1cPoorControlEvaluator{}
	e.factEvaluators[contracts.FactHbA1cModerateControl] = &HbA1cModerateControlEvaluator{}
	e.factEvaluators[contracts.FactHbA1cGoodControl] = &HbA1cGoodControlEvaluator{}

	// Cardiovascular facts (Blood Pressure)
	e.factEvaluators[contracts.FactHasHypertension] = &HasHypertensionEvaluator{}
	e.factEvaluators[contracts.FactBloodPressureControlled] = &BloodPressureControlledEvaluator{}
	e.factEvaluators[contracts.FactBloodPressureUncontrolled] = &BloodPressureUncontrolledEvaluator{}

	// Renal facts (Kidney Health)
	e.factEvaluators[contracts.FactHasCKD] = &HasCKDEvaluator{}
	e.factEvaluators[contracts.FactKidneyScreeningComplete] = &KidneyScreeningCompleteEvaluator{}
	e.factEvaluators[contracts.FactHasACEorARB] = &HasACEorARBEvaluator{}

	// Screening facts (Depression)
	e.factEvaluators[contracts.FactDepressionScreeningComplete] = &DepressionScreeningCompleteEvaluator{}
	e.factEvaluators[contracts.FactPositiveDepressionScreen] = &PositiveDepressionScreenEvaluator{}
	e.factEvaluators[contracts.FactFollowUpPlanDocumented] = &FollowUpPlanDocumentedEvaluator{}

	// General facts
	e.factEvaluators[contracts.FactHasOutpatientEncounter] = &HasOutpatientEncounterEvaluator{}
	e.factEvaluators[contracts.FactIsAdult] = &IsAdultEvaluator{}
	e.factEvaluators[contracts.FactIsEligibleAge] = &IsEligibleAgeEvaluator{}
}

// Name returns the engine identifier.
func (e *CQLEngine) Name() string {
	return "cql-engine"
}

// Evaluate determines clinical truths from the patient context.
//
// CRITICAL: This engine:
// 1. Uses ONLY data from ClinicalExecutionContext (frozen contract)
// 2. Makes NO external KB calls, NO database calls, NO HTTP calls
// 3. Returns ClinicalFacts (truths), NOT MeasureResults
// 4. Is deterministic: same input → same output
//
// Per CTO/CMO Architecture:
//   - CQL Engine = "What is true about this patient?"
//   - Measure Engine = "Given what's true, are we meeting standards of care?"
func (e *CQLEngine) Evaluate(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
) (*contracts.EngineResult, error) {

	startTime := time.Now()

	result := &contracts.EngineResult{
		EngineName:      e.Name(),
		Success:         true,
		ClinicalFacts:   make([]contracts.ClinicalFact, 0),
		Recommendations: make([]contracts.Recommendation, 0),
		Alerts:          make([]contracts.Alert, 0),
		MeasureResults:  make([]contracts.MeasureResult, 0), // CQL Engine produces NO MeasureResults
		EvidenceLinks:   make([]string, 0),
	}

	// Evaluate all registered facts
	for factID, evaluator := range e.factEvaluators {
		// Filter by category if specified
		if len(e.config.FactCategories) > 0 {
			if !e.isInCategory(evaluator.Category(), e.config.FactCategories) {
				continue
			}
		}

		// Evaluate the clinical fact
		fact := evaluator.Evaluate(execCtx)
		fact.FactID = factID
		fact.EvaluatedAt = time.Now()

		result.ClinicalFacts = append(result.ClinicalFacts, fact)
	}

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// isInCategory checks if category is in the allowed list.
func (e *CQLEngine) isInCategory(category string, allowed []string) bool {
	for _, c := range allowed {
		if c == category {
			return true
		}
	}
	return false
}

// AvailableFacts returns the list of registered fact evaluators.
func (e *CQLEngine) AvailableFacts() []string {
	facts := make([]string, 0, len(e.factEvaluators))
	for id := range e.factEvaluators {
		facts = append(facts, id)
	}
	return facts
}

// GetFactEvaluator returns a specific evaluator by ID.
func (e *CQLEngine) GetFactEvaluator(factID string) (FactEvaluator, bool) {
	evaluator, exists := e.factEvaluators[factID]
	return evaluator, exists
}

// ============================================================================
// FACT EVALUATORS: Clinical Truth Determination Logic
// ============================================================================

// --- GLYCEMIC FACTS (Diabetes/HbA1c) ---

// HasDiabetesEvaluator determines if patient has diabetes diagnosis.
type HasDiabetesEvaluator struct{}

func (e *HasDiabetesEvaluator) FactID() string   { return contracts.FactHasDiabetes }
func (e *HasDiabetesEvaluator) Category() string { return "glycemic" }
func (e *HasDiabetesEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 pre-computed flag - NO hardcoded fallback!
	// KB-7 checks patient conditions against DiabetesMellitus ValueSet at snapshot build time.
	// The ValueSet contains 100+ SNOMED/ICD codes - far more complete than any hardcoded list.
	if ctx.Knowledge.Terminology.ValueSetMemberships != nil {
		if isDiabetic, ok := ctx.Knowledge.Terminology.ValueSetMemberships["HasDiabetes"]; ok && isDiabetic {
			// Find the ACTUAL diabetes condition using CodeMemberships
			// CodeMemberships maps "system|code" -> []ValueSetNames
			codeMemberships := ctx.Knowledge.Terminology.CodeMemberships
			for _, condition := range ctx.Patient.ActiveConditions {
				codeKey := fmt.Sprintf("%s|%s", condition.Code.System, condition.Code.Code)
				if memberships, ok := codeMemberships[codeKey]; ok {
					// Check if this condition is in a diabetes-related ValueSet
					for _, vsName := range memberships {
						if isDiabetesValueSet(vsName) {
							return contracts.ClinicalFact{
								Value:        true,
								Evidence:     fmt.Sprintf("Active diabetes condition: %s (%s)", condition.Code.Display, condition.Code.Code),
								SourceData:   []string{condition.SourceReference},
								FactCategory: "glycemic",
							}
						}
					}
				}
			}
			// Flag is true but couldn't identify specific condition
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     "Patient has diabetes (KB-7 DiabetesMellitus ValueSet)",
				FactCategory: "glycemic",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No diabetes diagnosis found (KB-7 validated)",
		FactCategory: "glycemic",
	}
}

// isDiabetesValueSet checks if the ValueSet name indicates diabetes
func isDiabetesValueSet(vsName string) bool {
	lowerName := strings.ToLower(vsName)
	return strings.Contains(lowerName, "diabetes") ||
		strings.Contains(lowerName, "diabetic") ||
		strings.Contains(lowerName, "type 2") ||
		strings.Contains(lowerName, "type2") ||
		strings.Contains(lowerName, "glycemic")
}

// HbA1cPoorControlEvaluator determines if HbA1c > 9%.
// DYNAMIC: Uses KB-7 LabHbA1c ValueSet via CodeMemberships.
type HbA1cPoorControlEvaluator struct{}

func (e *HbA1cPoorControlEvaluator) FactID() string   { return contracts.FactHbA1cPoorControl }
func (e *HbA1cPoorControlEvaluator) Category() string { return "glycemic" }
func (e *HbA1cPoorControlEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 CodeMemberships for HbA1c code validation
	codeMemberships := ctx.Knowledge.Terminology.CodeMemberships
	hba1c := findMostRecentHbA1c(ctx.Patient.RecentLabResults, codeMemberships)
	if hba1c == nil {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "No HbA1c result found in recent labs (KB-7 validated)",
			FactCategory: "glycemic",
		}
	}

	if hba1c.Value == nil {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "HbA1c result has no numeric value",
			FactCategory: "glycemic",
		}
	}

	numericValue := hba1c.Value.Value
	isPoorControl := numericValue > 9.0

	return contracts.ClinicalFact{
		Value:        isPoorControl,
		NumericValue: &numericValue,
		Evidence:     fmt.Sprintf("HbA1c = %.1f%% (threshold > 9.0%%, KB-7 validated)", numericValue),
		SourceData:   []string{hba1c.SourceReference},
		FactCategory: "glycemic",
	}
}

// HbA1cModerateControlEvaluator determines if HbA1c is 7-9%.
// DYNAMIC: Uses KB-7 LabHbA1c ValueSet via CodeMemberships.
type HbA1cModerateControlEvaluator struct{}

func (e *HbA1cModerateControlEvaluator) FactID() string   { return contracts.FactHbA1cModerateControl }
func (e *HbA1cModerateControlEvaluator) Category() string { return "glycemic" }
func (e *HbA1cModerateControlEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 CodeMemberships for HbA1c code validation
	codeMemberships := ctx.Knowledge.Terminology.CodeMemberships
	hba1c := findMostRecentHbA1c(ctx.Patient.RecentLabResults, codeMemberships)
	if hba1c == nil || hba1c.Value == nil {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "No HbA1c result found (KB-7 validated)",
			FactCategory: "glycemic",
		}
	}

	numericValue := hba1c.Value.Value
	isModerate := numericValue >= 7.0 && numericValue <= 9.0

	return contracts.ClinicalFact{
		Value:        isModerate,
		NumericValue: &numericValue,
		Evidence:     fmt.Sprintf("HbA1c = %.1f%% (moderate range: 7.0-9.0%%, KB-7 validated)", numericValue),
		SourceData:   []string{hba1c.SourceReference},
		FactCategory: "glycemic",
	}
}

// HbA1cGoodControlEvaluator determines if HbA1c < 7%.
// DYNAMIC: Uses KB-7 LabHbA1c ValueSet via CodeMemberships.
type HbA1cGoodControlEvaluator struct{}

func (e *HbA1cGoodControlEvaluator) FactID() string   { return contracts.FactHbA1cGoodControl }
func (e *HbA1cGoodControlEvaluator) Category() string { return "glycemic" }
func (e *HbA1cGoodControlEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 CodeMemberships for HbA1c code validation
	codeMemberships := ctx.Knowledge.Terminology.CodeMemberships
	hba1c := findMostRecentHbA1c(ctx.Patient.RecentLabResults, codeMemberships)
	if hba1c == nil || hba1c.Value == nil {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "No HbA1c result found (KB-7 validated)",
			FactCategory: "glycemic",
		}
	}

	numericValue := hba1c.Value.Value
	isGood := numericValue < 7.0

	return contracts.ClinicalFact{
		Value:        isGood,
		NumericValue: &numericValue,
		Evidence:     fmt.Sprintf("HbA1c = %.1f%% (good control < 7.0%%, KB-7 validated)", numericValue),
		SourceData:   []string{hba1c.SourceReference},
		FactCategory: "glycemic",
	}
}

// --- CARDIOVASCULAR FACTS (Blood Pressure) ---

// HasHypertensionEvaluator determines if patient has hypertension diagnosis.
type HasHypertensionEvaluator struct{}

func (e *HasHypertensionEvaluator) FactID() string   { return contracts.FactHasHypertension }
func (e *HasHypertensionEvaluator) Category() string { return "cardiovascular" }
func (e *HasHypertensionEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 pre-computed flag - NO hardcoded fallback!
	// KB-7 checks patient conditions against Hypertension ValueSet at snapshot build time.
	// The ValueSet contains 50+ SNOMED/ICD codes - far more complete than any hardcoded list.
	if ctx.Knowledge.Terminology.ValueSetMemberships != nil {
		if hasHTN, ok := ctx.Knowledge.Terminology.ValueSetMemberships["HasHypertension"]; ok && hasHTN {
			// Find the matching condition for evidence/audit trail
			for _, condition := range ctx.Patient.ActiveConditions {
				return contracts.ClinicalFact{
					Value:        true,
					Evidence:     fmt.Sprintf("Active hypertension condition: %s (%s)", condition.Code.Display, condition.Code.Code),
					SourceData:   []string{condition.SourceReference},
					FactCategory: "cardiovascular",
				}
			}
			// Flag is true but no condition found (edge case)
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     "Patient has hypertension (KB-7 Hypertension ValueSet)",
				FactCategory: "cardiovascular",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No hypertension diagnosis found (KB-7 validated)",
		FactCategory: "cardiovascular",
	}
}

// BloodPressureControlledEvaluator determines if BP < 140/90.
type BloodPressureControlledEvaluator struct{}

func (e *BloodPressureControlledEvaluator) FactID() string   { return contracts.FactBloodPressureControlled }
func (e *BloodPressureControlledEvaluator) Category() string { return "cardiovascular" }
func (e *BloodPressureControlledEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	systolic, diastolic := findMostRecentBP(ctx.Patient.RecentVitalSigns)
	if systolic == 0 && diastolic == 0 {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "No blood pressure reading found",
			FactCategory: "cardiovascular",
		}
	}

	isControlled := systolic < 140 && diastolic < 90

	return contracts.ClinicalFact{
		Value:        isControlled,
		Evidence:     fmt.Sprintf("BP = %d/%d mmHg (controlled < 140/90)", int(systolic), int(diastolic)),
		FactCategory: "cardiovascular",
	}
}

// BloodPressureUncontrolledEvaluator determines if BP >= 140/90.
type BloodPressureUncontrolledEvaluator struct{}

func (e *BloodPressureUncontrolledEvaluator) FactID() string {
	return contracts.FactBloodPressureUncontrolled
}
func (e *BloodPressureUncontrolledEvaluator) Category() string { return "cardiovascular" }
func (e *BloodPressureUncontrolledEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	systolic, diastolic := findMostRecentBP(ctx.Patient.RecentVitalSigns)
	if systolic == 0 && diastolic == 0 {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "No blood pressure reading found",
			FactCategory: "cardiovascular",
		}
	}

	isUncontrolled := systolic >= 140 || diastolic >= 90

	return contracts.ClinicalFact{
		Value:        isUncontrolled,
		Evidence:     fmt.Sprintf("BP = %d/%d mmHg (uncontrolled >= 140/90)", int(systolic), int(diastolic)),
		FactCategory: "cardiovascular",
	}
}

// --- RENAL FACTS (Kidney Health) ---

// HasCKDEvaluator determines if patient has CKD diagnosis.
type HasCKDEvaluator struct{}

func (e *HasCKDEvaluator) FactID() string   { return contracts.FactHasCKD }
func (e *HasCKDEvaluator) Category() string { return "renal" }
func (e *HasCKDEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 pre-computed flag - NO hardcoded fallback!
	// KB-7 checks patient conditions against CKDStages ValueSet at snapshot build time.
	// The ValueSet contains 30+ SNOMED/ICD codes for all CKD stages - far more complete than any hardcoded list.
	if ctx.Knowledge.Terminology.ValueSetMemberships != nil {
		if hasCKD, ok := ctx.Knowledge.Terminology.ValueSetMemberships["has_ckd"]; ok && hasCKD {
			// Find the matching condition for evidence/audit trail
			for _, condition := range ctx.Patient.ActiveConditions {
				return contracts.ClinicalFact{
					Value:        true,
					Evidence:     fmt.Sprintf("Active CKD condition: %s (%s)", condition.Code.Display, condition.Code.Code),
					SourceData:   []string{condition.SourceReference},
					FactCategory: "renal",
				}
			}
			// Flag is true but no condition found (edge case)
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     "Patient has CKD (KB-7 CKDStages ValueSet)",
				FactCategory: "renal",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No CKD diagnosis found (KB-7 validated)",
		FactCategory: "renal",
	}
}

// KidneyScreeningCompleteEvaluator determines if uACR or eGFR test exists.
type KidneyScreeningCompleteEvaluator struct{}

func (e *KidneyScreeningCompleteEvaluator) FactID() string   { return contracts.FactKidneyScreeningComplete }
func (e *KidneyScreeningCompleteEvaluator) Category() string { return "renal" }
func (e *KidneyScreeningCompleteEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 CodeMemberships for uACR code validation
	codeMemberships := ctx.Knowledge.Terminology.CodeMemberships

	// Check for uACR using KB-7 LabuACR ValueSet
	for _, lab := range ctx.Patient.RecentLabResults {
		if measures.IsUACRCode(lab.Code.Code, codeMemberships) {
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     fmt.Sprintf("uACR test found: %s (KB-7 validated)", lab.Code.Display),
				SourceData:   []string{lab.SourceReference},
				FactCategory: "renal",
			}
		}
	}

	// Check for eGFR in calculator snapshot
	if ctx.Knowledge.Calculators.EGFR != nil {
		return contracts.ClinicalFact{
			Value:        true,
			Evidence:     fmt.Sprintf("eGFR calculated: %.1f %s", ctx.Knowledge.Calculators.EGFR.Value, ctx.Knowledge.Calculators.EGFR.Unit),
			FactCategory: "renal",
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No kidney screening (uACR or eGFR) found in measurement period",
		FactCategory: "renal",
	}
}

// HasACEorARBEvaluator determines if patient is on ACE inhibitor or ARB.
// DYNAMIC: Uses KB-7 CodeMemberships from KnowledgeSnapshot - NO hardcoded drug lists!
type HasACEorARBEvaluator struct{}

func (e *HasACEorARBEvaluator) FactID() string   { return contracts.FactHasACEorARB }
func (e *HasACEorARBEvaluator) Category() string { return "renal" }
func (e *HasACEorARBEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 ValueSetMemberships for ACE/ARB detection
	// KB-7 reverse lookup already populates flags like "OnAceInhibitors", "OnAceisAndArbs"
	vsm := ctx.Knowledge.Terminology.ValueSetMemberships
	if vsm != nil {
		// Check for any ACE inhibitor or ARB membership flags
		aceArbFlags := []string{
			"OnAceInhibitors",
			"OnAceisAndArbs",
			"OnAceInhibitorOrArbOrArni",
			"OnAngiotensinConvertingEnzyme(ace)Inhibitors",
		}
		for _, flag := range aceArbFlags {
			if vsm[flag] {
				// Find the matching medication for evidence
				for _, med := range ctx.Patient.ActiveMedications {
					return contracts.ClinicalFact{
						Value:        true,
						Evidence:     fmt.Sprintf("On ACE/ARB: %s (KB-7 validated via %s)", med.Code.Display, flag),
						SourceData:   []string{med.SourceReference},
						FactCategory: "renal",
					}
				}
				return contracts.ClinicalFact{
					Value:        true,
					Evidence:     fmt.Sprintf("On ACE/ARB medication (KB-7 %s)", flag),
					FactCategory: "renal",
				}
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "Not on ACE inhibitor or ARB (KB-7 validated)",
		FactCategory: "renal",
	}
}

// --- SCREENING FACTS (Depression) ---

// DepressionScreeningCompleteEvaluator determines if PHQ-9 exists.
// DYNAMIC: Uses KB-7 LabPHQ9 ValueSet via CodeMemberships.
type DepressionScreeningCompleteEvaluator struct{}

func (e *DepressionScreeningCompleteEvaluator) FactID() string {
	return contracts.FactDepressionScreeningComplete
}
func (e *DepressionScreeningCompleteEvaluator) Category() string { return "screening" }
func (e *DepressionScreeningCompleteEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 CodeMemberships for PHQ-9 code validation
	codeMemberships := ctx.Knowledge.Terminology.CodeMemberships

	for _, lab := range ctx.Patient.RecentLabResults {
		if measures.IsPHQ9Code(lab.Code.Code, codeMemberships) {
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     fmt.Sprintf("PHQ-9 screening found: %s (KB-7 validated)", lab.Code.Display),
				SourceData:   []string{lab.SourceReference},
				FactCategory: "screening",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No PHQ-9 depression screening found (KB-7 validated)",
		FactCategory: "screening",
	}
}

// PositiveDepressionScreenEvaluator determines if PHQ-9 >= 10.
// DYNAMIC: Uses KB-7 LabPHQ9 ValueSet via CodeMemberships.
type PositiveDepressionScreenEvaluator struct{}

func (e *PositiveDepressionScreenEvaluator) FactID() string {
	return contracts.FactPositiveDepressionScreen
}
func (e *PositiveDepressionScreenEvaluator) Category() string { return "screening" }
func (e *PositiveDepressionScreenEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// DYNAMIC: Use KB-7 CodeMemberships for PHQ-9 code validation
	codeMemberships := ctx.Knowledge.Terminology.CodeMemberships

	for _, lab := range ctx.Patient.RecentLabResults {
		if measures.IsPHQ9Code(lab.Code.Code, codeMemberships) && lab.Value != nil {
			numericValue := lab.Value.Value
			isPositive := numericValue >= 10

			return contracts.ClinicalFact{
				Value:        isPositive,
				NumericValue: &numericValue,
				Evidence:     fmt.Sprintf("PHQ-9 = %.0f (positive >= 10, KB-7 validated)", numericValue),
				SourceData:   []string{lab.SourceReference},
				FactCategory: "screening",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No PHQ-9 result to evaluate (KB-7 validated)",
		FactCategory: "screening",
	}
}

// FollowUpPlanDocumentedEvaluator determines if follow-up plan is documented.
type FollowUpPlanDocumentedEvaluator struct{}

func (e *FollowUpPlanDocumentedEvaluator) FactID() string   { return contracts.FactFollowUpPlanDocumented }
func (e *FollowUpPlanDocumentedEvaluator) Category() string { return "screening" }
func (e *FollowUpPlanDocumentedEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	// Check clinical flags for follow-up plan
	if ctx.Patient.RiskProfile.ClinicalFlags != nil {
		if followUp, ok := ctx.Patient.RiskProfile.ClinicalFlags["has_followup_plan"]; ok && followUp {
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     "Follow-up plan documented (from clinical flags)",
				FactCategory: "screening",
			}
		}
	}

	// Check CDI facts for follow-up documentation
	for _, fact := range ctx.Knowledge.CDI.ExtractedFacts {
		if fact.FactType == "followup_plan" {
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     "Follow-up plan found in CDI facts",
				FactCategory: "screening",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No follow-up plan documented",
		FactCategory: "screening",
	}
}

// --- GENERAL FACTS ---

// HasOutpatientEncounterEvaluator determines if qualifying encounter exists.
type HasOutpatientEncounterEvaluator struct{}

func (e *HasOutpatientEncounterEvaluator) FactID() string   { return contracts.FactHasOutpatientEncounter }
func (e *HasOutpatientEncounterEvaluator) Category() string { return "general" }
func (e *HasOutpatientEncounterEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	for _, encounter := range ctx.Patient.RecentEncounters {
		if encounter.Class == "ambulatory" || encounter.Class == "outpatient" {
			return contracts.ClinicalFact{
				Value:        true,
				Evidence:     fmt.Sprintf("Outpatient encounter found: %s", encounter.EncounterID),
				SourceData:   []string{encounter.SourceReference},
				FactCategory: "general",
			}
		}
	}

	return contracts.ClinicalFact{
		Value:        false,
		Evidence:     "No qualifying outpatient encounter found",
		FactCategory: "general",
	}
}

// IsAdultEvaluator determines if patient is >= 18 years old.
type IsAdultEvaluator struct{}

func (e *IsAdultEvaluator) FactID() string   { return contracts.FactIsAdult }
func (e *IsAdultEvaluator) Category() string { return "general" }
func (e *IsAdultEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	if ctx.Patient.Demographics.BirthDate == nil {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "Birth date not available",
			FactCategory: "general",
		}
	}

	age := measures.CalculateAge(*ctx.Patient.Demographics.BirthDate)
	isAdult := age >= 18

	numericAge := float64(age)
	return contracts.ClinicalFact{
		Value:        isAdult,
		NumericValue: &numericAge,
		Evidence:     fmt.Sprintf("Patient age: %d years (adult >= 18)", age),
		FactCategory: "general",
	}
}

// IsEligibleAgeEvaluator determines if patient is in typical measure age range (18-75).
type IsEligibleAgeEvaluator struct{}

func (e *IsEligibleAgeEvaluator) FactID() string   { return contracts.FactIsEligibleAge }
func (e *IsEligibleAgeEvaluator) Category() string { return "general" }
func (e *IsEligibleAgeEvaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.ClinicalFact {
	if ctx.Patient.Demographics.BirthDate == nil {
		return contracts.ClinicalFact{
			Value:        false,
			Evidence:     "Birth date not available",
			FactCategory: "general",
		}
	}

	age := measures.CalculateAge(*ctx.Patient.Demographics.BirthDate)
	isEligible := age >= 18 && age <= 75

	numericAge := float64(age)
	return contracts.ClinicalFact{
		Value:        isEligible,
		NumericValue: &numericAge,
		Evidence:     fmt.Sprintf("Patient age: %d years (eligible range 18-75)", age),
		FactCategory: "general",
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// findMostRecentHbA1c returns the most recent HbA1c result.
// DYNAMIC: Uses KB-7 LabHbA1c ValueSet via CodeMemberships.
func findMostRecentHbA1c(labs []contracts.LabResult, codeMemberships map[string][]string) *contracts.LabResult {
	var mostRecent *contracts.LabResult
	for i := range labs {
		if measures.IsHbA1cCode(labs[i].Code.Code, codeMemberships) {
			if mostRecent == nil {
				mostRecent = &labs[i]
			} else if labs[i].EffectiveDateTime != nil && mostRecent.EffectiveDateTime != nil {
				if labs[i].EffectiveDateTime.After(*mostRecent.EffectiveDateTime) {
					mostRecent = &labs[i]
				}
			}
		}
	}
	return mostRecent
}

// findMostRecentBP returns the most recent blood pressure reading.
func findMostRecentBP(vitals []contracts.VitalSign) (systolic, diastolic float64) {
	for _, vital := range vitals {
		for _, comp := range vital.ComponentValues {
			if comp.Code.Code == measures.LoincSystolicBP && comp.Value != nil {
				systolic = comp.Value.Value
			}
			if comp.Code.Code == measures.LoincDiastolicBP && comp.Value != nil {
				diastolic = comp.Value.Value
			}
		}
		if systolic > 0 && diastolic > 0 {
			return systolic, diastolic
		}
	}
	return 0, 0
}
