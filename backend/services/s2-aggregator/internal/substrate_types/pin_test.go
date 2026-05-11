package substrate_types

import (
	"reflect"
	"testing"
)

// TestPRNAdministrationFieldPinning structurally pins the field names of
// PRNAdministration so drift against the canonical prn_velocity.Administration
// is caught at CI time. If the canonical type adds or renames a field,
// update this list AND the local copy in lock-step.
//
// SOURCE OF TRUTH: shared/v2_substrate/prn_velocity/types.go (Administration).
func TestPRNAdministrationFieldPinning(t *testing.T) {
	want := []string{"ResidentID", "Class", "AdministeredAt"}
	got := fieldNames(PRNAdministration{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("PRNAdministration fields drifted: want %v got %v\n"+
			"if canonical type changed, update local copy + SOURCE OF TRUTH comment",
			want, got)
	}
}

// TestPRNVelocityResultFieldPinning pins PRNVelocityResult.
//
// SOURCE OF TRUTH: shared/v2_substrate/prn_velocity/types.go (VelocityResult).
func TestPRNVelocityResultFieldPinning(t *testing.T) {
	want := []string{
		"ResidentID", "Class", "EvaluatedAt",
		"Recent30dCount", "Baseline90dAvg", "VelocityRatio", "Severity",
	}
	got := fieldNames(PRNVelocityResult{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("PRNVelocityResult fields drifted: want %v got %v", want, got)
	}
}

func fieldNames(v interface{}) []string {
	t := reflect.TypeOf(v)
	out := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		out[i] = t.Field(i).Name
	}
	return out
}
