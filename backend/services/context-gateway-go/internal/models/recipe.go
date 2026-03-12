// Package models provides core data structures for clinical recipes
package models

import (
	"fmt"
	"strings"
	"time"
)

// DataSourceType represents different types of clinical data sources
type DataSourceType string

const (
	DataSourcePatientService    DataSourceType = "patient_service"
	DataSourceMedicationService DataSourceType = "medication_service"
	DataSourceObservationService DataSourceType = "observation_service"
	DataSourceFHIRStore         DataSourceType = "fhir_store"
	DataSourceSafetyGateway     DataSourceType = "safety_gateway"
	DataSourceWorkflowEngine    DataSourceType = "workflow_engine"
	DataSourceElasticsearch     DataSourceType = "elasticsearch"
	DataSourceApolloFederation  DataSourceType = "apollo_federation"
)

// DataPoint defines a specific piece of clinical data required by a recipe
type DataPoint struct {
	Name                 string           `json:"name" yaml:"name"`
	SourceType           DataSourceType   `json:"source_type" yaml:"source_type"`
	Fields               []string         `json:"fields" yaml:"fields"`
	Required             bool             `json:"required" yaml:"required"`
	MaxAgeHours          int32            `json:"max_age_hours" yaml:"max_age_hours"`
	QualityThreshold     float64          `json:"quality_threshold" yaml:"quality_threshold"`
	TimeoutMs            int32            `json:"timeout_ms" yaml:"timeout_ms"`
	RetryCount           int32            `json:"retry_count" yaml:"retry_count"`
	FallbackSources      []DataSourceType `json:"fallback_sources,omitempty" yaml:"fallback_sources,omitempty"`
	FreshnessRequirement int32            `json:"freshness_requirement" yaml:"freshness_requirement"`
}

// Validate checks if the DataPoint configuration is valid
func (dp *DataPoint) Validate() error {
	if dp.Name == "" {
		return fmt.Errorf("data point name cannot be empty")
	}
	if dp.SourceType == "" {
		return fmt.Errorf("data point source_type cannot be empty")
	}
	if len(dp.Fields) == 0 {
		return fmt.Errorf("data point must specify at least one field")
	}
	if dp.MaxAgeHours < 0 {
		return fmt.Errorf("max_age_hours cannot be negative")
	}
	if dp.QualityThreshold < 0 || dp.QualityThreshold > 1 {
		return fmt.Errorf("quality_threshold must be between 0 and 1")
	}
	if dp.TimeoutMs <= 0 {
		return fmt.Errorf("timeout_ms must be positive")
	}
	if dp.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}
	return nil
}

// ConditionalRule defines additional data requirements based on conditions
type ConditionalRule struct {
	Condition            string      `json:"condition" yaml:"condition"`
	AdditionalDataPoints []DataPoint `json:"additional_data_points" yaml:"additional_data_points"`
	Description          string      `json:"description" yaml:"description"`
}

// QualityConstraints defines data quality requirements for a recipe
type QualityConstraints struct {
	MinimumCompleteness float64  `json:"minimum_completeness" yaml:"minimum_completeness"`
	MaximumAgeHours     int32    `json:"maximum_age_hours" yaml:"maximum_age_hours"`
	RequiredFields      []string `json:"required_fields" yaml:"required_fields"`
	AccuracyThreshold   float64  `json:"accuracy_threshold" yaml:"accuracy_threshold"`
}

// SafetyRequirements defines safety constraints for clinical data assembly
type SafetyRequirements struct {
	MinimumCompletenessScore      float64 `json:"minimum_completeness_score" yaml:"minimum_completeness_score"`
	AbsoluteRequiredEnforcement   string  `json:"absolute_required_enforcement" yaml:"absolute_required_enforcement"`
	PreferredDataHandling         string  `json:"preferred_data_handling" yaml:"preferred_data_handling"`
	CriticalMissingDataAction     string  `json:"critical_missing_data_action" yaml:"critical_missing_data_action"`
	StaleDataAction               string  `json:"stale_data_action" yaml:"stale_data_action"`
}

// CacheStrategy defines caching behavior for recipe data
type CacheStrategy struct {
	L1TTLSeconds       int32    `json:"l1_ttl_seconds" yaml:"l1_ttl_seconds"`
	L2TTLSeconds       int32    `json:"l2_ttl_seconds" yaml:"l2_ttl_seconds"`
	L3TTLSeconds       int32    `json:"l3_ttl_seconds" yaml:"l3_ttl_seconds"`
	InvalidationEvents []string `json:"invalidation_events" yaml:"invalidation_events"`
	CacheKeyPattern    string   `json:"cache_key_pattern" yaml:"cache_key_pattern"`
}

