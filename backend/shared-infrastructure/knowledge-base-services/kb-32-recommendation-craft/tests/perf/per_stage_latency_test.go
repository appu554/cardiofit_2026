// Package perf_test validates per-stage latency budgets for the six-stage
// recommendation rendering pipeline using in-memory fakes.
//
// Recommendation Craft Guidelines Part 13 — performance test category.
//
// Stages 1–3 require I/O (substrate / HAPI / generator with ClinicalSnapshot
// substrate client) that is impractical to isolate for sub-millisecond
// measurement without infrastructure. These stages are deferred:
//
//   - Stage 1 (context assembly): requires substrate client network call;
//     covered by integration tests in tests/integration/.
//   - Stage 2 (reasoning/chain builder): requires HAPI client network call;
//     covered by integration tests in tests/integration/.
//   - Stage 3 (generator): allocates maps/UUIDs which adds noise; covered
//     by integration tests.
//
// This file measures Stages 4–6 (pure-function, zero I/O) against the hard
// caps defined in Recommendation Craft Guidelines §13:
//
//   - Stage 4 (appropriateness.Check):   p95 ≤ 50ms
//   - Stage 5 (framing.ContentHash):     p95 ≤ 30ms
//   - Stage 6 (formatter.Validate):      p95 ≤ 20ms
//
// VisibilityClass: perf — latency caps per Guidelines §13
package perf_test

import (
	"sort"
	"testing"
	"time"

	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/formatter"
	"github.com/cardiofit/kb32/internal/framing"
)

// Iterations is the number of measurement runs per stage. 100 runs produces
// a statistically stable p95 estimate for pure-function stages while keeping
// test wall time well under one second.
const Iterations = 100

// p95 returns the 95th-percentile value from a slice of durations.
// The slice is sorted in place (ascending); the index is clamped to the slice
// bounds to protect against rounding at the high end.
func p95(durs []time.Duration) time.Duration {
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	idx := int(0.95 * float64(len(durs)))
	if idx >= len(durs) {
		idx = len(durs) - 1
	}
	return durs[idx]
}

// TestPerStageLatency_Stage4AppropriatenessCheck validates that the
// appropriateness gate (Stage 4) satisfies its p95 latency budget.
//
// Guidelines §13 hard cap: Stage 4 appropriateness check ≤ 50ms p95.
// In practice this pure-function should be sub-microsecond; the 50ms cap is
// a conservative worst-case guard against scheduler jitter on CI runners.
func TestPerStageLatency_Stage4AppropriatenessCheck(t *testing.T) {
	t.Parallel()

	a := appropriateness.Assessment{
		ClinicalWarrant:        4,
		EvidenceSolidity:       4,
		AlternativesConsidered: 4,
		RestraintConsidered:    4,
		GoalsOfCareAlignment:   4,
	}

	durs := make([]time.Duration, Iterations)
	for i := 0; i < Iterations; i++ {
		start := time.Now()
		_ = appropriateness.Check(a)
		durs[i] = time.Since(start)
	}

	// Guidelines §13 hard cap: Stage 4 appropriateness check ≤ 50ms p95
	const cap = 50 * time.Millisecond
	if got := p95(durs); got > cap {
		t.Errorf(
			"Stage 4 appropriateness.Check: p95 latency = %v exceeds hard cap %v "+
				"(Guidelines §13)",
			got, cap,
		)
	}
}

// TestPerStageLatency_Stage5ContentHash validates that the framing content
// hash computation (Stage 5) satisfies its p95 latency budget.
//
// Guidelines §13 hard cap: Stage 5 ContentHash ≤ 30ms p95.
func TestPerStageLatency_Stage5ContentHash(t *testing.T) {
	t.Parallel()

	cc := framing.ClinicalContent{
		RuleID:          "STOP_PPI_001",
		Type:            "STOP",
		EvidenceAnchors: []string{"anchor-a", "anchor-b", "anchor-c"},
		Urgency:         "amber",
	}

	durs := make([]time.Duration, Iterations)
	for i := 0; i < Iterations; i++ {
		start := time.Now()
		_ = framing.ContentHash(cc)
		durs[i] = time.Since(start)
	}

	// Guidelines §13 hard cap: Stage 5 ContentHash ≤ 30ms p95
	const cap = 30 * time.Millisecond
	if got := p95(durs); got > cap {
		t.Errorf(
			"Stage 5 framing.ContentHash: p95 latency = %v exceeds hard cap %v "+
				"(Guidelines §13)",
			got, cap,
		)
	}
}

