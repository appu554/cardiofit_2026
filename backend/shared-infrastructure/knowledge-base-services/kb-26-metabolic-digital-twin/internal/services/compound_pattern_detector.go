package services

import (
	"fmt"
	"strings"

	"kb-26-metabolic-digital-twin/internal/models"
)

// CompoundContext carries patient-level context for compound pattern detection.
type CompoundContext struct {
	CKMStage            string
	DaysSinceDischarge  *int     // nil if not post-discharge
	NewMedications      []string // medications started in last 14 days
	MeasurementFreqDrop float64  // 0-1, percentage drop from average
}

// DetectCompoundPatterns checks a set of simultaneous deviations against known
// multi-organ syndrome templates. Returns all matching patterns.
func DetectCompoundPatterns(
	deviations []models.DeviationResult,
	context CompoundContext,
) []models.CompoundPatternMatch {
	var matches []models.CompoundPatternMatch

	if m, ok := detectCardiorenalSyndrome(deviations); ok {
		matches = append(matches, m)
	}
	if m, ok := detectInfectionCascade(deviations, context); ok {
		matches = append(matches, m)
	}
	if m, ok := detectMedicationCrisis(deviations, context); ok {
		matches = append(matches, m)
	}
	if m, ok := detectFluidOverloadTriad(deviations, context); ok {
		matches = append(matches, m)
	}
	if m, ok := detectPostDischargeDeterior(deviations, context); ok {
		matches = append(matches, m)
	}

	return matches
}

// ---------------------------------------------------------------------------
// Pattern: CARDIORENAL_SYNDROME
// eGFR dropping >=15% AND (SBP dropping >=15 mmHg OR weight gaining >=1.5kg)
// ---------------------------------------------------------------------------

func detectCardiorenalSyndrome(deviations []models.DeviationResult) (models.CompoundPatternMatch, bool) {
	egfr := findDeviation(deviations, "EGFR", "BELOW_BASELINE")
	if egfr == nil || egfr.DeviationPercent < 15 {
		return models.CompoundPatternMatch{}, false
	}

	sbp := findDeviation(deviations, "SBP", "BELOW_BASELINE")
	weight := findDeviation(deviations, "WEIGHT", "ABOVE_BASELINE")

	sbpMatch := sbp != nil && sbp.DeviationAbsolute >= 15
	weightMatch := weight != nil && weight.DeviationAbsolute >= 1.5

	if !sbpMatch && !weightMatch {
		return models.CompoundPatternMatch{}, false
	}

	matched := []models.DeviationResult{*egfr}
	if sbpMatch {
		matched = append(matched, *sbp)
	}
	if weightMatch {
		matched = append(matched, *weight)
	}

	highest := maxSeverity(matched)

	return models.CompoundPatternMatch{
		PatternName:         "CARDIORENAL_SYNDROME",
		MatchedDeviations:   matched,
		PatternConfidence:   "HIGH",
		ClinicalSyndrome:    "Cardiorenal syndrome — simultaneous renal and haemodynamic deterioration",
		RecommendedResponse: "Urgent nephrology/cardiology review. Assess volume status and cardiac output.",
		CompoundSeverity:    escalateSeverity(highest, 1),
	}, true
}

// ---------------------------------------------------------------------------
// Pattern: INFECTION_CASCADE
// Glucose rising >=30% AND SBP dropping >=15%
// Optional: MeasurementFreqDrop >=0.50 adds confidence
// ---------------------------------------------------------------------------

func detectInfectionCascade(deviations []models.DeviationResult, ctx CompoundContext) (models.CompoundPatternMatch, bool) {
	glucose := findDeviation(deviations, "GLUCOSE", "ABOVE_BASELINE")
	if glucose == nil || glucose.DeviationPercent < 30 {
		return models.CompoundPatternMatch{}, false
	}

	sbp := findDeviation(deviations, "SBP", "BELOW_BASELINE")
	if sbp == nil || sbp.DeviationPercent < 15 {
		return models.CompoundPatternMatch{}, false
	}

	matched := []models.DeviationResult{*glucose, *sbp}
	highest := maxSeverity(matched)

	confidence := "MODERATE"
	if ctx.MeasurementFreqDrop >= 0.50 {
		confidence = "HIGH"
	}

	return models.CompoundPatternMatch{
		PatternName:         "INFECTION_CASCADE",
		MatchedDeviations:   matched,
		PatternConfidence:   confidence,
		ClinicalSyndrome:    "Possible infection cascade — hyperglycaemia with haemodynamic compromise",
		RecommendedResponse: "Screen for infection source. Consider blood cultures and empiric therapy.",
		CompoundSeverity:    escalateSeverity(highest, 1),
	}, true
}

// ---------------------------------------------------------------------------
// Pattern: MEDICATION_CRISIS
// Any deviation with ClinicalSignificance >= MODERATE AND NewMedications non-empty
// ---------------------------------------------------------------------------

