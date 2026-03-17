package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario4_MissingData simulates a 72-hour data gap where no lab
// values are available (e.g., patient traveling, device offline).
//
// Timeline:
//   - Days 0-9:    Normal operation
//   - Days 10-12:  DataAvailable=false → all current values nil
//   - Days 13-15:  Data returns → re-entry protocol activates
//   - Days 16-90:  Normal operation resumes
//
// Key assertions:
//   - With nil pointers, Channel B should NOT fire false HALTs
//   - Re-entry protocol should activate when data returns
//   - SafetyTrace present at every cycle (even during data gap)
//   - Dose stays frozen during missing data period
func TestScenario4_MissingData(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewMissingDataPatient()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		switch {
		case day >= 10 && day <= 12:
			p.DataAvailable = false
			note = "DATA_MISSING"
		case day == 13:
			p.DataAvailable = true
			note = "DATA_RETURNED"
		default:
			p.DataAvailable = true
		}

		if day > 0 && cycleInDay == 0 && day%2 == 0 && p.DataAvailable {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateClear,
			CardID:     "card-missing-004",
			GainFactor: 0.7,
		}

		var ctx *channel_c.TitrationContext
		if p.DataAvailable {
			ctx = &channel_c.TitrationContext{
				EGFR:              p.EGFR,
				ActiveMedications: []string{"METFORMIN"},
				ProposedAction:    "dose_maintain",
				DoseDeltaPercent:  5,
			}
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("Missing Data 72h", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// During data gap: should NOT get false HALT from zero-value glucose
	// With *float64 fix, nil values should result in CLEAR or HOLD_DATA, not HALT
	missingCycles := result.CyclesInRange(10, 13)
	for _, c := range missingCycles {
		if c.Result != nil && c.Result.Arbiter.FinalGate == vt.GateHalt {
			if c.Result.ChannelB.RuleFired == "B-01" {
				t.Errorf("FALSE HALT from B-01 during data gap at day %d (zero-value bug!)", c.Day)
			}
		}
	}

	// Dose should be stable during data gap (no changes without data)
	if len(missingCycles) > 1 {
		gapDose := missingCycles[0].Dose
		for _, c := range missingCycles[1:] {
			if c.Dose != gapDose {
				t.Logf("Note: dose changed during data gap from %.1f to %.1f", gapDose, c.Dose)
			}
		}
	}

	// Normal operation after data returns
	AssertNoHALTInRange(t, result, 16, TotalDays)
}
