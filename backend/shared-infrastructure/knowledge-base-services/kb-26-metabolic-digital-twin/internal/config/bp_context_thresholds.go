package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BPContextThresholds is the loaded configuration consumed by the BP
// context classifier. Fields are flattened from the YAML hierarchy for
// ergonomic access at call sites.
type BPContextThresholds struct {
	// Clinic thresholds
	ClinicSBPElevated   float64
	ClinicDBPElevated   float64
	ClinicSBPElevatedDM float64
	ClinicDBPElevatedDM float64

	// Home thresholds
	HomeSBPElevated float64
	HomeDBPElevated float64

	// Data requirements
	MinClinicReadings int
	ClinicMaxAgeDays  int
	MinHomeReadings   int
	MinHomeDays       int
	HomeMaxAgeDays    int

	// White-coat effect
	WCEClinicallySignificant float64
	WCESevere                float64

	// Selection bias
	MinHomeForConfidence int
	FlagIfReadingsBelow  int
}

// rawSharedConfig matches the YAML structure of bp_context_thresholds.yaml.
type rawSharedConfig struct {
	Thresholds struct {
		Clinic struct {
			SBPElevated   float64 `yaml:"sbp_elevated"`
			DBPElevated   float64 `yaml:"dbp_elevated"`
			SBPElevatedDM float64 `yaml:"sbp_elevated_dm"`
			DBPElevatedDM float64 `yaml:"dbp_elevated_dm"`
		} `yaml:"clinic"`
		Home struct {
			SBPElevated float64 `yaml:"sbp_elevated"`
			DBPElevated float64 `yaml:"dbp_elevated"`
		} `yaml:"home"`
	} `yaml:"thresholds"`
	DataRequirements struct {
		Clinic struct {
			MinReadings int `yaml:"min_readings"`
			MaxAgeDays  int `yaml:"max_age_days"`
		} `yaml:"clinic"`
		Home struct {
			MinReadings int `yaml:"min_readings"`
			MinDays     int `yaml:"min_days"`
			MaxAgeDays  int `yaml:"max_age_days"`
		} `yaml:"home"`
	} `yaml:"data_requirements"`
	WhiteCoatEffect struct {
		ClinicallySignificant float64 `yaml:"clinically_significant"`
		Severe                float64 `yaml:"severe"`
	} `yaml:"white_coat_effect"`
	SelectionBias struct {
		MinHomeForConfidence int `yaml:"min_home_readings_for_confidence"`
		FlagIfReadingsBelow  int `yaml:"flag_if_readings_below"`
	} `yaml:"selection_bias"`
}

// rawOverrideConfig matches the YAML structure of *_overrides.yaml.
type rawOverrideConfig struct {
	ThresholdsOverride struct {
		Clinic *struct {
			SBPElevated   *float64 `yaml:"sbp_elevated"`
			DBPElevated   *float64 `yaml:"dbp_elevated"`
			SBPElevatedDM *float64 `yaml:"sbp_elevated_dm"`
			DBPElevatedDM *float64 `yaml:"dbp_elevated_dm"`
		} `yaml:"clinic"`
	} `yaml:"thresholds_override"`
	WhiteCoatEffectOverride *struct {
		ClinicallySignificant *float64 `yaml:"clinically_significant"`
		Severe                *float64 `yaml:"severe"`
	} `yaml:"white_coat_effect_override"`
}

// LoadBPContextThresholds reads shared thresholds from
// {configDir}/shared/bp_context_thresholds.yaml and applies the market
// override from {configDir}/{market}/bp_context_overrides.yaml if present.
// Unknown markets are NOT errors — only shared values are used.
func LoadBPContextThresholds(configDir, market string) (*BPContextThresholds, error) {
	sharedPath := filepath.Join(configDir, "shared", "bp_context_thresholds.yaml")
	sharedBytes, err := os.ReadFile(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("read shared BP context thresholds: %w", err)
	}

	var shared rawSharedConfig
	if err := yaml.Unmarshal(sharedBytes, &shared); err != nil {
		return nil, fmt.Errorf("parse shared BP context thresholds: %w", err)
	}

	t := &BPContextThresholds{
		ClinicSBPElevated:        shared.Thresholds.Clinic.SBPElevated,
		ClinicDBPElevated:        shared.Thresholds.Clinic.DBPElevated,
		ClinicSBPElevatedDM:      shared.Thresholds.Clinic.SBPElevatedDM,
		ClinicDBPElevatedDM:      shared.Thresholds.Clinic.DBPElevatedDM,
		HomeSBPElevated:          shared.Thresholds.Home.SBPElevated,
		HomeDBPElevated:          shared.Thresholds.Home.DBPElevated,
		MinClinicReadings:        shared.DataRequirements.Clinic.MinReadings,
		ClinicMaxAgeDays:         shared.DataRequirements.Clinic.MaxAgeDays,
		MinHomeReadings:          shared.DataRequirements.Home.MinReadings,
		MinHomeDays:              shared.DataRequirements.Home.MinDays,
		HomeMaxAgeDays:           shared.DataRequirements.Home.MaxAgeDays,
		WCEClinicallySignificant: shared.WhiteCoatEffect.ClinicallySignificant,
		WCESevere:                shared.WhiteCoatEffect.Severe,
		MinHomeForConfidence:     shared.SelectionBias.MinHomeForConfidence,
		FlagIfReadingsBelow:      shared.SelectionBias.FlagIfReadingsBelow,
	}

	overridePath := filepath.Join(configDir, market, "bp_context_overrides.yaml")
	overrideBytes, err := os.ReadFile(overridePath)
	if errors.Is(err, os.ErrNotExist) {
		// Unknown market or no override file — return shared-only thresholds.
		return t, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s override: %w", market, err)
	}

	var override rawOverrideConfig
	if err := yaml.Unmarshal(overrideBytes, &override); err != nil {
		return nil, fmt.Errorf("parse %s override: %w", market, err)
	}

	if override.ThresholdsOverride.Clinic != nil {
		c := override.ThresholdsOverride.Clinic
		if c.SBPElevated != nil {
			t.ClinicSBPElevated = *c.SBPElevated
		}
		if c.DBPElevated != nil {
			t.ClinicDBPElevated = *c.DBPElevated
		}
		if c.SBPElevatedDM != nil {
			t.ClinicSBPElevatedDM = *c.SBPElevatedDM
		}
		if c.DBPElevatedDM != nil {
			t.ClinicDBPElevatedDM = *c.DBPElevatedDM
		}
	}
	if override.WhiteCoatEffectOverride != nil {
		w := override.WhiteCoatEffectOverride
		if w.ClinicallySignificant != nil {
			t.WCEClinicallySignificant = *w.ClinicallySignificant
		}
		if w.Severe != nil {
			t.WCESevere = *w.Severe
		}
	}

	return t, nil
}
