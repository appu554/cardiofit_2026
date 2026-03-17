package physiology

import "math"

// RenalEngine models eGFR decline and renoprotective medication effects.
type RenalEngine struct {
	cfg *PopulationConfig
}

func NewRenalEngine(cfg *PopulationConfig) *RenalEngine {
	return &RenalEngine{cfg: cfg}
}

// RenalMedications tracks which renoprotective medications are active.
type RenalMedications struct {
	ACEiOrARBActive bool
	SGLT2iActive    bool
	GLP1RAActive    bool
}

// Step advances renal function by one cycle.
func (e *RenalEngine) Step(state *PhysiologyState, meds RenalMedications, currentSBP, currentGlucose float64) {
	rc := e.cfg.Renal

	// Base eGFR decline per cycle (config stores positive rate; negate for subtraction)
	declinePerCycle := -rc.NaturalEGFRDeclinePerYear / (365.0 * float64(e.cfg.Simulation.CyclesPerDay))

	// Renoprotective medication effects (reduce rate of decline)
	protection := 0.0
	if meds.ACEiOrARBActive {
		protection += rc.ACEiARBProtectionPct
	}
	if meds.SGLT2iActive {
		protection += rc.SGLT2iProtectionPct
	}
	if meds.GLP1RAActive {
		protection += rc.GLP1RAProtectionPct
	}
	// Cap protection at 80% (can't fully stop decline)
	if protection > 0.80 {
		protection = 0.80
	}

	effectiveDecline := declinePerCycle * (1.0 - protection)

	// Accelerated decline from uncontrolled hypertension
	if currentSBP > rc.UncontrolledSBPThreshold {
		excess := (currentSBP - rc.UncontrolledSBPThreshold) / 20.0 // per 20 mmHg above threshold
		effectiveDecline *= (1.0 + excess*0.5)
	}

	// Accelerated decline from high glucose
	if currentGlucose > rc.HighGlucoseThresholdMmol {
		excess := (currentGlucose - rc.HighGlucoseThresholdMmol) / 5.0
		effectiveDecline *= (1.0 + excess*0.3)
	}

	state.EGFRMlMin += effectiveDecline

	// Clamp eGFR
	if state.EGFRMlMin < 5 {
		state.EGFRMlMin = 5
	}
	if state.EGFRMlMin > 120 {
		state.EGFRMlMin = 120
	}

	// Creatinine inversely tracks eGFR (simplified CKD-EPI inverse)
	// Creatinine ≈ 7200 / eGFR (rough approximation for 80kg patient)
	if state.EGFRMlMin > 0 {
		state.CreatinineUmol = 7200.0 / state.EGFRMlMin
	}

	// Potassium: affected by renal function and ACEi/ARB
	basePotassium := 4.2
	if state.EGFRMlMin < 30 {
		basePotassium += (30 - state.EGFRMlMin) * 0.03 // rises as eGFR drops
	}
	if meds.ACEiOrARBActive {
		basePotassium += 0.3 // ACEi/ARB raises potassium
	}
	// Smooth toward target
	lagFraction := 1.0 / (3.0 * float64(e.cfg.Simulation.CyclesPerDay))
	state.PotassiumMmol += (basePotassium - state.PotassiumMmol) * lagFraction

	// Clamp potassium
	state.PotassiumMmol = math.Max(2.5, math.Min(7.0, state.PotassiumMmol))
}
