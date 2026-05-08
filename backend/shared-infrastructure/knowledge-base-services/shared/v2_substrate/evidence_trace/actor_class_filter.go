// Package evidence_trace — see graph.go for package-level documentation.
//
// Phase 1a Task 6 scope note:
// This file lands a post-traversal ActorClass filter against an injectable
// resolver interface. The plan's original sketch (a SQL WhereActorClass on a
// hypothetical Query builder) was not implementable as written: the Edge
// struct has no actor_class column, and there is no Query builder type — the
// query layer is BFS-based (TraceForward/TraceBackward).
//
// Adding actor_class to the edges table or implementing a cross-package join
// to the recommendations table is deferred. This filter is sufficient for the
// Phase 1a stated goal: "a Fair Work / AHPRA contestation can pull every
// algorithmic decision feeding a KPI" — by walking the lineage with
// TraceBackward and then filtering by ActorClass.
package evidence_trace

import (
	"context"

	"github.com/google/uuid"
)

// ActorClassResolver looks up the ActorClass associated with an EvidenceTrace node.
// Implementations resolve via the originating entity (e.g. a Recommendation node
// resolves to the Recommendation's recorded ActorClass, per Plan 0.1).
//
// Returns ("", false, nil) when the node has no associated actor class (e.g. pure
// substrate facts derived from observations, not authored decisions).
type ActorClassResolver interface {
	ActorClassFor(ctx context.Context, nodeID uuid.UUID) (string, bool, error)
}

// FilterByActorClass keeps only those traversal nodes whose resolved ActorClass
// matches actorClass. Nodes the resolver cannot classify are dropped.
//
// Use case: a Fair Work / AHPRA contestation walks an EvidenceTrace lineage and
// then filters down to algorithmic decisions only ("ActorClassAlgorithm") to
// produce the audit packet.
func FilterByActorClass(ctx context.Context, t Traversal, resolver ActorClassResolver, actorClass string) (Traversal, error) {
	out := Traversal{Depth: t.Depth}
	for _, id := range t.NodeIDs {
		got, ok, err := resolver.ActorClassFor(ctx, id)
		if err != nil {
			return Traversal{}, err
		}
		if !ok {
			continue
		}
		if got == actorClass {
			out.NodeIDs = append(out.NodeIDs, id)
		}
	}
	return out, nil
}
