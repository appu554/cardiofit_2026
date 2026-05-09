// Package restraint_test — surfacing.go tests.
//
// VisibilityClass: AD — restraint signals per Guidelines §10
package restraint_test

import (
	"testing"

	"github.com/cardiofit/kb32/internal/restraint"
)

// ---------------------------------------------------------------------------
// Surface tests
// ---------------------------------------------------------------------------

func TestSurface_EmptySignals(t *testing.T) {
	sd := restraint.Surface(nil)
	if sd.SignalCount != 0 {
		t.Errorf("expected SignalCount=0, got %d", sd.SignalCount)
	}
	if sd.HighestSeverity != "" {
		t.Errorf("expected empty HighestSeverity, got %q", sd.HighestSeverity)
	}
	if sd.Signals != nil {
		t.Errorf("expected nil Signals slice, got %v", sd.Signals)
	}
}

func TestSurface_EmptySlice(t *testing.T) {
	sd := restraint.Surface([]restraint.Signal{})
	if sd.SignalCount != 0 {
		t.Errorf("expected SignalCount=0, got %d", sd.SignalCount)
	}
	if sd.HighestSeverity != "" {
		t.Errorf("expected empty HighestSeverity, got %q", sd.HighestSeverity)
	}
}

func TestSurface_AllAmberHighestIsAmber(t *testing.T) {
	sigs := []restraint.Signal{
		{Type: "acb_increase", Severity: restraint.SeverityAmber, Reasoning: "ACB ≥ 3"},
		{Type: "polypharmacy_threshold", Severity: restraint.SeverityAmber, Reasoning: "DBI ≥ 1.0"},
	}
	sd := restraint.Surface(sigs)
	if sd.HighestSeverity != restraint.SeverityAmber {
		t.Errorf("expected Amber highest severity, got %q", sd.HighestSeverity)
	}
}

func TestSurface_RedTrumpsAmber(t *testing.T) {
	sigs := []restraint.Signal{
		{Type: "acb_increase", Severity: restraint.SeverityAmber, Reasoning: "ACB ≥ 3"},
		{Type: "recent_fall_72h", Severity: restraint.SeverityRed, Reasoning: "Resident fell within 72h"},
	}
	sd := restraint.Surface(sigs)
	if sd.HighestSeverity != restraint.SeverityRed {
		t.Errorf("expected Red to trump Amber, got %q", sd.HighestSeverity)
	}
}

func TestSurface_SignalCountMatches(t *testing.T) {
	sigs := []restraint.Signal{
		{Type: "recent_fall_72h", Severity: restraint.SeverityRed},
		{Type: "acb_increase", Severity: restraint.SeverityAmber},
		{Type: "family_distress", Severity: restraint.SeverityRed},
	}
	sd := restraint.Surface(sigs)
	if sd.SignalCount != 3 {
		t.Errorf("expected SignalCount=3, got %d", sd.SignalCount)
	}
	if len(sd.Signals) != 3 {
		t.Errorf("expected 3 signals in SurfaceData.Signals, got %d", len(sd.Signals))
	}
}

func TestSurface_SingleRedSignal(t *testing.T) {
	sigs := []restraint.Signal{
		{Type: "recent_fall_72h", Severity: restraint.SeverityRed, Reasoning: "Resident fell within 72h"},
	}
	sd := restraint.Surface(sigs)
	if sd.SignalCount != 1 {
		t.Errorf("expected SignalCount=1, got %d", sd.SignalCount)
	}
	if sd.HighestSeverity != restraint.SeverityRed {
		t.Errorf("expected Red, got %q", sd.HighestSeverity)
	}
}
