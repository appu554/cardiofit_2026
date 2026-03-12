// Package measures - CMS122: Diabetes Hemoglobin A1c (HbA1c) Poor Control (>9%)
//
// SOURCE OF TRUTH (ELM):
//   Library: DiabetesGlycemicStatusAssessmentGreaterThan9PercentFHIR
//   Version: 0.1.002
//   Path: clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS122/CMS122-DiabetesHbA1c.cql
//
// MEASURE DESCRIPTION:
//   Percentage of patients 18-75 years of age with diabetes who had hemoglobin
//   A1c > 9.0% during the measurement period.
//
// IMPORTANT - THIS IS AN INVERSE MEASURE:
//   Being IN the numerator = POOR control (bad outcome)
//   Care gap logic is INVERTED: gap = NOT in numerator when should have good control
//
// CLINICAL LOGIC:
//   Initial Population: Age 18-75, has diabetes, has qualifying encounter
//   Denominator: Same as Initial Population
//   Numerator: HbA1c > 9% OR no HbA1c test OR HbA1c without result
//   Care Gap: Patient is diabetic, in denominator, but has GOOD HbA1c (≤9%)
//             This means they DON'T have a care gap for this inverse measure
//
// VALUESET DEPENDENCIES (from KB-7):
//   - Diabetes: HasDiabetes flag
//   - HbA1c Laboratory Test: hba1c lab value
//
// IMPLEMENTATION NOTES:
//   - This evaluator uses ONLY precomputed data from KnowledgeSnapshot
//   - NO runtime KB-7 calls
//   - NO runtime KB-8 calls
//   - Pure function: same input → same output
package measures

