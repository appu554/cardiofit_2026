package evidence_trace

import (
	"context"

	"github.com/google/uuid"
)

// EdgeStore is the contract for reading and writing EvidenceTrace edges.
// Production: kb-20-patient-profile's V2SubstrateStore. Tests: an in-memory
// implementation backing the BFS traversal.
//
// OutEdges/InEdges accept a kind filter:
//   - empty Kind ("")  → return edges of any kind
//   - non-empty Kind   → return edges of exactly that kind
//
// Implementations MUST NOT return duplicate (From, To, Kind) triples; the
// canonical schema enforces uniqueness via the primary key on
// evidence_trace_edges. The traversal relies on this for cycle detection
// at the visited-set level.
type EdgeStore interface {
	InsertEdge(ctx context.Context, e Edge) error
	OutEdges(ctx context.Context, from uuid.UUID, kind EdgeKind) ([]Edge, error)
	InEdges(ctx context.Context, to uuid.UUID, kind EdgeKind) ([]Edge, error)
}
