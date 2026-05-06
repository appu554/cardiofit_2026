package evidence_trace

// Wave 5.2 — high-level query API for Layer 3 + audit consumers.
//
// Three pure-Go functions over a NodeStore + EdgeStore pair:
//
//   - LineageOf(nodeID)            — backward traversal to all evidence inputs
//   - ConsequencesOf(nodeID)       — forward traversal to all triggered nodes
//   - ReasoningWindow(resident...) — regulator-audit window query rollup
//
// These build on the existing TraceForward / TraceBackward primitives but
// shape the result for Layer 3's consumption. They are pure: they take a
// store interface and operate without DB knowledge so unit tests can run
// against an in-memory store.

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// NodeStore is the read contract needed by the Wave 5.2 query helpers.
// kb-20's V2SubstrateStore satisfies this directly via its existing
// GetEvidenceTraceNode / per-resident list methods. Tests use an in-memory
// implementation.
type NodeStore interface {
	GetEvidenceTraceNode(ctx context.Context, id uuid.UUID) (*models.EvidenceTraceNode, error)
	// ListEvidenceTraceNodesByResident returns all nodes for residentRef
	// recorded in [from, to). Implementations should order by recorded_at
	// ascending so callers can stream the audit window in chronological
	// order.
	ListEvidenceTraceNodesByResident(ctx context.Context, residentRef uuid.UUID, from, to time.Time) ([]models.EvidenceTraceNode, error)
}

// Lineage is the structured backward-traversal result for one target node.
//
// Nodes is the BFS-ordered list of upstream node summaries (excluding the
// target itself). MaxDepth is the deepest level reached (≤ caller's cap).
type Lineage struct {
	TargetNodeID uuid.UUID            `json:"target_node_id"`
	Nodes        []NodeSummary        `json:"nodes"`
	MaxDepth     int                  `json:"max_depth"`
}

// Consequences is the structured forward-traversal result.
type Consequences struct {
	SourceNodeID uuid.UUID     `json:"source_node_id"`
	Nodes        []NodeSummary `json:"nodes"`
	MaxDepth     int           `json:"max_depth"`
}

// NodeSummary is the slimmed-down node shape returned by Lineage /
// Consequences. Callers wanting the full node can re-fetch by ID.
type NodeSummary struct {
	ID              uuid.UUID `json:"id"`
	StateMachine    string    `json:"state_machine"`
	StateChangeType string    `json:"state_change_type"`
	RecordedAt      time.Time `json:"recorded_at"`
	OccurredAt      time.Time `json:"occurred_at"`
}

// ReasoningSummaryWindow is the regulator-audit rollup for a resident over
// a time window. Shape is intentionally JSON-friendly for ACQSC submission.
type ReasoningSummaryWindow struct {
	ResidentRef                       uuid.UUID                `json:"resident_ref"`
	From                              time.Time                `json:"from"`
	To                                time.Time                `json:"to"`
	TotalNodes                        int                      `json:"total_nodes"`
	NodesByStateMachine               map[string]int           `json:"nodes_by_state_machine"`
	RecommendationCount               int                      `json:"recommendation_count"`
	DecisionCount                     int                      `json:"decision_count"`
	AverageEvidencePerRecommendation  float64                  `json:"average_evidence_per_recommendation"`
	Nodes                             []NodeSummary            `json:"nodes"`
}

// ErrNilNodeStore is returned when a caller passes a nil NodeStore.
var ErrNilNodeStore = errors.New("evidence_trace: nil node store")

// Default depth cap used when the caller passes <=0 to Lineage / Consequences.
const defaultQueryDepth = 10

// LineageOf returns the backward traversal from nodeID — every node
// upstream that fed into this decision. Uses derived_from + evidence_for
// edge kinds; led_to is the forward-only kind so it's filtered out of the
// upstream walk.
//
// maxDepth: <=0 → defaultQueryDepth (10). Caps at the caller's value
// otherwise.
func LineageOf(ctx context.Context, nodeID uuid.UUID, nodeStore NodeStore, edgeStore EdgeStore, maxDepth int) (*Lineage, error) {
	if nodeStore == nil {
		return nil, ErrNilNodeStore
	}
	if edgeStore == nil {
		return nil, errors.New("evidence_trace: nil edge store")
	}
	if maxDepth <= 0 {
		maxDepth = defaultQueryDepth
	}
	t, err := TraceBackward(ctx, edgeStore, nodeID, maxDepth, []EdgeKind{EdgeKindDerivedFrom, EdgeKindEvidenceFor})
	if err != nil {
		return nil, err
	}
	summaries, err := loadSummaries(ctx, nodeStore, t.NodeIDs)
	if err != nil {
		return nil, err
	}
	return &Lineage{TargetNodeID: nodeID, Nodes: summaries, MaxDepth: t.Depth}, nil
}