// TestPerStageLatency_Stage6FormatterValidate validates that the formatter
// word-budget validation (Stage 6) satisfies its p95 latency budget.
//
// Guidelines §13 hard cap: Stage 6 formatter.Validate ≤ 20ms p95.
func TestPerStageLatency_Stage6FormatterValidate(t *testing.T) {
	t.Parallel()

	// Construct a LayerOutput that is within budget on both layers.
	out := formatter.LayerOutput{
		L1Signal:    "Stop PPI — no documented indication found for this resident.",
		L2Reasoning: "Proton pump inhibitor prescribed without a documented indication. Guidelines recommend review and deprescribing when no active peptic ulcer or GERD diagnosis is present. Risks include hypomagnesaemia and Clostridioides difficile infection.",
	}

	durs := make([]time.Duration, Iterations)
	for i := 0; i < Iterations; i++ {
		start := time.Now()
		_ = formatter.Validate(out)
		durs[i] = time.Since(start)
	}

	// Guidelines §13 hard cap: Stage 6 formatter.Validate ≤ 20ms p95
	const cap = 20 * time.Millisecond
	if got := p95(durs); got > cap {
		t.Errorf(
			"Stage 6 formatter.Validate: p95 latency = %v exceeds hard cap %v "+
				"(Guidelines §13)",
			got, cap,
		)
	}
}

// TestPerStageLatency_AllStagesUnderCap is a composite assertion that runs
// all three in-process stages sequentially and verifies that each individual
// p95 satisfies its Guidelines §13 hard cap.
//
// This test is intentionally redundant with the three individual tests above
// but provides a single point of failure for CI dashboards that track
// per-stage latency regression.
func TestPerStageLatency_AllStagesUnderCap(t *testing.T) {
	t.Parallel()

	stages := []struct {
		name string
		fn   func() // zero-alloc operation to measure
		cap  time.Duration
	}{
		{
			name: "stage4_appropriateness_check",
			fn: func() {
				a := appropriateness.Assessment{
					ClinicalWarrant: 4, EvidenceSolidity: 4,
					AlternativesConsidered: 4, RestraintConsidered: 4, GoalsOfCareAlignment: 4,
				}
				_ = appropriateness.Check(a)
			},
			// Guidelines §13 hard cap: Stage 4 appropriateness check ≤ 50ms p95
			cap: 50 * time.Millisecond,
		},
		{
			name: "stage5_content_hash",
			fn: func() {
				cc := framing.ClinicalContent{
					RuleID: "X", Type: "STOP",
					EvidenceAnchors: []string{"A", "B"}, Urgency: "amber",
				}
				_ = framing.ContentHash(cc)
			},
			// Guidelines §13 hard cap: Stage 5 ContentHash ≤ 30ms p95
			cap: 30 * time.Millisecond,
		},
		{
			name: "stage6_formatter_validate",
			fn: func() {
				out := formatter.LayerOutput{
					L1Signal:    "Short signal text here.",
					L2Reasoning: "Short reasoning text.",
				}
				_ = formatter.Validate(out)
			},
			// Guidelines §13 hard cap: Stage 6 formatter.Validate ≤ 20ms p95
			cap: 20 * time.Millisecond,
		},
	}

	for _, s := range stages {
		s := s // capture loop var
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()
			durs := make([]time.Duration, Iterations)
			for i := 0; i < Iterations; i++ {
				start := time.Now()
				s.fn()
				durs[i] = time.Since(start)
			}
			if got := p95(durs); got > s.cap {
				t.Errorf(
					"%s: p95 latency = %v exceeds hard cap %v (Guidelines §13)",
					s.name, got, s.cap,
				)
			}
		})
	}
}
