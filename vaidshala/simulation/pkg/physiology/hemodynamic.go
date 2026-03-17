package physiology

// HemodynamicEngine models blood pressure response to medications.
type HemodynamicEngine struct {
	cfg *PopulationConfig
}

func NewHemodynamicEngine(cfg *PopulationConfig) *HemodynamicEngine {
	return &HemodynamicEngine{cfg: cfg}
}

// MedicationBPEffect represents the BP-lowering effect of active medications.
type MedicationBPEffect struct {
	ACEiOrARBActive   bool
	ThiazideActive    bool
	CCBActive         bool
	BetaBlockerActive bool
	SGLT2iActive      bool
}

// Step advances hemodynamics by one cycle.
func (e *HemodynamicEngine) Step(state *PhysiologyState, meds MedicationBPEffect) {
	hc := e.cfg.Hemodynamic

	// Natural SBP drift (tends upward without treatment)
	driftPerCycle := hc.SBPDriftRate / float64(e.cfg.Simulation.CyclesPerDay)
	state.SBPMmHg += driftPerCycle

	// Medication effects (applied as fraction per cycle toward equilibrium)
	totalEffect := 0.0
	if meds.ACEiOrARBActive {
		totalEffect += hc.ACEiARBEffectMmHg
	}
	if meds.ThiazideActive {
		totalEffect += hc.ThiazideEffectMmHg
	}
	if meds.CCBActive {
		totalEffect += hc.CCBEffectMmHg
	}
	if meds.BetaBlockerActive {
		totalEffect += hc.BetaBlockerEffectMmHg
	}
	if meds.SGLT2iActive {
		totalEffect += hc.SGLT2iBPEffectMmHg
	}

	// Apply medication effect gradually (1/7 per day, fraction per cycle)
	effectPerCycle := totalEffect / (7.0 * float64(e.cfg.Simulation.CyclesPerDay))
	state.SBPMmHg += effectPerCycle

	// DBP tracks SBP with ~0.6 ratio from baseline
	state.DBPMmHg = 40 + (state.SBPMmHg-40)*0.55

	// Clamp to physiological range
	if state.SBPMmHg < 70 {
		state.SBPMmHg = 70
	}
	if state.SBPMmHg > 220 {
		state.SBPMmHg = 220
	}
}
