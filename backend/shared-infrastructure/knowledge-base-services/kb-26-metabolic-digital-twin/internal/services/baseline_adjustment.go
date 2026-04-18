package services

import "kb-26-metabolic-digital-twin/internal/models"

// BaselineAdjustmentController implements a 4-state post-discharge machine
// that manages baseline validity after hospital discharge. Hospital readings
// (IV fluids, bed rest, different meds) are irrelevant to home patterns, so
// this controller excludes them, builds a new home baseline, and transitions
// to steady state.
type BaselineAdjustmentController struct{}

// NewBaselineAdjustmentController creates a new controller instance.
func NewBaselineAdjustmentController() *BaselineAdjustmentController {
	return &BaselineAdjustmentController{}
}

// DetermineBaselineStage returns the baseline stage based on days since discharge
// and the number of post-discharge readings accumulated.
func (c *BaselineAdjustmentController) DetermineBaselineStage(
	daysSinceDischarge int,
	postDischargeReadingCount int,
) string {
	if daysSinceDischarge <= 2 {
		return models.BaselineStageHospitalInfluenced
	}
	if daysSinceDischarge > 30 {
		return models.BaselineStageSteadyState
	}
	if postDischargeReadingCount >= 5 && daysSinceDischarge >= 14 {
		return models.BaselineStagePostDischargeEvolving
	}
	return models.BaselineStageBuildingNew
}

// ShouldSuppressDeviation returns true if deviation detection should be
// suppressed for this baseline stage. CRITICAL absolute threshold breaches
// (SBP >180, eGFR <20, K+ >6.0) are NEVER suppressed.
func (c *BaselineAdjustmentController) ShouldSuppressDeviation(
	stage string,
	severity string,
) bool {
	if severity == "CRITICAL" {
		return false // CRITICAL always fires regardless of baseline state
	}
	return stage == models.BaselineStageHospitalInfluenced
}

// GetThresholdMultiplier returns the deviation threshold multiplier for a stage.
// During BUILDING stage, thresholds are widened (1.5x = less sensitive).
// During EVOLVING, standard thresholds apply.
// During STEADY_STATE, standard thresholds apply.
func (c *BaselineAdjustmentController) GetThresholdMultiplier(stage string) float64 {
	switch stage {
	case models.BaselineStageHospitalInfluenced:
		return 0 // suppressed entirely (handled by ShouldSuppressDeviation)
	case models.BaselineStageBuildingNew:
		return 1.5 // widened thresholds — less sensitive
	case models.BaselineStagePreAdmissionFallback:
		return 1.5
	default: // EVOLVING, STEADY_STATE
		return 1.0
	}
}

// SelectBaseline determines which baseline to use for deviation detection.
// Returns "POST_DISCHARGE" if enough post-discharge readings exist,
// "PRE_ADMISSION" as fallback, or "NONE" if neither available.
func (c *BaselineAdjustmentController) SelectBaseline(
	stage string,
	hasPostDischargeBaseline bool,
	hasPreAdmissionBaseline bool,
) string {
	switch stage {
	case models.BaselineStageHospitalInfluenced:
		return "NONE" // suppress all (except CRITICAL)
	case models.BaselineStageBuildingNew:
		if hasPreAdmissionBaseline {
			return "PRE_ADMISSION"
		}
		return "NONE"
	case models.BaselineStagePostDischargeEvolving:
		if hasPostDischargeBaseline {
			return "POST_DISCHARGE"
		}
		if hasPreAdmissionBaseline {
			return "PRE_ADMISSION"
		}
		return "NONE"
	default: // STEADY_STATE
		return "POST_DISCHARGE"
	}
}
