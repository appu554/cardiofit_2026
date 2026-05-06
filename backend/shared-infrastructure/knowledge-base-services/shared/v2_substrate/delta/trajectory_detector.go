package delta

// TrajectorySnapshot captures the trend signal computed alongside the
// running baseline at recompute time. Layer 2 doc §1.4 (lines 277-285)
// defines is_trending as 3+ consecutive observations moving in the same
// direction relative to the immediately prior reading; the snapshot
// also reports the raw consecutive count and the trailing direction
// so downstream rules ("eGFR has been falling for 5 consecutive
// readings") can read the count directly without re-running detection.
type TrajectorySnapshot struct {
	// IsTrending is true when ConsecutiveSameDirection >= 3 (per Layer 2
	// §1.4). The minimum-three-in-a-row threshold avoids triggering on
	// a single noisy reading; clinical rules that want a hairier
	// trigger (e.g. "any drop") should compare BaselineValue + StdDev
	// directly via ComputeDelta rather than this flag.
	IsTrending bool `json:"is_trending"`
	// ConsecutiveSameDirection is the count of trailing observations
	// moving in the same direction (most recent backwards, until a
	// direction reversal or flat-step). Always 0 when the trailing
	// step is flat (equal to prior). Always >= 1 when at least one
	// non-flat trailing step exists.
	ConsecutiveSameDirection int `json:"consecutive_same_direction"`
	// Direction is "up" or "down" describing the trailing trend. Empty
	// string when ConsecutiveSameDirection == 0 (flat or insufficient
	// data).
	Direction string `json:"direction,omitempty"`
}

// DetectTrajectory computes the trajectory snapshot for a value series.
// Pure function: zero DB I/O, zero allocations beyond the direction
// scratch slice. Designed to run inside the recompute critical section
// alongside Percentiles + ClassifyBaselineConfidence on the same value
// slice — no extra DB round-trip.
//
// Input convention: values are passed in observed_at-DESC order (most
// recent first), matching the convention used by recomputeAndUpsertWith
// throughout the BaselineStore. Internally we walk in chronological
// (ASC) order so "trailing" means "most recent" — the function
// reverses values once into an ASC scratch slice.
//
// Algorithm (per Layer 2 §1.4):
//  1. Reverse values to chronological order.
//  2. Compute pairwise direction between adjacent samples ("up", "down",
//     or "flat" depending on sign of the difference).
//  3. Walk directions from the most-recent end backwards counting
//     consecutive same-direction steps. Stop on the first reversal or
//     flat step.
//  4. is_trending = count >= 3.
//
// Edge cases:
//   - len(values) < 3 → returns {IsTrending: false, Count: 0,
//     Direction: ""}. We need at least 3 values to produce 2 pairwise
//     directions; the spec calls for 3+ consecutive same-direction
//     observations, which requires count(directions) >= 3, hence
//     count(values) >= 4. But we also surface count for diagnostic
//     use (e.g. "2-step decline, not yet trending") so we return what
//     we can compute.
//   - Trailing step is flat → returns {IsTrending: false, Count: 0,
//     Direction: ""}. Any flat tail kills the trajectory regardless
//     of what came before.
func DetectTrajectory(values []float64) TrajectorySnapshot {
	if len(values) < 3 {
		return TrajectorySnapshot{}
	}

	// Reverse to ASC order. Allocates once; the caller's values slice
	// is not mutated.
	asc := make([]float64, len(values))
	for i, v := range values {
		asc[len(values)-1-i] = v
	}

	// Pairwise directions. "flat" only on exact equality — float
	// comparisons here are intentionally unrounded so a tiny but
	// real drift counts as a direction. Clinical thresholds (the
	// "20% decline" velocity rule) live elsewhere; this detector
	// is purely about ordinal direction.
	directions := make([]string, 0, len(asc)-1)
	for i := 1; i < len(asc); i++ {
		switch {
		case asc[i] > asc[i-1]:
			directions = append(directions, "up")
		case asc[i] < asc[i-1]:
			directions = append(directions, "down")
		default:
			directions = append(directions, "flat")
		}
	}

	last := directions[len(directions)-1]
	if last == "flat" {
		return TrajectorySnapshot{}
	}

	count := 1
	for i := len(directions) - 2; i >= 0 && directions[i] == last; i-- {
		count++
	}

	return TrajectorySnapshot{
		IsTrending:               count >= 3,
		ConsecutiveSameDirection: count,
		Direction:                last,
	}
}
