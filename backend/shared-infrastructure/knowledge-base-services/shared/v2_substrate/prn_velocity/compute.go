package prn_velocity

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// Compute returns the VelocityResult for a (resident, class) administration
// stream evaluated at `now`. The function is pure: no I/O, no clock reads
// beyond the supplied `now`. The caller is expected to pre-filter
// administrations to a single resident and class; Compute will defensively
// skip any administration whose ResidentID or Class does not match.
//
// Severity bucketing (CAPE Guidelines v1.1 lines 283–289):
//
//	velocity_ratio > 4.0 → Severity 5  (400%+ increase)
//	velocity_ratio > 2.5 → Severity 4  (250%+ increase)
//	velocity_ratio > 1.5 → Severity 3  (150%+ increase)
//	velocity_ratio > 1.0 → Severity 2  (any increase)
//	else                 → Severity 1
//
// Special cases:
//
//   - Baseline empty and Recent30dCount > 0: VelocityRatio = +Inf,
//     Severity = 5 (emergent class use — CRITICAL).
//   - Baseline empty and Recent30dCount == 0: VelocityRatio = 0,
//     Severity = 1 (0/0 → 0 by convention).
//   - Baseline present but Recent30dCount == 0: VelocityRatio = 0,
//     Severity = 1.
//
// Window semantics are half-open; see package doc on types.go.
func Compute(administrations []Administration, residentID uuid.UUID, class PRNClass, now time.Time) VelocityResult {
	recentStart := now.Add(-30 * 24 * time.Hour)    // exclusive lower bound for recent window
	baselineStart := now.Add(-120 * 24 * time.Hour) // exclusive lower bound for baseline window

	var recentCount int
	var baselineCount int

	for _, a := range administrations {
		// Defensive filter — caller should already have filtered, but a
		// stray administration shouldn't corrupt the count.
		if a.ResidentID != residentID || a.Class != class {
			continue
		}

		t := a.AdministeredAt

		// Recent window: (now-30d, now]
		if t.After(recentStart) && !t.After(now) {
			recentCount++
			continue
		}

		// Baseline window: (now-120d, now-30d]
		// (t > baselineStart) AND (t <= recentStart)
		if t.After(baselineStart) && !t.After(recentStart) {
			baselineCount++
		}
	}

	baselineAvg := float64(baselineCount) / 3.0

	var ratio float64
	var severity int

	switch {
	case baselineAvg == 0 && recentCount > 0:
		ratio = math.Inf(+1)
		severity = 5
	case baselineAvg == 0 && recentCount == 0:
		ratio = 0
		severity = 1
	default:
		ratio = float64(recentCount) / baselineAvg
		severity = severityFromRatio(ratio)
	}

	return VelocityResult{
		ResidentID:     residentID,
		Class:          class,
		EvaluatedAt:    now,
		Recent30dCount: recentCount,
		Baseline90dAvg: baselineAvg,
		VelocityRatio:  ratio,
		Severity:       severity,
	}
}

// severityFromRatio applies the CAPE bucket table. Uses strict `>` per the
// CQL `when velocity_ratio > X.X` semantics — a ratio of exactly 4.0 falls
// in bucket 4, not 5.
func severityFromRatio(ratio float64) int {
	switch {
	case ratio > 4.0:
		return 5
	case ratio > 2.5:
		return 4
	case ratio > 1.5:
		return 3
	case ratio > 1.0:
		return 2
	default:
		return 1
	}
}
