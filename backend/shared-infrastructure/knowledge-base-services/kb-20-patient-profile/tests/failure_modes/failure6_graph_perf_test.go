// Wave 6.1 — Failure Mode 6: graph query performance.
//
// Layer 2 doc Part 6 Failure 6: "EvidenceTrace graph traversal must hit
// production SLOs on a 6-month-of-activity dataset. Defence: Wave 5.4
// load test demonstrates forward+backward depth=5 traversal p95 <200ms."
//
// The actual benchmark lives in
// shared/v2_substrate/evidence_trace/bench_test.go (in-process BFS) and
// in shared/v2_substrate/evidence_trace/loadgen (synthesizer). Production
// run against the kb-20 PostgreSQL store is deferred to V1 per the Wave
// 5.4 plan task.
//
// This file documents the cross-wave dependency and runs an in-process
// smoke that exercises a small forward+backward traversal so the failure-
// mode pack visibly covers all six modes.
package failure_modes

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
)

// memoryEdgeStore is a minimal local implementation of EdgeStore for
// the smoke test. (We deliberately don't import the loadgen package here
// to keep the failure-mode pack self-contained.)
type memoryEdgeStore struct{ edges []evidence_trace.Edge }

func (m *memoryEdgeStore) InsertEdge(_ context.Context, e evidence_trace.Edge) error {
	for _, ex := range m.edges {
		if ex.From == e.From && ex.To == e.To && ex.Kind == e.Kind {
			return nil
		}
	}
	m.edges = append(m.edges, e)
	return nil
}
func (m *memoryEdgeStore) OutEdges(_ context.Context, from uuid.UUID, kind evidence_trace.EdgeKind) ([]evidence_trace.Edge, error) {
	var out []evidence_trace.Edge
	for _, e := range m.edges {
		if e.From == from && (kind == "" || e.Kind == kind) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (m *memoryEdgeStore) InEdges(_ context.Context, to uuid.UUID, kind evidence_trace.EdgeKind) ([]evidence_trace.Edge, error) {
	var out []evidence_trace.Edge
	for _, e := range m.edges {
		if e.To == to && (kind == "" || e.Kind == kind) {
			out = append(out, e)
		}
	}
	return out, nil
}

func TestFailure6_GraphTraversalSmoke(t *testing.T) {
	// 50-node linear chain — depth=5 must surface 5 nodes.
	const N = 50
	store := &memoryEdgeStore{}
	nodes := make([]uuid.UUID, N)
	for i := range nodes {
		nodes[i] = uuid.New()
	}
	for i := 0; i < N-1; i++ {
		_ = store.InsertEdge(context.Background(), evidence_trace.Edge{
			From: nodes[i], To: nodes[i+1], Kind: evidence_trace.EdgeKindLedTo,
		})
	}
	out, err := evidence_trace.TraceForward(context.Background(), store, nodes[0], 5, []evidence_trace.EdgeKind{evidence_trace.EdgeKindLedTo})
	if err != nil {
		t.Fatalf("TraceForward: %v", err)
	}
	if len(out.NodeIDs) != 5 {
		t.Fatalf("depth=5 forward traversal must surface 5 nodes; got %d", len(out.NodeIDs))
	}
	bk, err := evidence_trace.TraceBackward(context.Background(), store, nodes[N-1], 5, []evidence_trace.EdgeKind{evidence_trace.EdgeKindLedTo})
	if err != nil {
		t.Fatalf("TraceBackward: %v", err)
	}
	if len(bk.NodeIDs) != 5 {
		t.Fatalf("depth=5 backward traversal must surface 5 nodes; got %d", len(bk.NodeIDs))
	}
}

func TestFailure6_BenchmarkRefersToWave54(t *testing.T) {
	// Documentation-only assertion: the production benchmark lives at
	// shared/v2_substrate/evidence_trace/bench_test.go. CI integration
	// of that benchmark against the kb-20 PostgreSQL store is deferred
	// to V1.
	t.Log("Wave 5.4 benchmark + loadgen package is the production-scale Failure 6 defence; CI integration deferred to V1.")
}
