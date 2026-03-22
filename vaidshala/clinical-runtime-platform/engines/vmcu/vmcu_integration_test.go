package vmcu

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	"vaidshala/clinical-runtime-platform/engines/vmcu/metabolic"
	"vaidshala/clinical-runtime-platform/engines/vmcu/titration"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// rulesPath returns the absolute path to protocol_rules.yaml.
func rulesPath(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "protocol_rules.yaml")
}

func newTestEngine(t *testing.T) *VMCUEngine {
	t.Helper()
	engine, err := NewVMCUEngine(VMCUConfig{
		ProtocolRulesPath: rulesPath(t),
		MaxDoseDeltaPct:   20.0,
		PhysioConfig:      channel_b.DefaultPhysioConfig(),
		CooldownConfig:    titration.DefaultCooldownConfig(),
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	return engine
}

// ═══════════════════════════════════════════════════════════════════════
// Phase 7.1: Full Titration Cycle Integration Test
// ═══════════════════════════════════════════════════════════════════════

func TestFullCycle_AllClear_DoseApplied(t *testing.T) {
	engine := newTestEngine(t)

	result, trace := engine.RunCycle(TitrationCycleInput{
		PatientID: "patient-001",
		ChannelAResult: vt.ChannelAResult{
			Gate:       vt.GateModify,
			CardID:     "card-123",
			GainFactor: 0.8,
		},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(75),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:              75,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  10,
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	// Channel A: MODIFY (from KB-23)
	if result.ChannelA.Gate != vt.GateModify {
		t.Errorf("Channel A should be MODIFY, got %s", result.ChannelA.Gate)
	}
	// Channel B: CLEAR (all labs normal)
	if result.ChannelB.Gate != vt.GateClear {
		t.Errorf("Channel B should be CLEAR, got %s", result.ChannelB.Gate)
	}
	// Channel C: CLEAR (no rules triggered)
	if result.ChannelC.Gate != vt.GateClear {
		t.Errorf("Channel C should be CLEAR, got %s", result.ChannelC.Gate)
	}
	// Arbiter: MODIFY (most restrictive = Channel A)
	if result.Arbiter.FinalGate != vt.GateModify {
		t.Errorf("Arbiter final gate should be MODIFY, got %s", result.Arbiter.FinalGate)
	}
	if result.Arbiter.DominantChannel != "A" {
		t.Errorf("dominant channel should be A, got %s", result.Arbiter.DominantChannel)
	}
	// Dose should be applied (MODIFY is non-blocking)
	if result.DoseApplied == nil {
		t.Fatal("dose should be applied when gate is MODIFY")
	}
	if result.BlockedBy != "" {
		t.Errorf("should not be blocked, got %s", result.BlockedBy)
	}
	// Gain factor applied: proposedDelta(10) * gain(0.8) = 8.0
	if *result.DoseDelta != 8.0 {
		t.Errorf("dose delta should be 8.0 (10 * 0.8), got %f", *result.DoseDelta)
	}

	// SafetyTrace populated
	if trace.TraceID == "" {
		t.Error("trace ID should not be empty")
	}
	if trace.PatientID != "patient-001" {
		t.Errorf("trace patient ID should be patient-001, got %s", trace.PatientID)
	}
	if trace.MCUGate != "MODIFY" {
		t.Errorf("trace MCU gate should be MODIFY, got %s", trace.MCUGate)
	}
	if trace.ProtocolRuleVsn == "" {
		t.Error("protocol rule version hash should be present")
	}
	if trace.GainFactor != 0.8 {
		t.Errorf("trace gain factor should be 0.8, got %f", trace.GainFactor)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Phase 7.3: Safety Scenario Tests
// ═══════════════════════════════════════════════════════════════════════

func TestScenario_Glucose35_ChannelBHalt(t *testing.T) {
	engine := newTestEngine(t)

	result, trace := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-002",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(3.5),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(75),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:           75,
			ProposedAction: "dose_increase",
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	if result.ChannelB.Gate != vt.GateHalt {
		t.Errorf("glucose 3.5 should trigger Channel B HALT, got %s", result.ChannelB.Gate)
	}
	if result.Arbiter.FinalGate != vt.GateHalt {
		t.Errorf("final gate should be HALT, got %s", result.Arbiter.FinalGate)
	}
	if result.DoseApplied != nil {
		t.Error("dose should NOT be applied when HALT")
	}
	if result.BlockedBy == "" {
		t.Error("blocked_by should be set")
	}
	if trace.BlockedBy == "" {
		t.Error("trace blocked_by should be set")
	}
}

func TestScenario_EGFR25_Metformin_ChannelCHalt(t *testing.T) {
	engine := newTestEngine(t)

	result, _ := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-003",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(7.0),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(25),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:              25,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  10,
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	if result.ChannelC.Gate != vt.GateHalt {
		t.Errorf("eGFR 25 + Metformin should trigger Channel C HALT (PG-01), got %s", result.ChannelC.Gate)
	}
	if result.Arbiter.FinalGate != vt.GateHalt {
		t.Errorf("final gate should be HALT, got %s", result.Arbiter.FinalGate)
	}
	if result.DoseApplied != nil {
		t.Error("dose should NOT be applied when HALT")
	}
}

func TestScenario_KB23Pause_AcuteIllness(t *testing.T) {
	engine := newTestEngine(t)

	result, _ := engine.RunCycle(TitrationCycleInput{
		PatientID: "patient-004",
		ChannelAResult: vt.ChannelAResult{
			Gate:      vt.GatePause,
			Rationale: "Acute illness perturbation active",
		},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(75),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:           75,
			ProposedAction: "dose_increase",
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	if result.Arbiter.FinalGate != vt.GatePause {
		t.Errorf("KB-23 PAUSE should propagate, got %s", result.Arbiter.FinalGate)
	}
	if result.Arbiter.DominantChannel != "A" {
		t.Errorf("dominant should be A (KB-23 drove the PAUSE), got %s", result.Arbiter.DominantChannel)
	}
	if result.DoseApplied != nil {
		t.Error("dose should NOT be applied when PAUSE")
	}
}

func TestScenario_EGFRDrop45pct_HoldData(t *testing.T) {
	engine := newTestEngine(t)

	prior := 52.0
	result, trace := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-005",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(28),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
			EGFRPrior48h:      &prior,
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:           28,
			ProposedAction: "dose_increase",
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	if result.ChannelB.Gate != vt.GateHoldData {
		t.Errorf("eGFR 46%% drop should trigger HOLD_DATA, got %s", result.ChannelB.Gate)
	}
	if !result.ChannelB.IsAnomaly {
		t.Error("should flag as anomaly")
	}
	if result.Arbiter.FinalGate != vt.GateHoldData {
		t.Errorf("final gate should be HOLD_DATA, got %s", result.Arbiter.FinalGate)
	}
	if result.DoseApplied != nil {
		t.Error("dose should be deferred on HOLD_DATA")
	}
	if trace.PhysioGate != "HOLD_DATA" {
		t.Errorf("trace should record HOLD_DATA, got %s", trace.PhysioGate)
	}
}

func TestScenario_Potassium28_InsulinIncrease_BothChannelsHalt(t *testing.T) {
	engine := newTestEngine(t)

	result, _ := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-006",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(3.5), // B-01 HALT (also sets ActiveHypoglycaemia for C)
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(2.8), // B-04 HALT
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(75),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:           75,
			ProposedAction: "insulin_increase",
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	// B fires first (B-01 glucose < 3.9)
	if result.ChannelB.Gate != vt.GateHalt {
		t.Errorf("Channel B should HALT, got %s", result.ChannelB.Gate)
	}
	// C also fires (PG-04 insulin into hypo, because B-01 sets ActiveHypoglycaemia)
	if result.ChannelC.Gate != vt.GateHalt {
		t.Errorf("Channel C should HALT (PG-04), got %s", result.ChannelC.Gate)
	}
	// Arbiter: HALT, Dominant=B (physiology priority)
	if result.Arbiter.FinalGate != vt.GateHalt {
		t.Errorf("final gate should be HALT, got %s", result.Arbiter.FinalGate)
	}
	if result.Arbiter.DominantChannel != "B" {
		t.Errorf("dominant should be B (physiology priority), got %s", result.Arbiter.DominantChannel)
	}
}

func TestScenario_AllThreeChannelsHalt(t *testing.T) {
	engine := newTestEngine(t)

	result, trace := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-007",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateHalt, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(3.0), // B-01 HALT
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(25),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:              25,
			ActiveMedications: []string{"METFORMIN"}, // PG-01 HALT
			ProposedAction:    "dose_increase",
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	if result.ChannelA.Gate != vt.GateHalt {
		t.Errorf("Channel A should HALT, got %s", result.ChannelA.Gate)
	}
	if result.ChannelB.Gate != vt.GateHalt {
		t.Errorf("Channel B should HALT, got %s", result.ChannelB.Gate)
	}
	if result.ChannelC.Gate != vt.GateHalt {
		t.Errorf("Channel C should HALT, got %s", result.ChannelC.Gate)
	}
	if result.Arbiter.FinalGate != vt.GateHalt {
		t.Errorf("final gate should be HALT, got %s", result.Arbiter.FinalGate)
	}
	if result.Arbiter.DominantChannel != "B" {
		t.Errorf("all HALT → dominant should be B, got %s", result.Arbiter.DominantChannel)
	}
	if trace.FinalGate != "HALT" {
		t.Errorf("trace final gate should be HALT, got %s", trace.FinalGate)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// FlushTraces integration
// ═══════════════════════════════════════════════════════════════════════

func TestFlushTraces_AccumulatesAndClears(t *testing.T) {
	engine := newTestEngine(t)

	input := TitrationCycleInput{
		PatientID:      "patient-flush",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80),
			PotassiumCurrent:  channel_b.Float64Ptr(4.5),
			SBPCurrent:        channel_b.Float64Ptr(120),
			WeightKgCurrent:   channel_b.Float64Ptr(70),
			EGFRCurrent:       channel_b.Float64Ptr(75),
			HbA1cCurrent:      channel_b.Float64Ptr(7.0),
		},
		CurrentDose:   100.0,
		ProposedDelta: 5.0,
	}

	engine.RunCycle(input)
	engine.RunCycle(input)
	engine.RunCycle(input)

	traces := engine.FlushTraces()
	if len(traces) != 3 {
		t.Errorf("expected 3 traces, got %d", len(traces))
	}

	// Second flush should be empty
	traces = engine.FlushTraces()
	if len(traces) != 0 {
		t.Errorf("expected 0 traces after flush, got %d", len(traces))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Integrator Freeze/Resume Wiring Tests
// ═══════════════════════════════════════════════════════════════════════

func TestWiring_IntegratorFreezesOnHalt(t *testing.T) {
	engine := newTestEngine(t)

	// Run a cycle that triggers HALT → integrator should freeze
	engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-freeze",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateHalt, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:  time.Now(),
			CreatinineCurrent: channel_b.Float64Ptr(80), PotassiumCurrent: channel_b.Float64Ptr(4.5),
			SBPCurrent: channel_b.Float64Ptr(120), WeightKgCurrent: channel_b.Float64Ptr(70),
			EGFRCurrent: channel_b.Float64Ptr(75), HbA1cCurrent: channel_b.Float64Ptr(7.0),
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
	})

	state, frozenDose := engine.GetIntegratorState("patient-freeze")
	if state != titration.IntegratorFrozen {
		t.Errorf("integrator should be FROZEN after HALT, got %s", state)
	}
	if frozenDose != 100.0 {
		t.Errorf("frozen dose should be 100.0, got %f", frozenDose)
	}
}

func TestWiring_IntegratorResumesOnClear(t *testing.T) {
	engine := newTestEngine(t)

	normalLabs := &channel_b.RawPatientData{
		GlucoseCurrent:    channel_b.Float64Ptr(6.5),
		GlucoseTimestamp:  time.Now(),
		CreatinineCurrent: channel_b.Float64Ptr(80), PotassiumCurrent: channel_b.Float64Ptr(4.5),
		SBPCurrent: channel_b.Float64Ptr(120), WeightKgCurrent: channel_b.Float64Ptr(70),
		EGFRCurrent: channel_b.Float64Ptr(75), HbA1cCurrent: channel_b.Float64Ptr(7.0),
	}

	// Cycle 1: HALT → freeze
	engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-resume",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateHalt, GainFactor: 1.0},
		RawLabs:        normalLabs,
		CurrentDose:    100.0,
		ProposedDelta:  10.0,
	})

	// Cycle 2: CLEAR → resume (dose restarts from frozen value)
	result, _ := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-resume",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs:        normalLabs,
		CurrentDose:    200.0, // caller might have drifted — engine should use frozen dose
		ProposedDelta:  10.0,
	})

	state, _ := engine.GetIntegratorState("patient-resume")
	if state != titration.IntegratorActive {
		t.Errorf("integrator should be ACTIVE after resume, got %s", state)
	}

	// Re-entry protocol is in MONITORING phase → dose changes blocked
	if result.DoseApplied != nil {
		t.Error("dose should be blocked during REENTRY:MONITORING phase")
	}
	if result.BlockedBy == "" {
		t.Error("blocked_by should indicate REENTRY:MONITORING")
	}

	// Verify re-entry is in MONITORING
	phase := engine.GetReentryPhase("patient-resume")
	if phase != titration.ReentryMonitoring {
		t.Errorf("should be in MONITORING phase, got %s", phase)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Re-entry Protocol Wiring Tests
// ═══════════════════════════════════════════════════════════════════════

func TestWiring_ReentryProtocolActivatesAfterResume(t *testing.T) {
	engine := newTestEngine(t)

	normalLabs := &channel_b.RawPatientData{
		GlucoseCurrent:    channel_b.Float64Ptr(6.5),
		GlucoseTimestamp:  time.Now(),
		CreatinineCurrent: channel_b.Float64Ptr(80), PotassiumCurrent: channel_b.Float64Ptr(4.5),
		SBPCurrent: channel_b.Float64Ptr(120), WeightKgCurrent: channel_b.Float64Ptr(70),
		EGFRCurrent: channel_b.Float64Ptr(75), HbA1cCurrent: channel_b.Float64Ptr(7.0),
	}

	// Cycle 1: HALT → freeze
	engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-reentry",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateHalt, GainFactor: 1.0},
		RawLabs:        normalLabs,
		CurrentDose:    100.0,
		ProposedDelta:  10.0,
	})

	// Cycle 2: CLEAR → resume, re-entry activates in MONITORING
	engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-reentry",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs:        normalLabs,
		CurrentDose:    100.0,
		ProposedDelta:  10.0,
	})

	phase := engine.GetReentryPhase("patient-reentry")
	if phase != titration.ReentryMonitoring {
		t.Errorf("re-entry should be MONITORING after first resume cycle, got %s", phase)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Cooldown Wiring Tests
// ═══════════════════════════════════════════════════════════════════════

func TestWiring_CooldownBlocksSecondDose(t *testing.T) {
	engine := newTestEngine(t)

	normalLabs := &channel_b.RawPatientData{
		GlucoseCurrent:    channel_b.Float64Ptr(6.5),
		GlucoseTimestamp:  time.Now(),
		CreatinineCurrent: channel_b.Float64Ptr(80), PotassiumCurrent: channel_b.Float64Ptr(4.5),
		SBPCurrent: channel_b.Float64Ptr(120), WeightKgCurrent: channel_b.Float64Ptr(70),
		EGFRCurrent: channel_b.Float64Ptr(75), HbA1cCurrent: channel_b.Float64Ptr(7.0),
	}
	ctx := &channel_c.TitrationContext{
		EGFR: 75, ProposedAction: "dose_increase", DoseDeltaPercent: 10,
	}

	// Cycle 1: dose applied with MedClass → cooldown starts
	// CurrentDose must be below BASAL_INSULIN autonomy ceiling (100 units)
	// so the proposed dose (80+10=90) clears the absolute ceiling check.
	result1, _ := engine.RunCycle(TitrationCycleInput{
		PatientID:        "patient-cd",
		ChannelAResult:   vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs:          normalLabs,
		TitrationContext: ctx,
		CurrentDose:      80.0,
		ProposedDelta:    10.0,
		MedClass:         titration.MedClassBasalInsulin,
	})
	if result1.DoseApplied == nil {
		t.Fatal("first dose should be applied")
	}

	// Cycle 2: same med class within 48h → cooldown blocks
	result2, _ := engine.RunCycle(TitrationCycleInput{
		PatientID:        "patient-cd",
		ChannelAResult:   vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs:          normalLabs,
		TitrationContext: ctx,
		CurrentDose:      90.0,
		ProposedDelta:    10.0,
		MedClass:         titration.MedClassBasalInsulin,
	})
	if result2.DoseApplied != nil {
		t.Error("second dose should be blocked by cooldown")
	}
	if result2.BlockedBy == "" {
		t.Error("blocked_by should indicate COOLDOWN")
	}

	// Verify cooldown is active via engine API
	if !engine.IsOnCooldown("patient-cd", titration.MedClassBasalInsulin) {
		t.Error("basal insulin should be on cooldown")
	}
	// Different med class should NOT be on cooldown
	if engine.IsOnCooldown("patient-cd", titration.MedClassRapidInsulin) {
		t.Error("rapid insulin should NOT be on cooldown")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Metabolic Engine (KB-24) Wiring Tests
// ═══════════════════════════════════════════════════════════════════════

func TestWiring_MetabolicEngineReducesGainDuringDawn(t *testing.T) {
	engine := newTestEngine(t)

	dawnTime := time.Date(2026, 3, 6, 5, 0, 0, 0, time.UTC) // 5 AM = dawn window

	result, _ := engine.RunCycle(TitrationCycleInput{
		PatientID:      "patient-dawn",
		ChannelAResult: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 1.0},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:    channel_b.Float64Ptr(8.0), // elevated fasting glucose
			GlucoseTimestamp:  dawnTime,
			CreatinineCurrent: channel_b.Float64Ptr(80), PotassiumCurrent: channel_b.Float64Ptr(4.5),
			SBPCurrent: channel_b.Float64Ptr(120), WeightKgCurrent: channel_b.Float64Ptr(70),
			EGFRCurrent: channel_b.Float64Ptr(75), HbA1cCurrent: channel_b.Float64Ptr(7.0),
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR: 75, ProposedAction: "dose_increase", DoseDeltaPercent: 10,
		},
		CurrentDose:   100.0,
		ProposedDelta: 10.0,
		MetabolicInput: &metabolic.MetabolicInput{
			GlucoseCurrent:   8.0,
			GlucoseTimestamp:  dawnTime,
			TimeOfDay:         dawnTime,
			IsPreBreakfast:    true,
			BMI:               28.0,
			CurrentInsulinDose: 40.0,
			InsulinType:       "basal",
			DaysOnTherapy:     30,
		},
	})

	if result.DoseApplied == nil {
		t.Fatal("dose should be applied during dawn")
	}
	// Gain = 1.0 * 0.7 (dawn adj) = 0.7
	// Delta = 10 * 0.7 = 7.0
	expectedDelta := 7.0
	if *result.DoseDelta != expectedDelta {
		t.Errorf("dawn gain should reduce delta to %.1f, got %.1f", expectedDelta, *result.DoseDelta)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Stop (graceful shutdown) test
// ═══════════════════════════════════════════════════════════════════════

func TestWiring_StopIsIdempotent(t *testing.T) {
	engine := newTestEngine(t)
	// Stop without async tracer should not panic
	engine.Stop()
	engine.Stop()
}
