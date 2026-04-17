package services

import (
	"fmt"
	"math"
	"strings"

	"kb-patient-profile/internal/models"
)

// ─── input type ─────────────────────────────────────────────────────────────

// EnhancedConfounderInput collects every confounder signal that should be
// evaluated when scoring an outcome window.
type EnhancedConfounderInput struct {
	ConcurrentMedCount   int
	AdherenceDrop        float64
	CalendarFactors      []models.ConfounderFactor
	ClinicalEventFactors []models.ConfounderFactor
	LifestyleFactors     []models.ConfounderFactor
	OutcomeType          string // DELTA_HBA1C, DELTA_SBP, DELTA_EGFR, DELTA_WEIGHT
	DeferOnRamadan       bool
	DeferOnSteroid       bool
}

// ─── scorer ─────────────────────────────────────────────────────────────────

// EnhancedConfounderScorer computes a composite confounder score from four
// independent subscores: medication, calendar, clinical-event, and lifestyle.
type EnhancedConfounderScorer struct{}

// NewEnhancedConfounderScorer returns a ready-to-use scorer.
func NewEnhancedConfounderScorer() *EnhancedConfounderScorer {
	return &EnhancedConfounderScorer{}
}

// Compute evaluates the input and returns a fully populated EnhancedConfounderResult.
func (s *EnhancedConfounderScorer) Compute(input EnhancedConfounderInput) models.EnhancedConfounderResult {
	var (
		result         models.EnhancedConfounderResult
		activeFactors  []models.ConfounderFactor
		narrativeParts []string
	)

	// ── 1. Medication subscore ──────────────────────────────────────────
	medScore := math.Min(float64(input.ConcurrentMedCount)*0.12, 0.40)
	adhScore := math.Min(input.AdherenceDrop*1.0, 0.25)
	result.MedicationScore = math.Min(medScore+adhScore, 0.45)

	if input.ConcurrentMedCount > 0 {
		narrativeParts = append(narrativeParts, fmt.Sprintf("%d concurrent medication change(s)", input.ConcurrentMedCount))
		activeFactors = append(activeFactors, models.ConfounderFactor{
			Category: models.ConfounderMedication,
			Name:     "CONCURRENT_MED_CHANGES",
			Weight:   medScore,
		})
	}
	if input.AdherenceDrop > 0.05 {
		narrativeParts = append(narrativeParts, fmt.Sprintf("adherence drop of %.0f%%", input.AdherenceDrop*100))
		activeFactors = append(activeFactors, models.ConfounderFactor{
			Category: models.ConfounderAdherence,
			Name:     "ADHERENCE_DROP",
			Weight:   adhScore,
		})
	}

	// ── 2. Calendar subscore ────────────────────────────────────────────
	var calSum float64
	for _, f := range input.CalendarFactors {
		if !affectsOutcome(f.AffectedOutcomes, input.OutcomeType) {
			continue
		}
		calSum += f.Weight
		activeFactors = append(activeFactors, f)
		narrativeParts = append(narrativeParts, f.Name)

		// Ramadan deferral
		if input.DeferOnRamadan && f.Name == "RAMADAN" && f.OverlapPct > 50 {
			result.ShouldDefer = true
			result.DeferReasonCode = "RAMADAN_ACTIVE"
			result.SuggestedRecheckWeeks = 6
		}
	}
	result.CalendarScore = math.Min(calSum, 0.30)

	// ── 3. Clinical event subscore ──────────────────────────────────────
	var clinSum float64
	for _, f := range input.ClinicalEventFactors {
		if !affectsOutcome(f.AffectedOutcomes, input.OutcomeType) {
			continue
		}
		clinSum += f.Weight
		activeFactors = append(activeFactors, f)
		narrativeParts = append(narrativeParts, f.Name)

		// Steroid deferral
		if input.DeferOnSteroid && f.Name == "STEROID_COURSE" {
			result.ShouldDefer = true
			result.DeferReasonCode = "STEROID_WASHOUT"
			result.SuggestedRecheckWeeks = 4
		}
	}
	result.ClinicalEventScore = math.Min(clinSum, 0.45)

	// High clinical event score forces deferral
	if result.ClinicalEventScore >= 0.35 {
		result.ShouldDefer = true
		if result.DeferReasonCode == "" {
			result.DeferReasonCode = "HIGH_CLINICAL_EVENT_CONFOUNDING"
		}
		if result.SuggestedRecheckWeeks < 6 {
			result.SuggestedRecheckWeeks = 6
		}
	}

	// ── 4. Lifestyle subscore ───────────────────────────────────────────
	var lifeSum float64
	for _, f := range input.LifestyleFactors {
		// Lifestyle affects everything — no outcome filtering
		lifeSum += f.Weight
		activeFactors = append(activeFactors, f)
		narrativeParts = append(narrativeParts, f.Name)
	}
	result.LifestyleScore = math.Min(lifeSum, 0.20)

	// ── Composite ───────────────────────────────────────────────────────
	raw := result.MedicationScore + result.CalendarScore + result.ClinicalEventScore + result.LifestyleScore
	result.CompositeScore = math.Round(math.Min(raw, 1.0)*100) / 100

	// ── Confidence ──────────────────────────────────────────────────────
	// Deferral forces LOW confidence — if we can't trust the outcome enough
	// to act on it, the confidence must reflect that.
	switch {
	case result.ShouldDefer:
		result.ConfidenceLevel = "LOW"
	case result.CompositeScore < 0.20:
		result.ConfidenceLevel = "HIGH"
	case result.CompositeScore <= 0.50:
		result.ConfidenceLevel = "MODERATE"
	default:
		result.ConfidenceLevel = "LOW"
	}

	// ── Active factors & narrative ──────────────────────────────────────
	result.ActiveFactors = activeFactors
	result.FactorCount = len(activeFactors)

	if len(narrativeParts) == 0 {
		result.Narrative = "No significant confounders identified for this outcome window."
	} else {
		result.Narrative = fmt.Sprintf(
			"Outcome confidence %s (score %.2f). Active confounders: %s.",
			result.ConfidenceLevel,
			result.CompositeScore,
			strings.Join(narrativeParts, ", "),
		)
	}

	return result
}

// affectsOutcome returns true if outcomeType is in the affectedOutcomes list,
// or if either side is empty (wildcard behaviour).
func affectsOutcome(affectedOutcomes []string, outcomeType string) bool {
	if len(affectedOutcomes) == 0 || outcomeType == "" {
		return true
	}
	for _, o := range affectedOutcomes {
		if o == outcomeType {
			return true
		}
	}
	return false
}
