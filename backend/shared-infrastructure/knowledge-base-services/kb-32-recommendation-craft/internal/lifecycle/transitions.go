// Package lifecycle provides kb-32's craft-engine gate for the
// detected → drafted recommendation transition.
//
// # Permissions middleware deferral
//
// Production PDP permissions wrapping for the HTTP endpoints that trigger
// lifecycle advances is deferred to Phase 2b. The gate itself enforces the
// appropriateness contract independently of transport-layer auth; the HTTP
// layer in internal/api documents the same deferral.
//
// # CraftEngineGate wiring
//
// The shared/v2_substrate/recommendation.Lifecycle accepts an optional
// CraftEngineGate (see lifecycle.go in that package). When configured, the
// gate's AdvanceDetectedToDrafted method is called before the substrate
// records the detected → drafted state change. If the gate returns
// ErrTransitionHeld, the transition is aborted and the recommendation remains
// in the detected state.
package lifecycle

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/appropriateness"
)

// ErrTransitionHeld is returned by AdvanceDetectedToDrafted when the
// appropriateness gate holds the recommendation in detected state.
// The caller (shared substrate Lifecycle) MUST NOT advance the state
// when this sentinel is returned.
var ErrTransitionHeld = errors.New("lifecycle: craft engine gate holds transition in detected state")

// AppropriatenessSource is the port through which the gate retrieves a
// scored appropriateness.Assessment for a given recommendation.
//
// The DefaultScorer (this package) returns a passing Assessment with all
// dimensions at 3. Real multi-dimension scoring is Phase 2b's responsibility.
type AppropriatenessSource interface {
	AssessFor(ctx context.Context, recID uuid.UUID) (appropriateness.Assessment, error)
}

// Gate implements the craft-engine appropriateness check for the
// detected → drafted transition. Wire it into the shared substrate's
// Lifecycle via Lifecycle.SetCraftEngineGate.
//
// Construct with NewGate; the zero value is not usable.
type Gate struct {
	src AppropriatenessSource
}

// NewGate constructs a Gate backed by the given AppropriatenessSource.
func NewGate(src AppropriatenessSource) *Gate {
	return &Gate{src: src}
}

// AdvanceDetectedToDrafted runs the appropriateness gate for the given
// recommendation ID. It returns nil when the assessment passes (all five
// dimensions > HoldThreshold) and ErrTransitionHeld when the assessment
// holds the recommendation.
//
// This method satisfies the CraftEngineGate interface defined in
// shared/v2_substrate/recommendation/lifecycle.go.
func (g *Gate) AdvanceDetectedToDrafted(ctx context.Context, recID uuid.UUID) error {
	assessment, err := g.src.AssessFor(ctx, recID)
	if err != nil {
		return err
	}

	if err := appropriateness.Check(assessment); err != nil {
		return ErrTransitionHeld
	}
	return nil
}

// DefaultScorer is an AppropriatenessSource that returns a passing
// Assessment with all five dimensions scored at 3 (above HoldThreshold=2).
//
// This is the Phase 2a default scorer. Real multi-dimension scoring
// (incorporating ClinicalSnapshot, evidence quality, and care-plan alignment)
// is deferred to Phase 2b. The default scorer exists so that the lifecycle
// gate is fully wired without blocking recommendation flow during Phase 2a
// shadow deployment.
//
// IMPORTANT: Do not use DefaultScorer in production without understanding
// that it always passes the gate. Replace with a real scorer in Phase 2b.
type DefaultScorer struct{}

// AssessFor returns a passing Assessment with all dimensions at 3.
// recID is accepted for interface compatibility but not used.
func (DefaultScorer) AssessFor(_ context.Context, _ uuid.UUID) (appropriateness.Assessment, error) {
	return appropriateness.Assessment{
		ClinicalWarrant:        3,
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}, nil
}
