package services

// HeightenedSurveillanceMode provides parameter modifiers for the
// 30-day post-discharge window. All methods are pure functions that
// take isActiveTransition as input — no state is maintained.
type HeightenedSurveillanceMode struct {
	deviationMultiplier float64 // 0.75 during transition
	paiContextBoost     float64 // 15.0 during transition
	engagementGapHours  int     // 72 during transition
}

// NewHeightenedSurveillanceMode returns a HeightenedSurveillanceMode with
// default post-discharge parameters: 25% tighter deviation thresholds,
// +15 PAI context boost, and 72-hour engagement gap alerting.
func NewHeightenedSurveillanceMode() *HeightenedSurveillanceMode {
	return &HeightenedSurveillanceMode{
		deviationMultiplier: 0.75,
		paiContextBoost:     15.0,
		engagementGapHours:  72,
	}
}

// GetDeviationMultiplier returns the threshold multiplier.
// 0.75 during transition (25% tighter), 1.0 normally.
func (h *HeightenedSurveillanceMode) GetDeviationMultiplier(isActiveTransition bool) float64 {
	if isActiveTransition {
		return h.deviationMultiplier
	}
	return 1.0
}

// GetPAIContextBoost returns points to add to PAI context dimension.
// 15.0 during transition, 0.0 normally.
func (h *HeightenedSurveillanceMode) GetPAIContextBoost(isActiveTransition bool) float64 {
	if isActiveTransition {
		return h.paiContextBoost
	}
	return 0.0
}

// GetEngagementGapHours returns the measurement gap threshold for alerting.
// 72 hours during transition, 168 (7 days) normally.
func (h *HeightenedSurveillanceMode) GetEngagementGapHours(isActiveTransition bool) int {
	if isActiveTransition {
		return h.engagementGapHours
	}
	return 168
}

// AmplifyEscalationTier upgrades tier by one level during transition.
// ROUTINE→URGENT, URGENT→IMMEDIATE, IMMEDIATE stays IMMEDIATE.
// Outside transition, the tier is returned unchanged.
func (h *HeightenedSurveillanceMode) AmplifyEscalationTier(tier string, isActiveTransition bool) string {
	if !isActiveTransition {
		return tier
	}
	switch tier {
	case "ROUTINE":
		return "URGENT"
	case "URGENT":
		return "IMMEDIATE"
	default:
		return tier // IMMEDIATE and any unknown tier stay as-is
	}
}
