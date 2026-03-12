package services

import "kb-21-behavioral-intelligence/internal/models"

// TrustEngine computes the loop_trust_score consumed by V-MCU for control authority gating.
// Implements the formula from Finding F-01 (Gap 1):
//
//	loop_trust_score = adherence_score * data_quality_weight * phenotype_weight * temporal_stability
//
// The THRESHOLDS for control authority mapping are V-MCU's responsibility.
// KB-21 only provides the composite trust score; V-MCU owns the decision logic.
type TrustEngine struct{}

func NewTrustEngine() *TrustEngine {
	return &TrustEngine{}
}

// ComputeLoopTrust calculates the composite trust score.
func (e *TrustEngine) ComputeLoopTrust(adherenceScore, dataQualityWeight, phenotypeWeight, temporalStability float64) float64 {
	score := adherenceScore * dataQualityWeight * phenotypeWeight * temporalStability
	return clamp(score, 0.0, 1.0)
}

// DataQualityWeight maps data quality classification to its weight in the trust formula.
// Per review Section 1.1:
//
//	HIGH = 1.0 (responded to ≥ 80% check-ins)
//	MODERATE = 0.75 (responded to 50–79%)
//	LOW = 0.50 (responded to < 50%)
func (e *TrustEngine) DataQualityWeight(quality models.DataQuality) float64 {
	switch quality {
	case "HIGH":
		return 1.0
	case "MODERATE":
		return 0.75
	case "LOW":
		return 0.50
	default:
		return 0.50
	}
}

// PhenotypeWeight maps behavioral phenotype to its weight in the trust formula.
// Per review Section 1.1:
//
//	CHAMPION = 1.0, STEADY = 0.90, SPORADIC = 0.65,
//	DECLINING = 0.40, DORMANT = 0.10, CHURNED = 0.0
func (e *TrustEngine) PhenotypeWeight(phenotype models.BehavioralPhenotype) float64 {
	switch phenotype {
	case models.PhenotypeChampion:
		return 1.0
	case models.PhenotypeSteady:
		return 0.90
	case models.PhenotypeSporadic:
		return 0.65
	case models.PhenotypeDeclining:
		return 0.40
	case models.PhenotypeDormant:
		return 0.10
	case models.PhenotypeChurned:
		return 0.0
	default:
		return 0.50
	}
}

// TemporalStability maps adherence trend to its weight in the trust formula.
// Per review Section 1.1:
//
//	STABLE or IMPROVING = 1.0
//	DECLINING = 0.70
//	CRITICAL = 0.40
func (e *TrustEngine) TemporalStability(trend models.AdherenceTrend) float64 {
	switch trend {
	case models.TrendStable, models.TrendImproving:
		return 1.0
	case models.TrendDeclining:
		return 0.70
	case models.TrendCritical:
		return 0.40
	default:
		return 1.0
	}
}

// RecommendAuthority produces an informational recommendation string based on the
// loop_trust_score. Note: these thresholds are INFORMATIONAL — V-MCU owns the actual
// control authority decision logic. KB-21 merely provides the trust input.
//
// Per review Section 1.1 Loop Authority Mapping:
//
//	≥ 0.75 → AUTO (full auto-titration)
//	0.55–0.74 → ASSISTED (auto with enhanced monitoring)
//	0.35–0.54 → CONFIRM (physician-confirmed titration)
//	0.20–0.34 → DISABLED (correction loop disabled, manual only)
//	< 0.20 → DISABLED
func (e *TrustEngine) RecommendAuthority(loopTrustScore float64) string {
	switch {
	case loopTrustScore >= 0.75:
		return "AUTO"
	case loopTrustScore >= 0.55:
		return "ASSISTED"
	case loopTrustScore >= 0.35:
		return "CONFIRM"
	default:
		return "DISABLED"
	}
}