import (
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS122 EVALUATOR
// ============================================================================

const (
	cms122MeasureID       = "CMS122"
	cms122MeasureName     = "Diabetes: Hemoglobin A1c (HbA1c) Poor Control (>9%)"
	cms122MeasureVersion  = "2024.0.0"
	cms122LogicVersion    = "1.0.0"
	cms122ELMLibrary      = "DiabetesGlycemicStatusAssessmentGreaterThan9PercentFHIR:0.1.002"
	cms122HbA1cThreshold  = 9.0 // HbA1c > 9% = poor control
	cms122MinAge          = 18
	cms122MaxAge          = 75
)

// CMS122Evaluator implements the CMS122 Diabetes HbA1c Poor Control measure.
//
// MEASURE TYPE: Inverse (higher numerator = worse performance)
//
// Being in the numerator indicates:
//   - HbA1c > 9% (poor glycemic control)
//   - OR no HbA1c test recorded
//   - OR HbA1c test without a result
//
// A "care gap" in the context of this inverse measure means:
//   - Patient SHOULD be tested but wasn't
//   - OR patient has poor control and needs intervention
type CMS122Evaluator struct{}

// NewCMS122Evaluator creates a new CMS122 evaluator instance.
func NewCMS122Evaluator() *CMS122Evaluator {
	return &CMS122Evaluator{}
}

// MeasureID returns "CMS122"
func (e *CMS122Evaluator) MeasureID() string {
	return cms122MeasureID
}

// MeasureName returns the human-readable name
func (e *CMS122Evaluator) MeasureName() string {
	return cms122MeasureName
}

// MeasureVersion returns the CMS-published version
func (e *CMS122Evaluator) MeasureVersion() string {
	return cms122MeasureVersion
}

// LogicVersion returns our Go implementation version
func (e *CMS122Evaluator) LogicVersion() string {
	return cms122LogicVersion
}

// ELMCorrespondence returns the CQL library this implements
func (e *CMS122Evaluator) ELMCorrespondence() string {
	return cms122ELMLibrary
}

// Evaluate runs CMS122 measure logic against the execution context.
//
// PURE FUNCTION: No side effects, no external calls, deterministic output.
//
// Logic flow (matches CQL exactly):
//  1. Check Initial Population (age 18-75, diabetic, qualifying encounter)
//  2. Denominator = Initial Population
//  3. Check Numerator (HbA1c > 9% OR missing)
//  4. Determine care gap status
func (e *CMS122Evaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.MeasureResult {
	// Extract read-only evaluation context
	evalCtx := ExtractEvaluationContext(ctx)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 1: Initial Population Check
	// CQL: AgeInYearsAt(end of "Measurement Period") in Interval[18, 75]
	//      and exists AdultOutpatientEncounters."Qualifying Encounters"
	//      and exists ([Condition: "Diabetes"])
	// ─────────────────────────────────────────────────────────────────────────

	// Age check: 18-75 at end of measurement period
	if evalCtx.PatientAge < cms122MinAge || evalCtx.PatientAge > cms122MaxAge {
		return NotInInitialPopulation(
			cms122MeasureID,
			cms122MeasureName,
			fmt.Sprintf("Age %d is outside eligible range [%d-%d]", evalCtx.PatientAge, cms122MinAge, cms122MaxAge),
			e,
		)
	}

	// Diabetes check (from KB-7 ValueSet membership)
	isDiabetic := evalCtx.ValueSetMemberships["HasDiabetes"]
	if !isDiabetic {
		return NotInInitialPopulation(
			cms122MeasureID,
			cms122MeasureName,
			"Patient does not have diabetes diagnosis",
			e,
		)
	}

	// Qualifying encounter check
	// In practice, if patient is in our system with recent data, they have encounters
	if !evalCtx.HasQualifyingEncounter {
		return NotInInitialPopulation(
			cms122MeasureID,
			cms122MeasureName,
			"No qualifying outpatient encounter during measurement period",
			e,
		)
	}

	// Patient is in Initial Population and Denominator
	// ─────────────────────────────────────────────────────────────────────────
	// STEP 2: Numerator Check
	// CQL: "Has Most Recent Glycemic Status Assessment Without Result"
	//      or "Has Most Recent Elevated Glycemic Status Assessment"
	//      or "Has No Record Of Glycemic Status Assessment"
	// ─────────────────────────────────────────────────────────────────────────

	// Check if HbA1c is available
	if evalCtx.LatestHbA1c == nil {
		// No HbA1c on record → IN NUMERATOR (poor control measure)
		return e.buildNumeratorResult(
			nil,
			"No HbA1c test recorded during measurement period",
		)
	}

	hba1cValue := *evalCtx.LatestHbA1c

	// HbA1c > 9% → IN NUMERATOR (poor control)
	if hba1cValue > cms122HbA1cThreshold {
		return e.buildNumeratorResult(
			&hba1cValue,
			fmt.Sprintf("HbA1c %.1f%% > %.1f%% threshold (poor glycemic control)", hba1cValue, cms122HbA1cThreshold),
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 3: Patient has good HbA1c control (≤9%)
	// For this INVERSE measure:
	//   - NOT in numerator = GOOD outcome
	//   - No care gap (patient is well-controlled)
	// ─────────────────────────────────────────────────────────────────────────

	return e.buildGoodControlResult(
		hba1cValue,
		fmt.Sprintf("HbA1c %.1f%% ≤ %.1f%% threshold (good glycemic control)", hba1cValue, cms122HbA1cThreshold),
	)
}

// buildNumeratorResult creates result for patient IN numerator (poor control).
// For CMS122, being in numerator is a BAD outcome.
func (e *CMS122Evaluator) buildNumeratorResult(hba1c *float64, rationale string) contracts.MeasureResult {
	resources := []string{"Patient"}
	if hba1c != nil {
		resources = append(resources, "Observation/HbA1c")
	}

	return contracts.MeasureResult{
		MeasureID:           cms122MeasureID,
		MeasureName:         cms122MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         true,                // IN numerator = poor control
		CareGapIdentified:   true,                // For inverse measure, numerator = care gap
		MeasureVersion:      cms122MeasureVersion,
		LogicVersion:        cms122LogicVersion,
		ELMCorrespondence:   cms122ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  resources,
	}
}

// buildGoodControlResult creates result for patient with good HbA1c control.
// For CMS122, NOT being in numerator is a GOOD outcome.
func (e *CMS122Evaluator) buildGoodControlResult(hba1c float64, rationale string) contracts.MeasureResult {
	return contracts.MeasureResult{
		MeasureID:           cms122MeasureID,
		MeasureName:         cms122MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         false,               // NOT in numerator = good control
		CareGapIdentified:   false,               // No care gap - patient is well-controlled
		MeasureVersion:      cms122MeasureVersion,
		LogicVersion:        cms122LogicVersion,
		ELMCorrespondence:   cms122ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  []string{"Patient", "Observation/HbA1c"},
	}
}

// ============================================================================
// CMS122 GOLDEN TEST HELPERS
// ============================================================================

// CMS122TestCase represents a test scenario for CMS122.
type CMS122TestCase struct {
	Name                    string
	Age                     int
	IsDiabetic              bool
	HbA1c                   *float64
	HasQualifyingEncounter  bool
	ExpectedInPopulation    bool
	ExpectedInNumerator     bool
	ExpectedCareGap         bool
}

// CMS122GoldenTestCases returns the canonical test cases for CMS122.
// These are the "golden" tests that must always pass.
func CMS122GoldenTestCases() []CMS122TestCase {
	hba1c_10_2 := 10.2
	hba1c_7_1 := 7.1
	hba1c_9_0 := 9.0
	hba1c_8_5 := 8.5

	return []CMS122TestCase{
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 1: Diabetic, HbA1c 10.2% → IN NUMERATOR (poor control)
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_HbA1c_10.2_PoorControl",
			Age:                    55,
			IsDiabetic:             true,
			HbA1c:                  &hba1c_10_2,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedInNumerator:    true,  // HbA1c > 9% = poor control
			ExpectedCareGap:        true,  // Care gap for inverse measure
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 2: Diabetic, HbA1c 7.1% → NOT IN NUMERATOR (good control)
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_HbA1c_7.1_GoodControl",
			Age:                    45,
			IsDiabetic:             true,
			HbA1c:                  &hba1c_7_1,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedInNumerator:    false, // HbA1c ≤ 9% = good control
			ExpectedCareGap:        false, // No care gap
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Exactly at threshold (9.0%)
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_HbA1c_9.0_AtThreshold",
			Age:                    60,
			IsDiabetic:             true,
			HbA1c:                  &hba1c_9_0,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedInNumerator:    false, // HbA1c = 9% exactly → NOT > 9%
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: No HbA1c test recorded
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_NoHbA1c_MissingTest",
			Age:                    50,
			IsDiabetic:             true,
			HbA1c:                  nil, // No test
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   true,
			ExpectedInNumerator:    true, // Missing test = poor control
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Non-diabetic patient
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "NonDiabetic_NotInPopulation",
			Age:                    55,
			IsDiabetic:             false,
			HbA1c:                  &hba1c_8_5,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Age below range
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_Age17_TooYoung",
			Age:                    17,
			IsDiabetic:             true,
			HbA1c:                  &hba1c_10_2,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Age above range
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Diabetic_Age76_TooOld",
			Age:                    76,
			IsDiabetic:             true,
			HbA1c:                  &hba1c_7_1,
			HasQualifyingEncounter: true,
			ExpectedInPopulation:   false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},
	}
}
