// Package measures - CMS165: Controlling High Blood Pressure
//
// SOURCE OF TRUTH (ELM):
//   Library: ControllingHighBloodPressureFHIR
//   Version: 0.1.000
//   Path: clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS165/CMS165-BloodPressure.cql
//
// MEASURE DESCRIPTION:
//   Percentage of patients 18-85 years of age who had a diagnosis of essential
//   hypertension starting within the first six months of the measurement period
//   or any time prior to the measurement period, and whose most recent blood
//   pressure was adequately controlled (systolic <140 and diastolic <90).
//
// THIS IS A STANDARD MEASURE:
//   Being IN the numerator = GOOD outcome (controlled BP)
//   Care gap = NOT in numerator (uncontrolled BP)
//
// CLINICAL LOGIC:
//   Initial Population: Age 18-85, has hypertension, has qualifying encounter
//   Denominator: Same as Initial Population
//   Denominator Exclusions: ESRD, CKD Stage 5, Pregnancy, Hospice, Palliative Care
//   Numerator: Most recent BP: Systolic < 140 AND Diastolic < 90
//   Care Gap: In denominator but BP not adequately controlled
//
// VALUESET DEPENDENCIES (from KB-7):
//   - Essential Hypertension: HasHypertension flag
//
// IMPLEMENTATION NOTES:
//   - This evaluator uses ONLY precomputed data from KnowledgeSnapshot
//   - BP values come from RecentVitalSigns with LOINC codes
//   - Pure function: same input → same output
package measures