// ConsequencesOf returns the forward traversal from nodeID — every node
// downstream that this triggered. Follows led_to edges only.
func ConsequencesOf(ctx context.Context, nodeID uuid.UUID, nodeStore NodeStore, edgeStore EdgeStore, maxDepth int) (*Consequences, error) {
	if nodeStore == nil {
		return nil, ErrNilNodeStore
	}
	if edgeStore == nil {
		return nil, errors.New("evidence_trace: nil edge store")
	}
	if maxDepth <= 0 {
		maxDepth = defaultQueryDepth
	}
	t, err := TraceForward(ctx, edgeStore, nodeID, maxDepth, []EdgeKind{EdgeKindLedTo})
	if err != nil {
		return nil, err
	}
	summaries, err := loadSummaries(ctx, nodeStore, t.NodeIDs)
	if err != nil {
		return nil, err
	}
	return &Consequences{SourceNodeID: nodeID, Nodes: summaries, MaxDepth: t.Depth}, nil
}

// ReasoningWindow returns the per-resident rollup of EvidenceTrace nodes
// recorded in [from, to). Suitable for ACQSC regulator submission.
//
// Aggregates: total node count, count by state_machine, recommendation
// count (a synonym for state_machine='Recommendation'), decision count
// (state_change_type containing "decided" / "accepted" / "rejected" — a
// permissive substring match used until Layer 3 codifies a closed set),
// average evidence-input count per Recommendation node.
func ReasoningWindow(ctx context.Context, residentRef uuid.UUID, from, to time.Time, nodeStore NodeStore) (*ReasoningSummaryWindow, error) {
	if nodeStore == nil {
		return nil, ErrNilNodeStore
	}
	if !to.After(from) {
		return nil, errors.New("evidence_trace: ReasoningWindow: to must be after from")
	}
	nodes, err := nodeStore.ListEvidenceTraceNodesByResident(ctx, residentRef, from, to)
	if err != nil {
		return nil, err
	}
	out := &ReasoningSummaryWindow{
		ResidentRef:         residentRef,
		From:                from,
		To:                  to,
		TotalNodes:          len(nodes),
		NodesByStateMachine: map[string]int{},
	}
	var (
		recCount      int
		evidenceTotal int
		decCount      int
	)
	for _, n := range nodes {
		out.NodesByStateMachine[n.StateMachine]++
		if n.StateMachine == models.EvidenceTraceStateMachineRecommendation {
			recCount++
			evidenceTotal += len(n.Inputs)
		}
		if isDecisionStateChange(n.StateChangeType) {
			decCount++
		}
		out.Nodes = append(out.Nodes, summariseNode(n))
	}
	out.RecommendationCount = recCount
	out.DecisionCount = decCount
	if recCount > 0 {
		out.AverageEvidencePerRecommendation = float64(evidenceTotal) / float64(recCount)
	}
	// Stabilise the order for tests / consumers expecting chronological output.
	sort.SliceStable(out.Nodes, func(i, j int) bool {
		return out.Nodes[i].RecordedAt.Before(out.Nodes[j].RecordedAt)
	})
	return out, nil
}

// loadSummaries fetches each node by ID and returns its summary. A missing
// node is skipped (logged-but-shown-not-implemented at this layer; a
// regulator query that hits a deleted ancestor should not fail outright).
func loadSummaries(ctx context.Context, store NodeStore, ids []uuid.UUID) ([]NodeSummary, error) {
	out := make([]NodeSummary, 0, len(ids))
	for _, id := range ids {
		n, err := store.GetEvidenceTraceNode(ctx, id)
		if err != nil {
			// Skip-not-fail: an upstream pointer that no longer resolves
			// shouldn't take down the entire lineage query. The traversal
			// still preserved the ID order so consumers know what was
			// missing if they cross-reference.
			continue
		}
		out = append(out, summariseNode(*n))
	}
	return out, nil
}

func summariseNode(n models.EvidenceTraceNode) NodeSummary {
	return NodeSummary{
		ID:              n.ID,
		StateMachine:    n.StateMachine,
		StateChangeType: n.StateChangeType,
		RecordedAt:      n.RecordedAt,
		OccurredAt:      n.OccurredAt,
	}
}

// isDecisionStateChange is a permissive heuristic for "this transition
// represents a decision worth counting in the regulator rollup". Layer 3
// will eventually publish a closed set; until then we substring-match on
// commonly-used decision verbs.
func isDecisionStateChange(s string) bool {
	if s == "" {
		return false
	}
	// Lowercase substring match; cheap and stable.
	low := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		low[i] = c
	}
	str := string(low)
	for _, kw := range []string{"decided", "accepted", "rejected", "approved", "declined"} {
		if containsSub(str, kw) {
			return true
		}
	}
	return false
}

// containsSub is a small substring check — avoiding a strings import since
// this package doesn't otherwise need it.
func containsSub(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
