// Package measures - CMS2: Preventive Care and Screening: Screening for Depression
//
// SOURCE OF TRUTH (ELM):
//   Library: PCSDepressionScreenAndFollowUpFHIR
//   Version: 0.2.000
//   Path: clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS2/CMS2-DepressionScreening.cql
//
// MEASURE DESCRIPTION:
//   Percentage of patients aged 12 years and older screened for depression on the
//   date of the encounter or up to 14 days prior to the date of the encounter using
//   an age-appropriate standardized depression screening tool AND if positive, a
//   follow-up plan is documented on the date of the eligible encounter.
//
// THIS IS A STANDARD MEASURE:
//   Being IN the numerator = GOOD outcome (screened with follow-up if needed)
//   Care gap = NOT in numerator (not screened or missing follow-up)
//
// CLINICAL LOGIC:
//   Initial Population: Age ≥12, has qualifying encounter
//   Denominator: Same as Initial Population
//   Denominator Exclusions: History of bipolar disorder
//   Numerator: Screening negative OR (Screening positive AND follow-up provided)
//   Care Gap: Not screened or positive without follow-up
//
// VALUESET DEPENDENCIES (from KB-7):
//   - Depression screening results (positive/negative)
//   - Bipolar disorder: has_bipolar_disorder flag
//   - Follow-up documentation
//
// IMPLEMENTATION NOTES:
//   - This evaluator uses ONLY precomputed data from KnowledgeSnapshot
//   - Pure function: same input → same output
package measures

