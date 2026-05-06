package evidence_trace

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Traversal is the result of a graph traversal: the distinct node IDs
// reached, in BFS visit order (excluding the start node), plus the actual
// maximum depth visited (≤ caller's maxDepth cap).
type Traversal struct {
	NodeIDs []uuid.UUID `json:"node_ids"`
	Depth   int         `json:"depth"`
}

// ErrInvalidDepth is returned when a caller passes a non-positive maxDepth.
var ErrInvalidDepth = errors.New("evidence_trace: maxDepth must be > 0")

// TraceForward performs a BFS starting at startNode, following outgoing
// edges (from → to), capped at maxDepth hops. Returns the distinct node
// IDs reached (excluding startNode), in BFS order.
//
// kindFilter: nil/empty → follow edges of all kinds. Non-empty → only
// follow edges whose Kind matches one of the listed kinds.
//
// Cycle handling: a visited-set short-circuits revisits, so cycles in the
// graph cannot cause infinite loops. The depth cap is a defence in depth
// (heh) for runaway graph sizes.
func TraceForward(ctx context.Context, store EdgeStore, startNode uuid.UUID, maxDepth int, kindFilter []EdgeKind) (Traversal, error) {
	return bfs(ctx, store, startNode, maxDepth, kindFilter, true)
}

// TraceBackward is the symmetric reverse traversal: BFS following incoming
// edges (to → from). Same semantics as TraceForward otherwise.
func TraceBackward(ctx context.Context, store EdgeStore, startNode uuid.UUID, maxDepth int, kindFilter []EdgeKind) (Traversal, error) {
	return bfs(ctx, store, startNode, maxDepth, kindFilter, false)
}

// bfs is the shared BFS implementation. forward=true follows OutEdges
// (downstream); forward=false follows InEdges (upstream).
func bfs(ctx context.Context, store EdgeStore, startNode uuid.UUID, maxDepth int, kindFilter []EdgeKind, forward bool) (Traversal, error) {
	if maxDepth <= 0 {
		return Traversal{}, ErrInvalidDepth
	}
	if store == nil {
		return Traversal{}, errors.New("evidence_trace: nil store")
	}

	// visited holds every node we have either started from or queued. The
	// start node goes in immediately so we never re-enqueue it via a cycle.
	visited := map[uuid.UUID]struct{}{startNode: {}}
	type queueEntry struct {
		id    uuid.UUID
		depth int
	}
	queue := []queueEntry{{id: startNode, depth: 0}}
	out := make([]uuid.UUID, 0)
	maxReached := 0

	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]

		if head.depth >= maxDepth {
			// Don't expand further; cap reached.
			continue
		}

		// Fetch neighbours by either all kinds or each filter kind in turn.
		neighbours, err := neighboursOf(ctx, store, head.id, kindFilter, forward)
		if err != nil {
			return Traversal{}, err
		}

		for _, e := range neighbours {
			next := edgeTarget(e, forward)
			if _, ok := visited[next]; ok {
				continue
			}
			visited[next] = struct{}{}
			out = append(out, next)
			nextDepth := head.depth + 1
			if nextDepth > maxReached {
				maxReached = nextDepth
			}
			queue = append(queue, queueEntry{id: next, depth: nextDepth})
		}
	}

	return Traversal{NodeIDs: out, Depth: maxReached}, nil
}

// neighboursOf collects edges out of (or into, depending on forward) node id,
// filtered by kindFilter (or all kinds if kindFilter is empty).
//
// When kindFilter has multiple entries, we issue one OutEdges/InEdges call
// per kind and concat. The concrete EdgeStore is expected to enforce
// (From, To, Kind) uniqueness, so simple concat is correct.
func neighboursOf(ctx context.Context, store EdgeStore, id uuid.UUID, kindFilter []EdgeKind, forward bool) ([]Edge, error) {
	if len(kindFilter) == 0 {
		if forward {
			return store.OutEdges(ctx, id, EdgeKind(""))
		}
		return store.InEdges(ctx, id, EdgeKind(""))
	}
	var collected []Edge
	for _, k := range kindFilter {
		var (
			edges []Edge
			err   error
		)
		if forward {
			edges, err = store.OutEdges(ctx, id, k)
		} else {
			edges, err = store.InEdges(ctx, id, k)
		}
		if err != nil {
			return nil, err
		}
		collected = append(collected, edges...)
	}
	return collected, nil
}

// edgeTarget returns the "other end" of the edge depending on traversal
// direction.
func edgeTarget(e Edge, forward bool) uuid.UUID {
	if forward {
		return e.To
	}
	return e.From
}
