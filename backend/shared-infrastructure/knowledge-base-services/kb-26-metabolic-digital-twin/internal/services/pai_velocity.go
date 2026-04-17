package services

import (
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// PAIConfig holds all tunable parameters for the Patient Acuity Index
// computation. Defined here (velocity dimension) and reused by all
// other PAI dimension engines (proximity, behavioral, context, attention,
// composite — Tasks 3-7).
type PAIConfig struct {
	// Dimension weights (must sum to 1.0)
	VelocityWeight   float64
	ProximityWeight  float64
	BehavioralWeight float64
	ContextWeight    float64
	AttentionWeight  float64

	// Velocity thresholds — slope breakpoints for piecewise mapping
	SevereDeclineSlope            float64
	ModerateDeclineSlope          float64
	MildDeclineSlope              float64
	StableSlope                   float64
	AcceleratingDeclineMultiplier float64
	DeceleratingDeclineMultiplier float64
	ConcordantBonus               float64
	PerAdditionalDomain           float64
	ConfounderDampeningEnabled    bool
	MaxVelocityDuringSeason       float64

	// Proximity
	ProximityExponent float64

	// Behavioral
	BehavioralCessationDays    int
	BehavioralReducedThreshold float64
	BehavioralSlightlyReduced  float64
	BehavioralCompoundBoth     float64

	// Tier thresholds
	CriticalThreshold float64
	HighThreshold     float64
	ModerateThreshold float64
	LowThreshold      float64
	SignificantDelta   float64
}

// ComputeVelocityScore maps the MHRI composite slope plus second-derivative
// acceleration, concordance, and seasonal context into a 0-100 velocity
// dimension score for the Patient Acuity Index.
//
// Scoring pipeline:
//  1. Nil guard — no slope → 0
//  2. Piecewise linear mapping of slope to base score
//  3. Second-derivative amplification / dampening
//  4. Concordant multi-domain bonus
//  5. Seasonal confounder cap
//  6. Clamp [0, 100]
func ComputeVelocityScore(input models.PAIDimensionInput, cfg *PAIConfig) float64 {
	// 1. Nil guard
	if input.MHRICompositeSlope == nil {
		return 0
	}
	slope := *input.MHRICompositeSlope

	// 2. Piecewise linear mapping
	var base float64
	switch {
	case slope <= cfg.SevereDeclineSlope:
		base = 100
	case slope <= cfg.ModerateDeclineSlope:
		// severe → moderate maps 100 → 60
		base = scaleLinear(slope, cfg.SevereDeclineSlope, cfg.ModerateDeclineSlope, 100, 60)
	case slope <= cfg.MildDeclineSlope:
		// moderate → mild maps 60 → 30
		base = scaleLinear(slope, cfg.ModerateDeclineSlope, cfg.MildDeclineSlope, 60, 30)
	case slope <= cfg.StableSlope:
		// mild → stable maps 30 → 0
		base = scaleLinear(slope, cfg.MildDeclineSlope, cfg.StableSlope, 30, 0)
	default:
		base = 0
	}

	// 3. Second-derivative amplification
	if input.SecondDerivative != nil {
		switch *input.SecondDerivative {
		case "ACCELERATING_DECLINE":
			base *= cfg.AcceleratingDeclineMultiplier
		case "DECELERATING_DECLINE":
			base *= cfg.DeceleratingDeclineMultiplier
		case "ACCELERATING_IMPROVEMENT":
			base *= 0.5
		}
	}

	// 4. Concordant multi-domain bonus
	if input.ConcordantDeterioration && input.DomainsDeterioriating >= 2 {
		base += cfg.ConcordantBonus + float64(input.DomainsDeterioriating-2)*cfg.PerAdditionalDomain
	}

	// 5. Seasonal confounder dampening
	if cfg.ConfounderDampeningEnabled && input.SeasonalWindow {
		if base > cfg.MaxVelocityDuringSeason {
			base = cfg.MaxVelocityDuringSeason
		}
	}

	// 6. Clamp [0, 100]
	return math.Max(0, math.Min(100, base))
}

// scaleLinear maps value from [fromMin, fromMax] to [toMin, toMax] linearly.
func scaleLinear(value, fromMin, fromMax, toMin, toMax float64) float64 {
	if fromMax == fromMin {
		return toMin
	}
	ratio := (value - fromMin) / (fromMax - fromMin)
	return toMin + ratio*(toMax-toMin)
}
