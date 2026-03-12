// Package risk provides risk stratification engine for KB-11 Population Health.
package risk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"gopkg.in/yaml.v3"
)

// ──────────────────────────────────────────────────────────────────────────────
// YAML Model Definitions (matching YAML structure)
// ──────────────────────────────────────────────────────────────────────────────

// YAMLRiskModel represents a risk model loaded from YAML.
type YAMLRiskModel struct {
	Name        string                 `yaml:"name"`
	Type        string                 `yaml:"type"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Governance  YAMLGovernance         `yaml:"governance"`
	Factors     []YAMLFactor           `yaml:"factors"`
	Tiers       map[string]YAMLTier    `yaml:"tiers"`
	Config      YAMLModelConfiguration `yaml:"configuration"`
}

// YAMLGovernance contains governance metadata for KB-18 integration.
type YAMLGovernance struct {
	Owner             string   `yaml:"owner"`
	ClinicalReviewer  string   `yaml:"clinical_reviewer"`
	LastApproved      string   `yaml:"last_approved"`
	ApprovalID        string   `yaml:"approval_id"`
	RequiresValidation bool    `yaml:"requires_validation"`
	ReviewCycleDays   int      `yaml:"review_cycle_days"`
	ClinicalEvidence  []string `yaml:"clinical_evidence"`
}

// YAMLFactor represents a risk factor in the model.
type YAMLFactor struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Weight      float64             `yaml:"weight"`
	Source      string              `yaml:"source"`
	LookbackDays int                `yaml:"lookback_days,omitempty"`
	Thresholds  []YAMLThreshold     `yaml:"thresholds,omitempty"`
	Scoring     []YAMLThreshold     `yaml:"scoring,omitempty"`
	Conditions  []YAMLCondition     `yaml:"conditions,omitempty"`
	Categories  []YAMLCategory      `yaml:"categories,omitempty"`
	SDOHFactors []YAMLSDOHFactor    `yaml:"sdoh_factors,omitempty"`
	HighRiskMeds []YAMLHighRiskMed  `yaml:"high_risk_medications,omitempty"`
}

// YAMLThreshold represents a scoring threshold.
type YAMLThreshold struct {
	Range []float64 `yaml:"range"`
	Score float64   `yaml:"score"`
}

// YAMLCondition represents a clinical condition in the model.
type YAMLCondition struct {
	Code          string  `yaml:"code"`
	System        string  `yaml:"system"`
	Description   string  `yaml:"description"`
	Score         float64 `yaml:"score,omitempty"`
	CharlsonWeight int    `yaml:"charlson_weight,omitempty"`
}

// YAMLCategory represents a categorical factor value.
type YAMLCategory struct {
	Type        string  `yaml:"type"`
	Description string  `yaml:"description,omitempty"`
	Score       float64 `yaml:"score"`
}

// YAMLSDOHFactor represents a social determinant of health factor.
type YAMLSDOHFactor struct {
	Name   string  `yaml:"name"`
	ZCode  string  `yaml:"z_code"`
	Score  float64 `yaml:"score"`
}

// YAMLHighRiskMed represents a high-risk medication class.
type YAMLHighRiskMed struct {
	Class      string  `yaml:"class"`
	Multiplier float64 `yaml:"multiplier"`
}

// YAMLTier represents a risk tier definition.
type YAMLTier struct {
	Range              []float64 `yaml:"range"`
	Intervention       string    `yaml:"intervention"`
	Description        string    `yaml:"description"`
	RecommendedActions []string  `yaml:"recommended_actions"`
}

// YAMLModelConfiguration contains model runtime configuration.
type YAMLModelConfiguration struct {
	CacheTTLMinutes            int     `yaml:"cache_ttl_minutes"`
	RecalculationIntervalDays  int     `yaml:"recalculation_interval_days,omitempty"`
	RecalculationTrigger       string  `yaml:"recalculation_trigger,omitempty"`
	MinimumDataCompleteness    float64 `yaml:"minimum_data_completeness"`
	ConfidenceThreshold        float64 `yaml:"confidence_threshold"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Model Loader
// ──────────────────────────────────────────────────────────────────────────────

// ModelLoader loads and manages risk models from YAML files.
type ModelLoader struct {
	modelsPath string
	models     map[string]*LoadedModel
}

// LoadedModel represents a fully loaded and parsed risk model.
type LoadedModel struct {
	YAML       *YAMLRiskModel
	Config     *ModelConfig
	Governance *GovernanceInfo
	Factors    []*FactorConfig
}

// GovernanceInfo contains parsed governance information.
type GovernanceInfo struct {
	Owner             string
	ClinicalReviewer  string
	LastApproved      string
	ApprovalID        string
	RequiresValidation bool
	ReviewCycleDays   int
	ClinicalEvidence  []string
}

// FactorConfig contains parsed factor configuration.
type FactorConfig struct {
	Name         string
	Description  string
	Weight       float64
	Source       string
	LookbackDays int
	Thresholds   []ThresholdRange
	Conditions   []ConditionConfig
	Categories   []CategoryConfig
}

// ThresholdRange represents a scoring range.
type ThresholdRange struct {
	Min   float64
	Max   float64
	Score float64
}

// ConditionConfig represents a condition scoring configuration.
type ConditionConfig struct {
	Code        string
	System      string
	Description string
	Score       float64
}

// CategoryConfig represents a categorical scoring configuration.
type CategoryConfig struct {
	Type        string
	Description string
	Score       float64
}

// NewModelLoader creates a new model loader.
func NewModelLoader(modelsPath string) *ModelLoader {
	return &ModelLoader{
		modelsPath: modelsPath,
		models:     make(map[string]*LoadedModel),
	}
}

// LoadAll loads all YAML models from the models directory.
func (l *ModelLoader) LoadAll() error {
	pattern := filepath.Join(l.modelsPath, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob model files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, _ := filepath.Glob(filepath.Join(l.modelsPath, "*.yml"))
	files = append(files, ymlFiles...)

	for _, file := range files {
		model, err := l.LoadFile(file)
		if err != nil {
			return fmt.Errorf("failed to load model %s: %w", file, err)
		}
		l.models[string(model.Config.Name)] = model
	}

	return nil
}

// LoadFile loads a single YAML model file.
func (l *ModelLoader) LoadFile(path string) (*LoadedModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var yamlModel YAMLRiskModel
	if err := yaml.Unmarshal(data, &yamlModel); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return l.parseModel(&yamlModel)
}

// parseModel converts a YAML model to a LoadedModel.
func (l *ModelLoader) parseModel(yamlModel *YAMLRiskModel) (*LoadedModel, error) {
	// Convert model type
	modelType := parseModelType(yamlModel.Type)

	// Build weights map from factors
	weights := make(map[string]float64)
	for _, factor := range yamlModel.Factors {
		weights[factor.Name] = factor.Weight
	}

	// Build thresholds from tiers
	thresholds := l.parseThresholds(yamlModel.Tiers)

	// Create ModelConfig
	config := &ModelConfig{
		Name:        modelType,
		Version:     yamlModel.Version,
		Description: yamlModel.Description,
		Weights:     weights,
		Thresholds:  thresholds,
		ValidDays:   yamlModel.Config.RecalculationIntervalDays,
	}

	if config.ValidDays == 0 {
		config.ValidDays = 30 // Default to 30 days
	}

	// Parse governance info
	governance := &GovernanceInfo{
		Owner:              yamlModel.Governance.Owner,
		ClinicalReviewer:   yamlModel.Governance.ClinicalReviewer,
		LastApproved:       yamlModel.Governance.LastApproved,
		ApprovalID:         yamlModel.Governance.ApprovalID,
		RequiresValidation: yamlModel.Governance.RequiresValidation,
		ReviewCycleDays:    yamlModel.Governance.ReviewCycleDays,
		ClinicalEvidence:   yamlModel.Governance.ClinicalEvidence,
	}

	// Parse factors
	factors := l.parseFactors(yamlModel.Factors)

	return &LoadedModel{
		YAML:       yamlModel,
		Config:     config,
		Governance: governance,
		Factors:    factors,
	}, nil
}

// parseThresholds converts YAML tiers to RiskThresholds.
func (l *ModelLoader) parseThresholds(tiers map[string]YAMLTier) RiskThresholds {
	thresholds := RiskThresholds{
		Low:      0.10, // Default
		Moderate: 0.30,
		High:     0.50,
		VeryHigh: 0.75,
		Rising:   0.15,
	}

	if tier, ok := tiers["LOW"]; ok && len(tier.Range) >= 1 {
		thresholds.Low = tier.Range[0]
	}
	if tier, ok := tiers["MODERATE"]; ok && len(tier.Range) >= 1 {
		thresholds.Moderate = tier.Range[0]
	}
	if tier, ok := tiers["HIGH"]; ok && len(tier.Range) >= 1 {
		thresholds.High = tier.Range[0]
	}
	if tier, ok := tiers["VERY_HIGH"]; ok && len(tier.Range) >= 1 {
		thresholds.VeryHigh = tier.Range[0]
	}

	return thresholds
}

// parseFactors converts YAML factors to FactorConfig.
func (l *ModelLoader) parseFactors(yamlFactors []YAMLFactor) []*FactorConfig {
	factors := make([]*FactorConfig, 0, len(yamlFactors))

	for _, yf := range yamlFactors {
		factor := &FactorConfig{
			Name:         yf.Name,
			Description:  yf.Description,
			Weight:       yf.Weight,
			Source:       yf.Source,
			LookbackDays: yf.LookbackDays,
		}

		// Parse thresholds (combine thresholds and scoring)
		allThresholds := append(yf.Thresholds, yf.Scoring...)
		for _, t := range allThresholds {
			if len(t.Range) >= 2 {
				factor.Thresholds = append(factor.Thresholds, ThresholdRange{
					Min:   t.Range[0],
					Max:   t.Range[1],
					Score: t.Score,
				})
			}
		}

		// Parse conditions
		for _, c := range yf.Conditions {
			factor.Conditions = append(factor.Conditions, ConditionConfig{
				Code:        c.Code,
				System:      c.System,
				Description: c.Description,
				Score:       c.Score,
			})
		}

		// Parse categories
		for _, cat := range yf.Categories {
			factor.Categories = append(factor.Categories, CategoryConfig{
				Type:        cat.Type,
				Description: cat.Description,
				Score:       cat.Score,
			})
		}

		factors = append(factors, factor)
	}

	return factors
}

// parseModelType converts a string to RiskModelType.
func parseModelType(typeStr string) models.RiskModelType {
	switch strings.ToUpper(typeStr) {
	case "HOSPITALIZATION":
		return models.RiskModelHospitalization
	case "READMISSION":
		return models.RiskModelReadmission
	case "ED_UTILIZATION":
		return models.RiskModelEDUtilization
	case "DIABETES_PROGRESSION":
		return models.RiskModelDiabetesProgression
	case "CHF_EXACERBATION":
		return models.RiskModelCHFExacerbation
	case "FRAILTY":
		return models.RiskModelFrailty
	default:
		return models.RiskModelHospitalization
	}
}

// GetModel retrieves a loaded model by name.
func (l *ModelLoader) GetModel(name string) (*LoadedModel, bool) {
	model, ok := l.models[name]
	return model, ok
}

// GetAllModels returns all loaded models.
func (l *ModelLoader) GetAllModels() map[string]*LoadedModel {
	return l.models
}

// ListModels returns a list of all loaded model names.
func (l *ModelLoader) ListModels() []string {
	names := make([]string, 0, len(l.models))
	for name := range l.models {
		names = append(names, name)
	}
	return names
}

// GetModelConfig returns the ModelConfig for a given model name.
func (l *ModelLoader) GetModelConfig(name string) (*ModelConfig, bool) {
	if model, ok := l.models[name]; ok {
		return model.Config, true
	}
	return nil, false
}

// ──────────────────────────────────────────────────────────────────────────────
// Scoring Helpers
// ──────────────────────────────────────────────────────────────────────────────

// ScoreByThreshold returns the score for a value based on thresholds.
func ScoreByThreshold(value float64, thresholds []ThresholdRange) float64 {
	for _, t := range thresholds {
		if value >= t.Min && value < t.Max {
			return t.Score
		}
	}
	// If value exceeds all ranges, use the last threshold's score
	if len(thresholds) > 0 {
		return thresholds[len(thresholds)-1].Score
	}
	return 0
}

// ScoreByCategory returns the score for a categorical value.
func ScoreByCategory(value string, categories []CategoryConfig) float64 {
	for _, c := range categories {
		if strings.EqualFold(c.Type, value) {
			return c.Score
		}
	}
	return 0
}

// HasCondition checks if a condition code matches any configured conditions.
func HasCondition(code string, conditions []ConditionConfig) (bool, float64) {
	for _, c := range conditions {
		// Handle wildcard matching (e.g., "I50.*")
		pattern := c.Code
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(code, prefix) {
				return true, c.Score
			}
		} else if code == c.Code {
			return true, c.Score
		}
	}
	return false, 0
}
