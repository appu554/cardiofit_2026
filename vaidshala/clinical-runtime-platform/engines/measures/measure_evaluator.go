// Package measures provides rule-based CQL measure evaluators.
//
// ARCHITECTURE (CTO/CMO APPROVED):
// These evaluators implement CMS measure logic as deterministic Go functions.
// They operate ONLY on precomputed data from KnowledgeSnapshot - NO runtime
// KB calls, NO Neo4j, NO terminology expansion.
//
// CQL is the SOURCE OF TRUTH (regulatory defensibility), but Go is the
// EXECUTION MODEL (performance, auditability, determinism).
//
// Each measure is implemented as a separate evaluator struct for:
//   - Independent versioning
//   - Independent testing
//   - Independent updates when CMS revises measures
//
// CRITICAL CONSTRAINTS:
//   - Evaluators may ONLY read from ClinicalExecutionContext
//   - NO database calls
//   - NO HTTP calls
//   - NO Redis calls
//   - NO Neo4j calls
//   - Pure function: same input → same output
package measures

import (
	"strings"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// MEASURE EVALUATOR INTERFACE
// ============================================================================

// MeasureEvaluator defines the contract for all CQL measure implementations.
// Each CMS measure (CMS122, CMS165, etc.) implements this interface.
type MeasureEvaluator interface {
	// MeasureID returns the CMS measure identifier (e.g., "CMS122")
	MeasureID() string

	// MeasureName returns human-readable name
	MeasureName() string

	// MeasureVersion returns the CMS-published version
	MeasureVersion() string

	// LogicVersion returns our implementation version
	LogicVersion() string

	// ELMCorrespondence returns the CQL library this implements
	ELMCorrespondence() string

	// Evaluate runs the measure logic against the execution context.
	// This is a PURE FUNCTION - no side effects, no external calls.
	Evaluate(ctx *contracts.ClinicalExecutionContext) contracts.MeasureResult
}

// ============================================================================
// EVALUATION CONTEXT (Read-Only View)
// ============================================================================

// EvaluationContext provides a read-only view of data needed for measure evaluation.
// This is extracted from ClinicalExecutionContext to enforce the constraint
// that evaluators can ONLY read precomputed data.
type EvaluationContext struct {
	// Patient demographics
	PatientAge    int
	PatientGender string

	// ValueSet memberships (precomputed by KB-7)
	// Key: flag name (e.g., "HasDiabetes", "HasHypertension")
	// Value: true if patient has the condition
	ValueSetMemberships map[string]bool

	// Lab values (from PatientContext)
	LatestHbA1c      *float64 // nil if no HbA1c on record
	LatestSystolicBP *float64 // nil if no BP on record
	LatestDiastolicBP *float64
	LatestEGFR       *float64 // from KB-8 calculator

	// Encounter info
	HasQualifyingEncounter bool

	// Measurement period
	MeasurementPeriodStart time.Time
	MeasurementPeriodEnd   time.Time
}

// LOINC codes for common clinical observations
const (
	// HbA1c LOINC codes (multiple valid codes per CMS)
	LoincHbA1c        = "4548-4"  // Hemoglobin A1c/Hemoglobin.total in Blood
	LoincHbA1cAlt     = "17856-6" // Hemoglobin A1c/Hemoglobin.total in Blood by HPLC
	LoincSystolicBP   = "8480-6"  // Systolic blood pressure
	LoincDiastolicBP  = "8462-4"  // Diastolic blood pressure
)

// ExtractEvaluationContext extracts read-only evaluation data from execution context.
func ExtractEvaluationContext(ctx *contracts.ClinicalExecutionContext) EvaluationContext {
	evalCtx := EvaluationContext{
		ValueSetMemberships:    make(map[string]bool),
		HasQualifyingEncounter: len(ctx.Patient.RecentEncounters) > 0,
	}

	// Calculate patient age from BirthDate
	if ctx.Patient.Demographics.BirthDate != nil {
		evalCtx.PatientAge = calculateAge(*ctx.Patient.Demographics.BirthDate, time.Now())
	}
	evalCtx.PatientGender = ctx.Patient.Demographics.Gender

	// Extract ValueSet memberships from KnowledgeSnapshot
	if ctx.Knowledge.Terminology.ValueSetMemberships != nil {
		for k, v := range ctx.Knowledge.Terminology.ValueSetMemberships {
			evalCtx.ValueSetMemberships[k] = v
		}
	}

	// Extract calculator results from KB-8
	if ctx.Knowledge.Calculators.EGFR != nil {
		val := ctx.Knowledge.Calculators.EGFR.Value
		evalCtx.LatestEGFR = &val
	}

	// Extract HbA1c from RecentLabResults (search by LOINC code)
	evalCtx.LatestHbA1c = findLatestLabValue(ctx.Patient.RecentLabResults, LoincHbA1c, LoincHbA1cAlt)

	// Extract BP from RecentVitalSigns
	// Blood pressure is typically a composite observation with systolic/diastolic components
	// CRITICAL: RecentVitalSigns is pre-sorted newest-first from KB-2A.
	// We must return on FIRST complete BP reading to get the most recent values.
	// Previous bug: Loop continued through all vitals, overwriting with older readings.
	for _, vital := range ctx.Patient.RecentVitalSigns {
		var foundSystolic, foundDiastolic bool
		// Check component values for BP readings
		for _, comp := range vital.ComponentValues {
			if comp.Code.Code == LoincSystolicBP && comp.Value != nil {
				val := comp.Value.Value
				evalCtx.LatestSystolicBP = &val
				foundSystolic = true
			}
			if comp.Code.Code == LoincDiastolicBP && comp.Value != nil {
				val := comp.Value.Value
				evalCtx.LatestDiastolicBP = &val
				foundDiastolic = true
			}
		}
		// Early exit: Once we have a complete BP reading, use it (it's the most recent)
		if foundSystolic && foundDiastolic {
			break
		}
	}

	// Measurement period
	if ctx.Runtime.MeasurementPeriod != nil {
		if ctx.Runtime.MeasurementPeriod.Start != nil {
			evalCtx.MeasurementPeriodStart = *ctx.Runtime.MeasurementPeriod.Start
		}
		if ctx.Runtime.MeasurementPeriod.End != nil {
			evalCtx.MeasurementPeriodEnd = *ctx.Runtime.MeasurementPeriod.End
		}
	}

	// Default to current calendar year if not set
	if evalCtx.MeasurementPeriodStart.IsZero() {
		now := time.Now()
		evalCtx.MeasurementPeriodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		evalCtx.MeasurementPeriodEnd = time.Date(now.Year(), 12, 31, 23, 59, 59, 999999999, time.UTC)
	}

	return evalCtx
}

// calculateAge computes age in years from birth date to reference date.
func calculateAge(birthDate, referenceDate time.Time) int {
	years := referenceDate.Year() - birthDate.Year()
	// Adjust if birthday hasn't occurred yet this year
	if referenceDate.Month() < birthDate.Month() ||
		(referenceDate.Month() == birthDate.Month() && referenceDate.Day() < birthDate.Day()) {
		years--
	}
	return years
}

// findLatestLabValue searches for the most recent lab result matching any of the given LOINC codes.
func findLatestLabValue(labs []contracts.LabResult, loincCodes ...string) *float64 {
	var latestTime time.Time
	var latestValue *float64

	for _, lab := range labs {
		// Check if this lab matches any of our target LOINC codes
		for _, targetCode := range loincCodes {
			if lab.Code.Code == targetCode {
				// Check if this is more recent than what we have
				if lab.EffectiveDateTime != nil && lab.Value != nil {
					if latestValue == nil || lab.EffectiveDateTime.After(latestTime) {
						latestTime = *lab.EffectiveDateTime
						val := lab.Value.Value
						latestValue = &val
					}
				}
				break
			}
		}
	}
	return latestValue
}

// ============================================================================
// RESULT BUILDERS (Helpers)
// ============================================================================

// NotInInitialPopulation creates a result for patients not in initial population.
func NotInInitialPopulation(measureID, measureName, rationale string, eval MeasureEvaluator) contracts.MeasureResult {
	return contracts.MeasureResult{
		MeasureID:           measureID,
		MeasureName:         measureName,
		InInitialPopulation: false,
		InDenominator:       false,
		InNumerator:         false,
		CareGapIdentified:   false,
		MeasureVersion:      eval.MeasureVersion(),
		LogicVersion:        eval.LogicVersion(),
		ELMCorrespondence:   eval.ELMCorrespondence(),
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
	}
}

// InNumerator creates a result for patients in numerator (met criteria).
func InNumerator(measureID, measureName, rationale string, eval MeasureEvaluator) contracts.MeasureResult {
	return contracts.MeasureResult{
		MeasureID:           measureID,
		MeasureName:         measureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         true,
		CareGapIdentified:   false, // In numerator = no gap (for standard measures)
		MeasureVersion:      eval.MeasureVersion(),
		LogicVersion:        eval.LogicVersion(),
		ELMCorrespondence:   eval.ELMCorrespondence(),
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
	}
}

// InDenominatorCareGap creates a result for patients with a care gap.
func InDenominatorCareGap(measureID, measureName, rationale string, eval MeasureEvaluator) contracts.MeasureResult {
	return contracts.MeasureResult{
		MeasureID:           measureID,
		MeasureName:         measureName,
		InInitialPopulation: true,
		InDenominator:       true,
		InNumerator:         false,
		CareGapIdentified:   true,
		MeasureVersion:      eval.MeasureVersion(),
		LogicVersion:        eval.LogicVersion(),
		ELMCorrespondence:   eval.ELMCorrespondence(),
		EvaluatedAt:         time.Now().UTC(),
		Rationale:           rationale,
	}
}

// DenominatorExclusion creates a result for excluded patients.
func DenominatorExclusion(measureID, measureName, rationale string, eval MeasureEvaluator) contracts.MeasureResult {
	return contracts.MeasureResult{
		MeasureID:              measureID,
		MeasureName:            measureName,
		InInitialPopulation:    true,
		InDenominator:          true,
		InNumerator:            false,
		InDenominatorExclusion: true,
		CareGapIdentified:      false,
		MeasureVersion:         eval.MeasureVersion(),
		LogicVersion:           eval.LogicVersion(),
		ELMCorrespondence:      eval.ELMCorrespondence(),
		EvaluatedAt:            time.Now().UTC(),
		Rationale:              rationale,
	}
}

// ============================================================================
// CODE CHECKER HELPERS (Exported for CQL Engine)
// ============================================================================

// CalculateAge computes age in years from birth date to now.
// Exported for use by CQL Engine fact evaluators.
func CalculateAge(birthDate time.Time) int {
	return calculateAge(birthDate, time.Now())
}

// ============================================================================
// DEPRECATED HARDCODED FUNCTIONS - ALL REMOVED
// ============================================================================
// The following functions have been REMOVED because they are obsolete:
// - IsDiabetesCode() - Use KB-7 "is_diabetic" flag from ValueSetMemberships
// - IsHypertensionCode() - Use KB-7 "has_hypertension" flag from ValueSetMemberships
// - IsCKDCode() - Use KB-7 "has_ckd" flag from ValueSetMemberships
// - IsHbA1cCode() - Use KB-7 "LabHbA1c" ValueSet via CodeMemberships
// - IsUACRCode() - Use KB-7 "LabuACR" ValueSet via CodeMemberships
// - IsPHQ9Code() - Use KB-7 "LabPHQ9" ValueSet via CodeMemberships
//
// KB-7 now has 34,500+ LOINC codes in the AllLOINCCodes ValueSet plus
// clinical ValueSets (LabHbA1c, LabuACR, LabPHQ9, etc.) with proper mappings.
// Use IsLabCodeInValueSet() for dynamic lookups - NO hardcoded lists!
// ============================================================================

// ============================================================================
// DYNAMIC VALUESET MEMBERSHIP CHECKS (KB-7 BACKED)
// These functions check CodeMemberships from the KnowledgeSnapshot.
// NO HARDCODED DRUG LISTS - all codes come from KB-7's dynamic expansion.
//
// NOTE ON VALUESET IDENTIFIERS:
// The constants below are "contract names" - agreed identifiers between KB-7
// and Go code. They are NOT hardcoded code lists. The actual clinical codes
// are fetched dynamically from KB-7 at build time and stored in CodeMemberships.
// ============================================================================

// Standard code system URLs (FHIR canonical URIs)
const (
	RxNormSystem = "http://www.nlm.nih.gov/research/umls/rxnorm"
	LoincSystem  = "http://loinc.org"
	SnomedSystem = "http://snomed.info/sct"
)

// ValueSet identifiers - these are KB-7 contract names, not hardcoded code lists.
// The actual clinical codes are fetched dynamically from KB-7.
const (
	// Medication ValueSets
	ValueSetACEInhibitors     = "ACEInhibitors"
	ValueSetARBs              = "ARBs"
	ValueSetACEInhibitorsARBs = "ACEInhibitorsARBs"

	// Lab ValueSets (LOINC-based)
	ValueSetLabHbA1c = "LabHbA1c"
	ValueSetLabuACR  = "LabuACR"
	ValueSetLabPHQ9  = "LabPHQ9"
)

// IsCodeInValueSet checks if a code belongs to a ValueSet using CodeMemberships.
// This is a pure in-memory lookup - NO network calls.
// codeMemberships is populated by KnowledgeSnapshotBuilder from KB-7.
// The key format is "system|code" (e.g., "http://www.nlm.nih.gov/research/umls/rxnorm|314076")
func IsCodeInValueSet(code string, system string, targetValueSets []string, codeMemberships map[string][]string) bool {
	if codeMemberships == nil {
		return false
	}

	// Build the key in the same format as KnowledgeSnapshotBuilder
	codeKey := system + "|" + code

	memberSets, exists := codeMemberships[codeKey]
	if !exists {
		return false
	}
	for _, memberSet := range memberSets {
		for _, target := range targetValueSets {
			if memberSet == target {
				return true
			}
		}
	}
	return false
}

// IsACEInhibitorOrARB checks if a code is an ACE inhibitor or ARB using KB-7 data.
// Uses CodeMemberships from KnowledgeSnapshot - NO hardcoded drug lists!
func IsACEInhibitorOrARB(code string, system string, codeMemberships map[string][]string) bool {
	return IsCodeInValueSet(code, system, []string{
		ValueSetACEInhibitors,
		ValueSetARBs,
		ValueSetACEInhibitorsARBs,
	}, codeMemberships)
}

// ============================================================================
// DYNAMIC LOINC VALUESET LOOKUPS (KB-7 BACKED)
// These replace the old hardcoded IsHbA1cCode, IsUACRCode, IsPHQ9Code functions.
// All 34,500+ LOINC codes are now in KB-7 - use CodeMemberships for lookups.
// ============================================================================

// IsLabCodeInValueSet checks if a LOINC code belongs to a specific lab ValueSet.
// Uses CodeMemberships from KnowledgeSnapshot - NO hardcoded LOINC lists!
func IsLabCodeInValueSet(code string, targetValueSet string, codeMemberships map[string][]string) bool {
	return IsCodeInValueSet(code, LoincSystem, []string{targetValueSet}, codeMemberships)
}

// IsHbA1cCode checks if the LOINC code is for HbA1c using KB-7 ValueSets.
// DYNAMIC: Uses CodeMemberships and checks for any HbA1c-related ValueSet names.
// KB-7 returns semantic names like "HemoglobinA1c", "Hba1cLaboratoryTest", etc.
func IsHbA1cCode(code string, codeMemberships map[string][]string) bool {
	if codeMemberships == nil {
		return false
	}

	codeKey := LoincSystem + "|" + code
	memberSets, exists := codeMemberships[codeKey]
	if !exists {
		return false
	}

	// Check if any membership is HbA1c-related
	for _, memberSet := range memberSets {
		lowerName := strings.ToLower(memberSet)
		if strings.Contains(lowerName, "hba1c") ||
			strings.Contains(lowerName, "hemoglobin") ||
			strings.Contains(lowerName, "a1c") ||
			strings.Contains(lowerName, "glycated") {
			return true
		}
	}
	return false
}

// IsUACRCode checks if the LOINC code is for uACR using KB-7 LabuACR ValueSet.
// DYNAMIC: Uses CodeMemberships instead of hardcoded list.
func IsUACRCode(code string, codeMemberships map[string][]string) bool {
	return IsLabCodeInValueSet(code, ValueSetLabuACR, codeMemberships)
}

// IsPHQ9Code checks if the LOINC code is for PHQ-9 using KB-7 LabPHQ9 ValueSet.
// DYNAMIC: Uses CodeMemberships instead of hardcoded list.
func IsPHQ9Code(code string, codeMemberships map[string][]string) bool {
	return IsLabCodeInValueSet(code, ValueSetLabPHQ9, codeMemberships)
}
