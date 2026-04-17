package services

import (
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ComputeAttentionScore computes the attention-gap dimension of the Patient
// Acuity Index. It combines two sub-components:
//
//   A. Clinician contact score — how long since the last clinician review
//   B. Unacknowledged card score — pending decision cards needing attention
//
// The final score is: max(A, B) + 0.20 * min(A, B), capped at 100.
func ComputeAttentionScore(input models.PAIDimensionInput, cfg *PAIConfig) float64 {
	days := input.DaysSinceLastClinician

	// A. Clinician contact score
	var clinicianScore float64
	switch {
	case days >= cfg.AttentionCriticalDays:
		clinicianScore = 100
	case days >= cfg.AttentionHighDays:
		clinicianScore = scaleLinear(float64(days),
			float64(cfg.AttentionHighDays), float64(cfg.AttentionCriticalDays),
			60, 100)
	case days >= cfg.AttentionModerateDays:
		clinicianScore = scaleLinear(float64(days),
			float64(cfg.AttentionModerateDays), float64(cfg.AttentionHighDays),
			30, 60)
	case days >= cfg.AttentionAdequateDays:
		clinicianScore = scaleLinear(float64(days),
			float64(cfg.AttentionAdequateDays), float64(cfg.AttentionModerateDays),
			10, 30)
	default:
		clinicianScore = scaleLinear(float64(days), 0, float64(cfg.AttentionAdequateDays), 0, 10)
	}

	// B. Unacknowledged card score
	var cardScore float64
	if input.HasUnacknowledgedCards {
		raw := float64(input.UnacknowledgedCardCount)*cfg.AttentionPerCard +
			float64(input.OldestUnacknowledgedDays)*cfg.AttentionPerDayOldest
		cardScore = math.Min(raw, cfg.AttentionCardCap)
	}

	// Combine: max(A,B) + 20% of min(A,B), capped at 100
	hi := math.Max(clinicianScore, cardScore)
	lo := math.Min(clinicianScore, cardScore)
	combined := hi + 0.20*lo

	return math.Max(0, math.Min(100, combined))
}
