// Package kpis_test — TDD tests for RIR computation.
//
// VisibilityClass: PFA — aggregation gate enforced by Phase 1a middleware.
package kpis

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRIR_BasicFormula(t *testing.T) {
	pharm := uuid.New()
	now := time.Now().UTC()
	recs := []RecRow{
		{AuthorID: pharm, State: "implemented", SubmittedAt: now.AddDate(0, 0, -45)},
		{AuthorID: pharm, State: "implemented", SubmittedAt: now.AddDate(0, 0, -50)},
		{AuthorID: pharm, State: "submitted", SubmittedAt: now.AddDate(0, 0, -40)}, // age > 30, denominator
		{AuthorID: pharm, State: "drafted", SubmittedAt: now.AddDate(0, 0, -10)},   // age < 30, excluded
	}
	rir := ComputeRIR(recs, pharm, now, 30)
	if got, want := rir, 2.0/3.0; got < want-0.01 || got > want+0.01 {
		t.Errorf("RIR = %v, want ~%v", got, want)
	}
}

func TestRIR_NoEligibleReturnsNaN(t *testing.T) {
	now := time.Now().UTC()
	rir := ComputeRIR([]RecRow{{State: "drafted", SubmittedAt: now}}, uuid.New(), now, 30)
	if !math.IsNaN(rir) {
		t.Errorf("expected NaN for empty denominator, got %v", rir)
	}
}

// TestRIR_AuthorFilterExcludesOthers verifies that records from other authors
// do not pollute the target pharmacist's RIR.
func TestRIR_AuthorFilterExcludesOthers(t *testing.T) {
	pharm := uuid.New()
	other := uuid.New()
	now := time.Now().UTC()
	recs := []RecRow{
		// Target pharmacist: 1 eligible, 1 implemented
		{AuthorID: pharm, State: "implemented", SubmittedAt: now.AddDate(0, 0, -45)},
		// Other pharmacist: 5 implemented — must be ignored
		{AuthorID: other, State: "implemented", SubmittedAt: now.AddDate(0, 0, -45)},
		{AuthorID: other, State: "implemented", SubmittedAt: now.AddDate(0, 0, -50)},
		{AuthorID: other, State: "implemented", SubmittedAt: now.AddDate(0, 0, -60)},
		{AuthorID: other, State: "implemented", SubmittedAt: now.AddDate(0, 0, -70)},
		{AuthorID: other, State: "implemented", SubmittedAt: now.AddDate(0, 0, -80)},
	}
	rir := ComputeRIR(recs, pharm, now, 30)
	if got, want := rir, 1.0; got < want-0.01 || got > want+0.01 {
		t.Errorf("RIR = %v, want %v (other authors must be excluded)", got, want)
	}
}

// TestRIR_StatesBeyondImplementedAlsoCount verifies that outcome_recorded and
// closed states count toward the numerator ("implemented or beyond").
func TestRIR_StatesBeyondImplementedAlsoCount(t *testing.T) {
	pharm := uuid.New()
	now := time.Now().UTC()
	recs := []RecRow{
		{AuthorID: pharm, State: "implemented", SubmittedAt: now.AddDate(0, 0, -31)},
		{AuthorID: pharm, State: "outcome_recorded", SubmittedAt: now.AddDate(0, 0, -35)},
		{AuthorID: pharm, State: "closed", SubmittedAt: now.AddDate(0, 0, -40)},
		{AuthorID: pharm, State: "submitted", SubmittedAt: now.AddDate(0, 0, -32)}, // eligible, not numerator
	}
	rir := ComputeRIR(recs, pharm, now, 30)
	// numerator = 3 (implemented + outcome_recorded + closed), denominator = 4
	if got, want := rir, 3.0/4.0; got < want-0.01 || got > want+0.01 {
		t.Errorf("RIR = %v, want ~%v", got, want)
	}
}
