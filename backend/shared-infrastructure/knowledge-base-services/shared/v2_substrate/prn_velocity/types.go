// Package prn_velocity implements the PRN (pro re nata, "as needed")
// administration velocity primitive — a CAPE Layer 1 resident stability
// signal.
//
// Spec: docs/superpowers/plans/CAPE_Implementation_Guidelines_v1_1.md
// lines 271–290 (CQL definition) and lines 569–571 (Phase 1 signal class
// names). The canonical CQL definition is preserved verbatim alongside
// this Go implementation in cql/prn_escalation_velocity.cql; the Go
// implementation is the runtime authority.
//
// This package is pure: no I/O, no clock reads beyond the supplied `now`,
// no package-level mutable state. Compute is safe for concurrent calls.
// Persistence (kb-33, Step 5) is the caller's concern.
//
// Window semantics (half-open):
//
//	Recent 30-day window:   (now-30d,  now]
//	Prior 90-day baseline: (now-120d, now-30d]
//
// An administration whose AdministeredAt equals exactly `now-30d` falls
// in the BASELINE window, not the recent window. An administration whose
// AdministeredAt equals exactly `now-120d` is OUTSIDE both windows
// (excluded). An administration whose AdministeredAt equals exactly `now`
// IS included in the recent window (closed right boundary).
//
// VisibilityClass: PDP (Pharmacist-Default-Private) — resident clinical
// signal derived from administration records.
package prn_velocity

import (
	"time"

	"github.com/google/uuid"
)

// PRNClass identifies a medication class for which administration velocity
// is computed. Phase 1 covers three classes per CAPE Guidelines lines
// 569–571. CAPE line 275 names additional candidates (antiemetic, laxative,
// sedative) that are NOT in scope for Phase 1.
type PRNClass string

const (
	PRNBenzodiazepine PRNClass = "benzodiazepine"
	PRNAntipsychotic  PRNClass = "antipsychotic"
	PRNAnalgesic      PRNClass = "analgesic"
)

// IsValidPRNClass reports whether s is one of the three Phase 1 PRN classes.
func IsValidPRNClass(s string) bool {
	switch PRNClass(s) {
	case PRNBenzodiazepine, PRNAntipsychotic, PRNAnalgesic:
		return true
	default:
		return false
	}
}

// Administration is a single PRN administration event. Compute consumes
// a slice of these; the caller is responsible for filtering by resident
// and PRN class before passing them in. Compute itself is defensive and
// will skip any administration whose ResidentID or Class does not match
// the requested target, but callers should not rely on that.
type Administration struct {
	ResidentID     uuid.UUID
	Class          PRNClass
	AdministeredAt time.Time
}

// VelocityResult is the output of Compute — a 30-day-vs-90-day-baseline
// escalation snapshot for one (resident, class) pair at one wall-clock
// time.
type VelocityResult struct {
	ResidentID  uuid.UUID
	Class       PRNClass
	EvaluatedAt time.Time

	// Recent30dCount is the count of administrations in (now-30d, now].
	Recent30dCount int

	// Baseline90dAvg is the mean per-30d administration count over the
	// prior 90 days, i.e. total count in (now-120d, now-30d] divided by 3.0.
	Baseline90dAvg float64

	// VelocityRatio = Recent30dCount / Baseline90dAvg, with these special
	// cases (see compute.go):
	//   - Baseline == 0 and Recent  > 0 → +Inf (emergent class use)
	//   - Baseline == 0 and Recent == 0 → 0    (0/0 → 0 by convention)
	//   - Baseline  > 0 and Recent == 0 → 0
	VelocityRatio float64

	// Severity is the 1..5 bucket per CAPE Guidelines lines 283–289.
	Severity int
}