// AssemblyRules defines how clinical context should be assembled
type AssemblyRules struct {
	ParallelExecution           bool  `json:"parallel_execution" yaml:"parallel_execution"`
	TimeoutBudgetMs             int32 `json:"timeout_budget_ms" yaml:"timeout_budget_ms"`
	CircuitBreakerEnabled       bool  `json:"circuit_breaker_enabled" yaml:"circuit_breaker_enabled"`
	RetryFailedSources          bool  `json:"retry_failed_sources" yaml:"retry_failed_sources"`
	ValidateDataFreshness       bool  `json:"validate_data_freshness" yaml:"validate_data_freshness"`
	EnforceQualityConstraints   bool  `json:"enforce_quality_constraints" yaml:"enforce_quality_constraints"`
}

// GovernanceMetadata contains recipe approval and audit information
type GovernanceMetadata struct {
	ApprovedBy              string    `json:"approved_by" yaml:"approved_by"`
	ApprovalDate            time.Time `json:"approval_date" yaml:"approval_date"`
	Version                 string    `json:"version" yaml:"version"`
	EffectiveDate           time.Time `json:"effective_date" yaml:"effective_date"`
	ExpiryDate              *time.Time `json:"expiry_date,omitempty" yaml:"expiry_date,omitempty"`
	ClinicalBoardApprovalID string    `json:"clinical_board_approval_id" yaml:"clinical_board_approval_id"`
	Tags                    []string  `json:"tags" yaml:"tags"`
	ChangeLog               []string  `json:"change_log" yaml:"change_log"`
}

// IsExpired checks if the recipe has expired based on governance metadata
func (gm *GovernanceMetadata) IsExpired() bool {
	if gm.ExpiryDate == nil {
		return false
	}
	return time.Now().UTC().After(*gm.ExpiryDate)
}

// IsApproved checks if the recipe is approved by the Clinical Governance Board
func (gm *GovernanceMetadata) IsApproved() bool {
	return strings.Contains(strings.ToLower(gm.ApprovedBy), "clinical governance board")
}

// WorkflowRecipe defines a complete clinical context recipe
type WorkflowRecipe struct {
	RecipeID              string              `json:"recipe_id" yaml:"recipe_id"`
	RecipeName            string              `json:"recipe_name" yaml:"recipe_name"`
	Version               string              `json:"version" yaml:"version"`
	ClinicalScenario      string              `json:"clinical_scenario" yaml:"clinical_scenario"`
	WorkflowCategory      string              `json:"workflow_category" yaml:"workflow_category"`
	ExecutionPattern      string              `json:"execution_pattern" yaml:"execution_pattern"`
	RequiredFields        []DataPoint         `json:"required_fields" yaml:"required_fields"`
	ConditionalRules      []ConditionalRule   `json:"conditional_rules,omitempty" yaml:"conditional_rules,omitempty"`
	QualityConstraints    QualityConstraints  `json:"quality_constraints" yaml:"quality_constraints"`
	SafetyRequirements    SafetyRequirements  `json:"safety_requirements" yaml:"safety_requirements"`
	CacheStrategy         CacheStrategy       `json:"cache_strategy" yaml:"cache_strategy"`
	AssemblyRules         AssemblyRules       `json:"assembly_rules" yaml:"assembly_rules"`
	GovernanceMetadata    GovernanceMetadata  `json:"governance_metadata" yaml:"governance_metadata"`
	SLAMs                 int32               `json:"sla_ms" yaml:"sla_ms"`
	BaseRecipeID          *string             `json:"base_recipe_id,omitempty" yaml:"base_recipe_id,omitempty"`
	ExtendsRecipes        []string            `json:"extends_recipes,omitempty" yaml:"extends_recipes,omitempty"`
}

