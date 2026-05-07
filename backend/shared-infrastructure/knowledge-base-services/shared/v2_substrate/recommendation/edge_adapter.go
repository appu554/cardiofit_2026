package recommendation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// NodeWriter is the persistence boundary for EvidenceTraceNodes. The
// production V2SubstrateStore in kb-20-patient-profile satisfies this
// interface (see kb-20-patient-profile/internal/storage/v2_substrate_store.go
// — UpsertEvidenceTraceNode). Tests use a fake.
type NodeWriter interface {
	UpsertEvidenceTraceNode(ctx context.Context,
		n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error)
}

// EvidenceTraceAdapter satisfies recommendation.EdgeStore by translating
// the recommendation lifecycle's lightweight EvidenceEdge into a
// metadata-rich models.EvidenceTraceNode that the substrate persists.
//
// Output convention: every emitted node carries one TraceOutput referencing
// the recommendation_id, so consumers can rejoin nodes by recommendation
// without scanning by state_machine alone.
//
// The adapter does NOT currently insert chain edges between successive
// transition nodes (Edge{From: prev, To: new, Kind: EdgeKindLedTo}).
// Chain edges require a recommendation_id-indexed lookup of the prior
// node, which the schema doesn't support efficiently today. Chronology
// is recoverable from RecordedAt; chain edges are deferred to a follow-up.
type EvidenceTraceAdapter struct {
	writer NodeWriter
	now    func() time.Time
}

// NewEvidenceTraceAdapter constructs an EvidenceTraceAdapter that writes
// EvidenceTraceNodes via writer. RecordedAt is sourced from time.Now().UTC()
// at emit time.
func NewEvidenceTraceAdapter(writer NodeWriter) *EvidenceTraceAdapter {
	return &EvidenceTraceAdapter{
		writer: writer,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// EmitEdge implements recommendation.EdgeStore.
func (a *EvidenceTraceAdapter) EmitEdge(ctx context.Context, e EvidenceEdge) error {
	// Bind a stable copy of the resident UUID for the pointer field on the
	// node. (Taking &e.ResidentID directly would point into the caller's
	// stack frame; safer to bind to a local.)
	resident := e.ResidentID

	// Bind ActorID similarly for TraceActor.PersonRef. ActorClass is captured
	// in ReasoningSummary.Text only when present — TraceActor has no field
	// for it, and it's already preserved on the EvidenceEdge for any future
	// adapter that wants to write it elsewhere.
	person := e.ActorID

	node := models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: e.FromState + " -> " + e.ToState,
		RecordedAt:      a.now(),
		OccurredAt:      e.OccurredAt,
		Actor: models.TraceActor{
			PersonRef: &person,
		},
		ResidentRef: &resident,
	}

	if e.ReasoningSummary != "" {
		node.ReasoningSummary = &models.ReasoningSummary{
			Text: e.ReasoningSummary,
		}
	}

	if len(e.InputRefs) > 0 {
		node.Inputs = make([]models.TraceInput, 0, len(e.InputRefs))
		for _, ref := range e.InputRefs {
			node.Inputs = append(node.Inputs, models.TraceInput{
				InputType:      models.TraceInputTypeOther,
				InputRef:       ref,
				RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
			})
		}
	}

	// Always emit the recommendation_id as an output so downstream queries
	// can recover the chain of transitions per recommendation.
	node.Outputs = []models.TraceOutput{{
		OutputType: "Recommendation",
		OutputRef:  e.RecommendationID,
	}}

	_, err := a.writer.UpsertEvidenceTraceNode(ctx, node)
	return err
}
