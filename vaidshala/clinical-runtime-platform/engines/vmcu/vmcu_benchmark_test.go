package vmcu

import (
	"sort"
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

	now := time.Now()
	input := TitrationCycleInput{
		PatientID: "bench-patient",
		ChannelAResult: vt.ChannelAResult{
			Gate:       vt.GateModify,
			GainFactor: 0.8,
		},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:           channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:         now,
			CreatinineCurrent:        channel_b.Float64Ptr(80),
			PotassiumCurrent:         channel_b.Float64Ptr(4.5),
			SBPCurrent:               channel_b.Float64Ptr(120),
			WeightKgCurrent:          channel_b.Float64Ptr(70),
			EGFRCurrent:              channel_b.Float64Ptr(75),
			HbA1cCurrent:             channel_b.Float64Ptr(7.0),
			EGFRLastMeasuredAt:       &now,
			HbA1cLastMeasuredAt:      &now,
			CreatinineLastMeasuredAt: &now,
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

// BenchmarkRunCycle is the benchmark referenced in the Final Proposal
// for the 13µs/cycle V-MCU latency claim. Runs a full cycle through
// all three channels + arbiter + integrator + titration engine.
func BenchmarkRunCycle(b *testing.B) {
	engine, err := NewVMCUEngine(VMCUConfig{
		ProtocolRulesPath: "protocol_rules.yaml",
		MaxDoseDeltaPct:   20.0,
		PhysioConfig:      channel_b.DefaultPhysioConfig(),
	})
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}

	now := time.Now()
	input := TitrationCycleInput{
		PatientID: "bench-13us",
		ChannelAResult: vt.ChannelAResult{
			Gate:       vt.GateClear,
			GainFactor: 1.0,
		},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:           channel_b.Float64Ptr(7.0),
			GlucoseTimestamp:         now,
			CreatinineCurrent:        channel_b.Float64Ptr(70),
			PotassiumCurrent:         channel_b.Float64Ptr(4.0),
			SBPCurrent:               channel_b.Float64Ptr(125),
			WeightKgCurrent:          channel_b.Float64Ptr(65),
			EGFRCurrent:              channel_b.Float64Ptr(85),
			HbA1cCurrent:             channel_b.Float64Ptr(6.8),
			EGFRLastMeasuredAt:       &now,
			HbA1cLastMeasuredAt:      &now,
			CreatinineLastMeasuredAt: &now,
		},
		TitrationContext: &channel_c.TitrationContext{
			EGFR:              85,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  5,
		},
		CurrentDose:   500.0,
		ProposedDelta: 25.0,
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

	now := time.Now()
	input := TitrationCycleInput{
		PatientID: "latency-patient",
		ChannelAResult: vt.ChannelAResult{
			Gate:       vt.GateModify,
			GainFactor: 0.8,
		},
		RawLabs: &channel_b.RawPatientData{
			GlucoseCurrent:           channel_b.Float64Ptr(6.5),
			GlucoseTimestamp:         now,
			CreatinineCurrent:        channel_b.Float64Ptr(80),
			PotassiumCurrent:         channel_b.Float64Ptr(4.5),
			SBPCurrent:               channel_b.Float64Ptr(120),
			WeightKgCurrent:          channel_b.Float64Ptr(70),
			EGFRCurrent:              channel_b.Float64Ptr(75),
			HbA1cCurrent:             channel_b.Float64Ptr(7.0),
			EGFRLastMeasuredAt:       &now,
			HbA1cLastMeasuredAt:      &now,
			CreatinineLastMeasuredAt: &now,
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

	// Measure 1000 cycles and check P99
	const iterations = 1000
	durations := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		engine.RunCycle(input)
		durations[i] = time.Since(start)
	}

	// Sort for percentile calculation
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	p50 := durations[iterations/2]
	p99 := durations[int(float64(iterations)*0.99)]
	maxDuration := durations[iterations-1]

	// Assert: P99 < 100µs (in-process, no I/O — validates the 13µs claim is achievable)
	if p99 > 100*time.Microsecond {
		t.Errorf("P99 latency exceeded: P99=%v, budget is 100µs", p99)
	}

	// Assert: max < 18ms (generous safety budget — handles GC pauses)
	if maxDuration > 18*time.Millisecond {
		t.Errorf("max latency budget exceeded: max cycle took %v, budget is 18ms", maxDuration)
	}
	t.Logf("latency across %d cycles: P50=%v  P99=%v  max=%v", iterations, p50, p99, maxDuration)
}
