package evidence_trace

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// memoryNodeStore is the unit-test-only NodeStore. Stable, no DB.
type memoryNodeStore struct {
	byID map[uuid.UUID]models.EvidenceTraceNode
}

func newMemoryNodeStore() *memoryNodeStore {
	return &memoryNodeStore{byID: map[uuid.UUID]models.EvidenceTraceNode{}}
}

func (m *memoryNodeStore) put(n models.EvidenceTraceNode) {
	m.byID[n.ID] = n
}

func (m *memoryNodeStore) GetEvidenceTraceNode(_ context.Context, id uuid.UUID) (*models.EvidenceTraceNode, error) {
	n, ok := m.byID[id]
	if !ok {
		return nil, ErrInvalidDepth // sentinel reuse — irrelevant for these tests
	}
	c := n
	return &c, nil
}

func (m *memoryNodeStore) ListEvidenceTraceNodesByResident(_ context.Context, residentRef uuid.UUID, from, to time.Time) ([]models.EvidenceTraceNode, error) {
	var out []models.EvidenceTraceNode
	for _, n := range m.byID {
		if n.ResidentRef == nil || *n.ResidentRef != residentRef {
			continue
		}
		if n.RecordedAt.Before(from) || !n.RecordedAt.Before(to) {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

// makeNode builds a node valid enough for these tests (we never call
// validation; the mappers aren't on the test path).
func makeNode(stateMachine, stateChange string, residentRef *uuid.UUID, when time.Time, evidenceInputs int) models.EvidenceTraceNode {
	n := models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    stateMachine,
		StateChangeType: stateChange,
		RecordedAt:      when,
		OccurredAt:      when,
		ResidentRef:     residentRef,
	}
	for i := 0; i < evidenceInputs; i++ {
		n.Inputs = append(n.Inputs, models.TraceInput{
			InputType:      "Observation",
			InputRef:       uuid.New(),
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		})
	}
	return n
}

// ---------------------------------------------------------------------------
// Fixture: a 100-node graph (the plan asks for fixture coverage).
// Shape: 90 leaf observation nodes, 10 recommendation nodes; each rec links
// to ~9 obs as derived_from, then leads to a synthetic outcome node.
// ---------------------------------------------------------------------------

type fixtureGraph struct {
	resident       uuid.UUID
	observations   []models.EvidenceTraceNode
	recommendations []models.EvidenceTraceNode
	outcomes       []models.EvidenceTraceNode
	edges          *memoryEdgeStore
	nodes          *memoryNodeStore
}

func buildFixture(t *testing.T) *fixtureGraph {
	t.Helper()
	resident := uuid.New()
	res := &fixtureGraph{
		resident: resident,
		edges:    newMemoryEdgeStore(),
		nodes:    newMemoryNodeStore(),
	}
	base := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)

	// 90 observation-seed nodes (state_machine=Monitoring, role=observation).
	for i := 0; i < 90; i++ {
		n := makeNode(models.EvidenceTraceStateMachineMonitoring, "observation_recorded", &resident, base.Add(time.Duration(i)*time.Minute), 0)
		res.observations = append(res.observations, n)
		res.nodes.put(n)
	}
	// 10 recommendation nodes — each links to obs[i*9 : i*9+9] as derived_from
	// and leads_to one outcome node.
	for i := 0; i < 10; i++ {
		rec := makeNode(models.EvidenceTraceStateMachineRecommendation, "decided_accepted", &resident, base.Add(time.Duration(i)*time.Hour), 9)
		res.recommendations = append(res.recommendations, rec)
		res.nodes.put(rec)
		// derived_from edges: obs → rec
		for j := 0; j < 9; j++ {
			mustInsert(t, res.edges, res.observations[i*9+j].ID, rec.ID, EdgeKindDerivedFrom)
		}
		// outcome
		outcome := makeNode(models.EvidenceTraceStateMachineClinicalState, "outcome_recorded", &resident, base.Add(time.Duration(i)*time.Hour+time.Minute), 1)
		res.outcomes = append(res.outcomes, outcome)
		res.nodes.put(outcome)
		mustInsert(t, res.edges, rec.ID, outcome.ID, EdgeKindLedTo)
	}
	return res
}

func TestLineageOf_Fixture(t *testing.T) {
	g := buildFixture(t)
	// Pick the 5th recommendation; expect 9 upstream observations.
	rec := g.recommendations[5]
	out, err := LineageOf(context.Background(), rec.ID, g.nodes, g.edges, 5)
	if err != nil {
		t.Fatalf("LineageOf: %v", err)
	}
	if out.TargetNodeID != rec.ID {
		t.Fatalf("target drift")
	}
	if len(out.Nodes) != 9 {
		t.Fatalf("want 9 upstream observations, got %d", len(out.Nodes))
	}
	for _, s := range out.Nodes {
		if s.StateMachine != models.EvidenceTraceStateMachineMonitoring {
			t.Fatalf("upstream summary should be Monitoring, got %s", s.StateMachine)
		}
	}
}

func TestConsequencesOf_Fixture(t *testing.T) {
	g := buildFixture(t)
	rec := g.recommendations[3]
	out, err := ConsequencesOf(context.Background(), rec.ID, g.nodes, g.edges, 5)
	if err != nil {
		t.Fatalf("ConsequencesOf: %v", err)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("want 1 downstream outcome, got %d", len(out.Nodes))
	}
	if out.Nodes[0].StateMachine != models.EvidenceTraceStateMachineClinicalState {
		t.Fatalf("downstream should be ClinicalState, got %s", out.Nodes[0].StateMachine)
	}
}

func TestConsequencesOf_DefaultDepth(t *testing.T) {
	g := buildFixture(t)
	// Pass maxDepth=0 → defaults to defaultQueryDepth=10.
	out, err := ConsequencesOf(context.Background(), g.recommendations[0].ID, g.nodes, g.edges, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("default depth should still resolve 1 hop, got %d", len(out.Nodes))
	}
}

func TestReasoningWindow_Fixture(t *testing.T) {
	g := buildFixture(t)
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	w, err := ReasoningWindow(context.Background(), g.resident, from, to, g.nodes)
	if err != nil {
		t.Fatalf("ReasoningWindow: %v", err)
	}
	// 90 obs + 10 rec + 10 outcomes = 110.
	if w.TotalNodes != 110 {
		t.Fatalf("want 110 nodes, got %d", w.TotalNodes)
	}
	if w.RecommendationCount != 10 {
		t.Fatalf("want 10 recommendations, got %d", w.RecommendationCount)
	}
	if w.AverageEvidencePerRecommendation != 9 {
		t.Fatalf("want avg 9 evidence per recommendation, got %v", w.AverageEvidencePerRecommendation)
	}
	if w.DecisionCount != 10 {
		// "decided_accepted" matches both 'decided' and 'accepted' → still
		// counted once per node.
		t.Fatalf("want 10 decisions, got %d", w.DecisionCount)
	}
	if got := w.NodesByStateMachine[models.EvidenceTraceStateMachineMonitoring]; got != 90 {
		t.Fatalf("Monitoring count: want 90 got %d", got)
	}
	// Ensure chronological ordering.
	for i := 1; i < len(w.Nodes); i++ {
		if w.Nodes[i].RecordedAt.Before(w.Nodes[i-1].RecordedAt) {
			t.Fatalf("nodes not chronologically ordered at index %d", i)
		}
	}
}

func TestReasoningWindow_RejectsInvertedRange(t *testing.T) {
	g := buildFixture(t)
	from := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if _, err := ReasoningWindow(context.Background(), g.resident, from, to, g.nodes); err == nil {
		t.Fatal("expected error for inverted range")
	}
}

func TestLineageOf_NilStore(t *testing.T) {
	if _, err := LineageOf(context.Background(), uuid.New(), nil, newMemoryEdgeStore(), 1); err == nil {
		t.Fatal("expected error on nil node store")
	}
}

func TestIsDecisionStateChange(t *testing.T) {
	if !isDecisionStateChange("Decided_Accepted") {
		t.Fatal("decided should match")
	}
	if !isDecisionStateChange("recommendation rejected by clinician") {
		t.Fatal("rejected should match")
	}
	if isDecisionStateChange("draft") {
		t.Fatal("draft should not match")
	}
	if isDecisionStateChange("") {
		t.Fatal("empty should not match")
	}
}
