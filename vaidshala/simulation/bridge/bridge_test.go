package bridge

import (
	"math"
	"testing"

	"vaidshala/clinical-runtime-platform/engines/vmcu/arbiter"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
	"vaidshala/simulation/pkg/patient"
	simtypes "vaidshala/simulation/pkg/types"
)

// ---------------------------------------------------------------------------
// RuleID normalization tests
// ---------------------------------------------------------------------------

func TestRuleIDNormalization_ExhaustiveMap(t *testing.T) {
	simRules := []string{
		"B-01", "B-02", "B-03", "B-04", "B-04+PG-14",
		"B-05", "B-06", "B-07", "B-08", "B-09",
		"B-10", "B-11", "B-12", "B-13", "B-14",
		"B-15", "B-16", "B-17", "B-18",
		"PG-01", "PG-02", "PG-03", "PG-04", "PG-05",
		"PG-06", "PG-07", "PG-08", "PG-14",
	}
	for _, ruleID := range simRules {
		prodID := NormalizeRuleID(ruleID, DirectionSimToProduction)
		if prodID == "" {
			t.Errorf("no production mapping for sim rule %q", ruleID)
		}
	}
}

func TestRuleIDNormalization_Scenario3Divergence(t *testing.T) {
	prodID := NormalizeRuleID("B-04+PG-14", DirectionSimToProduction)
	if prodID != "B-03-RAAS-SUPPRESSED" {
		t.Errorf("got %q, want %q", prodID, "B-03-RAAS-SUPPRESSED")
	}
}

func TestRuleIDNormalization_ProductionOnly(t *testing.T) {
	prodOnlyRules := []string{
		"B-10", "B-11", "B-19",
		"DA-02", "DA-03", "DA-04", "DA-05", "DA-08",
		"PG-09", "PG-10", "PG-11", "PG-12", "PG-13",
		"PG-15", "PG-16",
	}
	for _, ruleID := range prodOnlyRules {
		simID := NormalizeRuleID(ruleID, DirectionProdToSimulation)
		if simID != "PRODUCTION_ONLY" {
			t.Errorf("prod-only rule %q: got %q, want PRODUCTION_ONLY", ruleID, simID)
		}
	}
}

func TestRuleIDNormalization_RoundTrip(t *testing.T) {
	// Every sim→prod mapping should reverse cleanly (except composite keys
	// like B-04+PG-14 which may not round-trip through the reverse map
	// deterministically if multiple sim IDs map to overlapping prod IDs).
	for simID, prodID := range ruleIDSimToProd {
		backToSim := NormalizeRuleID(prodID, DirectionProdToSimulation)
		// The reverse lookup finds the first match in map iteration order,
		// so we verify the back-mapped sim ID produces the same prod ID.
		backToProd := NormalizeRuleID(backToSim, DirectionSimToProduction)
		if backToProd != prodID {
			t.Errorf("round-trip broken: sim %q → prod %q → sim %q → prod %q",
				simID, prodID, backToSim, backToProd)
		}
	}
}

func TestRuleIDNormalization_UnknownSimPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown simulation rule ID")
		}
	}()
	NormalizeRuleID("BOGUS-99", DirectionSimToProduction)
}

func TestRuleIDNormalization_UnknownProdPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown production rule ID")
		}
	}()
	NormalizeRuleID("BOGUS-99", DirectionProdToSimulation)
}

// ---------------------------------------------------------------------------
// ToSimulationResult tests
// ---------------------------------------------------------------------------

func TestToSimulationResult_BasicMapping(t *testing.T) {
	doseVal := 5.0
	deltaVal := 2.5
	prod := &vt.TitrationCycleResult{
		ChannelA: vt.ChannelAResult{Gate: vt.GateClear, GainFactor: 0.85},
		ChannelB: vt.ChannelBResult{Gate: vt.GatePause, RuleFired: "B-01"},
		ChannelC: vt.ChannelCResult{Gate: vt.GateClear, RuleID: "PG-01"},
		Arbiter:  vt.ArbiterOutput{FinalGate: vt.GatePause, DominantChannel: "B"},
		DoseApplied: &doseVal,
		DoseDelta:   &deltaVal,
		BlockedBy:   "PHYSIO_GATE",
	}

	sim := ToSimulationResult(prod)

	if sim.FinalGate != simtypes.PAUSE {
		t.Errorf("FinalGate: got %v, want PAUSE", sim.FinalGate)
	}
	if sim.DominantChannel != simtypes.ChannelB {
		t.Errorf("DominantChannel: got %q, want %q", sim.DominantChannel, simtypes.ChannelB)
	}
	if !sim.DoseApplied {
		t.Error("DoseApplied: got false, want true")
	}
	if sim.DoseDelta != 2.5 {
		t.Errorf("DoseDelta: got %.2f, want 2.5", sim.DoseDelta)
	}
	if sim.BlockedBy != "PHYSIO_GATE" {
		t.Errorf("BlockedBy: got %q, want PHYSIO_GATE", sim.BlockedBy)
	}
	if sim.PhysioRuleFired != "B-01" {
		t.Errorf("PhysioRuleFired: got %q, want B-01", sim.PhysioRuleFired)
	}
	if sim.ProtocolRuleFired != "PG-01" {
		t.Errorf("ProtocolRuleFired: got %q, want PG-01", sim.ProtocolRuleFired)
	}
	// SafetyTrace sub-fields
	if sim.SafetyTrace.MCUGate != simtypes.CLEAR {
		t.Errorf("SafetyTrace.MCUGate: got %v, want CLEAR", sim.SafetyTrace.MCUGate)
	}
	if sim.SafetyTrace.GainFactor != 0.85 {
		t.Errorf("SafetyTrace.GainFactor: got %.2f, want 0.85", sim.SafetyTrace.GainFactor)
	}
}

