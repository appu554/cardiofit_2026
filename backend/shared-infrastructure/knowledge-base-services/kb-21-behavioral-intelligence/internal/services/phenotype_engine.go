package services

import "kb-21-behavioral-intelligence/internal/models"

// PhenotypeEngine classifies patients into behavioral phenotypes based on
// adherence score, adherence trend, and recency of interaction.
//
// Phenotype definitions from the KB-21 specification:
//
//	CHAMPION  — adherence ≥ 0.90, trend STABLE or IMPROVING
//	STEADY    — adherence 0.70–0.89, trend STABLE
//	SPORADIC  — adherence 0.50–0.69, or erratic pattern
//	DECLINING — any adherence level, trend DECLINING or CRITICAL
//	DORMANT   — no interaction for 14+ days
//	CHURNED   — no interaction for 30+ days
type PhenotypeEngine struct{}

func NewPhenotypeEngine() *PhenotypeEngine {
	return &PhenotypeEngine{}
}

// Classify determines the patient's behavioral phenotype.
// daysSinceLastInteraction takes priority for DORMANT/CHURNED classification
// because absence of signal is the strongest behavioral indicator.
func (e *PhenotypeEngine) Classify(
	adherenceScore float64,
	trend models.AdherenceTrend,
	daysSinceLastInteraction int,
) models.BehavioralPhenotype {
	// Absence-based classification first (strongest signal)
	if daysSinceLastInteraction >= 30 {
		return models.PhenotypeChurned
	}
	if daysSinceLastInteraction >= 14 {
		return models.PhenotypeDormant
	}

	// Trend-based override: declining trend regardless of score level
	if trend == models.TrendDeclining || trend == models.TrendCritical {
		return models.PhenotypeDeclining
	}

	// Score-based classification
	switch {
	case adherenceScore >= 0.90:
		return models.PhenotypeChampion
	case adherenceScore >= 0.70:
		return models.PhenotypeSteady
	case adherenceScore >= 0.50:
		return models.PhenotypeSporadic
	default:
		// Very low adherence with non-declining trend — still SPORADIC
		// (DECLINING requires a downward trend, not just low absolute level)
		return models.PhenotypeSporadic
	}
}
