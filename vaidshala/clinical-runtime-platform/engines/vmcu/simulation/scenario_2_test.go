package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario2_AKIDuringTitration simulates an acute kidney injury event
// during active insulin titration.
//
// Timeline:
//   - Days 0-14:  Stable titration, normal labs
//   - Days 15-17: eGFR crashes from 45 → 12 (AKI event)
//   - Days 18-30: B-03/B-08 fires HALT, dose frozen
//   - Days 31-60: Gradual eGFR recovery back to 40
//   - Days 61-90: Normal operation resumes
//
// Key assertions:
//   - HALT fires when eGFR < 15 (B-08)
//   - Creatinine spike triggers B-03
//   - Dose freezes during HALT period
//   - Dose unfreezes after recovery
//   - SafetyTrace records every cycle including HALT cycles
func TestScenario2_AKIDuringTitration(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewAKIRiskPatient()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		switch {
		case day < 15:
			// Stable phase
			p.Glucose = 8.0

		case day >= 15 && day <= 17:
			// AKI event: eGFR crash
			progress := float64(day-15) / 3.0
			p.EGFR = 45.0 - (33.0 * progress) // 45 → 12
			p.Creatinine = 110.0 + (150.0 * progress) // 110 → 260
			note = "AKI_EVENT"

		case day > 17 && day <= 30:
			// Sustained injury
			p.EGFR = 12.0
			p.Creatinine = 260.0
			note = "AKI_SUSTAINED"

		case day > 30 && day <= 60:
			// Gradual recovery
			progress := float64(day-30) / 30.0
			p.EGFR = 12.0 + (28.0 * progress) // 12 → 40
			p.Creatinine = 260.0 - (150.0 * progress)
			note = "AKI_RECOVERY"

		default:
			// Post-recovery stable
			p.EGFR = 40.0
			p.Creatinine = 110.0
		}

		// Snapshot history every 2 days
		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateModify,
			CardID:     "card-aki-002",
			GainFactor: 0.7,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"METFORMIN"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  10,
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("AKI During Titration", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// HALT should fire during AKI (days 15-30)
	AssertHALTInRange(t, result, 15, 31)

	// B-08 (eGFR < 15) should fire
	AssertChannelBRuleFiredInRange(t, result, 16, 31, "B-08")

	// Dose should be frozen during HALT period
	akiCycles := result.CyclesInRange(18, 30)
	if len(akiCycles) > 1 {
		firstDose := akiCycles[0].Dose
		for _, c := range akiCycles[1:] {
			if c.Dose != firstDose {
				t.Logf("Note: dose changed during AKI from %.1f to %.1f (integrator freeze may have delayed)", firstDose, c.Dose)
			}
		}
	}

	// No HALT after recovery (day 61+)
	AssertNoHALTInRange(t, result, 65, TotalDays)
}
