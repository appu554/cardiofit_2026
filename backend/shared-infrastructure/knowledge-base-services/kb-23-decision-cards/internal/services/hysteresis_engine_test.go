package services

import (
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// gatesAtOrBelow — pure function that determines which gate values count as
// "confirming" evidence for a downgrade. We test this independently because
// it's the core predicate of the N-01 hysteresis algorithm.
// ---------------------------------------------------------------------------

func newTestHysteresisEngine() *HysteresisEngine {
	return NewHysteresisEngine(nil, nil, zap.NewNop())
}

func TestGatesAtOrBelow_Safe(t *testing.T) {
	he := newTestHysteresisEngine()
	result := he.gatesAtOrBelow(models.GateSafe)

	// SAFE is level 0 — only SAFE itself is at or below
	if len(result) != 1 {
		t.Fatalf("expected 1 gate, got %d: %v", len(result), result)
	}
	if result[0] != "SAFE" {
		t.Errorf("expected [SAFE], got %v", result)
	}
}

func TestGatesAtOrBelow_Modify(t *testing.T) {
	he := newTestHysteresisEngine()
	result := he.gatesAtOrBelow(models.GateModify)

	// MODIFY is level 1 — SAFE (0) and MODIFY (1) are at or below
	if len(result) != 2 {
		t.Fatalf("expected 2 gates, got %d: %v", len(result), result)
	}
	expected := map[string]bool{"SAFE": true, "MODIFY": true}
	for _, g := range result {
		if !expected[g] {
			t.Errorf("unexpected gate %q in result", g)
		}
	}
}

func TestGatesAtOrBelow_Pause(t *testing.T) {
	he := newTestHysteresisEngine()
	result := he.gatesAtOrBelow(models.GatePause)

	// PAUSE is level 2 — SAFE, MODIFY, PAUSE
	if len(result) != 3 {
		t.Fatalf("expected 3 gates, got %d: %v", len(result), result)
	}
	expected := map[string]bool{"SAFE": true, "MODIFY": true, "PAUSE": true}
	for _, g := range result {
		if !expected[g] {
			t.Errorf("unexpected gate %q in result", g)
		}
	}
}

func TestGatesAtOrBelow_Halt(t *testing.T) {
	he := newTestHysteresisEngine()
	result := he.gatesAtOrBelow(models.GateHalt)

	// HALT is level 3 — all gates are at or below
	if len(result) != 4 {
		t.Fatalf("expected 4 gates, got %d: %v", len(result), result)
	}
}

// ---------------------------------------------------------------------------
// Apply — testing the upgrade/same/downgrade decision logic.
// NOTE: countConfirmingSessions requires a DB, so we only test the
// immediate-return paths (same gate, upgrade).
// ---------------------------------------------------------------------------

func TestApply_SameGate_ReturnsUnchanged(t *testing.T) {
	he := newTestHysteresisEngine()

	gates := []models.MCUGate{models.GateSafe, models.GateModify, models.GatePause, models.GateHalt}
	for _, g := range gates {
		result, rationale := he.Apply(
			[16]byte{}, // zero UUID
			g, g,
		)
		if result != g {
			t.Errorf("Apply(%s, %s) = %s, want %s", g, g, result, g)
		}
		if rationale != "" {
			t.Errorf("same gate should return empty rationale, got %q", rationale)
		}
	}
}

func TestApply_Upgrade_Immediate(t *testing.T) {
	he := newTestHysteresisEngine()

	tests := []struct {
		name    string
		current models.MCUGate
		propose models.MCUGate
	}{
		{"SAFE -> MODIFY", models.GateSafe, models.GateModify},
		{"SAFE -> PAUSE", models.GateSafe, models.GatePause},
		{"SAFE -> HALT", models.GateSafe, models.GateHalt},
		{"MODIFY -> PAUSE", models.GateModify, models.GatePause},
		{"MODIFY -> HALT", models.GateModify, models.GateHalt},
		{"PAUSE -> HALT", models.GatePause, models.GateHalt},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, rationale := he.Apply([16]byte{}, tc.current, tc.propose)
			if result != tc.propose {
				t.Errorf("Apply(%s, %s) = %s, want %s (upgrade should be immediate)",
					tc.current, tc.propose, result, tc.propose)
			}
			if rationale != "N-01: gate upgrade immediate" {
				t.Errorf("expected immediate upgrade rationale, got %q", rationale)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultSLAConfig — verify production defaults match specification
// ---------------------------------------------------------------------------

func TestDefaultSLAConfig(t *testing.T) {
	cfg := DefaultSLAConfig()

	if cfg.HaltSLA.Minutes() != 15 {
		t.Errorf("HALT SLA = %v, want 15m", cfg.HaltSLA)
	}
	if cfg.PauseSLA.Hours() != 1 {
		t.Errorf("PAUSE SLA = %v, want 1h", cfg.PauseSLA)
	}
	if cfg.ModifySLA.Hours() != 4 {
		t.Errorf("MODIFY SLA = %v, want 4h", cfg.ModifySLA)
	}
}
