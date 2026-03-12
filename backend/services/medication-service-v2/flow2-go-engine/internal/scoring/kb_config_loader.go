// Package scoring provides KB configuration loading for compare-and-rank profiles
package scoring

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// KBConfig represents the complete KB configuration
type KBConfig struct {
	Profiles           map[string]ProfileConfig    `yaml:"profiles"`
	Penalties          PenaltiesConfig             `yaml:"penalties"`
	EfficacyBonuses    EfficacyBonusesConfig      `yaml:"efficacy_bonuses"`
	KnockoutRules      KnockoutRulesConfig        `yaml:"knockout_rules"`
	TieBreakers        []string                   `yaml:"tie_breakers"`
	Normalization      NormalizationConfig        `yaml:"normalization"`
	Thresholds         ThresholdsConfig           `yaml:"thresholds"`
	Metadata           MetadataConfig             `yaml:"metadata"`
	Validation         ValidationConfig           `yaml:"validation"`
	Audit              AuditConfig                `yaml:"audit"`
	Performance        PerformanceConfig          `yaml:"performance"`
	ClinicalValidation ClinicalValidationConfig   `yaml:"clinical_validation"`
	Integration        IntegrationConfig          `yaml:"integration"`
}

// ProfileConfig represents a single profile configuration
type ProfileConfig struct {
	Weights WeightProfile `yaml:"weights"`
}

// PenaltiesConfig represents penalty configurations
type PenaltiesConfig struct {
	Safety       SafetyPenaltiesConfig       `yaml:"safety"`
	Adherence    AdherencePenaltiesConfig    `yaml:"adherence"`
	Availability AvailabilityPenaltiesConfig `yaml:"availability"`
}

// SafetyPenaltiesConfig represents safety penalty configuration
type SafetyPenaltiesConfig struct {
	ResidualDDI map[string]float64 `yaml:"residual_ddi"`
	Hypo        map[string]float64 `yaml:"hypo"`
	WeightGain  float64            `yaml:"weight_gain"`
}

// AdherencePenaltiesConfig represents adherence penalty configuration
type AdherencePenaltiesConfig struct {
	Base                   float64            `yaml:"base"`
	FrequencyBonus         map[string]float64 `yaml:"frequency_bonus"`
	FDCBonus              float64            `yaml:"fdc_bonus"`
	InjectablePenalty     float64            `yaml:"injectable_penalty"`
	WeeklyInjectableBonus float64            `yaml:"weekly_injectable_bonus"`
	DeviceTrainingPenalty float64            `yaml:"device_training_penalty"`
}

// AvailabilityPenaltiesConfig represents availability penalty configuration
type AvailabilityPenaltiesConfig struct {
	TierFactor           map[int]float64 `yaml:"tier_factor"`
	OutOfStockMultiplier float64         `yaml:"out_of_stock_multiplier"`
}

// EfficacyBonusesConfig represents efficacy bonus configuration
type EfficacyBonusesConfig struct {
	CVBenefit          float64 `yaml:"cv_benefit"`
	HFBenefit          float64 `yaml:"hf_benefit"`
	CKDBenefit         float64 `yaml:"ckd_benefit"`
	MaxPhenotypeBonus  float64 `yaml:"max_phenotype_bonus"`
}

// KnockoutRulesConfig represents knockout rule configuration
type KnockoutRulesConfig struct {
	Stock  StockKnockoutConfig  `yaml:"stock"`
	Budget BudgetKnockoutConfig `yaml:"budget"`
}

// StockKnockoutConfig represents stock-related knockout rules
type StockKnockoutConfig struct {
	MaxLeadTimeDays int `yaml:"max_lead_time_days"`
}

// BudgetKnockoutConfig represents budget-related knockout rules
type BudgetKnockoutConfig struct {
	MinimalTierMax  int `yaml:"minimal_tier_max"`
	StandardTierMax int `yaml:"standard_tier_max"`
	AdvancedTierMax int `yaml:"advanced_tier_max"`
}

// NormalizationConfig represents normalization settings
type NormalizationConfig struct {
	Efficacy EfficacyNormConfig `yaml:"efficacy"`
	Cost     CostNormConfig     `yaml:"cost"`
	Safety   SafetyNormConfig   `yaml:"safety"`
}

// EfficacyNormConfig represents efficacy normalization settings
type EfficacyNormConfig struct {
	A1cDropMax float64 `yaml:"a1c_drop_max"`
	A1cDropCap float64 `yaml:"a1c_drop_cap"`
}

// CostNormConfig represents cost normalization settings
type CostNormConfig struct {
	UseRelativeNormalization bool `yaml:"use_relative_normalization"`
}

// SafetyNormConfig represents safety normalization settings
type SafetyNormConfig struct {
	BaseScore float64 `yaml:"base_score"`
	MinScore  float64 `yaml:"min_score"`
}

// ThresholdsConfig represents clinical decision thresholds
type ThresholdsConfig struct {
	MinimumSafetyScore   float64 `yaml:"minimum_safety_score"`
	MinimumEfficacyScore float64 `yaml:"minimum_efficacy_score"`
	TopSlotMinimumScore  float64 `yaml:"top_slot_minimum_score"`
}

// MetadataConfig represents configuration metadata
type MetadataConfig struct {
	Version                    string `yaml:"version"`
	LastUpdated               string `yaml:"last_updated"`
	ClinicalGovernanceApproved bool   `yaml:"clinical_governance_approved"`
	NextReviewDate            string `yaml:"next_review_date"`
}

