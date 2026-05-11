package substrate_types

import (
	"reflect"
	"testing"
)

// TestOverrideReasonFieldPinning pins the kb-32 OverrideReason dual-vocab shape.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/overrides/taxonomy.go
// (OverrideReason — Phase 2-completion Task 5).
func TestOverrideReasonFieldPinning(t *testing.T) {
	want := []string{
		"ID", "RecommendationID", "ReasonCode", "ReasonCodeShort",
		"AppropriatenessFlag", "Reasoning", "CapturedAt", "CapturedBy",
	}
	got := fieldNames(OverrideReason{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("OverrideReason fields drifted: want %v got %v\n"+
			"if canonical OverrideReason changed, update local copy + SOURCE OF TRUTH comment",
			want, got)
	}
}

// TestCanonicalOverrideReasonCodes pins the entire 20-pair dual-vocab list
// against kb-32 Phase 2-completion Task 5. Ordering matches kb-32
// ValidReasonCodes / ValidShortCodes index-for-index.
func TestCanonicalOverrideReasonCodes(t *testing.T) {
	want := []OverrideReasonCodePair{
		// 12 Wright/McCoy foundation
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
		// 8 ACOP extension
		{"goals_of_care_aligned", "GCA"},
		{"deprescribing_underway", "DUW"},
		{"frailty_consideration", "FRC"},
		{"family_consensus_pending", "FCP"},
		{"sdm_review_required", "SDR"},
		{"trial_period_active", "TPA"},
		{"audit_visit_imminent", "AVI"},
		{"cross_resident_pattern", "CRP"},
	}
	if !reflect.DeepEqual(want, CanonicalOverrideReasonCodes) {
		t.Fatalf("CanonicalOverrideReasonCodes drifted from kb-32 Phase 2-completion Task 5 mapping:\n"+
			"want: %v\ngot:  %v", want, CanonicalOverrideReasonCodes)
	}
	if len(CanonicalOverrideReasonCodes) != 20 {
		t.Fatalf("expected 20 dual-vocab pairs (12 foundation + 8 ACOP), got %d", len(CanonicalOverrideReasonCodes))
	}
}
