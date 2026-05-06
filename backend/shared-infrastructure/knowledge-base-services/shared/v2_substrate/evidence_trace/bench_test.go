package evidence_trace

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// BenchmarkTraceForward_Depth5 exercises the BFS forward traversal on a
// linear chain of 200 nodes with depth=5. The intent is regression
// detection on the pure-Go BFS — the production p95 SLO (Wave 5.4 plan
// table: <200ms at depth=5 on a 6-month synthetic dataset) is verified
// against the real kb-20 PostgreSQL store at V1, not here.
func BenchmarkTraceForward_Depth5(b *testing.B) {
	store := newMemoryEdgeStore()
	const N = 200
	nodes := make([]uuid.UUID, N)
	for i := range nodes {
		nodes[i] = uuid.New()
	}
	for i := 0; i < N-1; i++ {
		_ = store.InsertEdge(context.Background(), Edge{From: nodes[i], To: nodes[i+1], Kind: EdgeKindLedTo})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TraceForward(context.Background(), store, nodes[0], 5, []EdgeKind{EdgeKindLedTo})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTraceBackward_Depth5 is the symmetric reverse benchmark.
func BenchmarkTraceBackward_Depth5(b *testing.B) {
	store := newMemoryEdgeStore()
	const N = 200
	nodes := make([]uuid.UUID, N)
	for i := range nodes {
		nodes[i] = uuid.New()
	}
	for i := 0; i < N-1; i++ {
		_ = store.InsertEdge(context.Background(), Edge{From: nodes[i], To: nodes[i+1], Kind: EdgeKindDerivedFrom})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TraceBackward(context.Background(), store, nodes[N-1], 5, []EdgeKind{EdgeKindDerivedFrom})
		if err != nil {
			b.Fatal(err)
		}
	}
}
