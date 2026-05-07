// Package analytics is the kb-30 override-rate analytics surface.
//
// Wave 6 Task 1: tracks override events per rule_id, computes the
// rolling 30-day override rate, and flags rules whose rate exceeds the
// retirement threshold (default 70%). The output is consumed by the
// rule-retirement workflow (shared/cql-toolchain/rule_retirement_workflow.py)
// which generates retirement candidates with a clinical-lead-override
// option per Layer 3 v2 doc Part 4.2.
//
// The tracker is in-memory and append-only; persistence to the kb-30
// audit store is V2 work. The library-wide override-rate target is
// < 5% per the Wave 6 plan acceptance.
package analytics

import (
	"sort"
	"sync"
	"time"
)

// EventKind labels each tracked event so the override rate is the
// fraction of override events over the total fire+override events.
type EventKind string

const (
	// EventFire denotes a rule that fired and surfaced to a clinician.
	EventFire EventKind = "fire"
	// EventOverride denotes a clinician override of a rule fire.
	EventOverride EventKind = "override"
)

// Event is a single tracked event for a rule_id.
type Event struct {
	RuleID    string
	Kind      EventKind
	Timestamp time.Time
}

// RuleStats is the aggregated stats for a single rule_id over the
// rolling window.
type RuleStats struct {
	RuleID        string
	WindowDays    int
	FireCount     int
	OverrideCount int
	OverrideRate  float64 // 0..1
	FlagRetire    bool    // true when OverrideRate > threshold and FireCount >= MinFires
}

// Default thresholds. Wave 6 plan: retire when override rate exceeds
// 70% over 30 days unless clinical lead overrides.
const (
	DefaultWindowDays         = 30
	DefaultRetireThreshold    = 0.70
	DefaultMinFiresForRetire  = 5
	LibraryWideTargetMaxRate  = 0.05
)

// Tracker is an in-memory append-only event log keyed by rule_id.
//
// Concurrent-safe.
type Tracker struct {
	mu     sync.RWMutex
	events []Event
	now    func() time.Time
}

// NewTracker constructs a tracker that uses time.Now for the rolling window.
func NewTracker() *Tracker {
	return &Tracker{now: func() time.Time { return time.Now().UTC() }}
}

// withClock is used in tests to inject a deterministic clock.
func (t *Tracker) withClock(now func() time.Time) *Tracker {
	t.now = now
	return t
}

// Record appends an event. Empty rule_id and zero timestamps are rejected
// (Record returns silently in that case so callers don't have to handle).
func (t *Tracker) Record(ruleID string, kind EventKind, ts time.Time) {
	if ruleID == "" || ts.IsZero() {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, Event{RuleID: ruleID, Kind: kind, Timestamp: ts.UTC()})
}

// Stats computes per-rule stats over the trailing windowDays. Pass 0 to
// use DefaultWindowDays. The retirement threshold is configurable; pass
// 0 for DefaultRetireThreshold.
func (t *Tracker) Stats(windowDays int, retireThreshold float64) []RuleStats {
	if windowDays <= 0 {
		windowDays = DefaultWindowDays
	}
	if retireThreshold <= 0 {
		retireThreshold = DefaultRetireThreshold
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := t.now().Add(-time.Duration(windowDays) * 24 * time.Hour)
	per := make(map[string]*RuleStats)
	for _, e := range t.events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		s, ok := per[e.RuleID]
		if !ok {
			s = &RuleStats{RuleID: e.RuleID, WindowDays: windowDays}
			per[e.RuleID] = s
		}
		switch e.Kind {
		case EventFire:
			s.FireCount++
		case EventOverride:
			s.OverrideCount++
		}
	}
	out := make([]RuleStats, 0, len(per))
	for _, s := range per {
		total := s.FireCount + s.OverrideCount
		if total > 0 {
			s.OverrideRate = float64(s.OverrideCount) / float64(total)
		}
		s.FlagRetire = s.OverrideRate > retireThreshold && total >= DefaultMinFiresForRetire
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RuleID < out[j].RuleID })
	return out
}

// LibraryWideOverrideRate returns the aggregate override rate across
// all events in the trailing window. Used to track the < 5% target.
func (t *Tracker) LibraryWideOverrideRate(windowDays int) float64 {
	if windowDays <= 0 {
		windowDays = DefaultWindowDays
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := t.now().Add(-time.Duration(windowDays) * 24 * time.Hour)
	var fires, overrides int
	for _, e := range t.events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		switch e.Kind {
		case EventFire:
			fires++
		case EventOverride:
			overrides++
		}
	}
	total := fires + overrides
	if total == 0 {
		return 0
	}
	return float64(overrides) / float64(total)
}

// RetirementCandidates returns the subset of Stats() whose FlagRetire
// is true.
func (t *Tracker) RetirementCandidates(windowDays int, retireThreshold float64) []RuleStats {
	all := t.Stats(windowDays, retireThreshold)
	var out []RuleStats
	for _, s := range all {
		if s.FlagRetire {
			out = append(out, s)
		}
	}
	return out
}
