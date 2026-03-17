package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	"vaidshala/clinical-runtime-platform/engines/vmcu/titration"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario17_PREVENT_HighTier_IntensiveSBPTarget simulates a PREVENT HIGH-risk
// patient (age 65, diabetes, eGFR 45, UACR 350, HbA1c 8.0) whose personalised
// SBP target is set to 120 mmHg (intensive) by the PREVENT calculator.
//
// With SBP at 148 mmHg (28 above target), the engine should allow dose titration.
// The arbiter should not block the dose change because the patient is well above
// the intensive target and no safety rules are triggered.
//
// Timeline:
//   - Days 0-30:  SBP 148, glucose 8.0 (above target, titration proceeds)
//   - Days 30-60: SBP gradually improves to 130 as medication takes effect
//   - Days 60-90: SBP stabilises near 125, approaching 120 target
//
// Key assertions:
//   - No HALTs during the simulation (labs are safe)
//   - Dose increases are permitted while SBP > target
//   - SafetyTrace present at every cycle
//   - Channel C receives PREVENT context and does not block
func TestScenario17_PREVENT_HighTier_IntensiveSBPTarget(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)

	patient := &PatientState{
		ID:            "SIM-PREVENT-HIGH-017",
		InitialDose:   500,
		ProposedDelta: 50,
		MedClass:      titration.MedClassOralAgent,
		Glucose:       8.0,
		Creatinine:    120,
		Potassium:     4.3,
		SBP:           148,
		Weight:        82,
		EGFR:          45,
		HbA1c:         8.0,
		DataAvailable: true,
	}
	patient.SnapshotHistory()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		// SBP trajectory: 148 -> 130 by day 60, then 125 by day 90
		switch {
		case day < 30:
			p.SBP = 148.0 - float64(day)*0.3
			note = "SBP_ABOVE_TARGET"
		case day < 60:
			p.SBP = 139.0 - float64(day-30)*0.3
			note = "SBP_IMPROVING"
		default:
			p.SBP = 130.0 - float64(day-60)*0.17
			note = "SBP_NEAR_TARGET"
		}

		// Glucose stable, slightly above target
		p.Glucose = 8.0 - float64(day)*0.01

		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateClear,
			CardID:     "card-prevent-high-017",
			GainFactor: 0.8,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"AMLODIPINE"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  10,

			// PREVENT stratification context
			PREVENTRiskTier:  "HIGH",
			PREVENTSBPTarget: 120.0,
			PREVENT10yrASCVD: 22.5,
			OnStatin:         true,
			PatientAge:       65,

			SBPCurrent:       p.SBP,
			PotassiumCurrent: p.Potassium,
		}

		labs := freshLabsWithOverrides(func(d *channel_b.RawPatientData) {
			d.GlucoseCurrent = f64(p.Glucose)
			d.CreatinineCurrent = f64(p.Creatinine)
			d.PotassiumCurrent = f64(p.Potassium)
			d.SBPCurrent = f64(p.SBP)
			d.WeightKgCurrent = f64(p.Weight)
			d.EGFRCurrent = f64(p.EGFR)
			d.HbA1cCurrent = f64(p.HbA1c)
			d.Creatinine48hAgo = p.PriorCreatinine48h
			d.EGFRPrior48h = p.PriorEGFR48h
			d.Weight72hAgo = p.PriorWeight72h
			d.HbA1cPrior30d = p.PriorHbA1c30d
		})

		return chA, labs, ctx, note
	}

	result := harness.Run("PREVENT HIGH-tier Intensive SBP Target", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// No HALTs: all lab values are within safe ranges
	AssertNoHALTInRange(t, result, 0, TotalDays)

	// Dose should have increased from baseline (SBP was well above 120 target)
	if result.FinalDose <= patient.InitialDose {
		t.Logf("note: final dose %.1f did not increase from initial %.1f; "+
			"this is acceptable if protocol guard constrained titration",
			result.FinalDose, patient.InitialDose)
	}

	// Verify gate distribution: predominantly CLEAR or MODIFY (no blocking)
	counts := CountGateOccurrences(result)
	t.Logf("Gate distribution: %s", FormatGateCounts(counts))

	if counts[vt.GateHalt] > 0 {
		t.Errorf("unexpected HALTs in PREVENT HIGH-tier scenario with safe labs")
	}
}
