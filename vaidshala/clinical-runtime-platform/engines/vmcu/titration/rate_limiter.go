// rate_limiter.go implements Commitment 2: Rate limiter post-resume (Phase 8.2).
//
// After a freeze/resume, the max dose delta is reduced to 50%
// for ceil(pause_hours / 24) cycles. This prevents aggressive
// dose changes immediately after a safety pause.
package titration

import "math"

// RateLimiter constrains dose deltas after a resume event.
type RateLimiter struct {
	// PostResumeReductionPct is the fraction of normal max delta allowed
	// during the post-resume window. Default: 0.50 (50%).
	PostResumeReductionPct float64

	// RemainingLimitedCycles tracks how many more cycles use reduced deltas.
	RemainingLimitedCycles int
}

// NewRateLimiter creates a rate limiter with default 50% reduction.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		PostResumeReductionPct: 0.50,
	}
}

// ActivatePostResume sets up rate limiting for ceil(pauseHours/24) cycles.
func (rl *RateLimiter) ActivatePostResume(pauseHours float64) {
	cycles := int(math.Ceil(pauseHours / 24.0))
	if cycles < 1 {
		cycles = 1
	}
	rl.RemainingLimitedCycles = cycles
}

// IsLimited returns true if dose changes are currently rate-limited.
func (rl *RateLimiter) IsLimited() bool {
	return rl.RemainingLimitedCycles > 0
}

// ApplyLimit adjusts the max dose delta if rate-limited.
// Returns the adjusted maxDoseDeltaPct for this cycle.
// Decrements the remaining cycle count.
func (rl *RateLimiter) ApplyLimit(normalMaxDeltaPct float64) float64 {
	if rl.RemainingLimitedCycles <= 0 {
		return normalMaxDeltaPct
	}
	rl.RemainingLimitedCycles--
	return normalMaxDeltaPct * rl.PostResumeReductionPct
}
