// Package stability provides a generic state-transition stability engine
// with dwell policy, flap detection, and override bypass. It is consumed
// by the BP context phenotype orchestrator in Phase 4, but is intentionally
// decoupled from that caller so other consumers (engagement classification,
// phenotype clustering) can adopt it independently.
//
// The engine is purely functional: no internal state, no persistence.
// Consumers pass a History on every Evaluate call and are responsible
// for persisting the accepted states to their own storage.
package stability

import "time"

// Decision is the stability engine's verdict on a proposed transition.
type Decision string

const (
	// DecisionAccept: the proposed state becomes the new current state.
	DecisionAccept Decision = "ACCEPT"

	// DecisionDamp: the proposed state is held; consumers should keep
	// the current state instead of adopting the proposed one. Triggered
	// when dwell policy or flap-lock prevents the transition.
	DecisionDamp Decision = "DAMP"

	// DecisionOverride: an override event (medication change, hospitalization,
	// etc.) bypasses dwell and flap checks. Functionally equivalent to
	// DecisionAccept but carries different semantics for logging/audit.
	DecisionOverride Decision = "OVERRIDE"
)

// Result is the engine's full output for a transition decision.
type Result struct {
	Decision Decision
	Reason   string
}

// Policy controls stability behavior for one consumer. Different consumers
// (BP context, engagement, etc.) use different policies.
type Policy struct {
	// MinDwell is the minimum time a state must be held before another
	// transition is accepted. Transitions proposed before this elapses
	// are damped unless an override event applies.
	MinDwell time.Duration

	// FlapWindow is the lookback window for flap detection. If the
	// proposed transition would re-enter a state that was active within
	// this window, the engine counts it toward the flap total.
	FlapWindow time.Duration

	// MaxFlapsBeforeLock — after N state changes within FlapWindow, the
	// engine refuses all subsequent transitions until an override fires.
	// Set to 0 to disable flap-lock.
	MaxFlapsBeforeLock int
}

// Entry is one record in the state history.
type Entry struct {
	State     string
	EnteredAt time.Time
}

// History is the sequence of (state, timestamp) entries the engine consults.
// Consumers pass a slice of recent transitions on each Evaluate call; the
// engine reads but never mutates. Persistence is the consumer's responsibility.
//
// Entries should be ordered oldest-first. The last entry is treated as the
// current state.
type History struct {
	Entries []Entry
}

// LatestState returns the most recent state, or "" if history is empty.
func (h *History) LatestState() string {
	if len(h.Entries) == 0 {
		return ""
	}
	return h.Entries[len(h.Entries)-1].State
}

// LatestEnteredAt returns when the latest state was entered, or the
// zero time if history is empty.
func (h *History) LatestEnteredAt() time.Time {
	if len(h.Entries) == 0 {
		return time.Time{}
	}
	return h.Entries[len(h.Entries)-1].EnteredAt
}

// CountFlapsInWindow returns how many distinct state-to-state transitions
// occurred within the given window before `now`. A flap is any entry whose
// State differs from the previous entry's State AND whose EnteredAt is
// within the window.
func (h *History) CountFlapsInWindow(now time.Time, window time.Duration) int {
	if len(h.Entries) < 2 {
		return 0
	}
	cutoff := now.Add(-window)
	var flaps int
	for i := 1; i < len(h.Entries); i++ {
		if h.Entries[i].EnteredAt.Before(cutoff) {
			continue
		}
		if h.Entries[i].State != h.Entries[i-1].State {
			flaps++
		}
	}
	return flaps
}
