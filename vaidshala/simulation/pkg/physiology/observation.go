package physiology

import "math/rand"

// ObservationGenerator adds realistic measurement noise to physiological values.
type ObservationGenerator struct {
	cfg *PopulationConfig
	rng *rand.Rand
}

func NewObservationGenerator(cfg *PopulationConfig) *ObservationGenerator {
	return &ObservationGenerator{
		cfg: cfg,
		rng: rand.New(rand.NewSource(cfg.Simulation.RandomSeed)),
	}
}

// Observe returns a noisy observation of the true state.
// The returned state is a copy with noise added — the original is not modified.
func (o *ObservationGenerator) Observe(state PhysiologyState) PhysiologyState {
	obs := state // copy
	nc := o.cfg.ObservationNoise

	obs.GlucoseMmol += o.rng.NormFloat64() * nc.GlucoseStddevMmol
	obs.SBPMmHg += o.rng.NormFloat64() * nc.BPStddevMmHg
	obs.DBPMmHg += o.rng.NormFloat64() * nc.BPStddevMmHg * 0.6 // DBP noise is smaller
	obs.PotassiumMmol += o.rng.NormFloat64() * nc.PotassiumStddevMmol
	obs.CreatinineUmol += o.rng.NormFloat64() * nc.CreatinineStddevUmol
	obs.WeightKg += o.rng.NormFloat64() * nc.WeightStddevKg

	// Clamp to valid ranges after adding noise
	if obs.GlucoseMmol < 1.0 {
		obs.GlucoseMmol = 1.0
	}
	if obs.SBPMmHg < 60 {
		obs.SBPMmHg = 60
	}
	if obs.PotassiumMmol < 2.0 {
		obs.PotassiumMmol = 2.0
	}
	if obs.CreatinineUmol < 20 {
		obs.CreatinineUmol = 20
	}

	return obs
}
