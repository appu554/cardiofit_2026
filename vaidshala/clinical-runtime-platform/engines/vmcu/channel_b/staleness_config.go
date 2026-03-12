package channel_b

import "time"

// StalenessThresholds defines maximum acceptable age for each lab type.
// If a measurement timestamp is older than its threshold, Channel B
// treats it as absent (nil-equivalent) rather than using stale data
// for safety-critical decisions.
type StalenessThresholds struct {
	Glucose    time.Duration // rapid physiological change
	Potassium  time.Duration // moderate variability
	Creatinine time.Duration // slow-moving marker
	EGFR       time.Duration // derived, slow trend
	HbA1c      time.Duration // 3-month average marker
	SBP        time.Duration // rapid physiological change
	Weight     time.Duration // matches B-06 delta window
}

// DefaultStalenessThresholds returns clinically-validated staleness limits.
//
// Rationale for each threshold:
//   - Glucose (4h): can swing 5+ mmol/L in hours; stale glucose is dangerous
//   - Potassium (12h): moderate variability; renal/cardiac risk window
//   - Creatinine (48h): slow-moving; AKI detection requires recent but not real-time
//   - eGFR (7d): derived from creatinine; trajectory matters more than point-in-time
//   - HbA1c (90d): 3-month glycated hemoglobin average by definition
//   - SBP (4h): hemodynamic instability can develop rapidly
//   - Weight (72h): matches B-06 delta window for fluid retention detection
func DefaultStalenessThresholds() StalenessThresholds {
	return StalenessThresholds{
		Glucose:    4 * time.Hour,
		Potassium:  12 * time.Hour,
		Creatinine: 48 * time.Hour,
		EGFR:       7 * 24 * time.Hour,
		HbA1c:      90 * 24 * time.Hour,
		SBP:        4 * time.Hour,
		Weight:     72 * time.Hour,
	}
}

// IsStale returns true if the given timestamp is older than the max age,
// or if the timestamp is nil (treated as infinitely stale).
func IsStale(ts *time.Time, maxAge time.Duration) bool {
	if ts == nil {
		return true
	}
	return time.Since(*ts) > maxAge
}
