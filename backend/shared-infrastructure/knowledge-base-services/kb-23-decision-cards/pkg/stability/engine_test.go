package stability

import (
	"testing"
	"time"
)

func TestEngine_FirstClassification_Accepts(t *testing.T) {
	eng := NewEngine(Policy{
		MinDwell:           14 * 24 * time.Hour,
		FlapWindow:         30 * 24 * time.Hour,
		MaxFlapsBeforeLock: 3,
	})

	result := eng.Evaluate(History{}, "MASKED_HTN", time.Now(), false)
	if result.Decision != DecisionAccept {
		t.Errorf("expected ACCEPT for first classification, got %s", result.Decision)
	}
}

func TestEngine_SameStateProposed_Accepts(t *testing.T) {
	eng := NewEngine(Policy{MinDwell: 14 * 24 * time.Hour})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "SUSTAINED_HTN", EnteredAt: now.AddDate(0, 0, -5)},
	}}

	result := eng.Evaluate(history, "SUSTAINED_HTN", now, false)
	if result.Decision != DecisionAccept {
		t.Errorf("expected ACCEPT for same state, got %s", result.Decision)
	}
}

func TestEngine_DifferentStateWithinDwell_Damps(t *testing.T) {
	eng := NewEngine(Policy{MinDwell: 14 * 24 * time.Hour})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "SUSTAINED_HTN", EnteredAt: now.AddDate(0, 0, -5)},
	}}

	result := eng.Evaluate(history, "MASKED_HTN", now, false)
	if result.Decision != DecisionDamp {
		t.Errorf("expected DAMP (5d elapsed, 14d required), got %s", result.Decision)
	}
}

func TestEngine_DifferentStatePastDwell_Accepts(t *testing.T) {
	eng := NewEngine(Policy{MinDwell: 14 * 24 * time.Hour})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "SUSTAINED_HTN", EnteredAt: now.AddDate(0, 0, -20)},
	}}

	result := eng.Evaluate(history, "MASKED_HTN", now, false)
	if result.Decision != DecisionAccept {
		t.Errorf("expected ACCEPT (20d elapsed, 14d required), got %s", result.Decision)
	}
}

func TestEngine_OverrideBypassesDwell(t *testing.T) {
	eng := NewEngine(Policy{MinDwell: 14 * 24 * time.Hour})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "SUSTAINED_HTN", EnteredAt: now.AddDate(0, 0, -2)},
	}}

	result := eng.Evaluate(history, "MASKED_HTN", now, true)
	if result.Decision != DecisionOverride {
		t.Errorf("expected OVERRIDE (2d elapsed but override=true), got %s", result.Decision)
	}
}

func TestEngine_OverrideBypassesFlapLock(t *testing.T) {
	eng := NewEngine(Policy{
		MinDwell:           1 * time.Hour, // short so dwell is not the blocker
		FlapWindow:         30 * 24 * time.Hour,
		MaxFlapsBeforeLock: 2,
	})
	now := time.Now()
	// 3 flaps in the window — should be flap-locked
	history := History{Entries: []Entry{
		{State: "A", EnteredAt: now.AddDate(0, 0, -25)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -20)},
		{State: "A", EnteredAt: now.AddDate(0, 0, -15)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -10)}, // latest — current state is B
	}}

	// Without override — should be damped by flap-lock
	plain := eng.Evaluate(history, "A", now, false)
	if plain.Decision != DecisionDamp {
		t.Errorf("expected DAMP (flap-locked), got %s: %s", plain.Decision, plain.Reason)
	}

	// With override — should be accepted
	override := eng.Evaluate(history, "A", now, true)
	if override.Decision != DecisionOverride {
		t.Errorf("expected OVERRIDE to bypass flap-lock, got %s", override.Decision)
	}
}

func TestEngine_FlapLockTriggersAfterMaxFlaps(t *testing.T) {
	eng := NewEngine(Policy{
		MinDwell:           1 * time.Hour,
		FlapWindow:         30 * 24 * time.Hour,
		MaxFlapsBeforeLock: 3,
	})
	now := time.Now()
	// Exactly 3 flaps — should be locked
	history := History{Entries: []Entry{
		{State: "A", EnteredAt: now.AddDate(0, 0, -25)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -20)},
		{State: "A", EnteredAt: now.AddDate(0, 0, -15)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -10)},
	}}

	result := eng.Evaluate(history, "A", now, false)
	if result.Decision != DecisionDamp {
		t.Errorf("expected DAMP at flap count 3 (max 3), got %s", result.Decision)
	}
}

