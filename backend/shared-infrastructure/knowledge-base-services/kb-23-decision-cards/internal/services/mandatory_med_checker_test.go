package services

import (
	"testing"
)

func TestMandatoryMeds_4a_MissingStatin(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"METFORMIN", "ACEi"}

	gaps := checker.CheckMandatory("4a", "", activeMeds)

	hasStatinGap := false
	for _, g := range gaps {
		if g.MissingClass == "STATIN" {
			hasStatinGap = true
			if g.Urgency != "URGENT" {
				t.Errorf("expected URGENT for missing statin in 4a, got %s", g.Urgency)
			}
		}
	}
	if !hasStatinGap {
		t.Error("should flag missing statin for Stage 4a")
	}
}

func TestMandatoryMeds_4b_AllPresent(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"STATIN", "ASPIRIN", "ACEi", "BETA_BLOCKER", "SGLT2i"}

	gaps := checker.CheckMandatory("4b", "", activeMeds)

	for _, g := range gaps {
		if g.Urgency == "IMMEDIATE" {
			t.Errorf("unexpected IMMEDIATE gap when all mandatory present: %s", g.MissingClass)
		}
	}
}

func TestMandatoryMeds_4c_HFrEF_MissingPillars(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"ACEi", "METFORMIN"} // missing: BB, MRA, SGLT2i

	gaps := checker.CheckMandatory("4c", "HFrEF", activeMeds)

	missingClasses := map[string]bool{}
	for _, g := range gaps {
		missingClasses[g.MissingClass] = true
	}
	if !missingClasses["SGLT2i"] {
		t.Error("should flag missing SGLT2i for HFrEF")
	}
	if !missingClasses["BETA_BLOCKER_HF"] {
		t.Error("should flag missing beta-blocker for HFrEF")
	}
	if !missingClasses["MRA"] {
		t.Error("should flag missing MRA for HFrEF")
	}
}

func TestMandatoryMeds_4c_HFpEF_OnlySGLT2i(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"ACEi", "THIAZIDE", "METFORMIN"}

	gaps := checker.CheckMandatory("4c", "HFpEF", activeMeds)

	hasSGLT2iGap := false
	for _, g := range gaps {
		if g.MissingClass == "SGLT2i" {
			hasSGLT2iGap = true
		}
	}
	if !hasSGLT2iGap {
		t.Error("should flag missing SGLT2i — only mandatory disease-modifying therapy for HFpEF")
	}
}

// Issue 2: 4b beta-blocker is conditional on post-MI (CAPRICORN).
// Stroke without MI should NOT trigger beta-blocker gap.
func TestMandatoryMeds_4b_PostStroke_NoBetaBlockerRequired(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"STATIN", "ASPIRIN", "ACEi"} // no beta-blocker
	ctx := ClinicalContext{
		ASCVDEventTypes: []string{"STROKE"},
	}

	gaps := checker.CheckMandatory("4b", "", activeMeds, ctx)

	for _, g := range gaps {
		if g.MissingClass == "BETA_BLOCKER" {
			t.Errorf("post-stroke patient should NOT have beta-blocker gap — no outcome data post-stroke: %+v", g)
		}
	}
}

func TestMandatoryMeds_4b_PostMI_BetaBlockerRequired(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"STATIN", "ASPIRIN", "ACEi"} // no beta-blocker
	ctx := ClinicalContext{
		ASCVDEventTypes: []string{"MI"},
	}

	gaps := checker.CheckMandatory("4b", "", activeMeds, ctx)

	hasBBGap := false
	for _, g := range gaps {
		if g.MissingClass == "BETA_BLOCKER" {
			hasBBGap = true
			if g.SourceTrial != "CAPRICORN" {
				t.Errorf("post-MI BB gap should cite CAPRICORN, got %s", g.SourceTrial)
			}
		}
	}
	if !hasBBGap {
		t.Error("post-MI patient missing beta-blocker should be flagged (CAPRICORN)")
	}
}

