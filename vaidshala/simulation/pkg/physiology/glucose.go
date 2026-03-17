package physiology

// GlucoseEngine models glucose metabolism including beta-cell decline
// and glucotoxicity feedback loops.
type GlucoseEngine struct {
	cfg *PopulationConfig
}

func NewGlucoseEngine(cfg *PopulationConfig) *GlucoseEngine {
	return &GlucoseEngine{cfg: cfg}
}

// Step advances glucose metabolism by one cycle.
// medicationEffect is the net glucose-lowering effect of active medications (negative = lowering).
func (e *GlucoseEngine) Step(state *PhysiologyState, medicationEffect float64) {
	gc := e.cfg.Glucose

	// Beta-cell decline (irreversible, per-cycle fraction of annual rate)
	dailyDecline := gc.BetaCellDeclineRate / 365.0
	cycleDecline := dailyDecline / float64(e.cfg.Simulation.CyclesPerDay)
	state.BetaCellPct -= state.BetaCellPct * cycleDecline

	// Glucotoxicity accelerates decline when glucose is chronically high
	if state.GlucoseMmol > gc.GlucotoxicityThresholdMmol {
		excess := state.GlucoseMmol - gc.GlucotoxicityThresholdMmol
		state.BetaCellPct -= excess * gc.GlucotoxicityMultiplier * cycleDecline
	}

	// Clamp beta-cell function
	if state.BetaCellPct < 5 {
		state.BetaCellPct = 5
	}

	// Equilibrium drift: glucose drifts upward as beta-cell function declines
	betaCellDeficit := 1.0 - (state.BetaCellPct / 100.0)
	driftPerCycle := gc.EquilibriumDriftRate * betaCellDeficit / float64(e.cfg.Simulation.CyclesPerDay)
	state.GlucoseMmol += driftPerCycle

	// Apply medication effect
	state.GlucoseMmol += medicationEffect

	// Clamp glucose to physiological range
	if state.GlucoseMmol < 2.0 {
		state.GlucoseMmol = 2.0
	}
	if state.GlucoseMmol > 30.0 {
		state.GlucoseMmol = 30.0
	}

	// HbA1c tracks glucose with ~90-day lag (simplified: weighted average)
	// HbA1c moves 1/90th of the way toward the "implied" HbA1c each day
	impliedHbA1c := 2.15 + (state.GlucoseMmol * 0.568) // DCCT formula approximation
	lagFraction := 1.0 / (90.0 * float64(e.cfg.Simulation.CyclesPerDay))
	state.HbA1cPct += (impliedHbA1c - state.HbA1cPct) * lagFraction
}