func TestToSimulationResult_NilDose(t *testing.T) {
	prod := &vt.TitrationCycleResult{
		ChannelA: vt.ChannelAResult{Gate: vt.GateClear},
		ChannelB: vt.ChannelBResult{Gate: vt.GateHalt},
		ChannelC: vt.ChannelCResult{Gate: vt.GateClear},
		Arbiter:  vt.ArbiterOutput{FinalGate: vt.GateHalt, DominantChannel: "B"},
		DoseApplied: nil,
		DoseDelta:   nil,
	}

	sim := ToSimulationResult(prod)
	if sim.DoseApplied {
		t.Error("DoseApplied: got true, want false (nil dose)")
	}
	if sim.DoseDelta != 0 {
		t.Errorf("DoseDelta: got %.2f, want 0 (nil delta)", sim.DoseDelta)
	}
}

func TestToSimulationResult_DominantChannelMapping(t *testing.T) {
	tests := []struct {
		prodCh string
		wantCh simtypes.Channel
	}{
		{"A", simtypes.ChannelA},
		{"B", simtypes.ChannelB},
		{"C", simtypes.ChannelC},
		{"NONE", simtypes.ChannelB}, // NONE defaults to B
		{"", simtypes.ChannelB},     // empty defaults to B
	}
	for _, tt := range tests {
		prod := &vt.TitrationCycleResult{
			Arbiter: vt.ArbiterOutput{FinalGate: vt.GateClear, DominantChannel: tt.prodCh},
		}
		sim := ToSimulationResult(prod)
		if sim.DominantChannel != tt.wantCh {
			t.Errorf("DominantChannel(%q): got %q, want %q", tt.prodCh, sim.DominantChannel, tt.wantCh)
		}
	}
}

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

// ---------------------------------------------------------------------------
// RAAS tolerance context propagation test (Scenario 3)
// ---------------------------------------------------------------------------

func TestToProductionRawLabs_RAASToleranceContext(t *testing.T) {
	sim := &simtypes.RawPatientData{
		CreatinineCurrent:       120.0,
		CreatininePrevious:      90.0,
		PotassiumCurrent:        4.8,
		CreatinineRiseExplained: true,
	}
	ctx := &simtypes.TitrationContext{
		OliguriaReported: false,
		CKDStage:         "3a",
	}
	ts := PatientTimestamps{}
	prod := ToProductionRawLabs(sim, ctx, ts)

	if prod.PotassiumCurrent == nil || *prod.PotassiumCurrent != 4.8 {
		t.Errorf("PotassiumCurrent not propagated: got %v", prod.PotassiumCurrent)
	}
	if prod.OliguriaReported != false {
		t.Errorf("OliguriaReported should be false")
	}
	if !prod.CreatinineRiseExplained {
		t.Errorf("CreatinineRiseExplained should be true")
	}
}

// ---------------------------------------------------------------------------
// ProductionEngine construction test
// ---------------------------------------------------------------------------

func TestNewProductionEngine_Constructs(t *testing.T) {
	engine, err := NewProductionEngine(
		WithProtocolRulesPath("testdata/protocol_rules.yaml"),
	)
	if err != nil {
		t.Fatalf("NewProductionEngine failed: %v", err)
	}
	if engine == nil {
		t.Fatal("engine is nil")
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

// ---------------------------------------------------------------------------
// Arbiter compatibility test: 125 gate signal combinations
// ---------------------------------------------------------------------------

func TestArbiterCompatibility_125Combinations(t *testing.T) {
	gates := []simtypes.GateSignal{
		simtypes.CLEAR, simtypes.MODIFY, simtypes.PAUSE,
		simtypes.HOLD_DATA, simtypes.HALT,
	}

	passed := 0
	for _, a := range gates {
		for _, b := range gates {
			for _, c := range gates {
				// Simulation arbiter
				simInput := simtypes.ArbiterInput{
					MCUGate:      a,
					PhysioGate:   b,
					ProtocolGate: c,
				}
				simResult := simtypes.Arbitrate(simInput)

				// Production arbiter (via converted gate signals)
				prodInput := vt.ArbiterInput{
					MCUGate:      GateSignalToProduction(a),
					PhysioGate:   GateSignalToProduction(b),
					ProtocolGate: GateSignalToProduction(c),
				}
				prodResult := arbiter.Arbitrate(prodInput)

				// Compare: both must agree on FinalGate
				expectedGate := GateSignalToProduction(simResult.FinalGate)
				if prodResult.FinalGate != expectedGate {
					t.Errorf("Arbiter(%d,%d,%d): sim=%d→%q, prod=%q",
						a, b, c, simResult.FinalGate, expectedGate, prodResult.FinalGate)
				} else {
					passed++
				}
			}
		}
	}
	if passed != 125 {
		t.Fatalf("Arbiter: %d/125 passed", passed)
	}
	t.Logf("Arbiter: 125/125 combinations verified")
}