// ValidationConfig represents validation settings
type ValidationConfig struct {
	WeightSumTolerance float64           `yaml:"weight_sum_tolerance"`
	ScoreRange         ScoreRangeConfig  `yaml:"score_range"`
	RequiredProfiles   []string          `yaml:"required_profiles"`
}

// ScoreRangeConfig represents score range validation
type ScoreRangeConfig struct {
	Min float64 `yaml:"min"`
	Max float64 `yaml:"max"`
}

// AuditConfig represents audit settings
type AuditConfig struct {
	TrackProfileUsage       bool   `yaml:"track_profile_usage"`
	LogScoreDistributions   bool   `yaml:"log_score_distributions"`
	MonitorClinicalOutcomes bool   `yaml:"monitor_clinical_outcomes"`
	RegulatoryCompliance    string `yaml:"regulatory_compliance"`
}

// PerformanceConfig represents performance optimization settings
type PerformanceConfig struct {
	EnableDominancePruning      bool `yaml:"enable_dominance_pruning"`
	MaxCandidatesBeforePruning  int  `yaml:"max_candidates_before_pruning"`
	ParallelScoring            bool `yaml:"parallel_scoring"`
	CacheNormalizationRanges   bool `yaml:"cache_normalization_ranges"`
}

// ClinicalValidationConfig represents clinical validation settings
type ClinicalValidationConfig struct {
	EnableMonotonicityChecks bool     `yaml:"enable_monotonicity_checks"`
	EnableSensitivityAnalysis bool     `yaml:"enable_sensitivity_analysis"`
	EnableFairnessChecks     bool     `yaml:"enable_fairness_checks"`
	ValidationTestCases      []string `yaml:"validation_test_cases"`
}

// IntegrationConfig represents integration settings
type IntegrationConfig struct {
	KBUpdateFrequency   string `yaml:"kb_update_frequency"`
	HotReloadEnabled    bool   `yaml:"hot_reload_enabled"`
	FallbackToDefaults  bool   `yaml:"fallback_to_defaults"`
	ValidateOnLoad      bool   `yaml:"validate_on_load"`
}

// KBConfigLoader handles loading and validation of KB configuration
type KBConfigLoader struct {
	configPath string
	config     *KBConfig
	logger     *logrus.Logger
	lastLoaded time.Time
}

// NewKBConfigLoader creates a new KB configuration loader
func NewKBConfigLoader(configPath string, logger *logrus.Logger) *KBConfigLoader {
	if logger == nil {
		logger = logrus.New()
	}

	return &KBConfigLoader{
		configPath: configPath,
		logger:     logger,
	}
}

// LoadConfig loads the KB configuration from file
func (loader *KBConfigLoader) LoadConfig() (*KBConfig, error) {
	// Check if file exists
	if _, err := os.Stat(loader.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("KB config file not found: %s", loader.configPath)
	}

	// Read file
	data, err := ioutil.ReadFile(loader.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read KB config file: %w", err)
	}

	// Parse YAML
	var config KBConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse KB config YAML: %w", err)
	}

	// Validate configuration
	if err := loader.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("KB config validation failed: %w", err)
	}

	loader.config = &config
	loader.lastLoaded = time.Now()

	loader.logger.WithFields(logrus.Fields{
		"config_path":    loader.configPath,
		"version":        config.Metadata.Version,
		"profiles_count": len(config.Profiles),
	}).Info("KB configuration loaded successfully")

	return &config, nil
}

// validateConfig validates the loaded configuration
func (loader *KBConfigLoader) validateConfig(config *KBConfig) error {
	// Validate required profiles exist
	for _, requiredProfile := range config.Validation.RequiredProfiles {
		if _, exists := config.Profiles[requiredProfile]; !exists {
			return fmt.Errorf("required profile '%s' not found", requiredProfile)
		}
	}

	// Validate weight sums
	tolerance := config.Validation.WeightSumTolerance
	for profileName, profile := range config.Profiles {
		weights := profile.Weights
		sum := weights.Efficacy + weights.Safety + weights.Availability + 
			   weights.Cost + weights.Adherence + weights.Preference
		
		if math.Abs(sum-1.0) > tolerance {
			return fmt.Errorf("profile '%s' weights sum to %.3f, expected 1.0 (±%.3f)", 
				profileName, sum, tolerance)
		}
	}

	// Validate score ranges
	scoreRange := config.Validation.ScoreRange
	if scoreRange.Min < 0.0 || scoreRange.Max > 1.0 || scoreRange.Min >= scoreRange.Max {
		return fmt.Errorf("invalid score range: [%.2f, %.2f]", scoreRange.Min, scoreRange.Max)
	}

	loader.logger.Debug("KB configuration validation passed")
	return nil
}

// GetConfig returns the current configuration
func (loader *KBConfigLoader) GetConfig() *KBConfig {
	return loader.config
}

// NeedsReload checks if the configuration needs to be reloaded
func (loader *KBConfigLoader) NeedsReload() bool {
	if loader.config == nil {
		return true
	}

	// Check file modification time
	fileInfo, err := os.Stat(loader.configPath)
	if err != nil {
		loader.logger.WithError(err).Warn("Failed to check KB config file modification time")
		return false
	}

	return fileInfo.ModTime().After(loader.lastLoaded)
}

// ReloadIfNeeded reloads the configuration if the file has been modified
func (loader *KBConfigLoader) ReloadIfNeeded() error {
	if !loader.NeedsReload() {
		return nil
	}

	loader.logger.Info("Reloading KB configuration due to file changes")
	_, err := loader.LoadConfig()
	return err
}

// GetAbsoluteConfigPath returns the absolute path to the config file
func (loader *KBConfigLoader) GetAbsoluteConfigPath() (string, error) {
	return filepath.Abs(loader.configPath)
}
