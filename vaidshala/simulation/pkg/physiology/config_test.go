package physiology

import "testing"

func TestLoadPopulationConfig_Default(t *testing.T) {
	cfg, err := LoadPopulationConfig("../../config/default.yaml")
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}
	if cfg.Glucose.EquilibriumDriftRate != 0.10 {
		t.Errorf("glucose drift rate: got %v, want 0.10", cfg.Glucose.EquilibriumDriftRate)
	}
	if cfg.Simulation.RandomSeed != 42 {
		t.Errorf("random seed: got %v, want 42", cfg.Simulation.RandomSeed)
	}
	if cfg.Autonomy.SingleStepPct != 0.20 {
		t.Errorf("single step pct: got %v, want 0.20", cfg.Autonomy.SingleStepPct)
	}
	if cfg.Population != "default" {
		t.Errorf("population: got %q, want %q", cfg.Population, "default")
	}
}

func TestLoadPopulationConfig_SouthAsian(t *testing.T) {
	cfg, err := LoadPopulationConfig("../../config/default.yaml", "../../config/south_asian.yaml")
	if err != nil {
		t.Fatalf("failed to load south_asian config: %v", err)
	}
	if cfg.BodyComposition.VisceralFatInsulinThreshold != 1.2 {
		t.Errorf("VFI threshold: got %v, want 1.2", cfg.BodyComposition.VisceralFatInsulinThreshold)
	}
	if cfg.Glucose.CarbBaselineG != 350 {
		t.Errorf("carb baseline: got %v, want 350", cfg.Glucose.CarbBaselineG)
	}
	if cfg.Glucose.EquilibriumDriftRate != 0.10 {
		t.Errorf("drift rate should be default 0.10, got %v", cfg.Glucose.EquilibriumDriftRate)
	}
	if cfg.Population != "south_asian" {
		t.Errorf("population: got %q, want %q", cfg.Population, "south_asian")
	}
}

func TestLoadPopulationConfig_NoFiles(t *testing.T) {
	_, err := LoadPopulationConfig()
	if err == nil {
		t.Error("expected error for no config files")
	}
}

func TestLoadPopulationConfig_MissingFile(t *testing.T) {
	_, err := LoadPopulationConfig("../../config/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
