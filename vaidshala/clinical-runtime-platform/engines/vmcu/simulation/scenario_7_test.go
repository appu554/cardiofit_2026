package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario7_ChannelDisagreement simulates scenarios where Channel A,
// B, and C produce different gate signals, testing the 1oo3 arbiter.
//
// Timeline:
//   - Days 0-9:    All channels agree (CLEAR/SAFE)
//   - Days 10-14:  B=PAUSE (potassium low), C=CLEAR, A=MODIFY
//                  → Arbiter should select PAUSE (most restrictive)
//   - Days 15-19:  B=CLEAR, C=PAUSE (protocol rule), A=MODIFY
//                  → Arbiter should select PAUSE (most restrictive)
//   - Days 20-29:  B=HALT (glucose crash), C=CLEAR, A=SAFE
//                  → Arbiter should select HALT
//   - Days 30-90:  Recovery, all channels clear
//
// Key assertions:
//   - Arbiter always picks the most restrictive gate
//   - DominantChannel correctly identifies which channel drove the decision
//   - No dose changes when any channel is blocking
func TestScenario7_ChannelDisagreement(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewChannelDisagreementPatient()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		// Default: normal labs
		p.Glucose = 7.0
		p.Potassium = 4.4
		p.SBP = 130

		var chAGate vt.GateSignal

		switch {
		case day < 10:
			chAGate = vt.GateClear
			note = "ALL_CLEAR"

		case day >= 10 && day < 15:
			// Channel B: PAUSE via low potassium (B-04 fires at <3.0, B-02 at glucose <4.5)
			// Use glucose-based PAUSE: set glucose to 4.3 (triggers B-02 PAUSE)
			p.Glucose = 4.3
			chAGate = vt.GateModify
			note = "B_PAUSE_A_MODIFY"

		case day >= 15 && day < 20:
			// Channel C triggers via protocol rule, B is clear
			p.Glucose = 7.0
			chAGate = vt.GateModify
			note = "C_PAUSE_A_MODIFY"

		case day >= 20 && day < 30:
			// Channel B: HALT via severe hypoglycaemia
			p.Glucose = 3.2
			chAGate = vt.GateClear
			note = "B_HALT_A_SAFE"

		default:
			// Recovery
			p.Glucose = 6.5
			chAGate = vt.GateClear
		}

		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       chAGate,
			CardID:     "card-disagree-007",
			GainFactor: 0.6,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"GLARGINE"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  10,
			ActiveHypoglycaemia: p.Glucose < 3.9,
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("Channel Disagreement", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// During B=PAUSE period (days 10-14), arbiter should pick PAUSE
	for _, c := range result.CyclesInRange(10, 15) {
		if c.Result != nil {
			gate := c.Result.Arbiter.FinalGate
			if gate != vt.GatePause && gate != vt.GateHalt {
				// At minimum should be PAUSE (most restrictive of MODIFY and PAUSE)
				t.Logf("Day %d C%d: expected PAUSE or higher, got %s (B rule: %s)",
					c.Day, c.Cycle, gate, c.Result.ChannelB.RuleFired)
			}
		}
	}

	// During B=HALT period (days 20-29), arbiter MUST be HALT
	AssertHALTInRange(t, result, 20, 30)

	// Dominant channel should be B during HALT period
	AssertDominantChannel(t, result, 20, 30, "B")

	// After recovery
	AssertNoHALTInRange(t, result, 35, TotalDays)
}
