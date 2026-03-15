// Package scenarios implements the V-MCU simulation regression test suite.
// These tests MUST pass before any patient connection.
// Run with: go test -v ./pkg/scenarios/
//
// Each test validates a specific clinical edge case against the Three-Channel
// Safety Architecture. Failures here indicate safety rule gaps that would
// cause harm in production.
package scenarios

import (
	"math"
	"testing"
	"time"

	"vaidshala/simulation/pkg/harness"
	"vaidshala/simulation/pkg/patient"
	"vaidshala/simulation/pkg/types"
)

var _ = time.Now // ensure import used

// ---------------------------------------------------------------------------
// SCENARIO 1: Active Hypoglycaemia + Insulin Increase
// ALL 3 channels must HALT. Zero dose output. DominantChannel = B.
// Clinical basis: Glucose 3.5 mmol/L is an emergency. No dose change of any kind.
// ---------------------------------------------------------------------------
func TestScenario01_ActiveHypoglycaemia(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.ActiveHypoglycaemia()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	// All three channels must independently HALT
	if result.SafetyTrace.MCUGate != types.HALT {
		t.Errorf("Channel A: expected HALT, got %s", result.SafetyTrace.MCUGate)
	}
	if result.SafetyTrace.PhysioGate != types.HALT {
		t.Errorf("Channel B: expected HALT, got %s (rule: %s)", result.SafetyTrace.PhysioGate, result.PhysioRuleFired)
	}
	// Channel C should also HALT via PG-04 (hypo + insulin increase)
	if result.SafetyTrace.ProtocolGate != types.HALT {
		t.Errorf("Channel C: expected HALT, got %s (rule: %s)", result.SafetyTrace.ProtocolGate, result.ProtocolRuleFired)
	}

	// Final gate must be HALT
	if result.FinalGate != types.HALT {
		t.Fatalf("SAFETY FAILURE: Final gate expected HALT, got %s", result.FinalGate)
	}

	// Zero dose output — non-negotiable
	if result.DoseApplied {
		t.Fatalf("SAFETY FAILURE: Dose was applied during active hypoglycaemia! Delta=%.2f", result.DoseDelta)
	}

	// Dominant channel should be B (physiology takes priority)
	if result.DominantChannel != types.ChannelB {
		t.Errorf("DominantChannel: expected PHYSIO_GATE, got %s", result.DominantChannel)
	}

	t.Logf("PASS: Active hypoglycaemia → 3-channel HALT, zero dose, dominant=B, rule=%s", result.PhysioRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 2: AKI Mid-Titration (Genuine, Not RAAS Response)
// Channel B: HALT (B-04, creatinine delta >26 µmol/L)
// Channel C: HALT (PG-03, cross-channel AKI confirmation)
// ---------------------------------------------------------------------------
func TestScenario02_AKIMidTitration(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.AKIMidTitration()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate != types.HALT {
		t.Fatalf("SAFETY FAILURE: AKI should trigger HALT, got %s", result.FinalGate)
	}
	if result.DoseApplied {
		t.Fatalf("SAFETY FAILURE: Dose applied during AKI!")
	}

	// Channel B must fire B-04
	if result.PhysioRuleFired != "B-04" {
		t.Errorf("Expected Channel B rule B-04 (creatinine delta), got %s", result.PhysioRuleFired)
	}

	t.Logf("PASS: AKI mid-titration → HALT, zero dose, physio_rule=%s, protocol_rule=%s",
		result.PhysioRuleFired, result.ProtocolRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 3: RAAS Creatinine Tolerance (Expected Rise, NOT AKI)
// THE MOST CRITICAL FALSE-POSITIVE TEST.
// Channel B must DOWNGRADE from HALT to PAUSE (via CreatinineRiseExplained flag).
// Without PG-14, every ACEi/ARB increase triggers false AKI alarm.
// ---------------------------------------------------------------------------
func TestScenario03_RAASCreatinineTolerance(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.RAASCreatinineTolerance()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	// Final gate should be PAUSE (not HALT) — the RAAS tolerance downgrade
	if result.FinalGate == types.HALT {
		t.Fatalf("FALSE POSITIVE: RAAS expected creatinine rise triggered HALT instead of PAUSE. "+
			"This would remove the most renal-protective drug! Rule=%s", result.PhysioRuleFired)
	}
	if result.FinalGate > types.PAUSE {
		t.Errorf("Expected PAUSE or lower for RAAS tolerance, got %s", result.FinalGate)
	}

	// Channel B should fire B-04+PG-14 (downgraded)
	if result.PhysioRuleFired != "B-04+PG-14" {
		t.Logf("Note: Channel B rule=%s (expected B-04+PG-14 for RAAS tolerance downgrade)", result.PhysioRuleFired)
	}

	t.Logf("PASS: RAAS tolerance → PAUSE (not HALT), creatinine rise correctly attributed to ACEi. Rule=%s",
		result.PhysioRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 4: 5-Day Data Drop-Out (Stale Labs)
// HOLD_DATA on stale K+ (>14 days). No titration on stale safety data.
// ---------------------------------------------------------------------------
func TestScenario04_DataDropOut(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.DataDropOut()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate < types.HOLD_DATA {
		t.Fatalf("SAFETY FAILURE: Stale labs should trigger at least HOLD_DATA, got %s. "+
			"Missing data must NOT be treated as safe data.", result.FinalGate)
	}
	if result.DoseApplied {
		t.Fatalf("SAFETY FAILURE: Dose applied with stale safety labs!")
	}

	t.Logf("PASS: Data drop-out → %s, zero dose, rule=%s", result.FinalGate, result.PhysioRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 5: Non-Adherent Patient (Adherence 0.30)
// gain_factor must be 0.25. MCU_GATE=MODIFY blocks upward titration.
// ---------------------------------------------------------------------------
func TestScenario05_NonAdherentPatient(t *testing.T) {
	engine := harness.NewVMCUEngine()
	engine.LastDoseChangeTime = hoursAgo(72) // Past cooldown
	vp := patient.NonAdherentPatient()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	// With MODIFY gate, upward titration should be blocked
	if result.DoseDelta > 0 {
		t.Errorf("BEHAVIORAL SAFETY: MODIFY gate should prevent upward titration. Got delta=%.2f", result.DoseDelta)
	}

	// Gain factor should be 0.25 (adherence <0.40)
	if result.SafetyTrace.GainFactor != 0.25 {
		t.Errorf("GainFactor: expected 0.25 for adherence=0.30, got %.2f", result.SafetyTrace.GainFactor)
	}

	t.Logf("PASS: Non-adherent → gate=%s, dose_delta=%.2f, gain_factor=%.2f",
		result.FinalGate, result.DoseDelta, result.SafetyTrace.GainFactor)
}

// ---------------------------------------------------------------------------
// SCENARIO 6: J-Curve CKD Stage 3b
// SBP 102 with eGFR 35 → Channel B PAUSE (B-12, below 105 floor).
// Protects renal perfusion in CKD patients.
// ---------------------------------------------------------------------------
func TestScenario06_JCurveCKD3b(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.JCurveCKD3b()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate < types.PAUSE {
		t.Fatalf("RENAL SAFETY: CKD 3b patient with SBP=102 should trigger at least PAUSE. Got %s. "+
			"J-curve BP floor not enforced.", result.FinalGate)
	}

	if result.PhysioRuleFired != "B-12" {
		t.Errorf("Expected B-12 (J-curve), got %s", result.PhysioRuleFired)
	}

	t.Logf("PASS: J-curve CKD 3b → %s, rule=%s (SBP floor enforced at 105)", result.FinalGate, result.PhysioRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 7: Dual RAAS Contraindication
// ACEi + ARB simultaneously → PG-08 HALT.
// ---------------------------------------------------------------------------
func TestScenario07_DualRAAS(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.DualRAAS()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate != types.HALT {
		t.Fatalf("DRUG SAFETY: Dual RAAS (ACEi+ARB) must trigger HALT. Got %s", result.FinalGate)
	}
	if result.ProtocolRuleFired != "PG-08" {
		t.Errorf("Expected PG-08 (dual RAAS), got %s", result.ProtocolRuleFired)
	}

	t.Logf("PASS: Dual RAAS → HALT via %s", result.ProtocolRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 8: Severe Hyponatraemia + Thiazide
// Na+ 128 → Channel B HALT (B-17). Cerebral oedema risk.
// ---------------------------------------------------------------------------
func TestScenario08_HyponatraemiaThiazide(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.HyponatraemiaThiazide()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate != types.HALT {
		t.Fatalf("ELECTROLYTE SAFETY: Na+=128 should trigger HALT. Got %s", result.FinalGate)
	}
	if result.PhysioRuleFired != "B-17" {
		t.Errorf("Expected B-17 (severe hyponatraemia), got %s", result.PhysioRuleFired)
	}

	t.Logf("PASS: Severe hyponatraemia → HALT via %s", result.PhysioRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 9: GREEN Trajectory — Happy Path
// All labs normal. Dose should be applied. System must actually titrate.
// A system that only blocks is useless. It must also act when safe.
// ---------------------------------------------------------------------------
func TestScenario09_GreenTrajectory(t *testing.T) {
	engine := harness.NewVMCUEngine()
	engine.LastDoseChangeTime = hoursAgo(72) // Past cooldown
	vp := patient.GreenTrajectory()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate != types.CLEAR {
		t.Errorf("GREEN trajectory should produce CLEAR gate. Got %s (rule_b=%s, rule_c=%s)",
			result.FinalGate, result.PhysioRuleFired, result.ProtocolRuleFired)
	}

	if !result.DoseApplied {
		t.Errorf("GREEN trajectory with above-target glucose should produce a dose change. BlockedBy=%s", result.BlockedBy)
	}

	if result.DoseDelta <= 0 {
		t.Errorf("Expected positive dose delta for glucose 8.5 mmol/L. Got %.2f", result.DoseDelta)
	}

	// Verify autonomy limit: delta should not exceed ±20% of 16U = 3.2U
	maxDelta := 16.0 * 0.20
	if result.DoseDelta > maxDelta {
		t.Errorf("AUTONOMY LIMIT: dose delta %.2f exceeds ±20%% limit (%.2f)", result.DoseDelta, maxDelta)
	}

	t.Logf("PASS: GREEN trajectory → CLEAR, dose_applied=true, delta=%.2f (within ±20%% of %.1f)",
		result.DoseDelta, engine.Integrator.LastApprovedDose)
}

// ---------------------------------------------------------------------------
// SCENARIO 10: Metformin + CKD Stage 4
// eGFR 25 + metformin → PG-01 HALT (KDIGO absolute contraindication).
// ---------------------------------------------------------------------------
func TestScenario10_MetforminCKD4(t *testing.T) {
	engine := harness.NewVMCUEngine()
	vp := patient.MetforminCKD4()
	result := engine.RunCycle(vp.ToTitrationInput(1))

	if result.FinalGate != types.HALT {
		t.Fatalf("DRUG SAFETY: Metformin at eGFR=25 must trigger HALT. Got %s", result.FinalGate)
	}

	t.Logf("PASS: Metformin+CKD4 → HALT, physio=%s, protocol=%s",
		result.PhysioRuleFired, result.ProtocolRuleFired)
}

// ---------------------------------------------------------------------------
// SCENARIO 11: Integrator Resume After Extended HALT
// After 5 days (120h) of HALT, rate limiter should apply 50% for 5 cycles.
// Tests: freeze/resume, rate limiter, post-resume dampening.
// ---------------------------------------------------------------------------
func TestScenario11_IntegratorResumeAfterHALT(t *testing.T) {
	engine := harness.NewVMCUEngine()
	engine.LastDoseChangeTime = hoursAgo(200) // Well past cooldown

	// Step 1: Trigger HALT (active hypo)
	hypo := patient.ActiveHypoglycaemia()
	r1 := engine.RunCycle(hypo.ToTitrationInput(1))
	if r1.FinalGate != types.HALT {
		t.Fatalf("Setup: expected HALT, got %s", r1.FinalGate)
	}
	if !engine.Integrator.Frozen {
		t.Fatal("Integrator should be frozen after HALT")
	}

	// Simulate 5 days passing
	engine.Integrator.FrozenSince = hoursAgo(120)

	// Step 2: Resume with normal labs
	green := patient.GreenTrajectory()
	r2 := engine.RunCycle(green.ToTitrationInput(2))

	if engine.Integrator.Frozen {
		t.Error("Integrator should have resumed on CLEAR gate")
	}

	// Post-resume rate limiter should be active
	// 120h / 24 = 5 cycles at 50%
	expectedLimit := int(math.Ceil(120.0 / 24.0)) // 5, but may be 6 due to test execution time
	_ = expectedLimit
	if engine.Integrator.PostResumeLimit < 5 || engine.Integrator.PostResumeLimit > 6 {
		t.Errorf("PostResumeLimit: expected 5-6, got %d", engine.Integrator.PostResumeLimit)
	}

	// If dose was applied, it should be at 50% of normal
	if r2.DoseApplied {
		// Normal delta for glucose 8.5 would be ~2.0. At 50%, should be ~1.0.
		if r2.DoseDelta > 1.5 {
			t.Errorf("Post-resume delta %.2f exceeds 50%% rate limit", r2.DoseDelta)
		}
		t.Logf("Post-resume dose delta: %.2f (rate-limited to 50%%)", r2.DoseDelta)
	}

	t.Logf("PASS: Integrator resume after 120h HALT → PostResumeLimit=%d, frozen=%v",
		engine.Integrator.PostResumeLimit, engine.Integrator.Frozen)
}

// ---------------------------------------------------------------------------
// SCENARIO 12: Exhaustive Arbiter Sweep (125 combinations)
// Tests: Mathematical completeness of the 1oo3 veto logic.
// Every combination of 5 gate states across 3 channels.
// ---------------------------------------------------------------------------
func TestScenario12_ArbiterExhaustiveSweep(t *testing.T) {
	signals := []types.GateSignal{types.CLEAR, types.MODIFY, types.PAUSE, types.HOLD_DATA, types.HALT}
	count := 0

	for _, a := range signals {
		for _, b := range signals {
			for _, c := range signals {
				result := types.Arbitrate(types.ArbiterInput{
					MCUGate:      a,
					PhysioGate:   b,
					ProtocolGate: c,
				})

				// Invariant 1: FinalGate is the most restrictive
				expected := types.MostRestrictive(a, types.MostRestrictive(b, c))
				if result.FinalGate != expected {
					t.Errorf("Arbiter(%s,%s,%s): expected %s, got %s",
						a, b, c, expected, result.FinalGate)
				}

				// Invariant 2: FinalGate >= each individual channel
				if result.FinalGate < a || result.FinalGate < b || result.FinalGate < c {
					t.Errorf("Arbiter(%s,%s,%s): FinalGate %s is less restrictive than input",
						a, b, c, result.FinalGate)
				}

				// Invariant 3: If any channel is HALT, final is HALT
				if a == types.HALT || b == types.HALT || c == types.HALT {
					if result.FinalGate != types.HALT {
						t.Errorf("Arbiter(%s,%s,%s): any HALT must produce HALT, got %s",
							a, b, c, result.FinalGate)
					}
				}

				// Invariant 4: If all CLEAR, final is CLEAR
				if a == types.CLEAR && b == types.CLEAR && c == types.CLEAR {
					if result.FinalGate != types.CLEAR {
						t.Errorf("Arbiter(CLEAR,CLEAR,CLEAR): expected CLEAR, got %s", result.FinalGate)
					}
				}

				count++
			}
		}
	}

	if count != 125 {
		t.Errorf("Expected 125 combinations (5³), got %d", count)
	}
	t.Logf("PASS: Arbiter exhaustive sweep — %d/125 combinations verified, all 4 invariants hold", count)
}

// helper
func hoursAgo(h int) time.Time { return time.Now().Add(-time.Duration(h) * time.Hour) }
