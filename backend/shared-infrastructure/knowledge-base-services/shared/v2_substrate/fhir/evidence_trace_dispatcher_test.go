package fhir

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func validNode(stateMachine, stateChange string) models.EvidenceTraceNode {
	residentRef := uuid.New()
	return models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    stateMachine,
		StateChangeType: stateChange,
		RecordedAt:      time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		OccurredAt:      time.Date(2026, 5, 6, 8, 55, 0, 0, time.UTC),
		ResidentRef:     &residentRef,
	}
}

func TestRouteEvidenceTrace_AllStateMachinesGoToProvenance(t *testing.T) {
	cases := []string{
		models.EvidenceTraceStateMachineRecommendation,
		models.EvidenceTraceStateMachineMonitoring,
		models.EvidenceTraceStateMachineClinicalState,
		models.EvidenceTraceStateMachineAuthorisation,
		models.EvidenceTraceStateMachineConsent,
	}
	for _, sm := range cases {
		rt, err := RouteEvidenceTrace(validNode(sm, "draft -> submitted"))
		if err != nil {
			t.Fatalf("%s: unexpected error %v", sm, err)
		}
		if rt != ResourceTypeProvenance {
			t.Fatalf("%s: want Provenance got %s", sm, rt)
		}
	}
}

func TestRouteEvidenceTrace_SystemEventsGoToAuditEvent(t *testing.T) {
	for _, change := range []string{"rule_fire", "credential_check", "query_recorded", "login_propagated"} {
		// Even attached to a clinical state machine, the system-event
		// state_change_type wins.
		rt, err := RouteEvidenceTrace(validNode(models.EvidenceTraceStateMachineRecommendation, change))
		if err != nil {
			t.Fatalf("%s: %v", change, err)
		}
		if rt != ResourceTypeAuditEvent {
			t.Fatalf("%s: want AuditEvent got %s", change, rt)
		}
	}
}

func TestRouteEvidenceTrace_CaseInsensitiveSystemEvent(t *testing.T) {
	rt, err := RouteEvidenceTrace(validNode(models.EvidenceTraceStateMachineMonitoring, " RULE_FIRE "))
	if err != nil {
		t.Fatal(err)
	}
	if rt != ResourceTypeAuditEvent {
		t.Fatalf("want AuditEvent got %s", rt)
	}
}

func TestRouteEvidenceTrace_RejectsUnknownStateMachine(t *testing.T) {
	if _, err := RouteEvidenceTrace(validNode("Bogus", "x")); err == nil {
		t.Fatal("expected error for unknown state_machine")
	}
}

func TestMapEvidenceTrace_ProvenanceShape(t *testing.T) {
	n := validNode(models.EvidenceTraceStateMachineRecommendation, "draft -> submitted")
	rt, res, err := MapEvidenceTrace(n)
	if err != nil {
		t.Fatalf("MapEvidenceTrace: %v", err)
	}
	if rt != ResourceTypeProvenance {
		t.Fatalf("want Provenance got %s", rt)
	}
	if got := res["resourceType"]; got != "Provenance" {
		t.Fatalf("body resourceType drift: %v", got)
	}
}

func TestMapEvidenceTrace_AuditEventShape(t *testing.T) {
	n := validNode(models.EvidenceTraceStateMachineMonitoring, "rule_fire")
	rt, res, err := MapEvidenceTrace(n)
	if err != nil {
		t.Fatalf("MapEvidenceTrace: %v", err)
	}
	if rt != ResourceTypeAuditEvent {
		t.Fatalf("want AuditEvent got %s", rt)
	}
	if got := res["resourceType"]; got != "AuditEvent" {
		t.Fatalf("body resourceType drift: %v", got)
	}
}

func TestMapEvidenceTrace_MutualExclusion(t *testing.T) {
	// Run the dispatcher on a 50-node mix and assert exactly-one resource
	// per node.
	machines := []string{
		models.EvidenceTraceStateMachineRecommendation,
		models.EvidenceTraceStateMachineMonitoring,
		models.EvidenceTraceStateMachineClinicalState,
		models.EvidenceTraceStateMachineAuthorisation,
		models.EvidenceTraceStateMachineConsent,
	}
	changes := []string{
		"draft -> submitted",
		"rule_fire",
		"approved",
		"credential_check",
		"observation_recorded",
	}
	for i := 0; i < 50; i++ {
		sm := machines[i%len(machines)]
		ch := changes[i%len(changes)]
		n := validNode(sm, ch)
		rt, res, err := MapEvidenceTrace(n)
		if err != nil {
			t.Fatalf("[%d] %s/%s: %v", i, sm, ch, err)
		}
		if rt != ResourceTypeProvenance && rt != ResourceTypeAuditEvent {
			t.Fatalf("[%d] %s/%s: bogus resourceType %q", i, sm, ch, rt)
		}
		if res == nil {
			t.Fatalf("[%d] nil body", i)
		}
	}
}

func TestIsSystemEventStateChange(t *testing.T) {
	if !IsSystemEventStateChange("rule_fire") {
		t.Fatal("rule_fire should match")
	}
	if !IsSystemEventStateChange("CREDENTIAL_CHECK") {
		t.Fatal("case insensitive match expected")
	}
	if IsSystemEventStateChange("draft -> submitted") {
		t.Fatal("clinical transition should not match")
	}
	if IsSystemEventStateChange("") {
		t.Fatal("empty should not match")
	}
}
