package stability

import (
	"fmt"
	"time"
)

// Engine evaluates whether a proposed state transition should be accepted,
// damped, or overridden. Purely functional — no internal state, no
// persistence. Consumers pass a History on every call.
type Engine struct {
	policy Policy
}

// NewEngine constructs an engine with the given policy.
func NewEngine(policy Policy) *Engine {
	return &Engine{policy: policy}
}

// Evaluate returns a Decision for the proposed transition.
//
//   history: the recent state history for the subject (patient, session, etc.)
//   proposedState: the state the classifier wants to transition to
//   now: the current time (passed explicitly for testability)
//   override: true when the consumer has detected an event that should
//             bypass dwell/flap checks (e.g. medication change, hospitalization)
func (e *Engine) Evaluate(
	history History,
	proposedState string,
	now time.Time,
	override bool,
) Result {
	current := history.LatestState()

	// No history -> first classification, accept.
	if current == "" {
		return Result{Decision: DecisionAccept, Reason: "no prior state"}
	}

	// No state change -> trivially accept (idempotent).
	if current == proposedState {
		return Result{Decision: DecisionAccept, Reason: "no transition"}
	}

	// Override events bypass dwell and flap checks.
	if override {
		return Result{Decision: DecisionOverride, Reason: "override event bypasses dwell"}
	}

	// Dwell check.
	enteredAt := history.LatestEnteredAt()
	elapsed := now.Sub(enteredAt)
	if elapsed < e.policy.MinDwell {
		// Phase 5 P5-1: before damping, check whether the raw classifier
		// output has been consistently agreeing with the proposed state.
		// If the agreement rate within the dwell window meets the
		// configured threshold, override the dwell. The engine remains
		// the sole arbiter of transitions — no orchestrator escape hatch.
		if e.policy.MaxDwellOverrideRate > 0 {
			rate := history.RawMatchRate(now, e.policy.MinDwell, proposedState)
			if rate >= e.policy.MaxDwellOverrideRate {
				return Result{
					Decision: DecisionAccept,
					Reason: fmt.Sprintf(
						"dwell overridden: raw match rate %.0f%% >= %.0f%% threshold",
						rate*100, e.policy.MaxDwellOverrideRate*100),
				}
			}
		}
		return Result{
			Decision: DecisionDamp,
			Reason: fmt.Sprintf("dwell not met: %s elapsed of %s required",
				elapsed.Round(time.Hour), e.policy.MinDwell),
		}
	}

	// Flap-lock check.
	if e.policy.MaxFlapsBeforeLock > 0 {
		flaps := history.CountFlapsInWindow(now, e.policy.FlapWindow)
		if flaps >= e.policy.MaxFlapsBeforeLock {
			return Result{
				Decision: DecisionDamp,
				Reason: fmt.Sprintf("flap-locked: %d flaps in last %s (max %d)",
					flaps, e.policy.FlapWindow, e.policy.MaxFlapsBeforeLock),
			}
		}
	}

	return Result{Decision: DecisionAccept, Reason: "transition accepted"}
}
