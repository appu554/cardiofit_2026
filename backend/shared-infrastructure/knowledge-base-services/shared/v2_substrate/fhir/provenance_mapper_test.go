package fhir

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func nodeForRoundTripRecommendation() models.EvidenceTraceNode {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	roleID := uuid.New()
	personID := uuid.New()
	authID := uuid.New()
	residentID := uuid.New()
	return models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: "draft -> submitted",
		RecordedAt:      now,
		OccurredAt:      now.Add(-1 * time.Minute),
		Actor: models.TraceActor{
			RoleRef:           &roleID,
			PersonRef:         &personID,
			AuthorityBasisRef: &authID,
		},
		Inputs: []models.TraceInput{
			{InputType: models.TraceInputTypeObservation, InputRef: uuid.New(), RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
			{InputType: models.TraceInputTypeMedicineUse, InputRef: uuid.New(), RoleInDecision: models.TraceRoleInDecisionSupportive},
		},
		ReasoningSummary: &models.ReasoningSummary{
			Text:                          "BP elevated; suggest titration",
			RuleFires:                     []string{"BP_HIGH_001", "TITRATION_002"},
			SuppressionsEvaluated:         []string{"SUPP_FRAILTY"},
			SuppressionsFired:             nil,
			AlternativesConsidered:        []string{"ACE_INHIBITOR", "ARB"},
			AlternativeSelectionRationale: "ACEI contraindicated by recent K+ rise",
		},
		Outputs: []models.TraceOutput{
			{OutputType: "Recommendation", OutputRef: uuid.New()},
		},
		ResidentRef: &residentID,
	}
}

func nodeForRoundTripMonitoring() models.EvidenceTraceNode {
	now := time.Date(2026, 5, 6, 13, 30, 0, 0, time.UTC)
	residentID := uuid.New()
	return models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineMonitoring,
		StateChangeType: "activated",
		RecordedAt:      now,
		OccurredAt:      now,
		Outputs: []models.TraceOutput{
			{OutputType: "MonitoringPlan", OutputRef: uuid.New()},
		},
		ResidentRef: &residentID,
	}
}

func TestProvenanceRoundTrip_Recommendation(t *testing.T) {
	in := nodeForRoundTripRecommendation()
	prov, err := EvidenceTraceNodeToProvenance(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if rt, _ := prov["resourceType"].(string); rt != "Provenance" {
		t.Fatalf("resourceType: got %q", rt)
	}

	out, err := ProvenanceToEvidenceTraceNode(reMarshal(t, prov))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}

	if out.ID != in.ID {
		t.Errorf("ID round-trip: got %s, want %s", out.ID, in.ID)
	}
	if out.StateMachine != in.StateMachine {
		t.Errorf("StateMachine: got %q, want %q", out.StateMachine, in.StateMachine)
	}
	if out.StateChangeType != in.StateChangeType {
		t.Errorf("StateChangeType: got %q, want %q", out.StateChangeType, in.StateChangeType)
	}
	if !out.RecordedAt.Equal(in.RecordedAt) {
		t.Errorf("RecordedAt: got %v, want %v", out.RecordedAt, in.RecordedAt)
	}
	if !out.OccurredAt.Equal(in.OccurredAt) {
		t.Errorf("OccurredAt: got %v, want %v", out.OccurredAt, in.OccurredAt)
	}
	if out.Actor.RoleRef == nil || *out.Actor.RoleRef != *in.Actor.RoleRef {
		t.Errorf("RoleRef round-trip failed")
	}
	if out.Actor.PersonRef == nil || *out.Actor.PersonRef != *in.Actor.PersonRef {
		t.Errorf("PersonRef round-trip failed")
	}
	if out.Actor.AuthorityBasisRef == nil || *out.Actor.AuthorityBasisRef != *in.Actor.AuthorityBasisRef {
		t.Errorf("AuthorityBasisRef round-trip failed")
	}
	if out.ResidentRef == nil || *out.ResidentRef != *in.ResidentRef {
		t.Errorf("ResidentRef round-trip failed")
	}
	if len(out.Inputs) != len(in.Inputs) {
		t.Fatalf("inputs len: got %d, want %d", len(out.Inputs), len(in.Inputs))
	}
	for i := range in.Inputs {
		if out.Inputs[i].InputType != in.Inputs[i].InputType ||
			out.Inputs[i].InputRef != in.Inputs[i].InputRef ||
			out.Inputs[i].RoleInDecision != in.Inputs[i].RoleInDecision {
			t.Errorf("input[%d] mismatch: got %+v, want %+v", i, out.Inputs[i], in.Inputs[i])
		}
	}
	if len(out.Outputs) != len(in.Outputs) {
		t.Fatalf("outputs len: got %d, want %d", len(out.Outputs), len(in.Outputs))
	}
	if out.Outputs[0].OutputType != in.Outputs[0].OutputType ||
		out.Outputs[0].OutputRef != in.Outputs[0].OutputRef {
		t.Errorf("output mismatch: got %+v, want %+v", out.Outputs[0], in.Outputs[0])
	}
	if out.ReasoningSummary == nil {
		t.Fatal("ReasoningSummary lost")
	}
	if out.ReasoningSummary.Text != in.ReasoningSummary.Text {
		t.Errorf("ReasoningSummary.Text mismatch")
	}
	if len(out.ReasoningSummary.RuleFires) != len(in.ReasoningSummary.RuleFires) {
		t.Errorf("RuleFires mismatch")
	}
	if out.ReasoningSummary.AlternativeSelectionRationale != in.ReasoningSummary.AlternativeSelectionRationale {
		t.Errorf("AlternativeSelectionRationale mismatch")
	}
}

func TestProvenanceRoundTrip_Monitoring(t *testing.T) {
	in := nodeForRoundTripMonitoring()
	prov, err := EvidenceTraceNodeToProvenance(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	out, err := ProvenanceToEvidenceTraceNode(reMarshal(t, prov))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.StateMachine != in.StateMachine {
		t.Errorf("StateMachine round-trip failed")
	}
	if out.StateChangeType != in.StateChangeType {
		t.Errorf("StateChangeType round-trip failed")
	}
	if len(out.Outputs) != 1 || out.Outputs[0].OutputType != "MonitoringPlan" {
		t.Errorf("MonitoringPlan output round-trip failed")
	}
}

func TestProvenanceMapper_RejectsInvalidInput(t *testing.T) {
	// missing state_change_type should be rejected on egress.
	n := nodeForRoundTripRecommendation()
	n.StateChangeType = ""
	if _, err := EvidenceTraceNodeToProvenance(n); err == nil {
		t.Error("expected egress validation error")
	}
}

func TestProvenanceMapper_RejectsWrongResourceType(t *testing.T) {
	if _, err := ProvenanceToEvidenceTraceNode(map[string]interface{}{"resourceType": "Encounter"}); err == nil {
		t.Error("expected error for wrong resourceType")
	}
}
