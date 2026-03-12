package vmcu

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// ═══════════════════════════════════════════════════════════════════════
// Phase 7.2: Latency Budget Validation
//
// Target latency budget (synchronous):
//   Channel A read:    < 5ms  (pre-provided by caller, ~0ms in-process)
//   Channel B eval:    < 10ms
//   Channel C eval:    < 2ms
//   Arbiter:           < 1ms
//   Total synchronous: < 18ms
//   SafetyTrace write: < 5ms (non-blocking, measured separately)
// ═══════════════════════════════════════════════════════════════════════

func BenchmarkFullCycle(b *testing.B) {
	engine, err := NewVMCUEngine(VMCUConfig{
		ProtocolRulesPath: "protocol_rules.yaml",
		MaxDoseDeltaPct:   20.0,
		PhysioConfig:      channel_b.DefaultPhysioConfig(),
	})
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}

	input := TitrationCycleInput{
		PatientID: "bench-patient",
		ChannelAResult: vt.ChannelAResult{
			Gate:       vt.GateModify,
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
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.RunCycle(input)
	}
}

func TestLatencyBudget(t *testing.T) {
	engine, err := NewVMCUEngine(VMCUConfig{
		ProtocolRulesPath: "protocol_rules.yaml",
		MaxDoseDeltaPct:   20.0,
		PhysioConfig:      channel_b.DefaultPhysioConfig(),
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	input := TitrationCycleInput{
		PatientID: "latency-patient",
		ChannelAResult: vt.ChannelAResult{
			Gate:       vt.GateModify,
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
	}

	// Warm up
	engine.RunCycle(input)

	// Measure 100 cycles and check P95
	const iterations = 100
	durations := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		engine.RunCycle(input)
		durations[i] = time.Since(start)
	}

	// Calculate max (conservative — check every single cycle)
	var maxDuration time.Duration
	for _, d := range durations {
		if d > maxDuration {
			maxDuration = d
		}
	}

	// Assert: total synchronous < 18ms (generous — typical is <1ms in-process)
	if maxDuration > 18*time.Millisecond {
		t.Errorf("latency budget exceeded: max cycle took %v, budget is 18ms", maxDuration)
	}
	t.Logf("latency: max=%v across %d cycles", maxDuration, iterations)
}
