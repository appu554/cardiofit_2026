package bridge

import (
	"math"
	"testing"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
	"vaidshala/simulation/pkg/patient"
	simtypes "vaidshala/simulation/pkg/types"
)

// ---------------------------------------------------------------------------
// GateSignal mapper tests
// ---------------------------------------------------------------------------

func TestGateSignalToProduction_AllValues(t *testing.T) {
	tests := []struct {
		sim  simtypes.GateSignal
		want vt.GateSignal
	}{
		{simtypes.CLEAR, vt.GateClear},
		{simtypes.MODIFY, vt.GateModify},
		{simtypes.PAUSE, vt.GatePause},
		{simtypes.HOLD_DATA, vt.GateHoldData},
		{simtypes.HALT, vt.GateHalt},
	}
	for _, tt := range tests {
		got := GateSignalToProduction(tt.sim)
		if got != tt.want {
			t.Errorf("GateSignalToProduction(%d) = %q, want %q", tt.sim, got, tt.want)
		}
	}
}

func TestGateSignalToSimulation_AllValues(t *testing.T) {
	tests := []struct {
		prod vt.GateSignal
		want simtypes.GateSignal
	}{
		{vt.GateClear, simtypes.CLEAR},
		{vt.GateModify, simtypes.MODIFY},
		{vt.GatePause, simtypes.PAUSE},
		{vt.GateHoldData, simtypes.HOLD_DATA},
		{vt.GateHalt, simtypes.HALT},
	}
	for _, tt := range tests {
		got := GateSignalToSimulation(tt.prod)
		if got != tt.want {
			t.Errorf("GateSignalToSimulation(%q) = %d, want %d", tt.prod, got, tt.want)
		}
	}
}

func TestGateSignalRoundTrip(t *testing.T) {
	for sim := simtypes.CLEAR; sim <= simtypes.HALT; sim++ {
		prod := GateSignalToProduction(sim)
		back := GateSignalToSimulation(prod)
		if back != sim {
			t.Errorf("round-trip failed: %d → %q → %d", sim, prod, back)
		}
	}
}

func TestGateSignalOrderingPreserved(t *testing.T) {
	// Verify that the severity ordering is preserved across the mapping.
	// Production uses Level() for ordering; simulation uses int comparison.
	for i := simtypes.CLEAR; i < simtypes.HALT; i++ {
		prodI := GateSignalToProduction(i)
		prodJ := GateSignalToProduction(i + 1)
		if prodI.Level() >= prodJ.Level() {
			t.Errorf("ordering broken: sim %d (prod %q, level %d) >= sim %d (prod %q, level %d)",
				i, prodI, prodI.Level(), i+1, prodJ, prodJ.Level())
		}
	}
}

func TestGateSignalUnknownFailSafe(t *testing.T) {
	// Unknown simulation value → HALT (fail-safe)
	unknown := simtypes.GateSignal(99)
	if got := GateSignalToProduction(unknown); got != vt.GateHalt {
		t.Errorf("unknown sim → prod: got %q, want %q", got, vt.GateHalt)
	}

	// Unknown production value → HALT (fail-safe)
	if got := GateSignalToSimulation(vt.GateSignal("BOGUS")); got != simtypes.HALT {
		t.Errorf("unknown prod → sim: got %d, want %d", got, simtypes.HALT)
	}
}

// ---------------------------------------------------------------------------
// RawPatientData round-trip tests
// ---------------------------------------------------------------------------

