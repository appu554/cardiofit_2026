// Package kpis_test — TDD tests for class-specific rate computation.
//
// VisibilityClass: PFA — aggregation gate enforced by Phase 1a middleware.
package kpis

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestClassSpecific_ClassFilter verifies that ComputeClassSpecificRate correctly
// partitions results by drug class.
func TestClassSpecific_ClassFilter(t *testing.T) {
	pharm := uuid.New()
	now := time.Now().UTC()
	recs := []RecRow{
		// Class A: 2 eligible, 1 implemented → rate = 0.5
		{AuthorID: pharm, Class: "A", State: "implemented", SubmittedAt: now.AddDate(0, 0, -45)},
		{AuthorID: pharm, Class: "A", State: "submitted", SubmittedAt: now.AddDate(0, 0, -40)},
		// Class B: 2 eligible, 2 implemented → rate = 1.0
		{AuthorID: pharm, Class: "B", State: "implemented", SubmittedAt: now.AddDate(0, 0, -35)},
		{AuthorID: pharm, Class: "B", State: "implemented", SubmittedAt: now.AddDate(0, 0, -50)},
	}

	rateA := ComputeClassSpecificRate(recs, pharm, "A", now, 30)
	if got, want := rateA, 0.5; got < want-0.01 || got > want+0.01 {
		t.Errorf("class A rate = %v, want %v", got, want)
	}

	rateB := ComputeClassSpecificRate(recs, pharm, "B", now, 30)
	if got, want := rateB, 1.0; got < want-0.01 || got > want+0.01 {
		t.Errorf("class B rate = %v, want %v", got, want)
	}
}

// TestClassSpecific_UnknownClassReturnsNaN verifies that querying a class with
// no eligible records returns NaN (denominator is zero).
func TestClassSpecific_UnknownClassReturnsNaN(t *testing.T) {
	pharm := uuid.New()
	now := time.Now().UTC()
	recs := []RecRow{
		{AuthorID: pharm, Class: "A", State: "implemented", SubmittedAt: now.AddDate(0, 0, -45)},
	}
	rate := ComputeClassSpecificRate(recs, pharm, "Z", now, 30)
	if !math.IsNaN(rate) {
		t.Errorf("expected NaN for unknown class, got %v", rate)
	}
}
