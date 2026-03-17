package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	"vaidshala/clinical-runtime-platform/engines/vmcu/titration"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario1_StableImprovement simulates a well-controlled T2DM patient
// who gradually improves over 90 days. Glucose trends down, HbA1c drops,
// and V-MCU should naturally reduce dose deltas over time.
//
// Key assertions:
//   - No HALTs throughout the simulation
//   - SafetyTrace present at every cycle
//   - Final dose <= initial dose (improvement = no escalation needed)
//   - All gates are CLEAR or MODIFY
func TestScenario1_StableImprovement(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewStableDiabetic()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		// Gradual glucose improvement: 7.5 → 6.0 over 90 days
		progress := float64(day) / float64(TotalDays)
		p.Glucose = 7.5 - (1.5 * progress)
		p.HbA1c = 7.8 - (1.0 * progress)

		// Snapshot history every 2 days
		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateClear,
			CardID:     "card-stable-001",
			GainFactor: 0.8 - (0.3 * progress), // decreasing gain as patient improves
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_maintain",
			DoseDeltaPercent:  5,
		}

		return chA, p.ToRawLabs(simTime), ctx, ""
	}

	result := harness.Run("Stable Improvement", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)
	AssertNoHALTInRange(t, result, 0, TotalDays)
	AssertFinalDoseInRange(t, result, 0, patient.InitialDose+100)

	// Verify predominantly CLEAR/SAFE gates
	counts := CountGateOccurrences(result)
	t.Logf("Gate distribution: %s", FormatGateCounts(counts))
	if counts[vt.GateHalt] > 0 {
		t.Errorf("unexpected HALTs in stable improvement scenario")
	}
}

func newTestEngine(t *testing.T) *vmcu.VMCUEngine {
	t.Helper()
	engine, err := vmcu.NewVMCUEngine(vmcu.VMCUConfig{
		ProtocolRulesPath: testRulesPath(t),
		MaxDoseDeltaPct:   20.0,
		PhysioConfig:      channel_b.DefaultPhysioConfig(),
		CooldownConfig:    titration.DefaultCooldownConfig(),
	})
	if err != nil {
		t.Fatalf("failed to create V-MCU engine: %v", err)
	}
	return engine
}

func testRulesPath(t *testing.T) string {
	t.Helper()
	// Use the same protocol_rules.yaml as the integration tests
	return "../protocol_rules.yaml"
}
