// Package loadgen synthesises an EvidenceTrace graph for performance
// testing. Wave 5.4 — execution and SLO lock-in deferred to V1; this
// package ships the synthesizer, an in-memory benchmark harness, and an
// invocation README so an operator can run the synthesis against a real
// PostgreSQL store at any point.
//
// Default profile (from the Wave 5.4 plan task):
//
//   - 200 residents
//   - 6 months of activity each
//   - ~5 nodes/day per resident
//   - → ~180,000 nodes
//   - → ~500,000 edges
package loadgen

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// Profile parameters for a synthesis run.
type Profile struct {
	ResidentCount    int
	DaysPerResident  int
	NodesPerDay      int
	Seed             int64
}

// DefaultProfile is the Wave 5.4 plan target: ~180k nodes / ~500k edges.
func DefaultProfile() Profile {
	return Profile{
		ResidentCount:   200,
		DaysPerResident: 180,
		NodesPerDay:     5,
		Seed:            42,
	}
}

// SmallProfile is suitable for in-process benchmarks: 10 residents × 30
// days × 5 nodes/day = 1,500 nodes, ~4,000 edges.
func SmallProfile() Profile {
	return Profile{
		ResidentCount:   10,
		DaysPerResident: 30,
		NodesPerDay:     5,
		Seed:            7,
	}
}

// NodeSink is the write contract for the synthesizer. Any sink that can
// persist a node and an edge satisfies this — kb-20's V2SubstrateStore
// adapted via storage.EvidenceTraceEdgeAdapter, an in-memory store, etc.
type NodeSink interface {
	UpsertEvidenceTraceNode(ctx context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error)
	InsertEvidenceTraceEdge(ctx context.Context, e evidence_trace.Edge) error
}

// Stats summarises a synthesis run.
type Stats struct {
	Nodes       int
	Edges       int
	Residents   int
	StartedAt   time.Time
	CompletedAt time.Time
}

// Synthesize builds a graph against sink per profile p. Idempotent only at
// the per-node primary-key level — running twice with the same seed will
// reuse node IDs, but edge inserts may collide on the (from,to,kind)
// primary key (which the canonical store treats as idempotent).
//
// Per-resident shape (per day):
//
//   - 3 observation-seed Monitoring nodes
//   - 1 Recommendation node, with derived_from edges to the day's obs nodes
//   - 1 outcome ClinicalState node, with led_to edge from the Recommendation
//
// → 5 nodes + 4 edges per resident-day.
func Synthesize(ctx context.Context, sink NodeSink, p Profile) (*Stats, error) {
	if sink == nil {
		return nil, fmt.Errorf("loadgen.Synthesize: nil sink")
	}
	rng := rand.New(rand.NewSource(p.Seed))
	stats := &Stats{Residents: p.ResidentCount, StartedAt: time.Now().UTC()}
	startDate := time.Date(2025, 11, 1, 9, 0, 0, 0, time.UTC)

	for r := 0; r < p.ResidentCount; r++ {
		residentRef := newSeededUUID(rng)
		for d := 0; d < p.DaysPerResident; d++ {
			day := startDate.AddDate(0, 0, d)
			obsIDs := make([]uuid.UUID, 0, 3)
			// Three observation-seed Monitoring nodes per day.
			for k := 0; k < 3; k++ {
				obs := makeNode(rng, models.EvidenceTraceStateMachineMonitoring,
					"observation_recorded", &residentRef, day.Add(time.Duration(k)*time.Hour), 0)
				if _, err := sink.UpsertEvidenceTraceNode(ctx, obs); err != nil {
					return stats, fmt.Errorf("upsert obs r=%d d=%d k=%d: %w", r, d, k, err)
				}
				stats.Nodes++
				obsIDs = append(obsIDs, obs.ID)
			}
			// One Recommendation node, with derived_from edges from the day's obs.
			rec := makeNode(rng, models.EvidenceTraceStateMachineRecommendation,
				"draft -> submitted", &residentRef, day.Add(4*time.Hour), 3)
			if _, err := sink.UpsertEvidenceTraceNode(ctx, rec); err != nil {
				return stats, fmt.Errorf("upsert rec: %w", err)
			}
			stats.Nodes++
			for _, oid := range obsIDs {
				e := evidence_trace.Edge{From: oid, To: rec.ID, Kind: evidence_trace.EdgeKindDerivedFrom}
				if err := sink.InsertEvidenceTraceEdge(ctx, e); err != nil {
					return stats, fmt.Errorf("insert obs→rec edge: %w", err)
				}
				stats.Edges++
			}
			// One outcome ClinicalState node + led_to edge from the rec.
			outcome := makeNode(rng, models.EvidenceTraceStateMachineClinicalState,
				"outcome_recorded", &residentRef, day.Add(5*time.Hour), 1)
			if _, err := sink.UpsertEvidenceTraceNode(ctx, outcome); err != nil {
				return stats, fmt.Errorf("upsert outcome: %w", err)
			}
			stats.Nodes++
			ledTo := evidence_trace.Edge{From: rec.ID, To: outcome.ID, Kind: evidence_trace.EdgeKindLedTo}
			if err := sink.InsertEvidenceTraceEdge(ctx, ledTo); err != nil {
				return stats, fmt.Errorf("insert rec→outcome edge: %w", err)
			}
			stats.Edges++
		}
	}
	stats.CompletedAt = time.Now().UTC()
	return stats, nil
}

func makeNode(rng *rand.Rand, sm, sct string, residentRef *uuid.UUID, when time.Time, evidenceInputs int) models.EvidenceTraceNode {
	n := models.EvidenceTraceNode{
		ID:              newSeededUUID(rng),
		StateMachine:    sm,
		StateChangeType: sct,
		RecordedAt:      when,
		OccurredAt:      when,
		ResidentRef:     residentRef,
	}
	for i := 0; i < evidenceInputs; i++ {
		n.Inputs = append(n.Inputs, models.TraceInput{
			InputType:      models.TraceInputTypeObservation,
			InputRef:       newSeededUUID(rng),
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		})
	}
	return n
}

// newSeededUUID builds a v4-shaped UUID from the deterministic RNG so
// runs are reproducible.
func newSeededUUID(rng *rand.Rand) uuid.UUID {
	var b [16]byte
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	// Set the version (4) and variant bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	u, _ := uuid.FromBytes(b[:])
	return u
}
