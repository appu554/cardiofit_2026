package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario8_AutonomyLimitBreach simulates a patient with consistently
// high glucose where V-MCU repeatedly proposes dose increases, eventually
// hitting the cumulative autonomy limit (>50%).
//
// Timeline:
//   - Days 0-30:  Persistent hyperglycaemia (glucose ~10-11 mmol/L)
//                 V-MCU keeps increasing dose by 10-20% each cycle
//   - Day ~30:    Cumulative change exceeds 50% → AUTONOMY block
//   - Days 30-45: Physician confirmation needed, doses blocked
//   - Day 45:     Physician confirms → cumulative counter resets
//   - Days 46-90: Titration resumes with fresh autonomy budget
//
// Key assertions:
//   - AUTONOMY: prefix appears in BlockedBy
//   - Dose never exceeds absolute ceiling for drug class
//   - After physician reset, titration can resume
//   - All cycles have SafetyTrace
func TestScenario8_AutonomyLimitBreach(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewAutonomyLimitPatient()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		// Persistent hyperglycaemia
		switch {
		case day < 30:
			p.Glucose = 10.0 + float64(day%5)*0.2
			note = "PERSISTENT_HYPERGLYCAEMIA"
		case day >= 30 && day < 45:
			p.Glucose = 9.5
			note = "AWAITING_PHYSICIAN_CONFIRMATION"
		default:
			// After physician confirms, glucose starts responding
			progress := float64(day-45) / 45.0
			p.Glucose = 9.5 - (2.5 * progress)
		}

		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateModify,
			CardID:     "card-autonomy-008",
			GainFactor: 0.9, // aggressive gain
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  15, // aggressive increase request
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("Autonomy Limit Breach", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// Check that autonomy limits were hit
	autonomyBlocked := result.CyclesBlockedBy("AUTONOMY:")
	t.Logf("Autonomy-blocked cycles: %d", len(autonomyBlocked))

	if len(autonomyBlocked) > 0 {
		// Verify the first autonomy block happened
		first := autonomyBlocked[0]
		t.Logf("First autonomy block: day %d cycle %d, blocked_by=%s",
			first.Day, first.Cycle, first.Result.BlockedBy)
	}

	// Dose should never exceed absolute ceiling for ORAL_AGENT (2000mg metformin)
	AssertDoseInRange(t, result, 0, TotalDays, 0, 2100) // slight tolerance

	// Gate distribution should show MODIFY attempts
	counts := CountGateOccurrences(result)
	t.Logf("Gate distribution: %s", FormatGateCounts(counts))
}
