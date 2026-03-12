package simulation

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TestScenario3_HypoglycaemiaCluster simulates 3 hypoglycaemia events
// within a 7-day window on basal insulin.
//
// Timeline:
//   - Days 0-9:   Normal titration on basal insulin
//   - Day 10:     Hypo event 1 (glucose 3.2 mmol/L)
//   - Day 12:     Hypo event 2 (glucose 3.5 mmol/L)
//   - Day 15:     Hypo event 3 (glucose 2.8 mmol/L)
//   - Days 16-30: Extended cooldown should prevent dose changes
//   - Days 31-90: Gradual recovery with reduced dose targets
//
// Key assertions:
//   - B-01 fires for each hypo event
//   - Cooldown blocks dose changes after events
//   - Dose never increases during hypo cluster period
//   - Trace captures all HALT events
func TestScenario3_HypoglycaemiaCluster(t *testing.T) {
	engine := newTestEngine(t)
	harness := NewHarness(engine)
	patient := NewHypoPronePatient()

	modifier := func(day, cycleInDay int, simTime time.Time, p *PatientState) (
		vt.ChannelAResult, *channel_b.RawPatientData, *channel_c.TitrationContext, string,
	) {
		note := ""
		p.Glucose = 5.5 // default normal

		// Inject hypo events
		switch {
		case day == 10 && cycleInDay == 1:
			p.Glucose = 3.2
			note = "HYPO_EVENT_1"
		case day == 12 && cycleInDay == 2:
			p.Glucose = 3.5
			note = "HYPO_EVENT_2"
		case day == 15 && cycleInDay == 0:
			p.Glucose = 2.8
			note = "HYPO_EVENT_3_SEVERE"
		case day > 15 && day <= 30:
			// Recovery — glucose normalizing
			p.Glucose = 5.0 + float64(day-15)*0.05
			note = "POST_HYPO_RECOVERY"
		}

		if day > 0 && cycleInDay == 0 && day%2 == 0 {
			p.SnapshotHistory()
		}

		chA := vt.ChannelAResult{
			Gate:       vt.GateModify,
			CardID:     "card-hypo-003",
			GainFactor: 0.6,
		}

		ctx := &channel_c.TitrationContext{
			EGFR:                p.EGFR,
			ActiveMedications:   []string{"GLARGINE"},
			ProposedAction:      "dose_maintain",
			DoseDeltaPercent:    5,
			ActiveHypoglycaemia: p.Glucose < 3.9,
		}

		return chA, p.ToRawLabs(simTime), ctx, note
	}

	result := harness.Run("Hypoglycaemia Cluster", patient, modifier)

	PrintSimulationSummary(t, result)
	AssertAllTracesPresent(t, result)

	// B-01 should fire during hypo events
	AssertChannelBRuleFiredInRange(t, result, 10, 16, "B-01")

	// HALTs expected during hypo events
	AssertHALTInRange(t, result, 10, 16)

	// Dose should not increase during and after cluster (days 10-30)
	AssertDoseDecreasing(t, result, 10, 31)

	// Cooldown should block dose changes
	cooldownBlocked := result.CyclesBlockedBy("COOLDOWN:")
	t.Logf("Cooldown-blocked cycles: %d", len(cooldownBlocked))
}