// Validate performs comprehensive validation of the recipe
func (wr *WorkflowRecipe) Validate() (bool, []string, []string) {
	var errors []string
	var warnings []string
	
	// Basic field validation
	if wr.RecipeID == "" {
		errors = append(errors, "recipe_id is required")
	}
	if wr.RecipeName == "" {
		errors = append(errors, "recipe_name is required")
	}
	if wr.Version == "" {
		errors = append(errors, "version is required")
	}
	if wr.ClinicalScenario == "" {
		errors = append(errors, "clinical_scenario is required")
	}
	
	// Validate required data points
	if len(wr.RequiredFields) == 0 {
		errors = append(errors, "at least one required field must be specified")
	} else {
		for i, dp := range wr.RequiredFields {
			if err := dp.Validate(); err != nil {
				errors = append(errors, fmt.Sprintf("required_field[%d]: %v", i, err))
			}
		}
	}
	
	// Validate conditional rules
	for i, rule := range wr.ConditionalRules {
		if rule.Condition == "" {
			errors = append(errors, fmt.Sprintf("conditional_rule[%d]: condition cannot be empty", i))
		}
		for j, dp := range rule.AdditionalDataPoints {
			if err := dp.Validate(); err != nil {
				errors = append(errors, fmt.Sprintf("conditional_rule[%d].additional_data_points[%d]: %v", i, j, err))
			}
		}
	}
	
	// Validate quality constraints
	if wr.QualityConstraints.MinimumCompleteness < 0 || wr.QualityConstraints.MinimumCompleteness > 1 {
		errors = append(errors, "quality_constraints.minimum_completeness must be between 0 and 1")
	}
	
	// Validate SLA
	if wr.SLAMs <= 0 {
		errors = append(errors, "sla_ms must be positive")
	}
	if wr.SLAMs > 30000 { // 30 seconds maximum
		warnings = append(warnings, "sla_ms is very high (>30s), consider optimizing")
	}
	
	// Validate governance
	if wr.GovernanceMetadata.ApprovedBy == "" {
		warnings = append(warnings, "recipe is not approved by governance")
	} else if !wr.GovernanceMetadata.IsApproved() {
		warnings = append(warnings, "recipe is not approved by Clinical Governance Board")
	}
	
	// Check expiration
	if wr.GovernanceMetadata.IsExpired() {
		errors = append(errors, "recipe has expired")
	}
	
	// Validate cache strategy
	if wr.CacheStrategy.L1TTLSeconds <= 0 {
		warnings = append(warnings, "L1 cache TTL should be positive for better performance")
	}
	
	// Validate assembly rules
	if wr.AssemblyRules.TimeoutBudgetMs <= 0 {
		errors = append(errors, "assembly_rules.timeout_budget_ms must be positive")
	}
	if wr.AssemblyRules.TimeoutBudgetMs > wr.SLAMs {
		warnings = append(warnings, "assembly timeout exceeds SLA budget")
	}
	
	return len(errors) == 0, errors, warnings
}

// GetCacheKey generates a cache key for the recipe with given patient and provider IDs
func (wr *WorkflowRecipe) GetCacheKey(patientID string, providerID *string) string {
	baseKey := strings.ReplaceAll(wr.CacheStrategy.CacheKeyPattern, "{patient_id}", patientID)
	baseKey = strings.ReplaceAll(baseKey, "{recipe_id}", wr.RecipeID)
	
	if providerID != nil && *providerID != "" {
		baseKey = fmt.Sprintf("%s:provider:%s", baseKey, *providerID)
	}
	
	return baseKey
}

// IsValid checks if the recipe is currently valid (approved and not expired)
func (wr *WorkflowRecipe) IsValid() bool {
	return wr.GovernanceMetadata.IsApproved() && !wr.GovernanceMetadata.IsExpired()
}

// GetRequiredSourceTypes returns a unique list of data source types used by this recipe
func (wr *WorkflowRecipe) GetRequiredSourceTypes() []DataSourceType {
	sourceMap := make(map[DataSourceType]bool)
	
	// Collect from required fields
	for _, dp := range wr.RequiredFields {
		sourceMap[dp.SourceType] = true
		for _, fallback := range dp.FallbackSources {
			sourceMap[fallback] = true
		}
	}
	
	// Collect from conditional rules
	for _, rule := range wr.ConditionalRules {
		for _, dp := range rule.AdditionalDataPoints {
			sourceMap[dp.SourceType] = true
			for _, fallback := range dp.FallbackSources {
				sourceMap[fallback] = true
			}
		}
	}
	
	// Convert to slice
	var sources []DataSourceType
	for source := range sourceMap {
		sources = append(sources, source)
	}
	
	return sources
}

// RecipeValidationResult represents the result of recipe validation
type RecipeValidationResult struct {
	RecipeID             string    `json:"recipe_id"`
	Valid                bool      `json:"valid"`
	Errors               []string  `json:"errors"`
	Warnings             []string  `json:"warnings,omitempty"`
	ValidationDurationMs float64   `json:"validation_duration_ms"`
	ValidatedAt          time.Time `json:"validated_at"`
}