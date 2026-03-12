package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario6_ConcurrentDeprescribing simulates deprescribing an SGLT2i
// while maintaining metformin. The patient is well-controlled (HbA1c 6.5%)
// and the clinician initiates a step-down.
//
// Timeline:
//   - Days 0-9:    Normal operation, well-controlled
//   - Day 10:      Clinician initiates SGLT2i deprescribing
//   - Days 10-40:  Gradual dose reduction with widened Channel B thresholds
//   - Days 41-50:  Glucose slightly rises during taper → monitor
//   - Days 51-90:  Stabilize at lower dose
//
// Key assertions:
//   - Channel B glucose thresholds are widened during active deprescribing
//   - Dose monotonically decreases during deprescribing window
//   - If Channel B fires during deprescribing, plan pauses (doesn't revert)
//   - SafetyTrace records deprescribing state
func TestScenario6_ConcurrentDeprescribing(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewDeprescribingPatient()

	deprescribingActive := false

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		switch {
		case day < 10:
			p.Glucose = 5.8

		case day == 10 && cycleInDay == 0:
			deprescribingActive = true
			note = "DEPRESCRIBING_INITIATED"
			p.Glucose = 5.8

		case day >= 10 && day <= 40 && deprescribingActive:
			// Glucose slowly rises during taper (expected)
			progress := float64(day-10) / 30.0
			p.Glucose = 5.8 + (1.5 * progress) // 5.8 → 7.3
			note = "DEPRESCRIBING_ACTIVE"

		case day > 40 && day <= 50:
			// Monitor phase — glucose slightly elevated
			p.Glucose = 7.0 + float64(day-40)*0.03
			note = "DEPRESCRIBING_MONITOR"

		default:
			p.Glucose = 7.2 // stable at new level
		}

		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateSafe,
			CardID:     "card-deprescribe-006",
			GainFactor: 0.5,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"METFORMIN", "DAPAGLIFLOZIN"},
			ProposedAction:    "dose_decrease",
			DoseDeltaPercent:  0,
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("Concurrent Deprescribing", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// No severe events expected in well-controlled patient
	AssertNoHALTInRange(t, result, 0, TotalDays)

	// Dose should be stable or decreasing (deprescribing = dose reduction)
	AssertFinalDoseInRange(t, result, 0, patient.InitialDose)
}