func detectMedicationCrisis(deviations []models.DeviationResult, ctx CompoundContext) (models.CompoundPatternMatch, bool) {
	if len(ctx.NewMedications) == 0 {
		return models.CompoundPatternMatch{}, false
	}

	var matched []models.DeviationResult
	for _, d := range deviations {
		if severityRank[d.ClinicalSignificance] >= severityRank["MODERATE"] {
			matched = append(matched, d)
		}
	}
	if len(matched) == 0 {
		return models.CompoundPatternMatch{}, false
	}

	highest := maxSeverity(matched)
	medList := strings.Join(ctx.NewMedications, ", ")

	// Build recommended response mentioning each medication and the affected vital.
	var responses []string
	for _, d := range matched {
		for _, med := range ctx.NewMedications {
			responses = append(responses, fmt.Sprintf(
				"Consider stopping or reducing %s. Temporal correlation with %s deterioration.",
				med, d.VitalSignType,
			))
		}
	}
	response := strings.Join(responses, " ")

	return models.CompoundPatternMatch{
		PatternName:         "MEDICATION_CRISIS",
		MatchedDeviations:   matched,
		PatternConfidence:   "HIGH",
		ClinicalSyndrome:    fmt.Sprintf("Medication-induced crisis — new medications (%s) temporally correlated with deterioration", medList),
		RecommendedResponse: response,
		CompoundSeverity:    escalateSeverity(highest, 1),
	}, true
}

// ---------------------------------------------------------------------------
// Pattern: FLUID_OVERLOAD_TRIAD
// Weight gain >=1.5kg AND SBP rise >=10 mmHg
// CKMStage must start with "4"
// ---------------------------------------------------------------------------

func detectFluidOverloadTriad(deviations []models.DeviationResult, ctx CompoundContext) (models.CompoundPatternMatch, bool) {
	if !strings.HasPrefix(ctx.CKMStage, "4") {
		return models.CompoundPatternMatch{}, false
	}

	weight := findDeviation(deviations, "WEIGHT", "ABOVE_BASELINE")
	if weight == nil || weight.DeviationAbsolute < 1.5 {
		return models.CompoundPatternMatch{}, false
	}

	sbp := findDeviation(deviations, "SBP", "ABOVE_BASELINE")
	if sbp == nil || sbp.DeviationAbsolute < 10 {
		return models.CompoundPatternMatch{}, false
	}

	matched := []models.DeviationResult{*weight, *sbp}
	highest := maxSeverity(matched)

	return models.CompoundPatternMatch{
		PatternName:         "FLUID_OVERLOAD_TRIAD",
		MatchedDeviations:   matched,
		PatternConfidence:   "HIGH",
		ClinicalSyndrome:    "Fluid overload triad — weight gain with hypertension in advanced CKM",
		RecommendedResponse: "Assess fluid balance. Consider diuretic adjustment and sodium restriction.",
		CompoundSeverity:    escalateSeverity(highest, 1),
	}, true
}

// ---------------------------------------------------------------------------
// Pattern: POST_DISCHARGE_DETERIORATION
// DaysSinceDischarge != nil AND <= 30
// Any deviation with ClinicalSignificance >= MODERATE
// Severity escalated by 1 level
// ---------------------------------------------------------------------------

func detectPostDischargeDeterior(deviations []models.DeviationResult, ctx CompoundContext) (models.CompoundPatternMatch, bool) {
	if ctx.DaysSinceDischarge == nil || *ctx.DaysSinceDischarge > 30 {
		return models.CompoundPatternMatch{}, false
	}

	var matched []models.DeviationResult
	for _, d := range deviations {
		if severityRank[d.ClinicalSignificance] >= severityRank["MODERATE"] {
			matched = append(matched, d)
		}
	}
	if len(matched) == 0 {
		return models.CompoundPatternMatch{}, false
	}

	highest := maxSeverity(matched)

	return models.CompoundPatternMatch{
		PatternName:         "POST_DISCHARGE_DETERIORATION",
		MatchedDeviations:   matched,
		PatternConfidence:   "HIGH",
		ClinicalSyndrome:    fmt.Sprintf("Post-discharge deterioration — %d days since discharge with significant deviation", *ctx.DaysSinceDischarge),
		RecommendedResponse: "Urgent post-discharge review. Reassess medication reconciliation and volume status.",
		CompoundSeverity:    escalateSeverity(highest, 1),
	}, true
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// findDeviation finds the first deviation matching the given vital type and direction.
func findDeviation(deviations []models.DeviationResult, vitalType, direction string) *models.DeviationResult {
	for i := range deviations {
		if deviations[i].VitalSignType == vitalType && deviations[i].Direction == direction {
			return &deviations[i]
		}
	}
	return nil
}

// maxSeverity returns the highest ClinicalSignificance among the given deviations.
func maxSeverity(deviations []models.DeviationResult) string {
	best := 0
	for _, d := range deviations {
		r := severityRank[d.ClinicalSignificance]
		if r > best {
			best = r
		}
	}
	return rankToSeverity[best]
}
