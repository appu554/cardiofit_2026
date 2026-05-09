// Package restraint_test verifies the restraint signal detectors defined in
// signaler.go. Tests follow TDD: written before implementation, covering all 9
// detector functions, the structural registration guard, and compound scenarios.
//
// VisibilityClass: AD — restraint signals per Guidelines §10
package restraint_test

import (
	"testing"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/restraint"
)

// ---------------------------------------------------------------------------
// Structural test: all 9 detectors must be registered
// ---------------------------------------------------------------------------

// expectedSignalTypes is the authoritative list of the 9 restraint signal
// types defined in Guidelines Part 10.  This test guards against future
// engineers adding or removing detectors inconsistently.
var expectedSignalTypes = []string{
	"recent_fall_72h",
	"acb_increase",
	"family_distress",
	"end_of_life_proximity",
	"capacity_lapse",
	"polypharmacy_threshold",
	"frailty_step_change",
	"recent_admission_72h",
	"restrictive_practice_active",
}

func TestExpectedSignalTypesAllRegistered(t *testing.T) {
	for _, name := range expectedSignalTypes {
		if !restraint.DetectorRegistered(name) {
			t.Errorf("detector %q is not registered in detectorsByName", name)
		}
	}
}

// ---------------------------------------------------------------------------
// IsValidSeverity
// ---------------------------------------------------------------------------

func TestIsValidSeverity_ValidValues(t *testing.T) {
	for _, s := range []string{"red", "amber"} {
		if !restraint.IsValidSeverity(s) {
			t.Errorf("IsValidSeverity(%q) should be true", s)
		}
	}
}

func TestIsValidSeverity_InvalidValues(t *testing.T) {
	for _, s := range []string{"", "Red", "AMBER", "green", "yellow", "critical"} {
		if restraint.IsValidSeverity(s) {
			t.Errorf("IsValidSeverity(%q) should be false", s)
		}
	}
}

// ---------------------------------------------------------------------------
// Helper: fully-green snapshot (no signals should fire)
// ---------------------------------------------------------------------------

func greenSnapshot() kb32ctx.ClinicalSnapshot {
	return kb32ctx.ClinicalSnapshot{
		EGFR:                      75.0,
		DBI:                       0.5,   // < 1.0
		ACB:                       1,     // < 3
		CFS:                       3,
		CareIntensity:             "active",
		RecentFall72h:             false,
		RecentAdmission72h:        false,
		FamilyDistress:            false,
		CapacityLapse:             false,
		FrailtyStepIncrease30d:    false,
		RestrictivePracticeActive: false,
	}
}

// ---------------------------------------------------------------------------
// Individual detector tests — trigger + no-trigger pairs
// ---------------------------------------------------------------------------

func TestDetector_RecentFall72h_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.RecentFall72h = true
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "recent_fall_72h")
	if found == nil {
		t.Fatal("expected recent_fall_72h signal, got none")
	}
	if found.Severity != restraint.SeverityRed {
		t.Errorf("expected Red severity, got %q", found.Severity)
	}
	if found.Reasoning == "" {
		t.Error("expected non-empty Reasoning")
	}
}

func TestDetector_RecentFall72h_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.RecentFall72h = false
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "recent_fall_72h"); found != nil {
		t.Error("unexpected recent_fall_72h signal on green snapshot")
	}
}

func TestDetector_ACBIncrease_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.ACB = 3
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "acb_increase")
	if found == nil {
		t.Fatal("expected acb_increase signal for ACB=3, got none")
	}
	if found.Severity != restraint.SeverityAmber {
		t.Errorf("expected Amber severity, got %q", found.Severity)
	}
}

func TestDetector_ACBIncrease_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.ACB = 2 // below threshold
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "acb_increase"); found != nil {
		t.Errorf("unexpected acb_increase signal for ACB=2")
	}
}

func TestDetector_FamilyDistress_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.FamilyDistress = true
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "family_distress")
	if found == nil {
		t.Fatal("expected family_distress signal, got none")
	}
	if found.Severity != restraint.SeverityRed {
		t.Errorf("expected Red severity, got %q", found.Severity)
	}
}

func TestDetector_FamilyDistress_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.FamilyDistress = false
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "family_distress"); found != nil {
		t.Error("unexpected family_distress signal")
	}
}

func TestDetector_EndOfLifeProximity_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.CareIntensity = "end_of_life"
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "end_of_life_proximity")
	if found == nil {
		t.Fatal("expected end_of_life_proximity signal, got none")
	}
	if found.Severity != restraint.SeverityRed {
		t.Errorf("expected Red severity, got %q", found.Severity)
	}
	if found.Reasoning == "" {
		t.Error("expected non-empty Reasoning")
	}
}

