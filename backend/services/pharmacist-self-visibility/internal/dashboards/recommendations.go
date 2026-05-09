// Package dashboards provides the pharmacist self-visibility dashboard surfaces.
//
// VisibilityClass: PDP (Pharmacist-Default-Private) — this file implements
// Surface 2: My Recommendations. Employer access requires explicit pharmacist
// consent; see the Phase 1a permission middleware.
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// Lifecycle state string constants for Plan 0.1 Recommendation entity.
// These values are intentionally redefined here rather than imported from
// github.com/cardiofit/shared to avoid a cross-module dependency
// (pharmacist-self-visibility is its own Go module).
//
// Callers providing RecRow.State values MUST use these constants or the
// canonical strings they represent. IsValidLifecycleState validates at the
// boundary; the RecSource implementation is responsible for canonicalisation
// before rows reach MyRecommendations.For.
const (
	// LifecycleDetected indicates the KB surface has flagged a clinical signal
	// but the pharmacist has not yet drafted a formal recommendation.
	LifecycleDetected = "detected"
	// LifecycleDrafted indicates the pharmacist has begun drafting.
	LifecycleDrafted = "drafted"
	// LifecycleSubmitted indicates the draft has been submitted to the GP.
	LifecycleSubmitted = "submitted"
	// LifecycleViewed indicates the GP has opened the recommendation.
	LifecycleViewed = "viewed"
	// LifecycleDeferred indicates the GP or pharmacist has deferred action.
	LifecycleDeferred = "deferred"
	// LifecycleDecided indicates the GP has made a clinical decision.
	LifecycleDecided = "decided"
	// LifecycleImplemented indicates the decision has been actioned.
	LifecycleImplemented = "implemented"
	// LifecycleMonitoringActive indicates an active monitoring plan is running.
	LifecycleMonitoringActive = "monitoring-active"
	// LifecycleOutcomeRecorded indicates the outcome has been documented.
	LifecycleOutcomeRecorded = "outcome-recorded"
	// LifecycleClosed is the terminal state.
	LifecycleClosed = "closed"
	// LifecycleRejected indicates the GP rejected the recommendation.
	// This state is surfaced to the pharmacist with Framing="learning_opportunity"
	// to avoid framing clinical disagreement as personal failure.
	LifecycleRejected = "rejected"
)

// IsValidLifecycleState reports whether s is a recognised Plan 0.1
// recommendation lifecycle state value. It accepts both the 10 canonical
// states from the models.enums.go RecommendationState* constants and
// "rejected", which is the surface-visible framing state used by this
// dashboard layer.
//
// The source interface is responsible for canonicalisation; this function is
// a boundary guard called by For() before building RecommendationCards.
func IsValidLifecycleState(s string) bool {
	switch s {
	case LifecycleDetected, LifecycleDrafted, LifecycleSubmitted,
		LifecycleViewed, LifecycleDeferred, LifecycleDecided,
		LifecycleImplemented, LifecycleMonitoringActive,
		LifecycleOutcomeRecorded, LifecycleClosed, LifecycleRejected:
		return true
	}
	return false
}

// RecommendationCard is the pharmacist's read-only view of a single
// recommendation in their lifecycle history.
//
// VisibilityClass: PDP (Pharmacist-Default-Private) — the employer MUST NOT
// receive this data without a separately recorded explicit consent event.
//
// State values MUST be one of the Plan 0.1 lifecycle constants (see
// IsValidLifecycleState). The Framing field carries UI presentation intent:
// when State is "rejected" the Framing is set to "learning_opportunity" so
// the dashboard never surfaces clinical disagreement as a performance failure.
type RecommendationCard struct {
	// RecommendationID is the unique identifier of the recommendation entity.
	RecommendationID uuid.UUID
	// State is the current Plan 0.1 lifecycle state. Callers should treat this
	// as a display hint; business logic MUST use the source of truth in the
	// recommendation store.
	State string
	// Framing carries a UI presentation hint. "learning_opportunity" is set
	// when State == "rejected"; empty string for all other states.
	Framing string
	// RejectionReason is the GP's stated reason for rejection, if available.
	// Only populated when State == "rejected".
	RejectionReason string
}

// RecRow mirrors the row type used by RecSource implementations.
// It is the minimal projection the RecSource.ListByAuthor contract returns.
// Callers MUST use Plan 0.1 lifecycle state values (see IsValidLifecycleState).
//
// VisibilityClass: PDP — exported so that store/postgres implementations outside
// this package can satisfy the RecSource interface without reflection tricks.
type RecRow struct {
	// ID is the unique identifier of the recommendation entity.
	ID uuid.UUID
	// AuthorID is the pharmacist UUID that authored this recommendation.
	AuthorID uuid.UUID
	// State is the current Plan 0.1 lifecycle state string.
	State string
	// RejectionReason is the GP's stated reason for rejection, if available.
	// This field is populated only when State == "rejected". The Plan 0.1
	// migration 023 schema does not carry a rejection_reason column; this field
	// is reserved for a future schema extension or application-layer annotation.
	RejectionReason string
}

// RecSource is the data-access interface that backs MyRecommendations.
// Implementations must respect context cancellation and must return rows whose
// State values satisfy IsValidLifecycleState; canonicalisation is the
// implementer's responsibility.
//
// In a full deployment the production source would use a
// database-backed implementation of this interface.
type RecSource interface {
	// ListByAuthor returns all recommendation rows authored by the given
	// pharmacist UUID. The source is expected to apply Phase 1a permission
	// middleware so that only PDP-consented rows are returned.
	ListByAuthor(ctx context.Context, author uuid.UUID) ([]RecRow, error)
}

// MyRecommendations surfaces a pharmacist's own recommendation lifecycle view.
// Construct with NewMyRecommendations; call For to obtain the card list for a
// given pharmacist.
//
// VisibilityClass: PDP (Pharmacist-Default-Private)
type MyRecommendations struct{ src RecSource }

// NewMyRecommendations constructs a MyRecommendations backed by the given
// RecSource.
func NewMyRecommendations(src RecSource) *MyRecommendations {
	return &MyRecommendations{src: src}
}

// For returns the RecommendationCard slice for the given author (pharmacist
// UUID). The source is expected to pre-filter by author; For applies a
// secondary filter as a defensive guard.
//
// Rejected recommendations carry Framing="learning_opportunity" so that the
// dashboard UI never frames clinical disagreement as pharmacist failure.
//
// A defensive context cancellation check is applied before source access. If
// the context is already cancelled, ctx.Err() is returned immediately.
//
// When the source returns no rows, For returns a non-nil empty slice so
// callers can distinguish "no recommendations" from an uninitialised result.
func (m *MyRecommendations) For(ctx context.Context, author uuid.UUID) ([]RecommendationCard, error) {
	// Defensive context check: return early if context is already cancelled
	// so that callers receive an explicit signal rather than a silently empty
	// or partial result.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	rows, err := m.src.ListByAuthor(ctx, author)
	if err != nil {
		return nil, err
	}

	out := make([]RecommendationCard, 0, len(rows))
	for _, r := range rows {
		c := RecommendationCard{
			RecommendationID: r.ID,
			State:            r.State,
			RejectionReason:  r.RejectionReason,
		}
		if r.State == LifecycleRejected {
			c.Framing = "learning_opportunity"
		}
		out = append(out, c)
	}
	return out, nil
}
