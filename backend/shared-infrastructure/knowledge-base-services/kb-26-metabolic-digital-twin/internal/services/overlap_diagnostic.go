package services

import "kb-26-metabolic-digital-twin/internal/models"

// EvaluateOverlap is the single hard guard between the propensity model and the
// CATE learner. Spec §6.1 is explicit: "This is a hard guard and cannot be disabled."
// A propensity outside [band.Floor, band.Ceiling] short-circuits CATE estimation
// to an OverlapBelowFloor or OverlapAboveCeiling status; only a propensity inside
// the band passes.
func EvaluateOverlap(propensity float64, band models.OverlapBand) models.OverlapStatus {
	switch {
	case propensity < band.Floor:
		return models.OverlapBelowFloor
	case propensity > band.Ceiling:
		return models.OverlapAboveCeiling
	default:
		return models.OverlapPass
	}
}
