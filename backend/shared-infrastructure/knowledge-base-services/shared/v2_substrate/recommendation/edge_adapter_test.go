package recommendation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// fakeNodeWriter captures EvidenceTraceNode upserts for assertion.
type fakeNodeWriter struct {
	nodes []models.EvidenceTraceNode
}

func (f *fakeNodeWriter) UpsertEvidenceTraceNode(_ context.Context,
	n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error) {
	f.nodes = append(f.nodes, n)
	return &n, nil
}

func TestEvidenceTraceAdapter_UpsertsRichNode(t *testing.T) {
	writer := &fakeNodeWriter{}
	adapter := NewEvidenceTraceAdapter(writer)

	residentID := uuid.New()
	personID := uuid.New()
	recID := uuid.New()
	inputRef := uuid.New()

	edge := EvidenceEdge{
		RecommendationID: recID,
		ResidentID:       residentID,
		FromState:        "drafted",
		ToState:          "submitted",
		ActorID:          personID,
		ActorClass:       ActorClassHuman,
		OccurredAt:       time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC),
		ReasoningSummary: "pharmacist completed draft",
		InputRefs:        []uuid.UUID{inputRef},
	}
	if err := adapter.EmitEdge(context.Background(), edge); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if len(writer.nodes) != 1 {
		t.Fatalf("expected 1 node upserted; got %d", len(writer.nodes))
	}
	node := writer.nodes[0]
	if node.StateMachine != models.EvidenceTraceStateMachineRecommendation {
		t.Errorf("state_machine = %q, want %q",
			node.StateMachine, models.EvidenceTraceStateMachineRecommendation)
	}
	if node.StateChangeType != "drafted -> submitted" {
		t.Errorf("state_change_type = %q", node.StateChangeType)
	}
	if !node.OccurredAt.Equal(edge.OccurredAt) {
		t.Errorf("occurred_at not propagated: got %v want %v",
			node.OccurredAt, edge.OccurredAt)
	}
	if node.ResidentRef == nil || *node.ResidentRef != residentID {
		t.Errorf("resident_ref not propagated")
	}
	if node.Actor.PersonRef == nil || *node.Actor.PersonRef != personID {
		t.Errorf("actor.person_ref not propagated")
	}
	if node.ReasoningSummary == nil {
		t.Fatalf("reasoning_summary nil")
	}
	if !strings.Contains(node.ReasoningSummary.Text, "actor_class=human") {
		t.Errorf("reasoning_summary.text missing actor_class prefix; got %q",
			node.ReasoningSummary.Text)
	}
	if !strings.Contains(node.ReasoningSummary.Text, edge.ReasoningSummary) {
		t.Errorf("reasoning_summary.text = %q does not contain caller reasoning %q",
			node.ReasoningSummary.Text, edge.ReasoningSummary)
	}
	if len(node.Inputs) != 1 {
		t.Fatalf("expected 1 input; got %d", len(node.Inputs))
	}
	if node.Inputs[0].InputRef != inputRef {
		t.Errorf("input_ref not propagated: got %v want %v",
			node.Inputs[0].InputRef, inputRef)
	}
	// Outputs should contain the recommendation_id so future queries can
	// rejoin nodes by recommendation.
	if len(node.Outputs) != 1 {
		t.Fatalf("expected 1 output (the recommendation ref); got %d",
			len(node.Outputs))
	}
	if node.Outputs[0].OutputRef != recID {
		t.Errorf("output_ref = %v want %v", node.Outputs[0].OutputRef, recID)
	}
}

func TestEvidenceTraceAdapter_GeneratesUniqueNodeIDs(t *testing.T) {
	writer := &fakeNodeWriter{}
	adapter := NewEvidenceTraceAdapter(writer)
	for i := 0; i < 3; i++ {
		_ = adapter.EmitEdge(context.Background(), EvidenceEdge{
			RecommendationID: uuid.New(),
			ResidentID:       uuid.New(),
			FromState:        "detected",
			ToState:          "drafted",
			ActorID:          uuid.New(),
			ActorClass:       ActorClassAlgorithmic,
			OccurredAt:       time.Now().UTC(),
		})
	}
	seen := map[uuid.UUID]bool{}
	for _, n := range writer.nodes {
		if seen[n.ID] {
			t.Errorf("duplicate node ID %v", n.ID)
		}
		seen[n.ID] = true
		if n.ID == uuid.Nil {
			t.Errorf("node ID must not be uuid.Nil")
		}
	}
}

func TestEvidenceTraceAdapter_NoInputsNoReasoning(t *testing.T) {
	writer := &fakeNodeWriter{}
	adapter := NewEvidenceTraceAdapter(writer)

	err := adapter.EmitEdge(context.Background(), EvidenceEdge{
		RecommendationID: uuid.New(),
		ResidentID:       uuid.New(),
		FromState:        "submitted",
		ToState:          "viewed",
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		OccurredAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if len(writer.nodes) != 1 {
		t.Fatalf("expected 1 node; got %d", len(writer.nodes))
	}
	node := writer.nodes[0]
	if node.ReasoningSummary == nil {
		t.Fatalf("expected ReasoningSummary populated with actor_class even when caller's reasoning is empty")
	}
	if !strings.Contains(node.ReasoningSummary.Text, "actor_class=human") {
		t.Errorf("expected actor_class in reasoning; got %q", node.ReasoningSummary.Text)
	}
	if len(node.Inputs) != 0 {
		t.Errorf("inputs should be empty when edge has no input refs")
	}
	// Still must record the recommendation_id as output.
	if len(node.Outputs) != 1 {
		t.Errorf("expected 1 output (recommendation_id); got %d", len(node.Outputs))
	}
}
