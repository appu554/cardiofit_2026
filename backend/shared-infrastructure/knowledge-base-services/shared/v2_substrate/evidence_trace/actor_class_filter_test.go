package evidence_trace

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakeResolver is a test-only ActorClassResolver backed by a simple map.
// If a nodeID has no entry the resolver returns ("", false, nil) — unclassifiable.
type fakeResolver struct {
	entries map[uuid.UUID]struct {
		class      string
		classified bool
	}
	// forceErrorOn, if set, causes ActorClassFor to return an error for that ID.
	forceErrorOn *uuid.UUID
}

func (f *fakeResolver) ActorClassFor(_ context.Context, id uuid.UUID) (string, bool, error) {
	if f.forceErrorOn != nil && *f.forceErrorOn == id {
		return "", false, errors.New("resolver: simulated lookup failure")
	}
	e, ok := f.entries[id]
	if !ok {
		return "", false, nil
	}
	return e.class, e.classified, nil
}

// makeID is a small helper so tests don't repeat uuid.New() noise.
func makeID() uuid.UUID { return uuid.New() }

// TestFilterByActorClass_KeepsMatchingNodes verifies that only nodes whose
// resolved actor class matches the filter string are kept in the output.
func TestFilterByActorClass_KeepsMatchingNodes(t *testing.T) {
	ctx := context.Background()

	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = makeID()
	}

	// ids[0] and ids[2] and ids[4] → "algorithm"
	// ids[1] and ids[3] → "clinician"
	resolver := &fakeResolver{
		entries: map[uuid.UUID]struct {
			class      string
			classified bool
		}{
			ids[0]: {"algorithm", true},
			ids[1]: {"clinician", true},
			ids[2]: {"algorithm", true},
			ids[3]: {"clinician", true},
			ids[4]: {"algorithm", true},
		},
	}

	in := Traversal{NodeIDs: ids, Depth: 3}
	got, err := FilterByActorClass(ctx, in, resolver, "algorithm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []uuid.UUID{ids[0], ids[2], ids[4]}
	if len(got.NodeIDs) != len(want) {
		t.Fatalf("got %d nodes, want %d; nodes=%v", len(got.NodeIDs), len(want), got.NodeIDs)
	}
	for i, id := range want {
		if got.NodeIDs[i] != id {
			t.Errorf("NodeIDs[%d]: got %v, want %v", i, got.NodeIDs[i], id)
		}
	}
}

// TestFilterByActorClass_DropsUnclassifiable verifies that nodes for which the
// resolver returns (_, false, nil) are silently dropped from the result.
func TestFilterByActorClass_DropsUnclassifiable(t *testing.T) {
	ctx := context.Background()

	classified := makeID()
	unclassifiable1 := makeID()
	unclassifiable2 := makeID()

	resolver := &fakeResolver{
		entries: map[uuid.UUID]struct {
			class      string
			classified bool
		}{
			classified: {"algorithm", true},
			// unclassifiable1 and unclassifiable2 have no entry → classified=false
		},
	}

	in := Traversal{NodeIDs: []uuid.UUID{classified, unclassifiable1, unclassifiable2}, Depth: 1}
	got, err := FilterByActorClass(ctx, in, resolver, "algorithm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.NodeIDs) != 1 {
		t.Fatalf("got %d nodes, want 1; nodes=%v", len(got.NodeIDs), got.NodeIDs)
	}
	if got.NodeIDs[0] != classified {
		t.Errorf("got node %v, want %v", got.NodeIDs[0], classified)
	}
}

// TestFilterByActorClass_PropagatesResolverError verifies that an error from
// the resolver is surfaced immediately and an empty Traversal is returned.
func TestFilterByActorClass_PropagatesResolverError(t *testing.T) {
	ctx := context.Background()

	good := makeID()
	bad := makeID()

	resolver := &fakeResolver{
		entries: map[uuid.UUID]struct {
			class      string
			classified bool
		}{
			good: {"algorithm", true},
		},
		forceErrorOn: &bad,
	}

	in := Traversal{NodeIDs: []uuid.UUID{good, bad}, Depth: 2}
	_, err := FilterByActorClass(ctx, in, resolver, "algorithm")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestFilterByActorClass_PreservesDepth verifies that the Depth field from the
// input Traversal is carried through unchanged even when some nodes are filtered.
func TestFilterByActorClass_PreservesDepth(t *testing.T) {
	ctx := context.Background()

	keep := makeID()
	drop := makeID()

	resolver := &fakeResolver{
		entries: map[uuid.UUID]struct {
			class      string
			classified bool
		}{
			keep: {"algorithm", true},
			drop: {"clinician", true},
		},
	}

	const wantDepth = 4
	in := Traversal{NodeIDs: []uuid.UUID{keep, drop}, Depth: wantDepth}
	got, err := FilterByActorClass(ctx, in, resolver, "algorithm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Depth != wantDepth {
		t.Errorf("Depth: got %d, want %d", got.Depth, wantDepth)
	}
	if len(got.NodeIDs) != 1 {
		t.Errorf("NodeIDs length: got %d, want 1", len(got.NodeIDs))
	}
}
