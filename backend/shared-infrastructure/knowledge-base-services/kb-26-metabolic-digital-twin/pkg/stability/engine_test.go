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
