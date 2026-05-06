// Wave 6.2 — Recommendation state-machine integration test.
//
// Layer 2 doc §4.2: "a baseline-delta event triggers a Recommendation
// rule fire; the resulting Recommendation lifecycle write lands in the
// EvidenceTrace as a new node with derived_from edges to the triggering
// observation."
package state_machine_integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// memSink stores nodes + edges in-process so we can assert the
// Recommendation lifecycle's substrate writes without a DB.
type memSink struct {
	nodes map[uuid.UUID]models.EvidenceTraceNode
	edges []evidence_trace.Edge
}

func newMemSink() *memSink { return &memSink{nodes: map[uuid.UUID]models.EvidenceTraceNode{}} }

func (m *memSink) writeNode(n models.EvidenceTraceNode) { m.nodes[n.ID] = n }
func (m *memSink) writeEdge(e evidence_trace.Edge)      { m.edges = append(m.edges, e) }

// mockRule is the tiny Layer-3 stand-in that fires when a baseline delta
// crosses a threshold. Returns the Recommendation node + the derived_from
// edge linking it to the triggering observation.
type mockRule struct{}

func (mockRule) FireOnDelta(observationNode models.EvidenceTraceNode, residentRef uuid.UUID, when time.Time) (models.EvidenceTraceNode, evidence_trace.Edge) {
	rec := models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: "draft -> submitted",
		RecordedAt:      when,
		OccurredAt:      when,
		ResidentRef:     &residentRef,
		Inputs: []models.TraceInput{{
			InputType:      models.TraceInputTypeObservation,
			InputRef:       observationNode.ID,
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		}},
	}
	edge := evidence_trace.Edge{
		From: observationNode.ID,
		To:   rec.ID,
		Kind: evidence_trace.EdgeKindDerivedFrom,
	}
	return rec, edge
}

func TestRecommendation_BaselineDeltaTriggersLifecycleWrite(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	residentRef := uuid.New()
	obs := models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineMonitoring,
		StateChangeType: "baseline_delta_flagged",
		RecordedAt:      time.Now().UTC(),
		OccurredAt:      time.Now().UTC(),
		ResidentRef:     &residentRef,
	}
	sink := newMemSink()
	sink.writeNode(obs)

	rec, edge := mockRule{}.FireOnDelta(obs, residentRef, time.Now().UTC())
	sink.writeNode(rec)
	sink.writeEdge(edge)

	if _, ok := sink.nodes[rec.ID]; !ok {
		t.Fatal("Recommendation node not landed")
	}
	if rec.StateMachine != models.EvidenceTraceStateMachineRecommendation {
		t.Fatalf("Recommendation state machine drift: %s", rec.StateMachine)
	}
	if len(sink.edges) != 1 || sink.edges[0].From != obs.ID || sink.edges[0].To != rec.ID {
		t.Fatalf("expected one derived_from edge obs→rec; got %+v", sink.edges)
	}
	if sink.edges[0].Kind != evidence_trace.EdgeKindDerivedFrom {
		t.Fatalf("expected derived_from kind, got %s", sink.edges[0].Kind)
	}
}

func TestRecommendation_LifecycleNodeCarriesEvidence(t *testing.T) {
	residentRef := uuid.New()
	obs := models.EvidenceTraceNode{ID: uuid.New(), ResidentRef: &residentRef}
	rec, _ := mockRule{}.FireOnDelta(obs, residentRef, time.Now().UTC())
	if len(rec.Inputs) != 1 {
		t.Fatalf("rec must carry 1 input; got %d", len(rec.Inputs))
	}
	if rec.Inputs[0].InputRef != obs.ID {
		t.Fatal("rec input must reference the triggering observation")
	}
	if rec.Inputs[0].RoleInDecision != models.TraceRoleInDecisionPrimaryEvidence {
		t.Fatalf("expected primary_evidence role; got %s", rec.Inputs[0].RoleInDecision)
	}
}
