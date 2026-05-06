package delta

import (
	"math"
	"time"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// Threshold constants (in standard-deviation units).
//
// |dev| <= 1.0           → within_baseline
// 1.0 < dev <= 2.0       → elevated      (high side)
// dev > 2.0              → severely_elevated
// -2.0 <= dev < -1.0     → low           (low side)
// dev < -2.0             → severely_low
//
// Boundary semantics: thresholds are inclusive on the within_baseline side
// (|dev|=1.0 stays within_baseline; |dev|=2.0 stays elevated/low). This
// matches spec §3.7 description "elevated when 1<dev<=2".
const (
	thresholdWithin   = 1.0
	thresholdSevere   = 2.0
)

// ComputeDelta returns the directional Delta for obs given baseline.
// Pure function: no IO, no time.Now beyond stamping ComputedAt.
//
// Returns DeltaFlagNoBaseline (with zeroed numeric fields) when:
//   - baseline is nil
//   - obs.Value is nil (e.g. behavioural ValueText-only observation)
//   - obs.Kind == ObservationKindBehavioural (no numeric semantics — see spec §8 risk row)
//   - baseline.StdDev == 0 (would yield Inf/NaN; treat as insufficient data)
func ComputeDelta(obs models.Observation, baseline *Baseline) models.Delta {
	now := time.Now().UTC()

	if baseline == nil ||
		obs.Value == nil ||
		obs.Kind == models.ObservationKindBehavioural ||
		baseline.StdDev == 0 {
		return models.Delta{
			BaselineValue:   0,
			DeviationStdDev: 0,
			DirectionalFlag: models.DeltaFlagNoBaseline,
			ComputedAt:      now,
		}
	}

	deviation := (*obs.Value - baseline.BaselineValue) / baseline.StdDev
	abs := math.Abs(deviation)

	var flag string
	switch {
	case abs <= thresholdWithin:
		flag = models.DeltaFlagWithinBaseline
	case deviation > thresholdSevere:
		flag = models.DeltaFlagSeverelyElevated
	case deviation > thresholdWithin: // 1 < dev <= 2
		flag = models.DeltaFlagElevated
	case deviation < -thresholdSevere:
		flag = models.DeltaFlagSeverelyLow
	default: // -2 <= dev < -1
		flag = models.DeltaFlagLow
	}

	return models.Delta{
		BaselineValue:   baseline.BaselineValue,
		DeviationStdDev: deviation,
		DirectionalFlag: flag,
		ComputedAt:      now,
	}
}
