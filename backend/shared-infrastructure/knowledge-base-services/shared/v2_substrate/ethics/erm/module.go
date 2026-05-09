// Package erm implements the Ethical Reasoning Module (ERM) scaffold per
// Guidelines §4.1–4.2. The ERM identifies decisions requiring ethical
// attention and applies established review patterns through registered
// Reasoner implementations.
//
// # Non-autonomy commitment (Guidelines §4.6)
//
// ERM is NOT autonomous. It surfaces concerns and routes decisions to
// appropriate review patterns, but human judgment remains the final ethical
// authority for any non-routine case. Callers MUST honour OutcomeHold and
// OutcomeReject by escalating to a human reviewer rather than proceeding
// automatically.
//
// VisibilityClass: AD
package erm

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// DecisionType identifies the category of an algorithmic decision being
// reviewed by the ERM.
type DecisionType string

const (
	// DecisionTypeRecommendationDraft is a clinical recommendation produced by
	// the craft engine before presentation to a clinician.
	DecisionTypeRecommendationDraft DecisionType = "recommendation_draft"

	// DecisionTypeVisibilityAggregate is an aggregated visibility result from
	// the PFA permission middleware before release to an employer query.
	DecisionTypeVisibilityAggregate DecisionType = "visibility_aggregate"

	// DecisionTypeAuthorisation is a system-level authorisation decision
	// controlling resource access.
	DecisionTypeAuthorisation DecisionType = "authorisation"
)

// Outcome is the ERM verdict on a decision point. Callers MUST NOT bypass
// OutcomeHold or OutcomeReject without explicit human sign-off.
type Outcome string

const (
	// OutcomeApprove indicates no ethical concerns; the decision may proceed.
	OutcomeApprove Outcome = "approve"

	// OutcomeApproveWithMonitoring indicates the decision may proceed but
	// requires active monitoring and review at the next cycle.
	OutcomeApproveWithMonitoring Outcome = "approve_with_monitoring"

	// OutcomeHold blocks the decision pending human review. The caller MUST
	// escalate to a designated human reviewer and MUST NOT proceed
	// automatically.
	OutcomeHold Outcome = "hold"

	// OutcomeReject indicates the decision violates ethical constraints and
	// MUST NOT proceed. A human reviewer must be notified.
	OutcomeReject Outcome = "reject"
)

// IsValidOutcome returns true when s is one of the four canonical Outcome
// string values.
func IsValidOutcome(s string) bool {
	switch Outcome(s) {
	case OutcomeApprove, OutcomeApproveWithMonitoring, OutcomeHold, OutcomeReject:
		return true
	default:
		return false
	}
}

// Concern describes a specific ethical concern raised by a Reasoner.
type Concern struct {
	Principle    string // "P1".."P7" per Ethical Architecture Guidelines §1
	ConcernLevel int    // 1 (minor) .. 5 (critical)
	Reasoning    string
}

// DecisionPoint carries all context a Reasoner needs to evaluate an
// algorithmic decision.
type DecisionPoint struct {
	// DecisionID uniquely identifies this decision event. If zero, the Module
	// assigns a new UUID before routing.
	DecisionID uuid.UUID

	// Component is the name of the service or sub-system producing the
	// decision (e.g. "kb-23", "craft-engine").
	Component string

	// DecisionType determines which Reasoner handles this point.
	DecisionType DecisionType

	// Inputs are the raw inputs to the algorithmic decision. The registered
	// Reasoner is responsible for type-asserting to the expected struct.
	Inputs interface{}

	// ProposedOutput is the draft decision output. The Reasoner evaluates
	// whether it is ethically acceptable.
	ProposedOutput interface{}
}

// Reasoner is the interface every concrete ERM reasoner must implement.
// A Reasoner evaluates a DecisionPoint and returns an Outcome together with
// any Concerns raised. The Reasoner MUST NOT mutate dp.
type Reasoner interface {
	Review(ctx context.Context, dp DecisionPoint) (Outcome, []Concern)
}

// ReasonerFunc is a function adapter for the Reasoner interface.
type ReasonerFunc func(ctx context.Context, dp DecisionPoint) (Outcome, []Concern)

// Review implements Reasoner.
func (f ReasonerFunc) Review(ctx context.Context, dp DecisionPoint) (Outcome, []Concern) {
	return f(ctx, dp)
}

// ErrUnknownDecisionType is returned by Module.Review when no Reasoner has
// been registered for the requested DecisionType.
var ErrUnknownDecisionType = errors.New("erm: no reasoner registered for this decision type")

// Module routes DecisionPoints to registered Reasoners and collects their
// verdicts. It is safe to call Register and Review concurrently only if the
// caller provides its own synchronisation; Module itself is not goroutine-safe.
//
// Human escalation: when Review returns OutcomeHold or OutcomeReject the
// caller is responsible for notifying a human reviewer. No automatic
// progression past these outcomes is permitted (Guidelines §4.6).
type Module struct {
	reasoners map[DecisionType]Reasoner
}

// NewModule returns a Module with an empty reasoner registry.
func NewModule() *Module {
	return &Module{reasoners: map[DecisionType]Reasoner{}}
}

// Register associates r with dt. Calling Register with the same dt twice
// replaces the previous registration — no duplication occurs.
func (m *Module) Register(dt DecisionType, r Reasoner) {
	m.reasoners[dt] = r
}

// Review routes dp to the registered Reasoner for dp.DecisionType.
// If dp.DecisionID is the zero UUID, Review assigns a fresh one before
// routing. Returns ErrUnknownDecisionType when no Reasoner is registered.
//
// OutcomeHold and OutcomeReject REQUIRE human escalation by the caller.
func (m *Module) Review(ctx context.Context, dp DecisionPoint) (Outcome, []Concern, error) {
	r, ok := m.reasoners[dp.DecisionType]
	if !ok {
		return "", nil, ErrUnknownDecisionType
	}
	if dp.DecisionID == (uuid.UUID{}) {
		dp.DecisionID = uuid.New()
	}
	out, concerns := r.Review(ctx, dp)
	return out, concerns, nil
}
