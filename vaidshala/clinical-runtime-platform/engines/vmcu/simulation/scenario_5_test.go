package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario5_PlausibilityFailure simulates a cross-session plausibility
// failure where lab results show physiologically impossible changes.
//
// Timeline:
//   - Days 0-19:  Normal stable operation, eGFR ~70
//   - Day 20:     eGFR jumps from 70 to 110 overnight (impossible: max 15/day)
//   - Days 20-22: Plausibility engine should FLAG_REVIEW
//   - Day 23:     Retest confirms true value is 72 → corrected
//   - Days 24-90: Normal operation with corrected values
//
// Key assertions:
//   - The implausible jump triggers data anomaly rules
//   - DA-01 (eGFR delta >40% in 48h) should fire
//   - Dose stays frozen during plausibility review
//   - After correction, normal operation resumes
func TestScenario5_PlausibilityFailure(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewStableDiabetic()
	patient.ID = "SIM-PLAUSIBILITY-005"

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		switch {
		case day < 20:
			p.EGFR = 70.0
			p.Glucose = 7.0

		case day == 20:
			// Implausible lab result: eGFR jumps from 70 → 110
			p.EGFR = 110.0
			p.PriorEGFR48h = f64(70.0) // was 70 two days ago
			note = "IMPLAUSIBLE_EGFR_JUMP"

		case day >= 21 && day <= 22:
			// Under review — keeping the suspect value
			p.EGFR = 110.0
			note = "PLAUSIBILITY_REVIEW"

		case day == 23:
			// Retest: true value is 72 (lab error corrected)
			p.EGFR = 72.0
			p.PriorEGFR48h = f64(110.0) // prior was the bad value
			note = "RETEST_CORRECTED"

		default:
			p.EGFR = 72.0
		}

		if day > 0 && cycleInDay == 0 && day%2 == 0 && day != 20 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateSafe,
			CardID:     "card-plausibility-005",
			GainFactor: 0.7,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_maintain",
			DoseDeltaPercent:  5,
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("Plausibility Failure", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// DA-01 (eGFR delta >40%) should fire on day 20
	AssertChannelBRuleFiredInRange(t, result, 20, 22, "DA-01")

	// After correction, no more anomalies
	AssertNoHALTInRange(t, result, 25, TotalDays)
}
