package physiology

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PopulationConfig holds all tunable physiology coefficients.
type PopulationConfig struct {
	Population string `yaml:"population"`
	Version    string `yaml:"version"`
	Extends    string `yaml:"extends,omitempty"`

	BodyComposition  BodyCompositionConfig  `yaml:"body_composition"`
	Glucose          GlucoseConfig          `yaml:"glucose"`
	Hemodynamic      HemodynamicConfig      `yaml:"hemodynamic"`
	Renal            RenalConfig            `yaml:"renal"`
	ObservationNoise ObservationNoiseConfig `yaml:"observation_noise"`
	Simulation       SimulationConfig       `yaml:"simulation"`
	Autonomy         AutonomyConfig         `yaml:"autonomy"`
}

type BodyCompositionConfig struct {
	VisceralFatInsulinThreshold float64 `yaml:"visceral_fat_insulin_threshold"`
	MuscleSensitivityWeight     float64 `yaml:"muscle_sensitivity_weight"`
	SGLT2iCalorieLossKcal       float64 `yaml:"sglt2i_calorie_loss_kcal"`
	GLP1RAAppetiteReductionPct  float64 `yaml:"glp1ra_appetite_reduction_pct"`
}

type GlucoseConfig struct {
	EquilibriumDriftRate       float64 `yaml:"equilibrium_drift_rate"`
	BetaCellDeclineRate        float64 `yaml:"beta_cell_decline_rate"`
	GlucotoxicityThresholdMmol float64 `yaml:"glucotoxicity_threshold_mmol"`
	GlucotoxicityMultiplier    float64 `yaml:"glucotoxicity_multiplier"`
	CarbBaselineG              float64 `yaml:"carb_baseline_g"`
	PPBGSpikeCoefficient       float64 `yaml:"ppbg_spike_coefficient"`
}

type HemodynamicConfig struct {
	SBPDriftRate          float64 `yaml:"sbp_drift_rate"`
	ACEiARBEffectMmHg     float64 `yaml:"acei_arb_effect_mmhg"`
	ThiazideEffectMmHg    float64 `yaml:"thiazide_effect_mmhg"`
	CCBEffectMmHg         float64 `yaml:"ccb_effect_mmhg"`
	BetaBlockerEffectMmHg float64 `yaml:"beta_blocker_effect_mmhg"`
	SGLT2iBPEffectMmHg    float64 `yaml:"sglt2i_bp_effect_mmhg"`
}

type RenalConfig struct {
	NaturalEGFRDeclinePerYear float64 `yaml:"natural_egfr_decline_per_year"`
	ACEiARBProtectionPct      float64 `yaml:"acei_arb_protection_pct"`
	SGLT2iProtectionPct       float64 `yaml:"sglt2i_protection_pct"`
	GLP1RAProtectionPct       float64 `yaml:"glp1ra_protection_pct"`
	UncontrolledSBPThreshold  float64 `yaml:"uncontrolled_sbp_threshold"`
	HighGlucoseThresholdMmol  float64 `yaml:"high_glucose_threshold_mmol"`
}

type ObservationNoiseConfig struct {
	GlucoseStddevMmol    float64 `yaml:"glucose_stddev_mmol"`
	BPStddevMmHg         float64 `yaml:"bp_stddev_mmhg"`
	PotassiumStddevMmol  float64 `yaml:"potassium_stddev_mmol"`
	CreatinineStddevUmol float64 `yaml:"creatinine_stddev_umol"`
	WeightStddevKg       float64 `yaml:"weight_stddev_kg"`
}

type SimulationConfig struct {
	RandomSeed   int64 `yaml:"random_seed"`
	TotalDays    int   `yaml:"total_days"`
	CyclesPerDay int   `yaml:"cycles_per_day"`
}

type AutonomyConfig struct {
	SingleStepPct float64 `yaml:"single_step_pct"`
	CumulativePct float64 `yaml:"cumulative_pct"`
}

// LoadPopulationConfig loads one or more YAML files, merging overrides onto defaults.
// First file is the base, subsequent files override non-zero values.
func LoadPopulationConfig(paths ...string) (*PopulationConfig, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one config path required")
	}

	var cfg PopulationConfig

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
	}

	return &cfg, nil
}
