// Package scenarios provides a centralized registry of all V-MCU simulation scenarios
// with metadata (expected outcomes, tags, production-only flag) and production tests
// that validate scenarios against the real V-MCU engine through the bridge adapter.
package scenarios

import (
	"vaidshala/simulation/pkg/patient"
	"vaidshala/simulation/pkg/types"
)

// Scenario describes a single clinical simulation scenario with its expected
// outcome, tags for filtering, and whether it requires the production engine.
type Scenario struct {
	ID        int
	Name      string
	Archetype func() patient.VirtualPatient // returns by value
	Expected  ExpectedOutcome
	Tags      []string
	ProdOnly  bool // true = only runs in production_test.go (not simulation harness)
}

// ExpectedOutcome captures the expected safety outcome of a scenario.
type ExpectedOutcome struct {
	Gate         types.GateSignal
	DoseApplied  bool
	PhysioRule   string
	ProtocolRule string
}

// AllScenarios returns the full registry: 12 standard scenarios + 2 production-only
// (Scenario 13: SeasonalHyponatraemia, Scenario 16: FinerenoneHyperkalemia).
// Scenarios 11 (IntegratorResume) and 12 (ArbiterSweep) are structural tests handled separately.
func AllScenarios() []Scenario {
	return []Scenario{
		{1, "Active Hypoglycaemia", patient.ActiveHypoglycaemia,
			ExpectedOutcome{types.HALT, false, "B-01", "PG-04"},
			[]string{"B-01", "PG-04"}, false},

		{2, "AKI Mid-Titration", patient.AKIMidTitration,
			ExpectedOutcome{types.HALT, false, "B-04", "PG-03"},
			[]string{"B-04", "PG-03"}, false},

		{3, "RAAS Creatinine Tolerance", patient.RAASCreatinineTolerance,
			ExpectedOutcome{types.PAUSE, false, "B-04+PG-14", ""},
			[]string{"B-04", "PG-14"}, false},

		{4, "Data Drop-Out", patient.DataDropOut,
			ExpectedOutcome{types.HOLD_DATA, false, "B-10", ""},
			[]string{"B-10"}, false},

		{5, "Non-Adherent Patient", patient.NonAdherentPatient,
			ExpectedOutcome{types.MODIFY, false, "", ""},
			[]string{"MODIFY"}, false},

		{6, "J-Curve CKD3b", patient.JCurveCKD3b,
			ExpectedOutcome{types.PAUSE, false, "B-12", ""},
			[]string{"B-12"}, false},

		{7, "Dual RAAS", patient.DualRAAS,
			ExpectedOutcome{types.HALT, false, "", "PG-08"},
			[]string{"PG-08"}, false},

		{8, "Hyponatraemia + Thiazide", patient.HyponatraemiaThiazide,
			ExpectedOutcome{types.HALT, false, "B-17", ""},
			[]string{"B-17"}, false},

		{9, "GREEN Trajectory", patient.GreenTrajectory,
			ExpectedOutcome{types.CLEAR, true, "", ""},
			[]string{"CLEAR"}, false},

		{10, "Metformin CKD4", patient.MetforminCKD4,
			ExpectedOutcome{types.HALT, false, "", "PG-01"},
			[]string{"PG-01"}, false},

		{13, "Seasonal Hyponatraemia", patient.SeasonalHyponatraemia,
			ExpectedOutcome{types.PAUSE, false, "B-19", ""},
			[]string{"B-19"}, true},

		{14, "High Glucose Variability", patient.HighGlucoseVariability,
			ExpectedOutcome{types.PAUSE, false, "B-20", ""},
			[]string{"B-20"}, false},

		{15, "ACR A3 Without RAAS", patient.ACRA3NoRAAS,
			ExpectedOutcome{types.HALT, false, "", "PG-17-A3"},
			[]string{"PG-17"}, false},

		{16, "Finerenone Hyperkalemia", patient.FinerenoneHyperkalemia,
			ExpectedOutcome{types.HALT, false, "B-21", ""},
			[]string{"B-21"}, true}, // prodOnly: B-21 only in production Channel B
	}
}
