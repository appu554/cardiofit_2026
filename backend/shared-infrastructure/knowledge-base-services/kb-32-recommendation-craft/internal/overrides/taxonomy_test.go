// Package overrides — tests for override-reason taxonomy.
// VisibilityClass: AD — override capture for clinical-safety audit per Guidelines §5
package overrides

import (
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestExactly20Codes — exported slice must carry exactly 20 entries.
// ---------------------------------------------------------------------------

func TestExactly20Codes(t *testing.T) {
	if got := len(ValidReasonCodes); got != 20 {
		t.Errorf("ValidReasonCodes: want 20 codes, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// TestIsValidReasonCode_All20 — every listed code must pass; garbage must fail.
// ---------------------------------------------------------------------------

func TestIsValidReasonCode_All20(t *testing.T) {
	for _, code := range ValidReasonCodes {
		if !IsValidReasonCode(code) {
			t.Errorf("IsValidReasonCode(%q): expected true, got false", code)
		}
	}
	if IsValidReasonCode("garbage") {
		t.Error("IsValidReasonCode(\"garbage\"): expected false, got true")
	}
}

// ---------------------------------------------------------------------------
// TestIsValidReasonCode_CaseSensitive — mixed-case must be rejected.
// ---------------------------------------------------------------------------

func TestIsValidReasonCode_CaseSensitive(t *testing.T) {
	if IsValidReasonCode("Patient_Preference") {
		t.Error("IsValidReasonCode(\"Patient_Preference\"): expected false (case-sensitive check), got true")
	}
	if IsValidReasonCode("Alert_Fatigue") {
		t.Error("IsValidReasonCode(\"Alert_Fatigue\"): expected false (case-sensitive check), got true")
	}
}

// ---------------------------------------------------------------------------
// TestIsValidFlag_AllThree — the three appropriateness flags must pass; others fail.
// ---------------------------------------------------------------------------

func TestIsValidFlag_AllThree(t *testing.T) {
	flags := []string{"appropriate_override", "inappropriate_override", "mixed"}
	for _, f := range flags {
		if !IsValidFlag(f) {
			t.Errorf("IsValidFlag(%q): expected true, got false", f)
		}
	}
	if IsValidFlag("unknown_flag") {
		t.Error("IsValidFlag(\"unknown_flag\"): expected false, got true")
	}
}

// ---------------------------------------------------------------------------
// TestOverrideReason_Validate_HappyPath — a fully-populated struct must pass.
// ---------------------------------------------------------------------------

func TestOverrideReason_Validate_HappyPath(t *testing.T) {
	o := OverrideReason{
		ID:                 "or-001",
		RecommendationID:   "rec-123",
		ReasonCode:         "alert_fatigue",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:          "Patient has demonstrated low-risk profile; alert deemed low-yield.",
		CapturedAt:         time.Now(),
		CapturedBy:         "pharmacist-uuid-abc",
	}
	if err := o.Validate(); err != nil {
		t.Errorf("Validate() on valid OverrideReason: unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestOverrideReason_Validate_RejectsInvalidReasonCode
// ---------------------------------------------------------------------------

func TestOverrideReason_Validate_RejectsInvalidReasonCode(t *testing.T) {
	o := OverrideReason{
		ID:                 "or-002",
		RecommendationID:   "rec-124",
		ReasonCode:         "not_a_real_code",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:          "Some reasoning here.",
		CapturedAt:         time.Now(),
		CapturedBy:         "pharmacist-uuid-abc",
	}
	err := o.Validate()
	if err == nil {
		t.Fatal("Validate() with invalid ReasonCode: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidReasonCode) {
		t.Errorf("Validate() with invalid ReasonCode: want errors.Is(err, ErrInvalidReasonCode), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestOverrideReason_Validate_RejectsInvalidFlag
// ---------------------------------------------------------------------------

func TestOverrideReason_Validate_RejectsInvalidFlag(t *testing.T) {
	o := OverrideReason{
		ID:                 "or-003",
		RecommendationID:   "rec-125",
		ReasonCode:         "patient_preference",
		AppropriatenessFlag: "bad_flag",
		Reasoning:          "Some reasoning here.",
		CapturedAt:         time.Now(),
		CapturedBy:         "pharmacist-uuid-abc",
	}
	err := o.Validate()
	if err == nil {
		t.Fatal("Validate() with invalid AppropriatenessFlag: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidFlag) {
		t.Errorf("Validate() with invalid AppropriatenessFlag: want errors.Is(err, ErrInvalidFlag), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestOverrideReason_Validate_RejectsEmptyReasoning
// ---------------------------------------------------------------------------

func TestOverrideReason_Validate_RejectsEmptyReasoning(t *testing.T) {
	o := OverrideReason{
		ID:                 "or-004",
		RecommendationID:   "rec-126",
		ReasonCode:         "clinical_judgment",
		AppropriatenessFlag: "mixed",
		Reasoning:          "",
		CapturedAt:         time.Now(),
		CapturedBy:         "pharmacist-uuid-abc",
	}
	err := o.Validate()
	if err == nil {
		t.Fatal("Validate() with empty Reasoning: expected error, got nil")
	}
	if !errors.Is(err, ErrEmptyReasoning) {
		t.Errorf("Validate() with empty Reasoning: want errors.Is(err, ErrEmptyReasoning), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestIsWrightMcCoyFoundation — 12 foundation codes true; ACOP codes false.
// ---------------------------------------------------------------------------

func TestIsWrightMcCoyFoundation(t *testing.T) {
	foundation := []string{
		"alert_fatigue",
		"irrelevant_to_patient",
		"patient_preference",
		"clinical_judgment",
		"alternative_pursued",
		"monitoring_in_place",
		"low_priority",
		"documentation_concern",
		"uncertain_evidence",
		"system_error",
		"workflow_constraint",
		"duplicative_alert",
	}
	for _, code := range foundation {
		if !IsWrightMcCoyFoundation(code) {
			t.Errorf("IsWrightMcCoyFoundation(%q): expected true, got false", code)
		}
	}

	acop := []string{
		"goals_of_care_aligned",
		"deprescribing_underway",
		"frailty_consideration",
		"family_consensus_pending",
		"sdm_review_required",
		"trial_period_active",
		"audit_visit_imminent",
		"cross_resident_pattern",
	}
	for _, code := range acop {
		if IsWrightMcCoyFoundation(code) {
			t.Errorf("IsWrightMcCoyFoundation(%q): expected false (ACOP code), got true", code)
		}
	}
}

// ---------------------------------------------------------------------------
// TestIsACOPExtension — 8 ACOP codes true; foundation codes false.
// ---------------------------------------------------------------------------

func TestIsACOPExtension(t *testing.T) {
	acop := []string{
		"goals_of_care_aligned",
		"deprescribing_underway",
		"frailty_consideration",
		"family_consensus_pending",
		"sdm_review_required",
		"trial_period_active",
		"audit_visit_imminent",
		"cross_resident_pattern",
	}
	for _, code := range acop {
		if !IsACOPExtension(code) {
			t.Errorf("IsACOPExtension(%q): expected true, got false", code)
		}
	}

	foundation := []string{
		"alert_fatigue",
		"irrelevant_to_patient",
		"patient_preference",
		"clinical_judgment",
		"alternative_pursued",
		"monitoring_in_place",
		"low_priority",
		"documentation_concern",
		"uncertain_evidence",
		"system_error",
		"workflow_constraint",
		"duplicative_alert",
	}
	for _, code := range foundation {
		if IsACOPExtension(code) {
			t.Errorf("IsACOPExtension(%q): expected false (foundation code), got true", code)
		}
	}
}
