package urgency

import (
	"testing"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
)

// baseSnap returns a zero-risk ClinicalSnapshot (all flags false, scores at
// low values). Tests modify only the field(s) under test to keep cases focused.
func baseSnap() kb32ctx.ClinicalSnapshot {
	return kb32ctx.ClinicalSnapshot{
		ACB:                0,
		DBI:                0.0,
		CareIntensity:      "active",
		RecentFall72h:      false,
		RecentAdmission72h: false,
	}
}

// ---------------------------------------------------------------------------
// IsValidUrgency
// ---------------------------------------------------------------------------

func TestIsValidUrgency(t *testing.T) {
	valid := []string{UrgencyRed, UrgencyAmber, UrgencyGreen}
	for _, u := range valid {
		if !IsValidUrgency(u) {
			t.Errorf("IsValidUrgency(%q) = false, want true", u)
		}
	}

	invalid := []string{"RED", "AMBER", "GREEN", "", "orange", "critical"}
	for _, u := range invalid {
		if IsValidUrgency(u) {
			t.Errorf("IsValidUrgency(%q) = true, want false", u)
		}
	}
}

// ---------------------------------------------------------------------------
// Red triggers
// ---------------------------------------------------------------------------

func TestTag_RecentFall(t *testing.T) {
	snap := baseSnap()
	snap.RecentFall72h = true
	if got := Tag(snap); got != UrgencyRed {
		t.Errorf("Tag = %q, want %q", got, UrgencyRed)
	}
}

func TestTag_RecentAdmission(t *testing.T) {
	snap := baseSnap()
	snap.RecentAdmission72h = true
	if got := Tag(snap); got != UrgencyRed {
		t.Errorf("Tag = %q, want %q", got, UrgencyRed)
	}
}

// ---------------------------------------------------------------------------
// Amber triggers
// ---------------------------------------------------------------------------

func TestTag_ACBHigh(t *testing.T) {
	snap := baseSnap()
	snap.ACB = 3 // boundary value that triggers amber
	if got := Tag(snap); got != UrgencyAmber {
		t.Errorf("Tag (ACB=3) = %q, want %q", got, UrgencyAmber)
	}
}

func TestTag_DBIHighBoundary(t *testing.T) {
	// DBI exactly 1.0 must trigger amber.
	snap := baseSnap()
	snap.DBI = 1.0
	if got := Tag(snap); got != UrgencyAmber {
		t.Errorf("Tag (DBI=1.0) = %q, want %q", got, UrgencyAmber)
	}
}

func TestTag_DBIBelowBoundary(t *testing.T) {
	// DBI 0.99 must NOT trigger amber (or red), so must be green.
	snap := baseSnap()
	snap.DBI = 0.99
	if got := Tag(snap); got != UrgencyGreen {
		t.Errorf("Tag (DBI=0.99) = %q, want %q", got, UrgencyGreen)
	}
}

func TestTag_Palliative(t *testing.T) {
	snap := baseSnap()
	snap.CareIntensity = "palliative"
	if got := Tag(snap); got != UrgencyAmber {
		t.Errorf("Tag (palliative) = %q, want %q", got, UrgencyAmber)
	}
}

func TestTag_EndOfLife(t *testing.T) {
	snap := baseSnap()
	snap.CareIntensity = "end_of_life"
	if got := Tag(snap); got != UrgencyAmber {
		t.Errorf("Tag (end_of_life) = %q, want %q", got, UrgencyAmber)
	}
}

// ---------------------------------------------------------------------------
// Green default
// ---------------------------------------------------------------------------

func TestTag_GreenDefault(t *testing.T) {
	snap := baseSnap() // all zero/false — no signals
	if got := Tag(snap); got != UrgencyGreen {
		t.Errorf("Tag (no signals) = %q, want %q", got, UrgencyGreen)
	}
}

// ---------------------------------------------------------------------------
// Red short-circuits amber
// ---------------------------------------------------------------------------

func TestTag_RedTrumpsAmber(t *testing.T) {
	// Both a red signal and an amber signal are active — red must win.
	snap := baseSnap()
	snap.RecentFall72h = true
	snap.ACB = 3
	if got := Tag(snap); got != UrgencyRed {
		t.Errorf("Tag (fall + ACB=3) = %q, want %q (red short-circuit)", got, UrgencyRed)
	}
}
