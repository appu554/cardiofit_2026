package physiology

// BodyCompositionEngine models weight changes from SGLT2i and GLP-1RA.
type BodyCompositionEngine struct {
	cfg *PopulationConfig
}

func NewBodyCompositionEngine(cfg *PopulationConfig) *BodyCompositionEngine {
	return &BodyCompositionEngine{cfg: cfg}
}

// BodyMedications tracks medications affecting body composition.
type BodyMedications struct {
	SGLT2iActive bool
	GLP1RAActive bool
}

// Step advances body composition by one cycle.
func (e *BodyCompositionEngine) Step(state *PhysiologyState, meds BodyMedications) {
	bc := e.cfg.BodyComposition

	// SGLT2i caloric loss → weight loss (~0.5kg/week at 300 kcal/day)
	if meds.SGLT2iActive {
		// 7700 kcal ≈ 1 kg fat
		dailyWeightLoss := bc.SGLT2iCalorieLossKcal / 7700.0
		cycleWeightLoss := dailyWeightLoss / float64(e.cfg.Simulation.CyclesPerDay)
		state.WeightKg -= cycleWeightLoss
	}

	// GLP-1RA appetite reduction → weight loss
	if meds.GLP1RAActive {
		// Assume baseline caloric intake ~2000 kcal/day
		dailyDeficit := 2000 * bc.GLP1RAAppetiteReductionPct
		dailyWeightLoss := dailyDeficit / 7700.0
		cycleWeightLoss := dailyWeightLoss / float64(e.cfg.Simulation.CyclesPerDay)
		state.WeightKg -= cycleWeightLoss
	}

	// Weight floor
	if state.WeightKg < 40 {
		state.WeightKg = 40
	}

	// Visceral fat index tracks insulin resistance
	state.VisceralFatIdx = bc.VisceralFatInsulinThreshold * (state.WeightKg / 80.0)
}
