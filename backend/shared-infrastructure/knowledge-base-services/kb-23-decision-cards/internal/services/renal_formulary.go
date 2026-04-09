package services

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// YAML config shapes (internal — not exported)
// ---------------------------------------------------------------------------

type sharedRulesFile struct {
	StaleEGFR             staleEGFRYAML       `yaml:"stale_egfr"`
	RapidDeclineThreshold float64              `yaml:"rapid_decline_threshold"`
	DrugRules             []models.RenalDrugRule `yaml:"drug_rules"`
}

type staleEGFRYAML struct {
	WarningDays  int `yaml:"warning_days"`
	CriticalDays int `yaml:"critical_days"`
}

type marketOverrideFile struct {
	Market     string            `yaml:"market"`
	StaleEGFR  *marketStaleEGFR  `yaml:"stale_egfr,omitempty"`
	DrugOverrides []drugOverride `yaml:"drug_overrides,omitempty"`
}

type marketStaleEGFR struct {
	HardBlockOnCritical *bool `yaml:"hard_block_on_critical,omitempty"`
}

type drugOverride struct {
	DrugClass                   string  `yaml:"drug_class"`
	SubstituteClass             string  `yaml:"substitute_class,omitempty"`
	AvailabilityFlag            string  `yaml:"availability_flag,omitempty"`
	InitiationMinEGFR           float64 `yaml:"initiation_min_egfr,omitempty"`
	ContinuationMinEGFR         float64 `yaml:"continuation_min_egfr,omitempty"`
	RequiresSpecialistInitiation bool   `yaml:"requires_specialist_initiation,omitempty"`
}

// ---------------------------------------------------------------------------
// Exported types
// ---------------------------------------------------------------------------

// StaleEGFRConfig holds stale-eGFR policy for a market.
type StaleEGFRConfig struct {
	WarningDays         int
	CriticalDays        int
	HardBlockOnCritical bool
}

// RenalFormulary is the loaded, market-merged renal dose rules.
type RenalFormulary struct {
	DrugRules             map[string]*models.RenalDrugRule
	StaleEGFR             StaleEGFRConfig
	RapidDeclineThreshold float64
}

// ---------------------------------------------------------------------------
// Loader
// ---------------------------------------------------------------------------

// LoadRenalFormulary reads the shared renal_dose_rules.yaml and optionally
// merges market-specific overrides (e.g. "india", "australia").
// If market is empty, only the shared baseline is loaded.
func LoadRenalFormulary(configDir, market string) (*RenalFormulary, error) {
	// 1. Load shared baseline
	sharedPath := filepath.Join(configDir, "shared", "renal_dose_rules.yaml")
	data, err := os.ReadFile(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("read shared renal rules: %w", err)
	}

	var shared sharedRulesFile
	if err := yaml.Unmarshal(data, &shared); err != nil {
		return nil, fmt.Errorf("parse shared renal rules: %w", err)
	}

	// Build rule map keyed by drug_class.
	rules := make(map[string]*models.RenalDrugRule, len(shared.DrugRules))
	for i := range shared.DrugRules {
		r := shared.DrugRules[i] // copy
		rules[r.DrugClass] = &r
	}

	f := &RenalFormulary{
		DrugRules: rules,
		StaleEGFR: StaleEGFRConfig{
			WarningDays:         shared.StaleEGFR.WarningDays,
			CriticalDays:        shared.StaleEGFR.CriticalDays,
			HardBlockOnCritical: false, // default; overridden per market
		},
		RapidDeclineThreshold: shared.RapidDeclineThreshold,
	}

	// 2. Merge market overrides (if any)
	if market == "" {
		return f, nil
	}

	overridePath := filepath.Join(configDir, market, "renal_overrides.yaml")
	oData, err := os.ReadFile(overridePath)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil // no overrides for this market — that's fine
		}
		return nil, fmt.Errorf("read %s overrides: %w", market, err)
	}

	var mo marketOverrideFile
	if err := yaml.Unmarshal(oData, &mo); err != nil {
		return nil, fmt.Errorf("parse %s overrides: %w", market, err)
	}

	// Merge stale-eGFR policy
	if mo.StaleEGFR != nil && mo.StaleEGFR.HardBlockOnCritical != nil {
		f.StaleEGFR.HardBlockOnCritical = *mo.StaleEGFR.HardBlockOnCritical
	}

	// Merge per-drug overrides
	for _, ov := range mo.DrugOverrides {
		rule, ok := rules[ov.DrugClass]
		if !ok {
			continue // override for a drug not in the shared set — skip
		}
		if ov.SubstituteClass != "" {
			rule.SubstituteClass = ov.SubstituteClass
		}
		if ov.InitiationMinEGFR > 0 {
			rule.InitiationMinEGFR = ov.InitiationMinEGFR
		}
		if ov.ContinuationMinEGFR > 0 {
			rule.ContinuationMinEGFR = ov.ContinuationMinEGFR
		}
	}

	return f, nil
}

// GetRule returns the merged rule for a drug class, or nil if unknown.
func (f *RenalFormulary) GetRule(drugClass string) *models.RenalDrugRule {
	return f.DrugRules[drugClass]
}