func TestDetector_EndOfLifeProximity_NoTrigger(t *testing.T) {
	for _, ci := range []string{"active", "comfort", "palliative"} {
		snap := greenSnapshot()
		snap.CareIntensity = ci
		sigs := restraint.DetectAll(snap)
		if found := findSignal(sigs, "end_of_life_proximity"); found != nil {
			t.Errorf("unexpected end_of_life_proximity for CareIntensity=%q", ci)
		}
	}
}

func TestDetector_CapacityLapse_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.CapacityLapse = true
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "capacity_lapse")
	if found == nil {
		t.Fatal("expected capacity_lapse signal, got none")
	}
	if found.Severity != restraint.SeverityRed {
		t.Errorf("expected Red severity, got %q", found.Severity)
	}
}

func TestDetector_CapacityLapse_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.CapacityLapse = false
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "capacity_lapse"); found != nil {
		t.Error("unexpected capacity_lapse signal")
	}
}

func TestDetector_PolypharmacyThreshold_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.DBI = 1.0
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "polypharmacy_threshold")
	if found == nil {
		t.Fatal("expected polypharmacy_threshold signal for DBI=1.0, got none")
	}
	if found.Severity != restraint.SeverityAmber {
		t.Errorf("expected Amber severity, got %q", found.Severity)
	}
}

func TestDetector_PolypharmacyThreshold_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.DBI = 0.99 // strictly below 1.0
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "polypharmacy_threshold"); found != nil {
		t.Errorf("unexpected polypharmacy_threshold for DBI=0.99")
	}
}

func TestDetector_FrailtyStepChange_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.FrailtyStepIncrease30d = true
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "frailty_step_change")
	if found == nil {
		t.Fatal("expected frailty_step_change signal, got none")
	}
	if found.Severity != restraint.SeverityAmber {
		t.Errorf("expected Amber severity, got %q", found.Severity)
	}
}

func TestDetector_FrailtyStepChange_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.FrailtyStepIncrease30d = false
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "frailty_step_change"); found != nil {
		t.Error("unexpected frailty_step_change signal")
	}
}

func TestDetector_RecentAdmission72h_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.RecentAdmission72h = true
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "recent_admission_72h")
	if found == nil {
		t.Fatal("expected recent_admission_72h signal, got none")
	}
	if found.Severity != restraint.SeverityRed {
		t.Errorf("expected Red severity, got %q", found.Severity)
	}
}

func TestDetector_RecentAdmission72h_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.RecentAdmission72h = false
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "recent_admission_72h"); found != nil {
		t.Error("unexpected recent_admission_72h signal")
	}
}

func TestDetector_RestrictivePracticeActive_Triggers(t *testing.T) {
	snap := greenSnapshot()
	snap.RestrictivePracticeActive = true
	sigs := restraint.DetectAll(snap)
	found := findSignal(sigs, "restrictive_practice_active")
	if found == nil {
		t.Fatal("expected restrictive_practice_active signal, got none")
	}
	if found.Severity != restraint.SeverityRed {
		t.Errorf("expected Red severity, got %q", found.Severity)
	}
}

func TestDetector_RestrictivePracticeActive_NoTrigger(t *testing.T) {
	snap := greenSnapshot()
	snap.RestrictivePracticeActive = false
	sigs := restraint.DetectAll(snap)
	if found := findSignal(sigs, "restrictive_practice_active"); found != nil {
		t.Error("unexpected restrictive_practice_active signal")
	}
}

// ---------------------------------------------------------------------------
// DetectAll aggregate tests
// ---------------------------------------------------------------------------

func TestDetectAll_NoSignalsOnGreenSnapshot(t *testing.T) {
	sigs := restraint.DetectAll(greenSnapshot())
	if len(sigs) != 0 {
		t.Errorf("expected 0 signals on green snapshot, got %d: %v", len(sigs), sigs)
	}
}

func TestDetectAll_MultipleSignalsCompound(t *testing.T) {
	snap := greenSnapshot()
	snap.RecentFall72h = true
	snap.ACB = 3
	sigs := restraint.DetectAll(snap)
	if len(sigs) != 2 {
		t.Errorf("expected 2 signals (recent_fall_72h + acb_increase), got %d: %v", len(sigs), sigs)
	}
	if findSignal(sigs, "recent_fall_72h") == nil {
		t.Error("expected recent_fall_72h in compound result")
	}
	if findSignal(sigs, "acb_increase") == nil {
		t.Error("expected acb_increase in compound result")
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func findSignal(sigs []restraint.Signal, signalType string) *restraint.Signal {
	for i := range sigs {
		if sigs[i].Type == signalType {
			return &sigs[i]
		}
	}
	return nil
}