import (
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS165 EVALUATOR
// ============================================================================

const (
	cms165MeasureID        = "CMS165"
	cms165MeasureName      = "Controlling High Blood Pressure"
	cms165MeasureVersion   = "2024.0.0"
	cms165LogicVersion     = "1.0.0"
	cms165ELMLibrary       = "ControllingHighBloodPressureFHIR:0.1.000"
	cms165SystolicThreshold  = 140.0 // Systolic < 140 = controlled
	cms165DiastolicThreshold = 90.0  // Diastolic < 90 = controlled
	cms165MinAge           = 18
	cms165MaxAge           = 85
)

// CMS165Evaluator implements the CMS165 Controlling High Blood Pressure measure.
//
// MEASURE TYPE: Standard (higher numerator = better performance)
//
// Being in the numerator indicates:
//   - Most recent systolic BP < 140 mmHg
//   - AND most recent diastolic BP < 90 mmHg
//
// A "care gap" means:
//   - Patient has uncontrolled blood pressure
//   - OR no recent BP measurement on record
type CMS165Evaluator struct{}

// NewCMS165Evaluator creates a new CMS165 evaluator instance.
func NewCMS165Evaluator() *CMS165Evaluator {
	return &CMS165Evaluator{}
}

// MeasureID returns "CMS165"
func (e *CMS165Evaluator) MeasureID() string {
	return cms165MeasureID
}

// MeasureName returns the human-readable name
func (e *CMS165Evaluator) MeasureName() string {
	return cms165MeasureName
}

// MeasureVersion returns the CMS-published version
func (e *CMS165Evaluator) MeasureVersion() string {
	return cms165MeasureVersion
}

// LogicVersion returns our Go implementation version
func (e *CMS165Evaluator) LogicVersion() string {
	return cms165LogicVersion
}

// ELMCorrespondence returns the CQL library this implements
func (e *CMS165Evaluator) ELMCorrespondence() string {
	return cms165ELMLibrary
}

// Evaluate runs CMS165 measure logic against the execution context.
//
// PURE FUNCTION: No side effects, no external calls, deterministic output.
//
// Logic flow (matches CQL exactly):
//  1. Check Initial Population (age 18-85, hypertension, qualifying encounter)
//  2. Check Denominator Exclusions (ESRD, CKD5, pregnancy, hospice)
//  3. Check Numerator (BP < 140/90)
//  4. Determine care gap status
func (e *CMS165Evaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.MeasureResult {
	// Extract read-only evaluation context
	evalCtx := ExtractEvaluationContext(ctx)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 1: Initial Population Check
	// CQL: AgeInYearsAt(end of "Measurement Period") in Interval[18, 85]
	//      and exists "Essential Hypertension Diagnosis"
	//      and exists AdultOutpatientEncounters."Qualifying Encounters"
	// ─────────────────────────────────────────────────────────────────────────

	// Age check: 18-85 at end of measurement period
	if evalCtx.PatientAge < cms165MinAge || evalCtx.PatientAge > cms165MaxAge {
		return NotInInitialPopulation(
			cms165MeasureID,
			cms165MeasureName,
			fmt.Sprintf("Age %d is outside eligible range [%d-%d]", evalCtx.PatientAge, cms165MinAge, cms165MaxAge),
			e,
		)
	}

	// Hypertension check (from KB-7 ValueSet membership)
	hasHypertension := evalCtx.ValueSetMemberships["HasHypertension"]
	if !hasHypertension {
		return NotInInitialPopulation(
			cms165MeasureID,
			cms165MeasureName,
			"Patient does not have essential hypertension diagnosis",
			e,
		)
	}

	// Qualifying encounter check
	if !evalCtx.HasQualifyingEncounter {
		return NotInInitialPopulation(
			cms165MeasureID,
			cms165MeasureName,
			"No qualifying outpatient encounter during measurement period",
			e,
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 2: Denominator Exclusions
	// CQL: ESRD, CKD Stage 5, Pregnancy, Hospice, Palliative Care
	// ─────────────────────────────────────────────────────────────────────────

	// Check for ESRD or CKD Stage 5 (from KB-7)
	if evalCtx.ValueSetMemberships["HasESRD"] || evalCtx.ValueSetMemberships["HasCKDStage5"] {
		return DenominatorExclusion(
			cms165MeasureID,
			cms165MeasureName,
			"Excluded due to ESRD or CKD Stage 5 diagnosis",
			e,
		)
	}

	// Check for pregnancy (from KB-7)
	if evalCtx.ValueSetMemberships["IsPregnant"] {
		return DenominatorExclusion(
			cms165MeasureID,
			cms165MeasureName,
			"Excluded due to pregnancy",
			e,
		)
	}

	// Check for hospice/palliative care (from KB-7)
	if evalCtx.ValueSetMemberships["InHospice"] || evalCtx.ValueSetMemberships["InPalliativeCare"] {
		return DenominatorExclusion(
			cms165MeasureID,
			cms165MeasureName,
			"Excluded due to hospice or palliative care",
			e,
		)
	}

	// Patient is in Denominator
	// ─────────────────────────────────────────────────────────────────────────
	// STEP 3: Numerator Check
	// CQL: "Has Systolic Blood Pressure Less Than 140"
	//      and "Has Diastolic Blood Pressure Less Than 90"
	// ─────────────────────────────────────────────────────────────────────────

	// Check if BP values are available
	if evalCtx.LatestSystolicBP == nil || evalCtx.LatestDiastolicBP == nil {
		return e.buildCareGapResult(
			nil, nil,
			"No blood pressure measurement recorded during measurement period",
		)
	}

	systolic := *evalCtx.LatestSystolicBP
	diastolic := *evalCtx.LatestDiastolicBP

	// BP adequately controlled: Systolic < 140 AND Diastolic < 90
	if systolic < cms165SystolicThreshold && diastolic < cms165DiastolicThreshold {
		return e.buildNumeratorResult(
			systolic, diastolic,
			fmt.Sprintf("Blood pressure %.0f/%.0f mmHg is adequately controlled (< 140/90)", systolic, diastolic),
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 4: Care Gap - BP not adequately controlled
	// ─────────────────────────────────────────────────────────────────────────

	return e.buildCareGapResult(
		&systolic, &diastolic,
		fmt.Sprintf("Blood pressure %.0f/%.0f mmHg is not adequately controlled (requires < 140/90)", systolic, diastolic),
	)
}

// buildNumeratorResult creates result for patient IN numerator (controlled BP).
// For CMS165, being in numerator is a GOOD outcome.
func (e *CMS165Evaluator) buildNumeratorResult(systolic, diastolic float64, rationale string) contracts.MeasureResult {
	return contracts.MeasureResult{
		MeasureID:           cms165MeasureID,
		MeasureName:         cms165MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         true,              // IN numerator = controlled BP
		CareGapIdentified:   false,             // No care gap
		MeasureVersion:      cms165MeasureVersion,
		LogicVersion:        cms165LogicVersion,
		ELMCorrespondence:   cms165ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  []string{"Patient", "Observation/BloodPressure"},
	}
}

// buildCareGapResult creates result for patient with care gap (uncontrolled BP).
func (e *CMS165Evaluator) buildCareGapResult(systolic, diastolic *float64, rationale string) contracts.MeasureResult {
	resources := []string{"Patient"}
	if systolic != nil || diastolic != nil {
		resources = append(resources, "Observation/BloodPressure")
	}

	return contracts.MeasureResult{
		MeasureID:           cms165MeasureID,
		MeasureName:         cms165MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         false,              // NOT in numerator = uncontrolled
		CareGapIdentified:   true,               // Care gap exists
		MeasureVersion:      cms165MeasureVersion,
		LogicVersion:        cms165LogicVersion,
		ELMCorrespondence:   cms165ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  resources,
	}
}

// ============================================================================
// CMS165 GOLDEN TEST HELPERS
// ============================================================================

// CMS165TestCase represents a test scenario for CMS165.
type CMS165TestCase struct {
	Name                    string
	Age                     int
	HasHypertension         bool
	SystolicBP              *float64
	DiastolicBP             *float64
	HasQualifyingEncounter  bool
	// Exclusion flags
	HasESRD                 bool
	HasCKDStage5            bool
	IsPregnant              bool
	InHospice               bool
	// Expected outcomes
	ExpectedInPopulation    bool
	ExpectedExcluded        bool
	ExpectedInNumerator     bool
	ExpectedCareGap         bool
}

// CMS165GoldenTestCases returns the canonical test cases for CMS165.
func CMS165GoldenTestCases() []CMS165TestCase {
	sbp_120 := 120.0
	sbp_145 := 145.0
	sbp_138 := 138.0
	sbp_142 := 142.0
	dbp_80 := 80.0
	dbp_92 := 92.0
	dbp_88 := 88.0 // Used in edge cases
	dbp_85 := 85.0
	_ = dbp_88 // Suppress unused warning - value available for additional test cases

	return []CMS165TestCase{
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 1: Controlled BP (120/80) → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Hypertensive_BP_120_80_Controlled",
			Age:                    55,
			HasHypertension:        true,
			SystolicBP:             &sbp_120,
			DiastolicBP:            &dbp_80,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    true,  // BP < 140/90 = controlled
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 2: Uncontrolled BP (145/92) → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Hypertensive_BP_145_92_Uncontrolled",
			Age:                    60,
			HasHypertension:        true,
			SystolicBP:             &sbp_145,
			DiastolicBP:            &dbp_92,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false, // BP > 140/90 = uncontrolled
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Systolic high, diastolic controlled
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Hypertensive_BP_142_85_SystolicHigh",
			Age:                    50,
			HasHypertension:        true,
			SystolicBP:             &sbp_142,
			DiastolicBP:            &dbp_85,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false, // Systolic ≥ 140, even if diastolic OK
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Systolic controlled, diastolic high
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Hypertensive_BP_138_92_DiastolicHigh",
			Age:                    45,
			HasHypertension:        true,
			SystolicBP:             &sbp_138,
			DiastolicBP:            &dbp_92,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false, // Diastolic ≥ 90, even if systolic OK
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: No BP recorded
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Hypertensive_NoBP_MissingVitals",
			Age:                    55,
			HasHypertension:        true,
			SystolicBP:             nil,
			DiastolicBP:            nil,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false, // No BP = can't be controlled
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: No hypertension
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "NonHypertensive_NotInPopulation",
			Age:                    50,
			HasHypertension:        false,
			SystolicBP:             &sbp_145,
			DiastolicBP:            &dbp_92,
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
			Name:                   "Hypertensive_ESRD_Excluded",
			Age:                    60,
			HasHypertension:        true,
			SystolicBP:             &sbp_145,
			DiastolicBP:            &dbp_92,
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
			Name:                   "Hypertensive_Age17_TooYoung",
			Age:                    17,
			HasHypertension:        true,
			SystolicBP:             &sbp_145,
			DiastolicBP:            &dbp_92,
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
			Name:                   "Hypertensive_Age86_TooOld",
			Age:                    86,
			HasHypertension:        true,
			SystolicBP:             &sbp_120,
			DiastolicBP:            &dbp_80,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},
	}
}
