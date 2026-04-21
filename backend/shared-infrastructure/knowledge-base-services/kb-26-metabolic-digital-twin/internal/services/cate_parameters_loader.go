package services

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"kb-26-metabolic-digital-twin/internal/models"
)

// CATEParameters is the struct representation of cate_parameters.yaml (Task 2).
// Loaded once at service startup; consumed by the calibration monitor and the
// (Sprint 2) CATE estimate handler.
type CATEParameters struct {
	Version     string `yaml:"version"`
	OverlapBand struct {
		Default   bandYAML            `yaml:"default"`
		PerCohort map[string]bandYAML `yaml:"per_cohort"`
	} `yaml:"overlap_band"`
	Bootstrap struct {
		NResamples int     `yaml:"n_resamples"`
		CILevel    float64 `yaml:"ci_level"`
	} `yaml:"bootstrap"`
	MinTrainingN int `yaml:"min_training_n"`
	Calibration  struct {
		AlarmThresholdAbsDiff float64 `yaml:"alarm_threshold_abs_diff"`
		RollingWindowDays     int     `yaml:"rolling_window_days"`
		MinMatchedPairs       int     `yaml:"min_matched_pairs"`
	} `yaml:"calibration"`
	PrimaryLearner map[string]primaryLearnerYAML `yaml:"primary_learner"`
}

type bandYAML struct {
	Floor   float64 `yaml:"floor"`
	Ceiling float64 `yaml:"ceiling"`
}

type primaryLearnerYAML struct {
	Learner     string `yaml:"learner"`
	HorizonDays int    `yaml:"horizon_days"`
}

// LoadCATEParameters reads and parses cate_parameters.yaml. Returns a defaults-only
// CATEParameters + an error if the file is missing or malformed — caller decides
// whether to fail startup or proceed with defaults (matching the PAI/attribution
// loader pattern).
func LoadCATEParameters(path string) (*CATEParameters, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DefaultCATEParameters(), fmt.Errorf("read cate params: %w", err)
	}
	var p CATEParameters
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return DefaultCATEParameters(), fmt.Errorf("parse cate params: %w", err)
	}
	return &p, nil
}

// DefaultCATEParameters returns a conservative fallback used when the YAML is
// missing or unparseable. Values match the Sprint 1 defaults documented in
// cate_parameters.yaml.
func DefaultCATEParameters() *CATEParameters {
	p := &CATEParameters{Version: "1.0.0-fallback"}
	p.OverlapBand.Default = bandYAML{Floor: 0.05, Ceiling: 0.95}
	p.OverlapBand.PerCohort = map[string]bandYAML{}
	p.Bootstrap.NResamples = 500
	p.Bootstrap.CILevel = 0.90
	p.MinTrainingN = 40
	p.Calibration.AlarmThresholdAbsDiff = 0.05
	p.Calibration.RollingWindowDays = 90
	p.Calibration.MinMatchedPairs = 20
	p.PrimaryLearner = map[string]primaryLearnerYAML{}
	return p
}

// BandForCohort returns the cohort-specific overlap band, falling back to default.
func (p *CATEParameters) BandForCohort(cohortID string) models.OverlapBand {
	if b, ok := p.OverlapBand.PerCohort[cohortID]; ok {
		return models.OverlapBand{Floor: b.Floor, Ceiling: b.Ceiling}
	}
	return models.OverlapBand{Floor: p.OverlapBand.Default.Floor, Ceiling: p.OverlapBand.Default.Ceiling}
}

// CalibrationConfigFromYAML projects the YAML block onto the CalibrationConfig
// struct expected by NewCATECalibrationMonitor.
func (p *CATEParameters) CalibrationConfigFromYAML() CalibrationConfig {
	return CalibrationConfig{
		AbsDiffAlarm:      p.Calibration.AlarmThresholdAbsDiff,
		RollingWindowDays: p.Calibration.RollingWindowDays,
		MinMatchedPairs:   p.Calibration.MinMatchedPairs,
	}
}
