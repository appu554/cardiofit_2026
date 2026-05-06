package fhir

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func systemNodeAuthorisation() models.EvidenceTraceNode {
	now := time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC)
	roleID := uuid.New()
	personID := uuid.New()
	authID := uuid.New()
	return models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineAuthorisation,
		StateChangeType: "credential verified",
		RecordedAt:      now,
		OccurredAt:      now,
		Actor: models.TraceActor{
			RoleRef:           &roleID,
			PersonRef:         &personID,
			AuthorityBasisRef: &authID,
		},
		Inputs: []models.TraceInput{
			{InputType: "Credential", InputRef: uuid.New(), RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
		},
		Outputs: []models.TraceOutput{
			{OutputType: "AuthorisationDecision", OutputRef: uuid.New()},
		},
		// system-only: no ResidentRef
	}
}

func TestAuditEventRoundTrip_Authorisation(t *testing.T) {
	in := systemNodeAuthorisation()
	ae, err := EvidenceTraceNodeToAuditEvent(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if rt, _ := ae["resourceType"].(string); rt != "AuditEvent" {
		t.Fatalf("resourceType: got %q", rt)
	}

	out, err := AuditEventToEvidenceTraceNode(reMarshal(t, ae))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}

	if out.ID != in.ID {
		t.Errorf("ID round-trip failed")
	}
	if out.StateMachine != in.StateMachine {
		t.Errorf("StateMachine: got %q, want %q", out.StateMachine, in.StateMachine)
	}
	if out.StateChangeType != in.StateChangeType {
		t.Errorf("StateChangeType: got %q, want %q", out.StateChangeType, in.StateChangeType)
	}
	if !out.RecordedAt.Equal(in.RecordedAt) {
		t.Errorf("RecordedAt mismatch")
	}
	if !out.OccurredAt.Equal(in.OccurredAt) {
		t.Errorf("OccurredAt mismatch")
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
	if out.ResidentRef != nil {
		t.Errorf("ResidentRef should remain nil for system node")
	}
	if len(out.Inputs) != 1 {
		t.Fatalf("inputs len: got %d, want 1", len(out.Inputs))
	}
	if out.Inputs[0].InputType != in.Inputs[0].InputType ||
		out.Inputs[0].InputRef != in.Inputs[0].InputRef ||
		out.Inputs[0].RoleInDecision != in.Inputs[0].RoleInDecision {
		t.Errorf("input round-trip mismatch: got %+v, want %+v", out.Inputs[0], in.Inputs[0])
	}
	if len(out.Outputs) != 1 {
		t.Fatalf("outputs len: got %d, want 1", len(out.Outputs))
	}
	if out.Outputs[0].OutputType != in.Outputs[0].OutputType ||
		out.Outputs[0].OutputRef != in.Outputs[0].OutputRef {
		t.Errorf("output round-trip mismatch")
	}
}

func TestAuditEventMapper_RejectsWrongResourceType(t *testing.T) {
	if _, err := AuditEventToEvidenceTraceNode(map[string]interface{}{"resourceType": "Provenance"}); err == nil {
		t.Error("expected error for wrong resourceType")
	}
}

func TestAuditEventMapper_RejectsInvalidNode(t *testing.T) {
	n := systemNodeAuthorisation()
	n.RecordedAt = time.Time{}
	if _, err := EvidenceTraceNodeToAuditEvent(n); err == nil {
		t.Error("expected egress validation error for zero RecordedAt")
	}
}
