package citations

import (
	"testing"
	"time"
)

// TestVersionStatusConstants verifies the four canonical status string values
// have not been accidentally changed.
func TestVersionStatusConstants(t *testing.T) {
	cases := []struct {
		status VersionStatus
		want   string
	}{
		{StatusActive, "active"},
		{StatusAmended, "amended"},
		{StatusRetracted, "retracted"},
		{StatusSuperseded, "superseded"},
	}
	for _, tc := range cases {
		if string(tc.status) != tc.want {
			t.Errorf("VersionStatus constant: got %q, want %q", string(tc.status), tc.want)
		}
	}
}

// TestActiveAtOpenInterval verifies that a nil EffectiveTo (open interval)
// returns true for any asOf at or after EffectiveFrom.
func TestActiveAtOpenInterval(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sv := SourceVersion{
		SourceID:      "src-open",
		Version:       "1",
		EffectiveFrom: from,
		EffectiveTo:   nil,
		Status:        StatusActive,
	}

	if !sv.ActiveAt(from) {
		t.Error("ActiveAt(from) on open interval = false, want true")
	}
	if !sv.ActiveAt(from.Add(365 * 24 * time.Hour)) {
		t.Error("ActiveAt(far future) on open interval = false, want true")
	}
	if sv.ActiveAt(from.Add(-time.Nanosecond)) {
		t.Error("ActiveAt(before from) on open interval = true, want false")
	}
}

// TestActiveAtClosedOpen verifies closed-open interval semantics for a version
// with a set EffectiveTo.
func TestActiveAtClosedOpen(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	sv := SourceVersion{
		SourceID:      "src-closed-open",
		Version:       "1",
		EffectiveFrom: from,
		EffectiveTo:   &to,
		Status:        StatusAmended,
	}

	// Closed lower bound.
	if !sv.ActiveAt(from) {
		t.Error("ActiveAt(from) = false, want true (inclusive lower bound)")
	}

	// Open upper bound: exactly at EffectiveTo is NOT active.
	if sv.ActiveAt(to) {
		t.Error("ActiveAt(to) = true, want false (exclusive upper bound)")
	}

	// Just before EffectiveTo is still active.
	if !sv.ActiveAt(to.Add(-time.Nanosecond)) {
		t.Error("ActiveAt(to - 1ns) = false, want true")
	}

	// After EffectiveTo is not active.
	if sv.ActiveAt(to.Add(time.Second)) {
		t.Error("ActiveAt(after to) = true, want false")
	}
}