// approxEqual compares floats with tolerance for int→float64→int round-trip.
func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestRawPatientDataRoundTrip_AllArchetypes(t *testing.T) {
	// All 10 standard scenarios + SeasonalHyponatraemia
	scenarios := append(patient.AllScenarios(), patient.SeasonalHyponatraemia())

	for _, vp := range scenarios {
		t.Run(vp.Archetype, func(t *testing.T) {
			// Extract timestamps from simulation's inline fields
			ts := PatientTimestamps{
				LastGlucose:    vp.Labs.GlucoseTimestamp,
				LastCreatinine: vp.Labs.CreatinineTimestamp,
				LastPotassium:  vp.Labs.PotassiumTimestamp,
				LastHbA1c:      vp.Labs.HbA1cTimestamp,
				LastEGFR:       vp.Labs.EGFRTimestamp,
			}

			// Forward: sim → prod
			prod := ToProductionRawLabs(&vp.Labs, &vp.Context, ts)

			// Reverse: prod → sim
			back := ToSimulationRawLabs(prod)

			// Verify key lab values survive the round-trip
			if !approxEqual(back.GlucoseCurrent, vp.Labs.GlucoseCurrent, 0.01) {
				t.Errorf("GlucoseCurrent: got %.2f, want %.2f", back.GlucoseCurrent, vp.Labs.GlucoseCurrent)
			}
			if !approxEqual(back.CreatinineCurrent, vp.Labs.CreatinineCurrent, 0.01) {
				t.Errorf("CreatinineCurrent: got %.2f, want %.2f", back.CreatinineCurrent, vp.Labs.CreatinineCurrent)
			}
			if !approxEqual(back.PotassiumCurrent, vp.Labs.PotassiumCurrent, 0.01) {
				t.Errorf("PotassiumCurrent: got %.2f, want %.2f", back.PotassiumCurrent, vp.Labs.PotassiumCurrent)
			}
			if !approxEqual(back.EGFR, vp.Labs.EGFR, 0.01) {
				t.Errorf("EGFR: got %.2f, want %.2f", back.EGFR, vp.Labs.EGFR)
			}
			if !approxEqual(back.SodiumCurrent, vp.Labs.SodiumCurrent, 0.01) {
				t.Errorf("SodiumCurrent: got %.2f, want %.2f", back.SodiumCurrent, vp.Labs.SodiumCurrent)
			}
			if !approxEqual(back.HbA1c, vp.Labs.HbA1c, 0.01) {
				t.Errorf("HbA1c: got %.2f, want %.2f", back.HbA1c, vp.Labs.HbA1c)
			}
			if !approxEqual(back.Weight, vp.Labs.Weight, 0.01) {
				t.Errorf("Weight: got %.2f, want %.2f", back.Weight, vp.Labs.Weight)
			}
			if !approxEqual(back.WeightPrevious, vp.Labs.WeightPrevious, 0.01) {
				t.Errorf("WeightPrevious: got %.2f, want %.2f", back.WeightPrevious, vp.Labs.WeightPrevious)
			}

			// Int fields: SBP, DBP, HeartRate (int → *float64 → int)
			if back.SBP != vp.Labs.SBP {
				t.Errorf("SBP: got %d, want %d", back.SBP, vp.Labs.SBP)
			}
			if back.DBP != vp.Labs.DBP {
				t.Errorf("DBP: got %d, want %d", back.DBP, vp.Labs.DBP)
			}
			if back.HeartRate != vp.Labs.HeartRate {
				t.Errorf("HeartRate: got %d, want %d", back.HeartRate, vp.Labs.HeartRate)
			}

			// Boolean flags
			if back.BetaBlockerActive != vp.Labs.BetaBlockerActive {
				t.Errorf("BetaBlockerActive: got %v, want %v", back.BetaBlockerActive, vp.Labs.BetaBlockerActive)
			}
			if back.CreatinineRiseExplained != vp.Labs.CreatinineRiseExplained {
				t.Errorf("CreatinineRiseExplained: got %v, want %v", back.CreatinineRiseExplained, vp.Labs.CreatinineRiseExplained)
			}
			if back.RecentDoseIncrease != vp.Labs.RecentDoseIncrease {
				t.Errorf("RecentDoseIncrease: got %v, want %v", back.RecentDoseIncrease, vp.Labs.RecentDoseIncrease)
			}

			// String fields
			if back.HeartRateRegularity != vp.Labs.HeartRateRegularity {
				t.Errorf("HeartRateRegularity: got %q, want %q", back.HeartRateRegularity, vp.Labs.HeartRateRegularity)
			}

			// CreatininePrevious round-trip
			if !approxEqual(back.CreatininePrevious, vp.Labs.CreatininePrevious, 0.01) {
				t.Errorf("CreatininePrevious: got %.2f, want %.2f", back.CreatininePrevious, vp.Labs.CreatininePrevious)
			}

			// GlucosePrevious round-trip (only if original was non-zero)
			if vp.Labs.GlucosePrevious != 0 {
				if !approxEqual(back.GlucosePrevious, vp.Labs.GlucosePrevious, 0.01) {
					t.Errorf("GlucosePrevious: got %.2f, want %.2f", back.GlucosePrevious, vp.Labs.GlucosePrevious)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TitrationContext mapper tests
// ---------------------------------------------------------------------------

func TestToProductionContext_MedicationMapping(t *testing.T) {
	sim := &simtypes.TitrationContext{
		ActiveMedications: []simtypes.ActiveMedication{
			{DrugClass: "METFORMIN", Dose: 1000},
			{DrugClass: "ACEi", Dose: 10},
			{DrugClass: "SGLT2I", Dose: 10},
		},
		CurrentDose:       16.0,
		ProposedDoseDelta: 2.0,
		EGFRCurrent:       65,
		HypoWithin7d:      true,
	}

	prod := ToProductionContext(sim)

	// Check medication list
	if len(prod.ActiveMedications) != 3 {
		t.Fatalf("ActiveMedications: got %d, want 3", len(prod.ActiveMedications))
	}
	expected := []string{"METFORMIN", "ACEi", "SGLT2I"}
	for i, want := range expected {
		if prod.ActiveMedications[i] != want {
			t.Errorf("ActiveMedications[%d]: got %q, want %q", i, prod.ActiveMedications[i], want)
		}
	}

	// Check EGFR
	if prod.EGFR != 65 {
		t.Errorf("EGFR: got %.2f, want 65", prod.EGFR)
	}

	// Check ProposedAction
	if prod.ProposedAction != "dose_increase" {
		t.Errorf("ProposedAction: got %q, want %q", prod.ProposedAction, "dose_increase")
	}

	// Check DoseDeltaPercent (2.0/16.0 * 100 = 12.5%)
	if !approxEqual(prod.DoseDeltaPercent, 12.5, 0.01) {
		t.Errorf("DoseDeltaPercent: got %.2f, want 12.5", prod.DoseDeltaPercent)
	}

	// Check HypoglycaemiaWithin7d
	if !prod.HypoglycaemiaWithin7d {
		t.Error("HypoglycaemiaWithin7d: got false, want true")
	}
}

func TestToProductionContext_DoseDecrease(t *testing.T) {
	sim := &simtypes.TitrationContext{
		CurrentDose:       20.0,
		ProposedDoseDelta: -4.0,
	}
	prod := ToProductionContext(sim)
	if prod.ProposedAction != "dose_decrease" {
		t.Errorf("ProposedAction: got %q, want %q", prod.ProposedAction, "dose_decrease")
	}
	if !approxEqual(prod.DoseDeltaPercent, 20.0, 0.01) {
		t.Errorf("DoseDeltaPercent: got %.2f, want 20.0", prod.DoseDeltaPercent)
	}
}

func TestToProductionContext_DoseHold(t *testing.T) {
	sim := &simtypes.TitrationContext{
		CurrentDose:       20.0,
		ProposedDoseDelta: 0,
	}
	prod := ToProductionContext(sim)
	if prod.ProposedAction != "dose_hold" {
		t.Errorf("ProposedAction: got %q, want %q", prod.ProposedAction, "dose_hold")
	}
	if prod.DoseDeltaPercent != 0 {
		t.Errorf("DoseDeltaPercent: got %.2f, want 0", prod.DoseDeltaPercent)
	}
}

func TestToProductionRawLabs_OnRAASAgent(t *testing.T) {
	sim := &simtypes.RawPatientData{GlucoseCurrent: 8.0}

	// ACEi active → OnRAASAgent true
	ctx := &simtypes.TitrationContext{ACEiActive: true}
	prod := ToProductionRawLabs(sim, ctx, PatientTimestamps{})
	if !prod.OnRAASAgent {
		t.Error("OnRAASAgent: got false when ACEiActive=true")
	}

	// ARB active → OnRAASAgent true
	ctx = &simtypes.TitrationContext{ARBActive: true}
	prod = ToProductionRawLabs(sim, ctx, PatientTimestamps{})
	if !prod.OnRAASAgent {
		t.Error("OnRAASAgent: got false when ARBActive=true")
	}

	// Neither → OnRAASAgent false
	ctx = &simtypes.TitrationContext{}
	prod = ToProductionRawLabs(sim, ctx, PatientTimestamps{})
	if prod.OnRAASAgent {
		t.Error("OnRAASAgent: got true when neither ACEi nor ARB active")
	}
}

func TestToProductionRawLabs_ContextFieldTransfer(t *testing.T) {
	sim := &simtypes.RawPatientData{GlucoseCurrent: 7.5}
	ctx := &simtypes.TitrationContext{
		ThiazideActive:   true,
		Season:           "SUMMER",
		CKDStage:         "3b",
		OliguriaReported: true,
	}

	prod := ToProductionRawLabs(sim, ctx, PatientTimestamps{})

	if !prod.ThiazideActive {
		t.Error("ThiazideActive not transferred")
	}
	if prod.Season != "SUMMER" {
		t.Errorf("Season: got %q, want SUMMER", prod.Season)
	}
	if prod.CKDStage != "3b" {
		t.Errorf("CKDStage: got %q, want 3b", prod.CKDStage)
	}
	if !prod.OliguriaReported {
		t.Error("OliguriaReported not transferred")
	}
}
