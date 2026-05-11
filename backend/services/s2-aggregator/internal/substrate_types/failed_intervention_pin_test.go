package substrate_types

import (
	"reflect"
	"testing"
)

// TestFailedInterventionRecordFieldPinning structurally pins the s2-side
// shape of FailedInterventionRecord against the canonical type at
// shared/v2_substrate/failed_interventions/types.go. If the canonical
// type adds or renames a field, update the local copy AND this
// expected-name list in lock-step.
//
// SOURCE OF TRUTH: shared/v2_substrate/failed_interventions/types.go
// (FailedInterventionRecord).
func TestFailedInterventionRecordFieldPinning(t *testing.T) {
	want := []string{
		"ResidentID",
		"InterventionType",
		"AttemptDate",
		"Outcome",
		"DocumentedReason",
		"RetryEligibleDate",
		"DocumentedBy",
	}
	got := fieldNames(FailedInterventionRecord{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("FailedInterventionRecord fields drifted: want %v got %v\n"+
			"if canonical type changed, update local copy + SOURCE OF TRUTH comment",
			want, got)
	}
}

// TestFailedInterventionOutcomeConstants pins the closed vocabulary so
// drift between the s2 mirror and the canonical failed_interventions
// package is caught at CI.
func TestFailedInterventionOutcomeConstants(t *testing.T) {
	cases := []struct {
		got, want string
		name      string
	}{
		{OutcomeReversedDueToBPSDRecurrence, "reversed_due_to_BPSD_recurrence", "BPSDRecurrence"},
		{OutcomeReversedDueToFamilyRequest, "reversed_due_to_family_request", "FamilyRequest"},
		{OutcomeReversedDueToClinicalDecline, "reversed_due_to_clinical_decline", "ClinicalDecline"},
		{OutcomeReversedDueToFrailty, "reversed_due_to_frailty", "Frailty"},
		{OutcomeGoalsOfCareAligned, "goals_of_care_aligned", "GoalsOfCareAligned"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("Outcome %s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