import (
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS2 EVALUATOR
// ============================================================================

const (
	cms2MeasureID      = "CMS2"
	cms2MeasureName    = "Preventive Care and Screening: Screening for Depression and Follow-Up Plan"
	cms2MeasureVersion = "2024.0.0"
	cms2LogicVersion   = "1.0.0"
	cms2ELMLibrary     = "PCSDepressionScreenAndFollowUpFHIR:0.2.000"
	cms2MinAge         = 12 // Age 12 and older

	// LOINC codes for depression screening
	LoincAdolescentDepressionScreening = "73831-0" // Adolescent depression screening assessment
	LoincAdultDepressionScreening      = "73832-8" // Adult depression screening assessment
)

// DepressionScreeningResult represents the outcome of a depression screening
type DepressionScreeningResult string

const (
	ScreeningNotDone DepressionScreeningResult = "not_done"
	ScreeningNegative DepressionScreeningResult = "negative"
	ScreeningPositive DepressionScreeningResult = "positive"
)

// CMS2Evaluator implements the CMS2 Depression Screening measure.
//
// MEASURE TYPE: Standard (higher numerator = better performance)
//
// Being in the numerator indicates:
//   - Depression screening was performed AND
//   - Either screening was negative
//   - OR screening was positive with follow-up documented
//
// A "care gap" means:
//   - Patient not screened for depression
//   - OR screening positive without documented follow-up
type CMS2Evaluator struct{}

// NewCMS2Evaluator creates a new CMS2 evaluator instance.
func NewCMS2Evaluator() *CMS2Evaluator {
	return &CMS2Evaluator{}
}

// MeasureID returns "CMS2"
func (e *CMS2Evaluator) MeasureID() string {
	return cms2MeasureID
}

// MeasureName returns the human-readable name
func (e *CMS2Evaluator) MeasureName() string {
	return cms2MeasureName
}

// MeasureVersion returns the CMS-published version
func (e *CMS2Evaluator) MeasureVersion() string {
	return cms2MeasureVersion
}

// LogicVersion returns our Go implementation version
func (e *CMS2Evaluator) LogicVersion() string {
	return cms2LogicVersion
}

// ELMCorrespondence returns the CQL library this implements
func (e *CMS2Evaluator) ELMCorrespondence() string {
	return cms2ELMLibrary
}

// Evaluate runs CMS2 measure logic against the execution context.
//
// PURE FUNCTION: No side effects, no external calls, deterministic output.
//
// Logic flow (simplified from CQL):
//  1. Check Initial Population (age ≥12, qualifying encounter)
//  2. Check Denominator Exclusions (bipolar disorder)
//  3. Check Numerator (screened negative OR positive with follow-up)
//  4. Determine care gap status
func (e *CMS2Evaluator) Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.MeasureResult {
	// Extract read-only evaluation context
	evalCtx := ExtractEvaluationContext(ctx)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 1: Initial Population Check
	// CQL: AgeInYearsAt(start of "Measurement Period") >= 12
	//      and exists "Qualifying Encounter During Measurement Period"
	// ─────────────────────────────────────────────────────────────────────────

	// Age check: 12 and older at start of measurement period
	if evalCtx.PatientAge < cms2MinAge {
		return NotInInitialPopulation(
			cms2MeasureID,
			cms2MeasureName,
			fmt.Sprintf("Age %d is below minimum age %d", evalCtx.PatientAge, cms2MinAge),
			e,
		)
	}

	// Qualifying encounter check
	if !evalCtx.HasQualifyingEncounter {
		return NotInInitialPopulation(
			cms2MeasureID,
			cms2MeasureName,
			"No qualifying encounter during measurement period",
			e,
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 2: Denominator Exclusions
	// CQL: exists "History of Bipolar Diagnosis Before Qualifying Encounter"
	// ─────────────────────────────────────────────────────────────────────────

	if evalCtx.ValueSetMemberships["has_bipolar_disorder"] {
		return DenominatorExclusion(
			cms2MeasureID,
			cms2MeasureName,
			"Excluded due to history of bipolar disorder",
			e,
		)
	}

	// Patient is in Denominator
	// ─────────────────────────────────────────────────────────────────────────
	// STEP 3: Numerator Check
	// CQL: (Screening Negative) OR (Screening Positive AND Follow-Up Provided)
	// ─────────────────────────────────────────────────────────────────────────

	// Determine screening result from ValueSet memberships
	screeningResult := getDepressionScreeningResult(evalCtx)
	hasFollowUp := evalCtx.ValueSetMemberships["has_depression_followup"]

	// Screening negative → IN NUMERATOR
	if screeningResult == ScreeningNegative {
		return e.buildNumeratorResult(
			ScreeningNegative, hasFollowUp,
			"Depression screening completed with negative result",
		)
	}

	// Screening positive with follow-up → IN NUMERATOR
	if screeningResult == ScreeningPositive && hasFollowUp {
		return e.buildNumeratorResult(
			ScreeningPositive, hasFollowUp,
			"Depression screening positive with documented follow-up plan",
		)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 4: Care Gap
	// ─────────────────────────────────────────────────────────────────────────

	if screeningResult == ScreeningNotDone {
		return e.buildCareGapResult(
			ScreeningNotDone, hasFollowUp,
			"Depression screening not performed during measurement period",
		)
	}

	// Screening positive without follow-up
	return e.buildCareGapResult(
		ScreeningPositive, hasFollowUp,
		"Depression screening positive but no follow-up plan documented",
	)
}

// getDepressionScreeningResult determines the screening result from context
func getDepressionScreeningResult(evalCtx EvaluationContext) DepressionScreeningResult {
	// Check for screening result flags from KB-7
	if evalCtx.ValueSetMemberships["depression_screening_negative"] {
		return ScreeningNegative
	}
	if evalCtx.ValueSetMemberships["depression_screening_positive"] {
		return ScreeningPositive
	}
	return ScreeningNotDone
}

// buildNumeratorResult creates result for patient IN numerator.
func (e *CMS2Evaluator) buildNumeratorResult(screening DepressionScreeningResult, hasFollowUp bool, rationale string) contracts.MeasureResult {
	resources := []string{"Patient", "Observation/DepressionScreening"}
	if hasFollowUp {
		resources = append(resources, "ServiceRequest/FollowUpPlan")
	}

	return contracts.MeasureResult{
		MeasureID:           cms2MeasureID,
		MeasureName:         cms2MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         true,
		CareGapIdentified:   false,
		MeasureVersion:      cms2MeasureVersion,
		LogicVersion:        cms2LogicVersion,
		ELMCorrespondence:   cms2ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  resources,
	}
}

// buildCareGapResult creates result for patient with care gap.
func (e *CMS2Evaluator) buildCareGapResult(screening DepressionScreeningResult, hasFollowUp bool, rationale string) contracts.MeasureResult {
	resources := []string{"Patient"}
	if screening != ScreeningNotDone {
		resources = append(resources, "Observation/DepressionScreening")
	}

	return contracts.MeasureResult{
		MeasureID:           cms2MeasureID,
		MeasureName:         cms2MeasureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         false,
		CareGapIdentified:   true,
		MeasureVersion:      cms2MeasureVersion,
		LogicVersion:        cms2LogicVersion,
		ELMCorrespondence:   cms2ELMLibrary,
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
		EvaluatedResources:  resources,
	}
}

// ============================================================================
// CMS2 GOLDEN TEST HELPERS
// ============================================================================

// CMS2TestCase represents a test scenario for CMS2.
type CMS2TestCase struct {
	Name                     string
	Age                      int
	HasQualifyingEncounter   bool
	// Exclusions
	HasBipolarDisorder       bool
	// Screening results
	ScreeningResult          DepressionScreeningResult
	HasFollowUp              bool
	// Expected outcomes
	ExpectedInPopulation     bool
	ExpectedExcluded         bool
	ExpectedInNumerator      bool
	ExpectedCareGap          bool
}

// CMS2GoldenTestCases returns the canonical test cases for CMS2.
func CMS2GoldenTestCases() []CMS2TestCase {
	return []CMS2TestCase{
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 1: Screening negative → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adult_ScreeningNegative_InNumerator",
			Age:                    45,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningNegative,
			HasFollowUp:            false,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    true,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// GOLDEN TEST 2: Screening positive with follow-up → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adult_ScreeningPositiveWithFollowUp",
			Age:                    35,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningPositive,
			HasFollowUp:            true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    true,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// CARE GAP: Screening positive without follow-up
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adult_ScreeningPositiveNoFollowUp_CareGap",
			Age:                    40,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningPositive,
			HasFollowUp:            false,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// CARE GAP: No screening performed
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adult_NoScreening_CareGap",
			Age:                    50,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningNotDone,
			HasFollowUp:            false,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        true,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// DENOMINATOR EXCLUSION: Bipolar disorder
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adult_BipolarDisorder_Excluded",
			Age:                    38,
			HasQualifyingEncounter: true,
			HasBipolarDisorder:     true,
			ScreeningResult:        ScreeningNotDone,
			HasFollowUp:            false,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       true,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// NOT IN POPULATION: Too young (age 11)
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Age11_TooYoung_NotInPopulation",
			Age:                    11,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningNegative,
			HasFollowUp:            false,
			ExpectedInPopulation:   false,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    false,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Adolescent (age 12) screened negative
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adolescent_Age12_ScreeningNegative",
			Age:                    12,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningNegative,
			HasFollowUp:            false,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    true,
			ExpectedCareGap:        false,
		},

		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// EDGE CASE: Adolescent (age 15) positive with follow-up
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		{
			Name:                   "Adolescent_Age15_PositiveWithFollowUp",
			Age:                    15,
			HasQualifyingEncounter: true,
			ScreeningResult:        ScreeningPositive,
			HasFollowUp:            true,
			ExpectedInPopulation:   true,
			ExpectedExcluded:       false,
			ExpectedInNumerator:    true,
			ExpectedCareGap:        false,
		},
	}
}
