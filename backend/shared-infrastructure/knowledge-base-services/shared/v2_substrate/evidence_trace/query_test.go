package evidence_trace

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// memoryEdgeStore is a pure in-memory EdgeStore for testing the BFS
// traversal logic without a database. The underlying edges slice is
// kept de-duplicated by (From, To, Kind) to mirror the canonical PK.
type memoryEdgeStore struct {
	edges []Edge
}

func newMemoryEdgeStore() *memoryEdgeStore { return &memoryEdgeStore{} }

func (m *memoryEdgeStore) InsertEdge(_ context.Context, e Edge) error {
	for _, existing := range m.edges {
		if existing.From == e.From && existing.To == e.To && existing.Kind == e.Kind {
			return nil // idempotent
		}
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	m.edges = append(m.edges, e)
	return nil
}

func (m *memoryEdgeStore) OutEdges(_ context.Context, from uuid.UUID, kind EdgeKind) ([]Edge, error) {
	var out []Edge
	for _, e := range m.edges {
		if e.From != from {
			continue
		}
		if kind != "" && e.Kind != kind {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (m *memoryEdgeStore) InEdges(_ context.Context, to uuid.UUID, kind EdgeKind) ([]Edge, error) {
	var out []Edge
	for _, e := range m.edges {
		if e.To != to {
			continue
		}
		if kind != "" && e.Kind != kind {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func mustInsert(t *testing.T, m *memoryEdgeStore, from, to uuid.UUID, kind EdgeKind) {
	t.Helper()
	if err := m.InsertEdge(context.Background(), Edge{From: from, To: to, Kind: kind}); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Linear chain — depth cap halts traversal
// ---------------------------------------------------------------------------

func TestTraceForward_LinearChain_DepthCapHalts(t *testing.T) {
	store := newMemoryEdgeStore()
	const N = 100
	nodes := make([]uuid.UUID, N)
	for i := range nodes {
		nodes[i] = uuid.New()
	}
	for i := 0; i < N-1; i++ {
		mustInsert(t, store, nodes[i], nodes[i+1], EdgeKindLedTo)
	}

	// Cap at 5 hops should reach exactly 5 nodes (excluding start).
	tr, err := TraceForward(context.Background(), store, nodes[0], 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 5 {
		t.Errorf("depth-5 cap: got %d nodes, want 5", len(tr.NodeIDs))
	}
	if tr.Depth != 5 {
		t.Errorf("max depth: got %d, want 5", tr.Depth)
	}
	for i, id := range tr.NodeIDs {
		if id != nodes[i+1] {
			t.Errorf("BFS order: idx %d got %s, want %s", i, id, nodes[i+1])
		}
	}

	// Cap at 200 (> chain length) should reach all 99 successors.
	tr, err = TraceForward(context.Background(), store, nodes[0], 200, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != N-1 {
		t.Errorf("uncapped: got %d nodes, want %d", len(tr.NodeIDs), N-1)
	}
}

func TestTraceBackward_LinearChain(t *testing.T) {
	store := newMemoryEdgeStore()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	mustInsert(t, store, a, b, EdgeKindLedTo)
	mustInsert(t, store, b, c, EdgeKindLedTo)

	tr, err := TraceBackward(context.Background(), store, c, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 2 {
		t.Fatalf("got %d ancestors, want 2", len(tr.NodeIDs))
	}
	if tr.NodeIDs[0] != b || tr.NodeIDs[1] != a {
		t.Errorf("backward BFS order wrong: %v", tr.NodeIDs)
	}
}

// ---------------------------------------------------------------------------
// Diamond pattern — visited-set prevents re-counting
// ---------------------------------------------------------------------------

func TestTraceForward_Diamond_NoDoubleCount(t *testing.T) {
	store := newMemoryEdgeStore()
	//   A
	//  / \
	// B   C
	//  \ /
	//   D
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mustInsert(t, store, a, b, EdgeKindLedTo)
	mustInsert(t, store, a, c, EdgeKindLedTo)
	mustInsert(t, store, b, d, EdgeKindLedTo)
	mustInsert(t, store, c, d, EdgeKindLedTo)

	tr, err := TraceForward(context.Background(), store, a, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 3 { // B, C, D — D appears exactly once
		t.Errorf("diamond: got %d nodes, want 3 (B,C,D)", len(tr.NodeIDs))
	}
	count := map[uuid.UUID]int{}
	for _, id := range tr.NodeIDs {
		count[id]++
	}
	if count[d] != 1 {
		t.Errorf("D appears %d times, want 1", count[d])
	}
}

// ---------------------------------------------------------------------------
// Cycle — visited-set prevents infinite loop
// ---------------------------------------------------------------------------

func TestTraceForward_Cycle_TerminatesWithoutInfiniteLoop(t *testing.T) {
	store := newMemoryEdgeStore()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	mustInsert(t, store, a, b, EdgeKindLedTo)
	mustInsert(t, store, b, c, EdgeKindLedTo)
	mustInsert(t, store, c, a, EdgeKindLedTo) // cycle back

	tr, err := TraceForward(context.Background(), store, a, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 2 { // B, C — A is start, never re-visited
		t.Errorf("cycle: got %d nodes, want 2 (B,C)", len(tr.NodeIDs))
	}
}

func TestTraceBackward_Cycle_TerminatesWithoutInfiniteLoop(t *testing.T) {
	store := newMemoryEdgeStore()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	mustInsert(t, store, a, b, EdgeKindLedTo)
	mustInsert(t, store, b, c, EdgeKindLedTo)
	mustInsert(t, store, c, a, EdgeKindLedTo)

	tr, err := TraceBackward(context.Background(), store, a, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 2 {
		t.Errorf("backward cycle: got %d nodes, want 2", len(tr.NodeIDs))
	}
}

// ---------------------------------------------------------------------------
// Multi-edge-kind graph + kindFilter
// ---------------------------------------------------------------------------

func TestTraceForward_KindFilter_Selective(t *testing.T) {
	store := newMemoryEdgeStore()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mustInsert(t, store, a, b, EdgeKindLedTo)
	mustInsert(t, store, a, c, EdgeKindEvidenceFor)
	mustInsert(t, store, a, d, EdgeKindSuppressed)

	// Only follow led_to edges → reach B only.
	tr, err := TraceForward(context.Background(), store, a, 5, []EdgeKind{EdgeKindLedTo})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 1 || tr.NodeIDs[0] != b {
		t.Errorf("kindFilter led_to: got %v, want [B]", tr.NodeIDs)
	}

	// Follow led_to + evidence_for → reach B and C.
	tr, err = TraceForward(context.Background(), store, a, 5, []EdgeKind{EdgeKindLedTo, EdgeKindEvidenceFor})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 2 {
		t.Errorf("kindFilter led_to+evidence_for: got %d nodes, want 2", len(tr.NodeIDs))
	}

	// No filter → reach all three.
	tr, err = TraceForward(context.Background(), store, a, 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 3 {
		t.Errorf("no filter: got %d nodes, want 3", len(tr.NodeIDs))
	}
}

// ---------------------------------------------------------------------------
// Argument validation
// ---------------------------------------------------------------------------

func TestTraceForward_RejectsNonPositiveDepth(t *testing.T) {
	store := newMemoryEdgeStore()
	a := uuid.New()
	if _, err := TraceForward(context.Background(), store, a, 0, nil); err == nil {
		t.Error("expected error for depth=0")
	}
	if _, err := TraceForward(context.Background(), store, a, -1, nil); err == nil {
		t.Error("expected error for negative depth")
	}
}

func TestTraceForward_RejectsNilStore(t *testing.T) {
	if _, err := TraceForward(context.Background(), nil, uuid.New(), 1, nil); err == nil {
		t.Error("expected error for nil store")
	}
}

func TestTraceForward_NodeWithNoEdges(t *testing.T) {
	store := newMemoryEdgeStore()
	a := uuid.New()
	tr, err := TraceForward(context.Background(), store, a, 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.NodeIDs) != 0 {
		t.Errorf("isolated node: got %d, want 0", len(tr.NodeIDs))
	}
	if tr.Depth != 0 {
		t.Errorf("isolated depth: got %d, want 0", tr.Depth)
	}
}