// Issue 3: ARNI upgrade safety — 36h washout + SBP ≥100 guard.
func TestMandatoryMeds_4c_HFrEF_OnACEi_FlagsARNIUpgrade(t *testing.T) {
	checker := NewMandatoryMedChecker()
	// Patient has full four-pillar coverage except ACEi instead of ARNI
	activeMeds := []string{"ACEi", "BETA_BLOCKER_HF", "MRA", "SGLT2i"}
	ctx := ClinicalContext{SBPmmHg: 125}

	gaps := checker.CheckMandatory("4c", "HFrEF", activeMeds, ctx)

	hasUpgrade := false
	for _, g := range gaps {
		if g.MissingClass == "ARNI_UPGRADE" {
			hasUpgrade = true
			if g.SafetyPrecautions == "" {
				t.Error("ARNI_UPGRADE must include safety precautions (washout + SBP)")
			}
			if !contains(g.SafetyPrecautions, "36-hour") && !contains(g.SafetyPrecautions, "washout") {
				t.Errorf("ARNI_UPGRADE safety precautions must mention 36h washout: %s", g.SafetyPrecautions)
			}
			if !contains(g.SafetyPrecautions, "100") {
				t.Errorf("ARNI_UPGRADE safety precautions must mention SBP ≥100: %s", g.SafetyPrecautions)
			}
		}
	}
	if !hasUpgrade {
		t.Error("HFrEF patient on ACEi without ARNI should be flagged for upgrade")
	}
}

func TestMandatoryMeds_4c_HFrEF_ARNIUpgrade_DeferredAtLowSBP(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"ACEi", "BETA_BLOCKER_HF", "MRA", "SGLT2i"}
	ctx := ClinicalContext{SBPmmHg: 92} // below 100

	gaps := checker.CheckMandatory("4c", "HFrEF", activeMeds, ctx)

	for _, g := range gaps {
		if g.MissingClass == "ARNI_UPGRADE" {
			if g.Urgency != "ROUTINE" {
				t.Errorf("ARNI upgrade should be ROUTINE (deferred) at SBP <100, got %s", g.Urgency)
			}
			if !contains(g.SafetyPrecautions, "deferred") {
				t.Errorf("ARNI upgrade at low SBP should note deferral: %s", g.SafetyPrecautions)
			}
		}
	}
}

// Issue 4: Unknown HF subtype → IMMEDIATE echo gap.
func TestMandatoryMeds_4c_UnknownHFSubtype_RequiresEcho(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"METFORMIN"} // nothing HF-related

	// Empty hfType = unknown subtype
	gaps := checker.CheckMandatory("4c", "", activeMeds)

	hasEcho := false
	for _, g := range gaps {
		if g.MissingClass == "ECHOCARDIOGRAM" {
			hasEcho = true
			if g.Urgency != "IMMEDIATE" {
				t.Errorf("echo gap should be IMMEDIATE, got %s", g.Urgency)
			}
		}
	}
	if !hasEcho {
		t.Error("unknown HF subtype should trigger IMMEDIATE echocardiogram gap")
	}

	// Also should flag SGLT2i (safe across spectrum) but NOT full GDMT
	hasSGLT2i := false
	hasBBGap := false
	for _, g := range gaps {
		if g.MissingClass == "SGLT2i" {
			hasSGLT2i = true
		}
		if g.MissingClass == "BETA_BLOCKER_HF" {
			hasBBGap = true
		}
	}
	if !hasSGLT2i {
		t.Error("should still flag SGLT2i — safe to initiate before EF known")
	}
	if hasBBGap {
		t.Error("should NOT flag beta-blocker before EF known — dose depends on HFrEF vs HFpEF")
	}
}

// Cross-domain: Renal gate blocks SGLT2i in HFpEF → CRITICAL conflict.
// (Tested more thoroughly in conflict_detector_crossdomain_test.go)
func TestMandatoryMeds_RenalSuppression(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"ACEi", "BETA_BLOCKER_HF"} // missing MRA + SGLT2i
	ctx := ClinicalContext{
		BlockedByRenal: []string{"MRA", "SGLT2i"},
	}

	gaps := checker.CheckMandatory("4c", "HFrEF", activeMeds, ctx)

	suppressedMRA := false
	suppressedSGLT2i := false
	for _, g := range gaps {
		if g.MissingClass == "MRA" && g.Suppressed {
			suppressedMRA = true
		}
		if g.MissingClass == "SGLT2i" && g.Suppressed {
			suppressedSGLT2i = true
		}
	}
	if !suppressedMRA {
		t.Error("MRA gap should be marked Suppressed when renal gate blocks it")
	}
	if !suppressedSGLT2i {
		t.Error("SGLT2i gap should be marked Suppressed when renal gate blocks it")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