func TestEngine_FlapLockDisabledByZero(t *testing.T) {
	eng := NewEngine(Policy{
		MinDwell:           1 * time.Hour,
		FlapWindow:         30 * 24 * time.Hour,
		MaxFlapsBeforeLock: 0, // disabled
	})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "A", EnteredAt: now.AddDate(0, 0, -25)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -20)},
		{State: "A", EnteredAt: now.AddDate(0, 0, -15)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -10)},
	}}

	result := eng.Evaluate(history, "A", now, false)
	if result.Decision != DecisionAccept {
		t.Errorf("expected ACCEPT (flap-lock disabled), got %s", result.Decision)
	}
}

func TestEngine_FlapDetectionRespectsWindowBoundary(t *testing.T) {
	eng := NewEngine(Policy{
		MinDwell:           1 * time.Hour,
		FlapWindow:         7 * 24 * time.Hour,
		MaxFlapsBeforeLock: 2,
	})
	now := time.Now()
	// Old flaps outside the 7-day window — should NOT count
	history := History{Entries: []Entry{
		{State: "A", EnteredAt: now.AddDate(0, 0, -30)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -28)},
		{State: "A", EnteredAt: now.AddDate(0, 0, -26)},
		{State: "B", EnteredAt: now.AddDate(0, 0, -24)},
		{State: "A", EnteredAt: now.AddDate(0, 0, -10)}, // single transition within window
	}}

	result := eng.Evaluate(history, "B", now, false)
	if result.Decision != DecisionAccept {
		t.Errorf("expected ACCEPT (only 1 flap in 7d window, max 2), got %s", result.Decision)
	}
}

func TestHistory_LatestState_Empty(t *testing.T) {
	h := History{}
	if h.LatestState() != "" {
		t.Errorf("expected empty string for empty history, got %s", h.LatestState())
	}
}

func TestHistory_CountFlapsInWindow_TwoEntriesSameState(t *testing.T) {
	now := time.Now()
	h := History{Entries: []Entry{
		{State: "A", EnteredAt: now.AddDate(0, 0, -5)},
		{State: "A", EnteredAt: now.AddDate(0, 0, -3)}, // not a flap (same state)
	}}
	if got := h.CountFlapsInWindow(now, 30*24*time.Hour); got != 0 {
		t.Errorf("expected 0 flaps for repeated same state, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// Phase 5 P5-1: raw-vs-stable disagreement override of dwell
//
// When the stability engine has been damping a transition for several days
// but the raw classifier output has been consistently agreeing with the
// proposed transition (not noise), the engine should override the dwell
// and accept the transition. The override threshold is the fraction of
// raw entries inside the dwell window that match the proposed state.
// Implemented as a single new Policy field (MaxDwellOverrideRate) and a
// new History method (RawMatchRate) — the engine remains the sole arbiter
// of transitions, no orchestrator escape hatches.
// ---------------------------------------------------------------------------

func TestHistory_RawMatchRate_NoEntries_ReturnsZero(t *testing.T) {
	h := History{}
	if got := h.RawMatchRate(time.Now(), 14*24*time.Hour, "MASKED_HTN"); got != 0 {
		t.Errorf("expected 0 for empty history, got %f", got)
	}
}

func TestHistory_RawMatchRate_AllRawMatchProposed_ReturnsOne(t *testing.T) {
	now := time.Now()
	h := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -10)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -7)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -3)},
	}}
	if got := h.RawMatchRate(now, 14*24*time.Hour, "MASKED_HTN"); got != 1.0 {
		t.Errorf("expected rate 1.0, got %f", got)
	}
}

func TestHistory_RawMatchRate_PartialMatch_ReturnsFraction(t *testing.T) {
	now := time.Now()
	h := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -10)},
		{State: "NORMOTENSION", Raw: "NORMOTENSION", EnteredAt: now.AddDate(0, 0, -8)}, // mismatch
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -5)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -2)},
	}}
	got := h.RawMatchRate(now, 14*24*time.Hour, "MASKED_HTN")
	if got != 0.75 {
		t.Errorf("expected 0.75 (3/4), got %f", got)
	}
}

