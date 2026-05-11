package substrate_types

import (
	"reflect"
	"testing"
)

// TestRestraintSignalFieldPinning pins the s2-side RestraintSignal stub
// shape. kb-32 does not yet define a persistent restraint signal type
// (see drift note in restraint.go); this pin captures the shape s2
// commits to until kb-32 catches up.
func TestRestraintSignalFieldPinning(t *testing.T) {
	want := []string{
		"SignalID", "Type", "Severity", "PairedRecommendationID",
		"TriggeredAt", "SubstrateID", "SubstrateSource",
	}
	got := fieldNames(RestraintSignal{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("RestraintSignal fields drifted: want %v got %v", want, got)
	}
}

// TestRestraintAcknowledgmentFieldPinning pins the pharmacist
// acknowledgment shape per S2 v1.0 Part 7.2.
func TestRestraintAcknowledgmentFieldPinning(t *testing.T) {
	want := []string{"SignalID", "PharmacistID", "AcknowledgedAt", "Decision"}
	got := fieldNames(RestraintAcknowledgment{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("RestraintAcknowledgment fields drifted: want %v got %v", want, got)
	}
}

// TestRestraintDecisionConstants pins the closed set of acknowledgment
// decision values per Phase 1 advisory-only commitment (v1.0 Part 7.2 +
// Part 7.4 safety-critical bypass).
func TestRestraintDecisionConstants(t *testing.T) {
	if RestraintDecisionAcknowledgeAdvisory != "acknowledge_advisory" {
		t.Errorf("RestraintDecisionAcknowledgeAdvisory = %q, want acknowledge_advisory", RestraintDecisionAcknowledgeAdvisory)
	}
	if RestraintDecisionSafetyCriticalBypass != "invoke_safety_critical_bypass" {
		t.Errorf("RestraintDecisionSafetyCriticalBypass = %q, want invoke_safety_critical_bypass", RestraintDecisionSafetyCriticalBypass)
	}
}
