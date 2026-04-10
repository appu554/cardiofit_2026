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
