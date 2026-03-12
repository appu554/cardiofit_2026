// Package measures - CMS134: Diabetes: Medical Attention for Nephropathy
//
// SOURCE OF TRUTH (ELM):
//   Library: KidneyHealthEvaluationFHIR
//   Version: 0.1.000
//   Path: clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS134/CMS134-DiabeticNephropathy.cql
//
// MEASURE DESCRIPTION:
//   The percentage of patients 18-85 years of age with diabetes who had a
//   kidney health evaluation during the measurement period, defined as both:
//   - An eGFR test, AND
//   - A urine albumin-creatinine ratio (uACR) test
//
// THIS IS A STANDARD MEASURE:
//   Being IN the numerator = GOOD outcome (kidney health monitored)
//   Care gap = NOT in numerator (missing kidney tests)
//
// CLINICAL LOGIC:
//   Initial Population: Age 18-85, has diabetes, has qualifying encounter
//   Denominator: Same as Initial Population
//   Denominator Exclusions: CKD Stage 5, ESRD, Hospice, Palliative Care
//   Numerator: Has both eGFR AND uACR during measurement period
//   Care Gap: Diabetic without complete kidney panel testing
//
// VALUESET DEPENDENCIES (from KB-7/KB-8):
//   - Diabetes: HasDiabetes flag
//   - eGFR: from KB-8 calculator or lab results
//   - uACR: from lab results
//
// IMPLEMENTATION NOTES:
//   - This evaluator uses ONLY precomputed data from KnowledgeSnapshot
//   - eGFR can come from KB-8 calculator or lab tests
//   - Pure function: same input → same output
package measures

