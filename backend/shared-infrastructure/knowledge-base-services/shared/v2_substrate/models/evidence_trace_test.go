package models

import "testing"

func TestIsValidEvidenceTraceStateMachine(t *testing.T) {
	valid := []string{
		EvidenceTraceStateMachineAuthorisation,
		EvidenceTraceStateMachineRecommendation,
		EvidenceTraceStateMachineMonitoring,
		EvidenceTraceStateMachineClinicalState,
		EvidenceTraceStateMachineConsent,
	}
	for _, s := range valid {
		if !IsValidEvidenceTraceStateMachine(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	for _, s := range []string{"", "Other", "recommendation", "AUTHORISATION"} {
		if IsValidEvidenceTraceStateMachine(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestIsValidTraceRoleInDecision(t *testing.T) {
	for _, s := range []string{
		TraceRoleInDecisionSupportive,
		TraceRoleInDecisionPrimaryEvidence,
		TraceRoleInDecisionSecondaryEvidence,
		TraceRoleInDecisionCounterEvidence,
	} {
		if !IsValidTraceRoleInDecision(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if IsValidTraceRoleInDecision("primary") {
		t.Error("expected 'primary' to be invalid (must be 'primary_evidence')")
	}
}

func TestIsSystemEvidenceTraceStateMachine(t *testing.T) {
	if !IsSystemEvidenceTraceStateMachine(EvidenceTraceStateMachineAuthorisation) {
		t.Error("Authorisation should be system")
	}
	if !IsSystemEvidenceTraceStateMachine(EvidenceTraceStateMachineConsent) {
		t.Error("Consent should be system")
	}
	if IsSystemEvidenceTraceStateMachine(EvidenceTraceStateMachineRecommendation) {
		t.Error("Recommendation should NOT be system")
	}
	if IsSystemEvidenceTraceStateMachine(EvidenceTraceStateMachineMonitoring) {
		t.Error("Monitoring should NOT be system")
	}
	if IsSystemEvidenceTraceStateMachine(EvidenceTraceStateMachineClinicalState) {
		t.Error("ClinicalState should NOT be system")
	}
}