func TestHistory_RawMatchRate_IgnoresLegacyEmptyRaw(t *testing.T) {
	now := time.Now()
	h := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "", EnteredAt: now.AddDate(0, 0, -12)}, // legacy
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -8)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -3)},
	}}
	got := h.RawMatchRate(now, 14*24*time.Hour, "MASKED_HTN")
	if got != 1.0 {
		t.Errorf("expected 1.0 (legacy entry skipped), got %f", got)
	}
}

func TestHistory_RawMatchRate_OutsideWindow_Excluded(t *testing.T) {
	now := time.Now()
	h := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -30)}, // outside 14d
		{State: "NORMOTENSION", Raw: "NORMOTENSION", EnteredAt: now.AddDate(0, 0, -3)},
	}}
	got := h.RawMatchRate(now, 14*24*time.Hour, "MASKED_HTN")
	if got != 0.0 {
		t.Errorf("expected 0.0 (only in-window entry doesn't match), got %f", got)
	}
}

func TestEngine_DwellOverridden_WhenRawConsistentlyAgreesWithProposed(t *testing.T) {
	// Patient held at NORMOTENSION for 5 days (less than 14d dwell), but the
	// raw classifier output has been MASKED_HTN on 4 of the last 5 snapshots.
	// At a 0.7 (70%) override threshold, the dwell should yield and the
	// engine should accept the MASKED_HTN transition.
	eng := NewEngine(Policy{
		MinDwell:             14 * 24 * time.Hour,
		MaxDwellOverrideRate: 0.7,
	})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -5)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -4)},
		{State: "NORMOTENSION", Raw: "NORMOTENSION", EnteredAt: now.AddDate(0, 0, -3)}, // single dissent
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -2)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -1)},
	}}

	result := eng.Evaluate(history, "MASKED_HTN", now, false)
	if result.Decision != DecisionAccept {
		t.Errorf("expected ACCEPT (raw match rate 4/5 = 0.8 >= 0.7), got %s: %s", result.Decision, result.Reason)
	}
}

func TestEngine_DwellHeld_WhenRawAgreementBelowOverrideRate(t *testing.T) {
	// Same shape but raw agrees on only 2 of 5 snapshots — below 0.7 — dwell holds.
	eng := NewEngine(Policy{
		MinDwell:             14 * 24 * time.Hour,
		MaxDwellOverrideRate: 0.7,
	})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -5)},
		{State: "NORMOTENSION", Raw: "NORMOTENSION", EnteredAt: now.AddDate(0, 0, -4)},
		{State: "NORMOTENSION", Raw: "WHITE_COAT_HTN", EnteredAt: now.AddDate(0, 0, -3)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -2)},
		{State: "NORMOTENSION", Raw: "NORMOTENSION", EnteredAt: now.AddDate(0, 0, -1)},
	}}

	result := eng.Evaluate(history, "MASKED_HTN", now, false)
	if result.Decision != DecisionDamp {
		t.Errorf("expected DAMP (raw match rate 2/5 = 0.4 < 0.7), got %s: %s", result.Decision, result.Reason)
	}
}

func TestEngine_DwellHeld_WhenOverrideRateUnconfigured(t *testing.T) {
	// Backward compatibility: a Policy that doesn't set MaxDwellOverrideRate
	// must behave exactly like the pre-Phase-5 engine — dwell always damps.
	eng := NewEngine(Policy{
		MinDwell: 14 * 24 * time.Hour,
		// MaxDwellOverrideRate intentionally omitted (== 0)
	})
	now := time.Now()
	history := History{Entries: []Entry{
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -5)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -4)},
		{State: "NORMOTENSION", Raw: "MASKED_HTN", EnteredAt: now.AddDate(0, 0, -3)},
	}}

	result := eng.Evaluate(history, "MASKED_HTN", now, false)
	if result.Decision != DecisionDamp {
		t.Errorf("expected DAMP when override rate is 0 (disabled), got %s: %s", result.Decision, result.Reason)
	}
}
