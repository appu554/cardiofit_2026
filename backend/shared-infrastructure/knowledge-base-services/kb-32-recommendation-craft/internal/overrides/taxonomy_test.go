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

// ---------------------------------------------------------------------------
// Dual-vocabulary tests (Phase 2-completion Task 5)
// ---------------------------------------------------------------------------

// canonicalMapping is the single source of truth for the test layer; if the
// implementation map drifts, these tests fail. Order matches ValidReasonCodes.
var canonicalMapping = []struct {
	Snake string
	Short string
}{
	// Wright/McCoy foundation (12)
	{"alert_fatigue", "ALF"},
	{"irrelevant_to_patient", "IRP"},
	{"patient_preference", "PPF"},
	{"clinical_judgment", "CJG"},
	{"alternative_pursued", "AAP"},
	{"monitoring_in_place", "MIP"},
	{"low_priority", "LPR"},
	{"documentation_concern", "DCN"},
	{"uncertain_evidence", "UNE"},
	{"system_error", "SYS"},
	{"workflow_constraint", "WFC"},
	{"duplicative_alert", "DPA"},
	// ACOP extension (8)
	{"goals_of_care_aligned", "GCA"},
	{"deprescribing_underway", "DUW"},
	{"frailty_consideration", "FRC"},
	{"family_consensus_pending", "FCP"},
	{"sdm_review_required", "SDR"},
	{"trial_period_active", "TPA"},
	{"audit_visit_imminent", "AVI"},
	{"cross_resident_pattern", "CRP"},
}

func TestToShortCode_All20(t *testing.T) {
	for _, tc := range canonicalMapping {
		got, ok := ToShortCode(tc.Snake)
		if !ok {
			t.Errorf("ToShortCode(%q): expected ok=true, got ok=false", tc.Snake)
			continue
		}
		if got != tc.Short {
			t.Errorf("ToShortCode(%q): got %q; want %q", tc.Snake, got, tc.Short)
		}
	}
	if _, ok := ToShortCode("garbage"); ok {
		t.Error("ToShortCode(\"garbage\"): expected ok=false, got ok=true")
	}
}

func TestToReasonCode_All20(t *testing.T) {
	for _, tc := range canonicalMapping {
		got, ok := ToReasonCode(tc.Short)
		if !ok {
			t.Errorf("ToReasonCode(%q): expected ok=true, got ok=false", tc.Short)
			continue
		}
		if got != tc.Snake {
			t.Errorf("ToReasonCode(%q): got %q; want %q", tc.Short, got, tc.Snake)
		}
	}
	if _, ok := ToReasonCode("ZZZ"); ok {
		t.Error("ToReasonCode(\"ZZZ\"): expected ok=false, got ok=true")
	}
}

func TestNormalizeCode_AcceptsBothVocabularies(t *testing.T) {
	for _, tc := range canonicalMapping {
		// Snake input → expect (snake, short, nil).
		snake, short, err := NormalizeCode(tc.Snake)
		if err != nil {
			t.Errorf("NormalizeCode(%q): unexpected error: %v", tc.Snake, err)
		}
		if snake != tc.Snake || short != tc.Short {
			t.Errorf("NormalizeCode(%q): got (%q,%q); want (%q,%q)",
				tc.Snake, snake, short, tc.Snake, tc.Short)
		}
		// Short input → same canonical pair.
		snake2, short2, err := NormalizeCode(tc.Short)
		if err != nil {
			t.Errorf("NormalizeCode(%q): unexpected error: %v", tc.Short, err)
		}
		if snake2 != tc.Snake || short2 != tc.Short {
			t.Errorf("NormalizeCode(%q): got (%q,%q); want (%q,%q)",
				tc.Short, snake2, short2, tc.Snake, tc.Short)
		}
	}
}

func TestNormalizeCode_RejectsGarbage(t *testing.T) {
	_, _, err := NormalizeCode("not_a_code")
	if !errors.Is(err, ErrInvalidReasonCode) {
		t.Errorf("NormalizeCode(\"not_a_code\"): want ErrInvalidReasonCode, got %v", err)
	}
}

func TestIsValidShortCode(t *testing.T) {
	for _, tc := range canonicalMapping {
		if !IsValidShortCode(tc.Short) {
			t.Errorf("IsValidShortCode(%q): expected true, got false", tc.Short)
		}
	}
	if IsValidShortCode("alf") {
		t.Error("IsValidShortCode(\"alf\"): expected false (case-sensitive), got true")
	}
	if IsValidShortCode("ZZZ") {
		t.Error("IsValidShortCode(\"ZZZ\"): expected false, got true")
	}
}

func TestValidShortCodes_MatchesCanonical(t *testing.T) {
	if got := len(ValidShortCodes); got != 20 {
		t.Errorf("ValidShortCodes: want 20 codes, got %d", got)
	}
	if len(ValidShortCodes) != len(canonicalMapping) {
		t.Fatalf("ValidShortCodes length %d != canonicalMapping length %d",
			len(ValidShortCodes), len(canonicalMapping))
	}
	for i, want := range canonicalMapping {
		if ValidShortCodes[i] != want.Short {
			t.Errorf("ValidShortCodes[%d] = %q; want %q", i, ValidShortCodes[i], want.Short)
		}
	}
}

func TestOverrideReason_Validate_DerivesShortFromSnake(t *testing.T) {
	o := OverrideReason{
		RecommendationID:    "rec-1",
		ReasonCode:          "patient_preference",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "rationale",
		CapturedAt:          time.Now(),
		CapturedBy:          "rx-1",
	}
	if err := o.Validate(); err != nil {
		t.Fatalf("Validate: unexpected error: %v", err)
	}
	if o.ReasonCodeShort != "PPF" {
		t.Errorf("derived ReasonCodeShort = %q; want PPF", o.ReasonCodeShort)
	}
}

func TestOverrideReason_Validate_AcceptsConsistentBothSet(t *testing.T) {
	o := OverrideReason{
		RecommendationID:    "rec-2",
		ReasonCode:          "alert_fatigue",
		ReasonCodeShort:     "ALF",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "rationale",
		CapturedAt:          time.Now(),
		CapturedBy:          "rx-2",
	}
	if err := o.Validate(); err != nil {
		t.Errorf("Validate with consistent pair: unexpected error: %v", err)
	}
}

func TestOverrideReason_Validate_RejectsInconsistentBothSet(t *testing.T) {
	o := OverrideReason{
		RecommendationID:    "rec-3",
		ReasonCode:          "alert_fatigue",
		ReasonCodeShort:     "PPF", // wrong: ALF would be canonical
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "rationale",
		CapturedAt:          time.Now(),
		CapturedBy:          "rx-3",
	}
	err := o.Validate()
	if err == nil {
		t.Fatal("Validate with inconsistent pair: expected error, got nil")
	}
	if !errors.Is(err, ErrInconsistentReasonCodes) {
		t.Errorf("Validate with inconsistent pair: want ErrInconsistentReasonCodes, got %v", err)
	}
}
