// Package evidence_trace provides the pure graph types and traversal
// algorithms for the v2 substrate EvidenceTrace audit graph (Layer 2 doc
// §1.6, Recommendation 3 of Part 7).
//
// The graph itself is queryable in BOTH directions from day 1:
//   - Forward: given a recommendation node, what did it produce?
//   - Backward: given an outcome node, what reasoning produced it?
//
// This package is pure (no DB, no network) — it expresses the graph types
// and a BFS traversal that operates over an EdgeStore interface. The
// canonical implementation of EdgeStore lives in kb-20-patient-profile;
// in-memory implementations are used for tests.
package evidence_trace

import (
	"time"

	"github.com/google/uuid"
)

// EdgeKind enumerates the relationships between EvidenceTrace nodes.
//
// Semantics (Layer 2 doc §1.6):
//   - led_to        — A directly produced B (forward causation in reasoning)
//   - derived_from  — A used B as input (backward provenance)
//   - evidence_for  — A is an evidence node supporting decision B
//   - suppressed    — A would have fired but was suppressed by B (negative
//                     causation, retained for audit completeness)
type EdgeKind string

const (
	EdgeKindLedTo       EdgeKind = "led_to"
	EdgeKindDerivedFrom EdgeKind = "derived_from"
	EdgeKindEvidenceFor EdgeKind = "evidence_for"
	EdgeKindSuppressed  EdgeKind = "suppressed"
)

// IsValidEdgeKind reports whether s is a recognised edge kind.
func IsValidEdgeKind(s string) bool {
	switch EdgeKind(s) {
	case EdgeKindLedTo, EdgeKindDerivedFrom, EdgeKindEvidenceFor, EdgeKindSuppressed:
		return true
	}
	return false
}

// Edge is a directed relationship between two EvidenceTrace nodes.
type Edge struct {
	From      uuid.UUID `json:"from_node"`
	To        uuid.UUID `json:"to_node"`
	Kind      EdgeKind  `json:"edge_kind"`
	CreatedAt time.Time `json:"created_at"`
}
