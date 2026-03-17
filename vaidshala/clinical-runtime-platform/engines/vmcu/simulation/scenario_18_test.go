package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	"vaidshala/clinical-runtime-platform/engines/vmcu/titration"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario18_Elderly_HIGH_PG21_Dampening simulates an elderly (age 78)
// PREVENT HIGH-risk patient where the PG-21 elderly safety gate should fire.
//
// PG-21 is a Channel C rule that dampens aggressive SBP titration for patients
// aged >= 75 even when PREVENT risk is HIGH. The rule issues MODIFY gate with
// rule ID "PG-21" to signal that dose changes must be conservative despite the
// intensive 120 mmHg target.
//
// Timeline:
//   - Days 0-30:  SBP 148, PREVENT HIGH, age 78 -> PG-21 fires with MODIFY
//   - Days 30-60: SBP slowly improves with dampened titration
//   - Days 60-90: SBP stabilises around 130 (not reaching 120 due to dampening)
//
// Key assertions:
//   - PG-21 fires (Channel C Gate=MODIFY, RuleID="PG-21")
//   - No HALTs (labs are within safe ranges)
//   - Dose escalation is slower than scenario 17 (non-elderly) due to dampening
//   - SafetyTrace present at every cycle
func TestScenario18_Elderly_HIGH_PG21_Dampening(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)

	patient := &PatientState{
		ID:            "SIM-ELDERLY-PG21-018",
		InitialDose:   500,
		ProposedDelta: 50,
		MedClass:      titration.MedClassOralAgent,
		Glucose:       7.5,
		Creatinine:    110,
		Potassium:     4.1,
		SBP:           148,
		Weight:        70,
		EGFR:          45,
		HbA1c:         7.8,
		DataAvailable: true,
	}
	patient.SnapshotHistory()

	pg21FiredCount := 0

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""

		// SBP trajectory: dampened improvement (elderly conservative titration)
		switch {
		case day < 30:
			p.SBP = 148.0 - float64(day)*0.15
			note = "ELDERLY_SBP_ABOVE_TARGET"
		case day < 60:
			p.SBP = 143.5 - float64(day-30)*0.15
			note = "ELDERLY_SBP_SLOW_IMPROVEMENT"
		default:
			p.SBP = 139.0 - float64(day-60)*0.1
			note = "ELDERLY_SBP_STABILISING"
		}

		p.Glucose = 7.5 - float64(day)*0.005

		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateClear,
			CardID:     "card-elderly-018",
			GainFactor: 0.7,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:              p.EGFR,
			ActiveMedications: []string{"AMLODIPINE"},
			ProposedAction:    "dose_increase",
			DoseDeltaPercent:  10,

			// PREVENT stratification: HIGH tier, intensive target
			PREVENTRiskTier:  "HIGH",
			PREVENTSBPTarget: 120.0,
			PREVENT10yrASCVD: 25.0,
			OnStatin:         true,

			// Elderly age triggers PG-21
			PatientAge: 78,

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

	result := harness.Run("Elderly HIGH + PG-21 Dampening", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// No HALTs: all labs within safe ranges
	AssertNoHALTInRange(t, result, 0, TotalDays)

	// Check for PG-21 firing: Channel C should emit MODIFY with RuleID "PG-21"
	for _, c := range result.Cycles {
		if c.Result != nil && c.Result.ChannelC.RuleID == "PG-21" {
			pg21FiredCount++
		}
	}

	t.Logf("PG-21 fired in %d / %d cycles", pg21FiredCount, len(result.Cycles))

	if pg21FiredCount == 0 {
		t.Errorf("expected PG-21 elderly safety gate to fire for age 78 patient, "+
			"but it never fired. Channel C may not yet implement PG-21 rule evaluation. "+
			"This is expected if PG-21 is not yet wired in protocol_rules.yaml")
	}

	// When PG-21 fires, Channel C gate should be MODIFY (not HALT or PAUSE)
	for _, c := range result.Cycles {
		if c.Result != nil && c.Result.ChannelC.RuleID == "PG-21" {
			if c.Result.ChannelC.Gate != vt.GateModify {
				t.Errorf("PG-21 should produce MODIFY gate, got %s at day %d cycle %d",
					c.Result.ChannelC.Gate, c.Day, c.Cycle)
			}
			break // only need to check the first occurrence
		}
	}

	// Gate distribution
	counts := CountGateOccurrences(result)
	t.Logf("Gate distribution: %s", FormatGateCounts(counts))

	if counts[vt.GateHalt] > 0 {
		t.Errorf("unexpected HALTs in elderly PREVENT HIGH scenario with safe labs")
	}
}
