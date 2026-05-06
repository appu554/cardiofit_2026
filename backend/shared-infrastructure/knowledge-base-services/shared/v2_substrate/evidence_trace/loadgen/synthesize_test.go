package loadgen

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// memSink is the in-memory NodeSink for unit tests / benchmarks.
type memSink struct {
	mu    sync.Mutex
	nodes map[uuid.UUID]models.EvidenceTraceNode
	edges []evidence_trace.Edge
}

func newMemSink() *memSink {
	return &memSink{nodes: map[uuid.UUID]models.EvidenceTraceNode{}}
}

func (m *memSink) UpsertEvidenceTraceNode(_ context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes[n.ID] = n
	c := n
	return &c, nil
}

func (m *memSink) InsertEvidenceTraceEdge(_ context.Context, e evidence_trace.Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ex := range m.edges {
		if ex.From == e.From && ex.To == e.To && ex.Kind == e.Kind {
			return nil
		}
	}
	m.edges = append(m.edges, e)
	return nil
}

func TestSynthesize_SmallProfile(t *testing.T) {
	sink := newMemSink()
	p := SmallProfile()
	stats, err := Synthesize(context.Background(), sink, p)
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	expectedNodes := p.ResidentCount * p.DaysPerResident * p.NodesPerDay
	if stats.Nodes != expectedNodes {
		t.Fatalf("nodes: want %d got %d", expectedNodes, stats.Nodes)
	}
	// 4 edges per resident-day (3 derived_from + 1 led_to).
	expectedEdges := p.ResidentCount * p.DaysPerResident * 4
	if stats.Edges != expectedEdges {
		t.Fatalf("edges: want %d got %d", expectedEdges, stats.Edges)
	}
	if len(sink.nodes) != expectedNodes {
		t.Fatalf("sink node count drift: want %d got %d", expectedNodes, len(sink.nodes))
	}
}

func TestSynthesize_RejectsNilSink(t *testing.T) {
	if _, err := Synthesize(context.Background(), nil, SmallProfile()); err == nil {
		t.Fatal("expected error for nil sink")
	}
}

func TestDefaultProfile_ApproxScale(t *testing.T) {
	p := DefaultProfile()
	nodes := p.ResidentCount * p.DaysPerResident * p.NodesPerDay
	if nodes < 150_000 || nodes > 250_000 {
		t.Fatalf("default profile node count out of expected band: %d", nodes)
	}
}