import (
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS134 EVALUATOR
// ============================================================================

const (
	cms134MeasureID      = "CMS134"
	cms134MeasureName    = "Diabetes: Medical Attention for Nephropathy"
	cms134MeasureVersion = "2024.0.0"
	cms134LogicVersion   = "1.0.0"
	cms134ELMLibrary     = "KidneyHealthEvaluationFHIR:0.1.000"
	cms134MinAge         = 18
	cms134MaxAge         = 85

	// LOINC codes for kidney tests
	LoincEGFR  = "33914-3" // eGFR by CKD-EPI
	LoincUACR  = "9318-7"  // Urine albumin/creatinine ratio
)

// CMS134Evaluator implements the CMS134 Diabetic Nephropathy measure.
//
// MEASURE TYPE: Standard (higher numerator = better performance)
//
// Being in the numerator indicates:
//   - Patient had eGFR test during measurement period
//   - AND patient had uACR test during measurement period
//
// A "care gap" means:
//   - Diabetic patient missing kidney health evaluation
type CMS134Evaluator struct{}

// NewCMS134Evaluator creates a new CMS134 evaluator instance.
func NewCMS134Evaluator() *CMS134Evaluator {
	return &CMS134Evaluator{}
}

// MeasureID returns "CMS134"
func (e *CMS134Evaluator) MeasureID() string {
	return cms134MeasureID
}

// MeasureName returns the human-readable name
func (e *CMS134Evaluator) MeasureName() string {
	return cms134MeasureName
}

// MeasureVersion returns the CMS-published version
func (e *CMS134Evaluator) MeasureVersion() string {
	return cms134MeasureVersion
}

// LogicVersion returns our Go implementation version
func (e *CMS134Evaluator) LogicVersion() string {
	return cms134LogicVersion
}

// ELMCorrespondence returns the CQL library this implements
func (e *CMS134Evaluator) ELMCorrespondence() string {
	return cms134ELMLibrary
}

// Evaluate runs CMS134 measure logic against the execution context.
//
// PURE FUNCTION: No side effects, no external calls, deterministic output.
//
// Logic flow (matches CQL exactly):
//  1. Check Initial Population (age 18-85, diabetic, qualifying encounter)
//  2. Check Denominator Exclusions (CKD5, ESRD, hospice, palliative)
//  3. Check Numerator (has both eGFR AND uACR)
//  4. Determine care gap status
func (e *CMS134Evaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.MeasureResult {
	// Extract read-only evaluation context
	evalCtx := ExtractEvaluationContext(ctx)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 1: Initial Population Check
	// CQL: AgeInYearsAt(start of "Measurement Period") in Interval[18, 85]
	//      and "Has Active Diabetes Overlaps Measurement Period"
	//      and "Has Outpatient Visit During Measurement Period"
	// ─────────────────────────────────────────────────────────────────────────

	// Age check: 18-85 at start of measurement period
	if evalCtx.PatientAge < cms134MinAge || evalCtx.PatientAge > cms134MaxAge {
		return NotInInitialPopulation(
			cms134MeasureID,
			cms134MeasureName,
			fmt.Sprintf("Age %d is outside eligible range [%d-%d]", evalCtx.PatientAge, cms134MinAge, cms134MaxAge),
			e,
		)
	}

	// Diabetes check (from KB-7 ValueSet membership)
	isDiabetic := evalCtx.ValueSetMemberships["HasDiabetes"]
	if !isDiabetic {
		return NotInInitialPopulation(
			cms134MeasureID,
			cms134MeasureName,
			"Patient does not have active diabetes diagnosis",
			e,
		)
	}

	// Qualifying encounter check
	if !evalCtx.HasQualifyingEncounter {
		return NotInInitialPopulation(
			cms134MeasureID,
			cms134MeasureName,
			"No qualifying outpatient visit during measurement period",
			e,
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 2: Denominator Exclusions
	// CQL: CKD Stage 5, ESRD, Hospice, Palliative Care
	// ─────────────────────────────────────────────────────────────────────────

	if evalCtx.ValueSetMemberships["HasCKDStage5"] || evalCtx.ValueSetMemberships["HasESRD"] {
		return DenominatorExclusion(
			cms134MeasureID,
			cms134MeasureName,
			"Excluded due to CKD Stage 5 or ESRD diagnosis",
			e,
		)
	}

	if evalCtx.ValueSetMemberships["InHospice"] || evalCtx.ValueSetMemberships["InPalliativeCare"] {
		return DenominatorExclusion(
			cms134MeasureID,
			cms134MeasureName,
			"Excluded due to hospice or palliative care",
			e,
		)
	}

	// Patient is in Denominator
	// ─────────────────────────────────────────────────────────────────────────
	// STEP 3: Numerator Check
	// CQL: "Has Kidney Panel Performed During Measurement Period"
	//      = exists eGFR test AND exists uACR test
	// ─────────────────────────────────────────────────────────────────────────

	// Check for eGFR (can come from KB-8 calculator or lab)
	hasEGFR := evalCtx.LatestEGFR != nil || hasLabTest(ctx.Patient.RecentLabResults, LoincEGFR)

	// Check for uACR (must come from lab)
	hasUACR := hasLabTest(ctx.Patient.RecentLabResults, LoincUACR)

	// Need BOTH tests for numerator
	if hasEGFR && hasUACR {
		return e.buildNumeratorResult(
			hasEGFR, hasUACR,
			"Patient has complete kidney health evaluation (eGFR and uACR) during measurement period",
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 4: Care Gap - Missing kidney panel tests
	// ─────────────────────────────────────────────────────────────────────────

	var missingTests string
	if !hasEGFR && !hasUACR {
		missingTests = "missing both eGFR and uACR tests"
	} else if !hasEGFR {
		missingTests = "missing eGFR test"
	} else {
		missingTests = "missing uACR test"
	}

	return e.buildCareGapResult(
		hasEGFR, hasUACR,
		fmt.Sprintf("Diabetic patient %s during measurement period", missingTests),
	)
}

// hasLabTest checks if any lab result has the given LOINC code
func hasLabTest(labs []contracts.LabResult, loincCode string) bool {
	for _, lab := range labs {
		if lab.Code.Code == loincCode && lab.Value != nil {
			return true
		}
	}
	return false
}

// buildNumeratorResult creates result for patient IN numerator (complete kidney panel).
func (e *CMS134Evaluator) buildNumeratorResult(hasEGFR, hasUACR bool, rationale string) contracts.MeasureResult {
	resources := []string{"Patient"}
	if hasEGFR {
		resources = append(resources, "Observation/eGFR")
	}
	if hasUACR {
		resources = append(resources, "Observation/uACR")
	}

	return contracts.MeasureResult{
		MeasureID:           cms134MeasureID,
		MeasureName:         cms134MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         true,
		CareGapIdentified:   false,
		MeasureVersion:      cms134MeasureVersion,
		LogicVersion:        cms134LogicVersion,
		ELMCorrespondence:   cms134ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  resources,
	}
}

// buildCareGapResult creates result for patient with care gap (missing tests).
func (e *CMS134Evaluator) buildCareGapResult(hasEGFR, hasUACR bool, rationale string) contracts.MeasureResult {
	resources := []string{"Patient"}
	if hasEGFR {
		resources = append(resources, "Observation/eGFR")
	}
	if hasUACR {
		resources = append(resources, "Observation/uACR")
	}

	return contracts.MeasureResult{
		MeasureID:           cms134MeasureID,
		MeasureName:         cms134MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         false,
		CareGapIdentified:   true,
		MeasureVersion:      cms134MeasureVersion,
		LogicVersion:        cms134LogicVersion,
		ELMCorrespondence:   cms134ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  resources,
	}
}

// ============================================================================
// CMS134 GOLDEN TEST HELPERS
// ============================================================================

// CMS134TestCase represents a test scenario for CMS134.
type CMS134TestCase struct {
	Name                    string
	Age                     int
	IsDiabetic              bool
	HasEGFR                 bool
	HasUACR                 bool
	HasQualifyingEncounter  bool
	// Exclusion flags
	HasCKDStage5            bool
	HasESRD                 bool
	InHospice               bool
	// Expected outcomes
	ExpectedInPopulation    bool
	ExpectedExcluded        bool
	ExpectedInNumerator     bool
	ExpectedCareGap         bool
}

// CMS134GoldenTestCases returns the canonical test cases for CMS134.
func CMS134GoldenTestCases() []CMS134TestCase {
	return []CMS134TestCase{
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 1: Complete kidney panel → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_CompletePanel_HasBothTests",
			Age:                    55,
			IsDiabetic:             true,
			HasEGFR:                true,
			HasUACR:                true,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    true,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 2: Missing uACR → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_MissingUACR_CareGap",
			Age:                    60,
			IsDiabetic:             true,
			HasEGFR:                true,
			HasUACR:                false,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Missing eGFR → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_MissingEGFR_CareGap",
			Age:                    50,
			IsDiabetic:             true,
			HasEGFR:                false,
			HasUACR:                true,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Missing both tests → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_MissingBothTests_CareGap",
			Age:                    45,
			IsDiabetic:             true,
			HasEGFR:                false,
			HasUACR:                false,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Non-diabetic
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "NonDiabetic_NotInPopulation",
			Age:                    55,
			IsDiabetic:             false,
			HasEGFR:                true,
			HasUACR:                true,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// DENOMINATOR EXCLUSION: ESRD
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_ESRD_Excluded",
			Age:                    60,
			IsDiabetic:             true,
			HasEGFR:                false,
			HasUACR:                false,
			HasQualifyingEncounter: true,
			HasESRD:                true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       true,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Age too young
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_Age17_TooYoung",
			Age:                    17,
			IsDiabetic:             true,
			HasEGFR:                true,
			HasUACR:                true,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Age too old
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_Age86_TooOld",
			Age:                    86,
			IsDiabetic:             true,
			HasEGFR:                true,
			HasUACR:                true,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},
	}
}
