package titration

import (
	"testing"
	"time"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// ═══════════════════════════════════════════════════════════════════════
// ComputeDose tests
// ═══════════════════════════════════════════════════════════════════════

func TestComputeDose_ClearGate_AppliesDose(t *testing.T) {
	eng := NewTitrationEngine(20.0)
	result := eng.ComputeDose(
		vt.ArbiterOutput{FinalGate: vt.GateClear},
		100.0, 10.0, 1.0,
	)
	if result.Blocked {
		t.Error("CLEAR gate should not block")
	}
	if result.NewDose != 110.0 {
		t.Errorf("expected 110, got %f", result.NewDose)
	}
}

func TestComputeDose_HaltGate_Blocks(t *testing.T) {
	eng := NewTitrationEngine(20.0)
	result := eng.ComputeDose(
		vt.ArbiterOutput{FinalGate: vt.GateHalt, DominantChannel: "B"},
		100.0, 10.0, 1.0,
	)
	if !result.Blocked {
		t.Error("HALT gate should block")
	}
	if result.BlockedBy == "" {
		t.Error("blocked_by should be set")
	}
}

func TestComputeDose_GainModulation(t *testing.T) {
	eng := NewTitrationEngine(20.0)
	result := eng.ComputeDose(
		vt.ArbiterOutput{FinalGate: vt.GateClear},
		100.0, 10.0, 0.5,
	)
	if result.DoseDelta != 5.0 {
		t.Errorf("delta should be 5.0 (10 * 0.5), got %f", result.DoseDelta)
	}
}

func TestComputeDose_MaxDeltaCapped(t *testing.T) {
	eng := NewTitrationEngine(20.0)
	result := eng.ComputeDose(
		vt.ArbiterOutput{FinalGate: vt.GateClear},
		100.0, 50.0, 1.0, // 50% requested but max is 20%
	)
	if result.DoseDelta != 20.0 {
		t.Errorf("delta should be capped at 20.0, got %f", result.DoseDelta)
	}
}

func TestComputeDose_NegativeDeltaCapped(t *testing.T) {
	eng := NewTitrationEngine(20.0)
	result := eng.ComputeDose(
		vt.ArbiterOutput{FinalGate: vt.GateClear},
		100.0, -50.0, 1.0,
	)
	if result.DoseDelta != -20.0 {
		t.Errorf("negative delta should be capped at -20.0, got %f", result.DoseDelta)
	}
}

func TestComputeDose_DoseNeverNegative(t *testing.T) {
	eng := NewTitrationEngine(100.0) // allow 100% change
	result := eng.ComputeDose(
		vt.ArbiterOutput{FinalGate: vt.GateClear},
		10.0, -50.0, 1.0,
	)
	if result.NewDose != 0.0 {
		t.Errorf("dose should floor at 0, got %f", result.NewDose)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Integrator freeze/resume tests (Commitment 1)
// ═══════════════════════════════════════════════════════════════════════

func TestIntegrator_FreezeAndResume(t *testing.T) {
	ig := NewIntegrator(100.0)

	if ig.IsFrozen() {
		t.Error("new integrator should not be frozen")
	}
	if ig.State() != IntegratorActive {
		t.Errorf("state should be ACTIVE, got %s", ig.State())
	}

	ig.Freeze(95.0, "CH_B:HALT")
	if !ig.IsFrozen() {
		t.Error("should be frozen after Freeze()")
	}
	if ig.FrozenDose() != 95.0 {
		t.Errorf("frozen dose should be 95.0, got %f", ig.FrozenDose())
	}

	ig.Resume()
	if ig.IsFrozen() {
		t.Error("should not be frozen after Resume()")
	}
	if ig.FrozenDose() != 95.0 {
		t.Error("frozen dose should be preserved after resume")
	}
}

func TestIntegrator_PauseHours(t *testing.T) {
	ig := NewIntegrator(100.0)
	ig.Freeze(100.0, "test")
	time.Sleep(10 * time.Millisecond)
	hours := ig.PauseHours()
	if hours <= 0 {
		t.Error("pause hours should be > 0 while frozen")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Rate limiter tests (Commitment 2)
// ═══════════════════════════════════════════════════════════════════════

func TestRateLimiter_PostResume(t *testing.T) {
	rl := NewRateLimiter()

	// Not limited initially
	if rl.IsLimited() {
		t.Error("should not be limited initially")
	}
	if rl.ApplyLimit(20.0) != 20.0 {
		t.Error("unlimited should return full delta")
	}

	// Activate after 36 hours pause → ceil(36/24) = 2 cycles
	rl.ActivatePostResume(36.0)
	if !rl.IsLimited() {
		t.Error("should be limited after activation")
	}

	// Cycle 1: limited
	limited := rl.ApplyLimit(20.0)
	if limited != 10.0 {
		t.Errorf("cycle 1 should be 10.0 (50%% of 20), got %f", limited)
	}

	// Cycle 2: still limited
	limited = rl.ApplyLimit(20.0)
	if limited != 10.0 {
		t.Errorf("cycle 2 should still be 10.0, got %f", limited)
	}

	// Cycle 3: no longer limited
	if rl.IsLimited() {
		t.Error("should not be limited after all cycles consumed")
	}
	full := rl.ApplyLimit(20.0)
	if full != 20.0 {
		t.Errorf("post-limit should return full 20.0, got %f", full)
	}
}

func TestRateLimiter_MinOneCycle(t *testing.T) {
	rl := NewRateLimiter()
	rl.ActivatePostResume(0.5) // < 24h → ceil(0.5/24) = 1 cycle
	if rl.RemainingLimitedCycles != 1 {
		t.Errorf("expected 1 limited cycle, got %d", rl.RemainingLimitedCycles)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Re-entry protocol tests (Commitment 3)
// ═══════════════════════════════════════════════════════════════════════

func TestReentryProtocol_ThreePhases(t *testing.T) {
	rp := NewReentryProtocol()

	if rp.IsActive() {
		t.Error("should not be active initially")
	}

	rp.Activate()
	if rp.Phase() != ReentryMonitoring {
		t.Errorf("should start in MONITORING, got %s", rp.Phase())
	}
	if rp.AllowsDoseChange() {
		t.Error("MONITORING should not allow dose changes")
	}
	if rp.MaxDeltaMultiplier() != 0.0 {
		t.Error("MONITORING multiplier should be 0.0")
	}

	// Advance through monitoring (2 cycles)
	rp.AdvanceCycle()
	if rp.Phase() != ReentryMonitoring {
		t.Errorf("should still be MONITORING after 1 cycle, got %s", rp.Phase())
	}
	rp.AdvanceCycle()
	if rp.Phase() != ReentryConservative {
		t.Errorf("should be CONSERVATIVE after 2 cycles, got %s", rp.Phase())
	}

	// Conservative phase
	if !rp.AllowsDoseChange() {
		t.Error("CONSERVATIVE should allow dose changes")
	}
	if rp.MaxDeltaMultiplier() != 0.50 {
		t.Errorf("CONSERVATIVE multiplier should be 0.50, got %f", rp.MaxDeltaMultiplier())
	}

	// Advance through conservative (3 cycles)
	rp.AdvanceCycle()
	rp.AdvanceCycle()
	rp.AdvanceCycle()
	if rp.Phase() != ReentryNormal {
		t.Errorf("should be NORMAL after 3 conservative cycles, got %s", rp.Phase())
	}

	// One more advance completes the protocol
	rp.AdvanceCycle()
	if rp.Phase() != ReentryNone {
		t.Errorf("should be NONE after normal phase, got %s", rp.Phase())
	}
	if rp.IsActive() {
		t.Error("protocol should no longer be active")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Dose cooldown tests (Commitment 6)
// ═══════════════════════════════════════════════════════════════════════

func TestCooldownTracker_BasalInsulin48h(t *testing.T) {
	ct := NewCooldownTracker(DefaultCooldownConfig())

	// No prior dose → not on cooldown
	if ct.IsOnCooldown("p1", MedClassBasalInsulin) {
		t.Error("should not be on cooldown with no prior dose")
	}

	// Record a dose 1 hour ago
	ct.RecordDoseChange(DoseEvent{
		PatientID: "p1",
		MedClass:  MedClassBasalInsulin,
		AppliedAt: time.Now().Add(-1 * time.Hour),
		DoseDelta: 5.0,
	})

	if !ct.IsOnCooldown("p1", MedClassBasalInsulin) {
		t.Error("should be on cooldown (48h not elapsed)")
	}

	remaining := ct.RemainingCooldown("p1", MedClassBasalInsulin)
	if remaining < 46*time.Hour || remaining > 48*time.Hour {
		t.Errorf("remaining cooldown should be ~47h, got %v", remaining)
	}
}

func TestCooldownTracker_RapidInsulin6h(t *testing.T) {
	ct := NewCooldownTracker(DefaultCooldownConfig())

	ct.RecordDoseChange(DoseEvent{
		PatientID: "p1",
		MedClass:  MedClassRapidInsulin,
		AppliedAt: time.Now().Add(-7 * time.Hour), // 7 hours ago > 6h
		DoseDelta: 2.0,
	})

	if ct.IsOnCooldown("p1", MedClassRapidInsulin) {
		t.Error("should NOT be on cooldown (6h elapsed)")
	}
}

func TestCooldownTracker_DifferentMedClasses(t *testing.T) {
	ct := NewCooldownTracker(DefaultCooldownConfig())

	ct.RecordDoseChange(DoseEvent{
		PatientID: "p1",
		MedClass:  MedClassBasalInsulin,
		AppliedAt: time.Now(),
	})

	if ct.IsOnCooldown("p1", MedClassRapidInsulin) {
		t.Error("rapid insulin should not be on cooldown when only basal was changed")
	}
	if !ct.IsOnCooldown("p1", MedClassBasalInsulin) {
		t.Error("basal insulin should be on cooldown")
	}
}
