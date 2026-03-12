package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RecipeResolver resolves clinical recipes and creates immutable snapshots
type RecipeResolver struct {
	recipeStore RecipeRepository
	cache       map[string]*ClinicalRecipe // In-memory cache for fast lookups
	logger      *zap.Logger
}

// RecipeRepository defines the interface for recipe persistence
type RecipeRepository interface {
	FindByPatientAndProtocol(ctx context.Context, patientCriteria PatientCriteria, protocol string) ([]*ClinicalRecipe, error)
	GetByID(ctx context.Context, recipeID string) (*ClinicalRecipe, error)
	Store(ctx context.Context, recipe *ClinicalRecipe) error
}

// ClinicalRecipe represents a clinical calculation recipe
type ClinicalRecipe struct {
	RecipeID          string                 `json:"recipe_id"`
	Name              string                 `json:"name"`
	Version           string                 `json:"version"`
	ProtocolID        string                 `json:"protocol_id"`
	PatientCriteria   PatientCriteria        `json:"patient_criteria"`
	CalculationSteps  []CalculationStep      `json:"calculation_steps"`
	ValidationRules   []ValidationRule       `json:"validation_rules"`
	OptimizationHints []OptimizationHint     `json:"optimization_hints"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	Active            bool                   `json:"active"`
	Priority          int                    `json:"priority"`
}

// PatientCriteria defines the criteria for recipe matching
type PatientCriteria struct {
	AgeRange          *AgeRange             `json:"age_range,omitempty"`
	WeightRange       *WeightRange          `json:"weight_range,omitempty"`
	Conditions        []string              `json:"conditions,omitempty"`
	Allergies         []string              `json:"allergies,omitempty"`
	RenalFunction     *RenalFunctionRange   `json:"renal_function,omitempty"`
	HepaticFunction   *HepaticFunctionRange `json:"hepatic_function,omitempty"`
	ConcurrentMeds    []string              `json:"concurrent_medications,omitempty"`
	ExclusionCriteria []string              `json:"exclusion_criteria,omitempty"`
}

// CalculationStep defines a single step in the clinical calculation
type CalculationStep struct {
	StepID       string                 `json:"step_id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // DOSE_CALCULATION, FREQUENCY_ADJUSTMENT, etc.
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	Formula      string                 `json:"formula,omitempty"`
	Conditions   []StepCondition        `json:"conditions,omitempty"`
	Order        int                    `json:"order"`
}

// ValidationRule defines validation requirements for the recipe
type ValidationRule struct {
	RuleID      string                 `json:"rule_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // RANGE_CHECK, INTERACTION_CHECK, etc.
	Parameters  map[string]interface{} `json:"parameters"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Conditions  []RuleCondition        `json:"conditions,omitempty"`
}

// OptimizationHint provides guidance for calculation optimization
type OptimizationHint struct {
	HintID      string                 `json:"hint_id"`
	Type        string                 `json:"type"` // CACHE_KEY, PARALLEL_SAFE, etc.
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
}

// RecipeSnapshot represents an immutable snapshot of a resolved recipe
type RecipeSnapshot struct {
	SnapshotID    string                 `json:"snapshot_id"`
	RecipeID      string                 `json:"recipe_id"`
	WorkflowID    string                 `json:"workflow_id"`
	PatientID     string                 `json:"patient_id"`
	PatientData   map[string]interface{} `json:"patient_data"`
	ResolvedSteps []ResolvedStep         `json:"resolved_steps"`
	Context       map[string]interface{} `json:"context"`
	CreatedAt     time.Time              `json:"created_at"`
	Immutable     bool                   `json:"immutable"`
	Hash          string                 `json:"hash"`
}

// ResolvedStep represents a calculation step with resolved parameters
type ResolvedStep struct {
	StepID            string                 `json:"step_id"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"`
	ResolvedParams    map[string]interface{} `json:"resolved_parameters"`
	ComputedValues    map[string]interface{} `json:"computed_values,omitempty"`
	ExecutionOrder    int                    `json:"execution_order"`
	CacheKey          string                 `json:"cache_key,omitempty"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
}

// Range and condition types
type AgeRange struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

type WeightRange struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

type RenalFunctionRange struct {
	CrCl_Min *float64 `json:"crcl_min,omitempty"`
	CrCl_Max *float64 `json:"crcl_max,omitempty"`
}

type HepaticFunctionRange struct {
	ChildPughScore_Max *int    `json:"child_pugh_score_max,omitempty"`
	ALT_Max           *float64 `json:"alt_max,omitempty"`
}

type StepCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // GT, LT, EQ, IN, etc.
	Value    interface{} `json:"value"`
}

type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// NewRecipeResolver creates a new RecipeResolver instance
func NewRecipeResolver(recipeStore RecipeRepository, logger *zap.Logger) *RecipeResolver {
	return &RecipeResolver{
		recipeStore: recipeStore,
		cache:       make(map[string]*ClinicalRecipe),
		logger:      logger,
	}
}

// ResolveRecipe finds and resolves the best recipe for a given patient and protocol
func (r *RecipeResolver) ResolveRecipe(
	ctx context.Context,
	patientID string,
	protocol string,
	patientData map[string]interface{},
) (*ClinicalRecipe, error) {
	startTime := time.Now()

	// Extract patient criteria from patient data
	criteria := r.extractPatientCriteria(patientData)

	r.logger.Info("Resolving recipe",
		zap.String("patient_id", patientID),
		zap.String("protocol", protocol),
		zap.Any("criteria", criteria))

	// Find matching recipes
	recipes, err := r.recipeStore.FindByPatientAndProtocol(ctx, criteria, protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to find recipes: %w", err)
	}

	if len(recipes) == 0 {
		return nil, fmt.Errorf("no recipes found for protocol %s with given patient criteria", protocol)
	}

	// Select the best recipe based on specificity and priority
	selectedRecipe := r.selectBestRecipe(recipes, criteria)

	r.logger.Info("Recipe resolved",
		zap.String("recipe_id", selectedRecipe.RecipeID),
		zap.String("recipe_name", selectedRecipe.Name),
		zap.Duration("resolution_time", time.Since(startTime)))

	return selectedRecipe, nil
}

// CreateSnapshot creates an immutable snapshot from a recipe and patient data
func (r *RecipeResolver) CreateSnapshot(
	ctx context.Context,
	recipe *ClinicalRecipe,
	workflowID string,
	patientID string,
	patientData map[string]interface{},
) (*RecipeSnapshot, error) {
	startTime := time.Now()

	// Generate unique snapshot ID
	snapshotID := r.generateSnapshotID(recipe.RecipeID, patientID, patientData)

	// Resolve all calculation steps with patient data
	resolvedSteps, err := r.resolveCalculationSteps(recipe.CalculationSteps, patientData)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve calculation steps: %w", err)
	}

	// Create immutable snapshot
	snapshot := &RecipeSnapshot{
		SnapshotID:    snapshotID,
		RecipeID:      recipe.RecipeID,
		WorkflowID:    workflowID,
		PatientID:     patientID,
		PatientData:   r.deepCopyMap(patientData),
		ResolvedSteps: resolvedSteps,
		Context: map[string]interface{}{
			"recipe_name":     recipe.Name,
			"recipe_version":  recipe.Version,
			"protocol_id":     recipe.ProtocolID,
			"resolution_time": time.Since(startTime).String(),
		},
		CreatedAt: time.Now(),
		Immutable: true,
	}

	// Calculate hash for integrity verification
	snapshot.Hash = r.calculateSnapshotHash(snapshot)

	r.logger.Info("Recipe snapshot created",
		zap.String("snapshot_id", snapshotID),
		zap.String("recipe_id", recipe.RecipeID),
		zap.Int("resolved_steps", len(resolvedSteps)),
		zap.Duration("snapshot_time", time.Since(startTime)))

	return snapshot, nil
}

// GetSnapshotByID retrieves a snapshot by its ID (would typically be from cache/storage)
func (r *RecipeResolver) GetSnapshotByID(ctx context.Context, snapshotID string) (*RecipeSnapshot, error) {
	// In a real implementation, this would fetch from Redis/database
	// For now, return error as snapshots are created on-demand
	return nil, fmt.Errorf("snapshot retrieval not implemented: %s", snapshotID)
}

// ValidateSnapshot verifies the integrity of a snapshot
func (r *RecipeResolver) ValidateSnapshot(snapshot *RecipeSnapshot) error {
	if !snapshot.Immutable {
		return fmt.Errorf("snapshot %s is not marked as immutable", snapshot.SnapshotID)
	}

	// Verify hash integrity
	expectedHash := r.calculateSnapshotHash(snapshot)
	if snapshot.Hash != expectedHash {
		return fmt.Errorf("snapshot %s hash mismatch: expected %s, got %s",
			snapshot.SnapshotID, expectedHash, snapshot.Hash)
	}

	return nil
}

// Helper methods

// extractPatientCriteria extracts criteria from patient data for recipe matching
func (r *RecipeResolver) extractPatientCriteria(patientData map[string]interface{}) PatientCriteria {
	criteria := PatientCriteria{}

	// Extract age
	if age, exists := patientData["age"].(float64); exists {
		criteria.AgeRange = &AgeRange{Min: &age, Max: &age}
	}

	// Extract weight
	if weight, exists := patientData["weight"].(float64); exists {
		criteria.WeightRange = &WeightRange{Min: &weight, Max: &weight}
	}

	// Extract conditions
	if conditions, exists := patientData["conditions"].([]interface{}); exists {
		criteria.Conditions = make([]string, len(conditions))
		for i, condition := range conditions {
			if condStr, ok := condition.(string); ok {
				criteria.Conditions[i] = condStr
			}
		}
	}

	// Extract allergies
	if allergies, exists := patientData["allergies"].([]interface{}); exists {
		criteria.Allergies = make([]string, len(allergies))
		for i, allergy := range allergies {
			if allergyStr, ok := allergy.(string); ok {
				criteria.Allergies[i] = allergyStr
			}
		}
	}

	// Extract renal function
	if crcl, exists := patientData["creatinine_clearance"].(float64); exists {
		criteria.RenalFunction = &RenalFunctionRange{CrCl_Min: &crcl, CrCl_Max: &crcl}
	}

	// Extract concurrent medications
	if meds, exists := patientData["current_medications"].([]interface{}); exists {
		criteria.ConcurrentMeds = make([]string, len(meds))
		for i, med := range meds {
			if medStr, ok := med.(string); ok {
				criteria.ConcurrentMeds[i] = medStr
			}
		}
	}

	return criteria
}

// selectBestRecipe selects the most appropriate recipe based on criteria matching and priority
func (r *RecipeResolver) selectBestRecipe(recipes []*ClinicalRecipe, criteria PatientCriteria) *ClinicalRecipe {
	if len(recipes) == 1 {
		return recipes[0]
	}

	// Score recipes based on specificity and priority
	bestRecipe := recipes[0]
	bestScore := r.calculateRecipeScore(bestRecipe, criteria)

	for _, recipe := range recipes[1:] {
		score := r.calculateRecipeScore(recipe, criteria)
		if score > bestScore {
			bestScore = score
			bestRecipe = recipe
		}
	}

	return bestRecipe
}

// calculateRecipeScore calculates a score for recipe matching
func (r *RecipeResolver) calculateRecipeScore(recipe *ClinicalRecipe, criteria PatientCriteria) float64 {
	score := float64(recipe.Priority) // Base score from recipe priority

	// Add points for specific criteria matches
	if recipe.PatientCriteria.AgeRange != nil && criteria.AgeRange != nil {
		score += 10.0 // Age-specific recipe
	}

	if recipe.PatientCriteria.WeightRange != nil && criteria.WeightRange != nil {
		score += 5.0 // Weight-specific recipe
	}

	if len(recipe.PatientCriteria.Conditions) > 0 && len(criteria.Conditions) > 0 {
		// Score based on condition overlap
		overlap := r.calculateStringSliceOverlap(recipe.PatientCriteria.Conditions, criteria.Conditions)
		score += overlap * 15.0
	}

	if recipe.PatientCriteria.RenalFunction != nil && criteria.RenalFunction != nil {
		score += 20.0 // Renal-specific recipe
	}

	if recipe.PatientCriteria.HepaticFunction != nil {
		score += 15.0 // Hepatic-specific recipe
	}

	return score
}

// calculateStringSliceOverlap calculates overlap between two string slices
func (r *RecipeResolver) calculateStringSliceOverlap(slice1, slice2 []string) float64 {
	if len(slice1) == 0 || len(slice2) == 0 {
		return 0.0
	}

	overlap := 0
	set1 := make(map[string]bool)
	for _, item := range slice1 {
		set1[item] = true
	}

	for _, item := range slice2 {
		if set1[item] {
			overlap++
		}
	}

	return float64(overlap) / float64(len(slice1))
}

// resolveCalculationSteps resolves all calculation steps with patient data
func (r *RecipeResolver) resolveCalculationSteps(steps []CalculationStep, patientData map[string]interface{}) ([]ResolvedStep, error) {
	resolvedSteps := make([]ResolvedStep, len(steps))

	for i, step := range steps {
		resolvedParams := r.resolveStepParameters(step, patientData)

		resolvedSteps[i] = ResolvedStep{
			StepID:            step.StepID,
			Name:              step.Name,
			Type:              step.Type,
			ResolvedParams:    resolvedParams,
			ExecutionOrder:    step.Order,
			CacheKey:          r.generateStepCacheKey(step.StepID, resolvedParams),
			EstimatedDuration: r.estimateStepDuration(step),
		}
	}

	return resolvedSteps, nil
}

// resolveStepParameters resolves parameters for a single step
func (r *RecipeResolver) resolveStepParameters(step CalculationStep, patientData map[string]interface{}) map[string]interface{} {
	resolved := make(map[string]interface{})

	for key, value := range step.Parameters {
		// Check if value is a reference to patient data
		if strValue, ok := value.(string); ok && len(strValue) > 0 && strValue[0] == '$' {
			// Parameter reference: $patient_weight -> patientData["patient_weight"]
			dataKey := strValue[1:] // Remove $ prefix
			if patientValue, exists := patientData[dataKey]; exists {
				resolved[key] = patientValue
			} else {
				// Keep original value if reference not found
				resolved[key] = value
			}
		} else {
			// Direct value
			resolved[key] = value
		}
	}

	return resolved
}

// generateSnapshotID generates a unique snapshot ID
func (r *RecipeResolver) generateSnapshotID(recipeID, patientID string, patientData map[string]interface{}) string {
	// Create deterministic ID based on recipe, patient, and data hash
	dataBytes, _ := json.Marshal(patientData)
	hash := sha256.Sum256(dataBytes)
	return fmt.Sprintf("snapshot_%s_%s_%x", recipeID[:8], patientID[:8], hash[:8])
}

// generateStepCacheKey generates a cache key for a resolved step
func (r *RecipeResolver) generateStepCacheKey(stepID string, params map[string]interface{}) string {
	paramBytes, _ := json.Marshal(params)
	hash := sha256.Sum256(paramBytes)
	return fmt.Sprintf("step_%s_%x", stepID, hash[:8])
}

// calculateSnapshotHash calculates a hash for snapshot integrity
func (r *RecipeResolver) calculateSnapshotHash(snapshot *RecipeSnapshot) string {
	// Create hash from critical snapshot fields
	hashData := map[string]interface{}{
		"snapshot_id":    snapshot.SnapshotID,
		"recipe_id":      snapshot.RecipeID,
		"patient_data":   snapshot.PatientData,
		"resolved_steps": snapshot.ResolvedSteps,
		"created_at":     snapshot.CreatedAt.Unix(),
	}

	hashBytes, _ := json.Marshal(hashData)
	hash := sha256.Sum256(hashBytes)
	return fmt.Sprintf("%x", hash)
}

// estimateStepDuration estimates execution duration for a step
func (r *RecipeResolver) estimateStepDuration(step CalculationStep) time.Duration {
	// Simple estimation based on step type
	switch step.Type {
	case "DOSE_CALCULATION":
		return 10 * time.Millisecond
	case "FREQUENCY_ADJUSTMENT":
		return 5 * time.Millisecond
	case "INTERACTION_CHECK":
		return 20 * time.Millisecond
	case "RENAL_ADJUSTMENT":
		return 15 * time.Millisecond
	default:
		return 10 * time.Millisecond
	}
}

// deepCopyMap creates a deep copy of a map
func (r *RecipeResolver) deepCopyMap(original map[string]interface{}) map[string]interface{} {
	jsonBytes, _ := json.Marshal(original)
	var copy map[string]interface{}
	json.Unmarshal(jsonBytes, &copy)
	return copy
}

// Recipe creation helpers for testing/setup

// NewClinicalRecipe creates a new clinical recipe
func NewClinicalRecipe(name, version, protocolID string) *ClinicalRecipe {
	return &ClinicalRecipe{
		RecipeID:          fmt.Sprintf("recipe_%d", time.Now().Unix()),
		Name:              name,
		Version:           version,
		ProtocolID:        protocolID,
		PatientCriteria:   PatientCriteria{},
		CalculationSteps:  make([]CalculationStep, 0),
		ValidationRules:   make([]ValidationRule, 0),
		OptimizationHints: make([]OptimizationHint, 0),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Active:            true,
		Priority:          50, // Normal priority
	}
}

// AddCalculationStep adds a calculation step to the recipe
func (recipe *ClinicalRecipe) AddCalculationStep(stepID, name, stepType string, parameters map[string]interface{}, order int) {
	step := CalculationStep{
		StepID:       stepID,
		Name:         name,
		Type:         stepType,
		Parameters:   parameters,
		Dependencies: make([]string, 0),
		Order:        order,
		Conditions:   make([]StepCondition, 0),
	}
	recipe.CalculationSteps = append(recipe.CalculationSteps, step)
}

// AddValidationRule adds a validation rule to the recipe
func (recipe *ClinicalRecipe) AddValidationRule(ruleID, name, ruleType, severity, message string, parameters map[string]interface{}) {
	rule := ValidationRule{
		RuleID:     ruleID,
		Name:       name,
		Type:       ruleType,
		Parameters: parameters,
		Severity:   severity,
		Message:    message,
		Conditions: make([]RuleCondition, 0),
	}
	recipe.ValidationRules = append(recipe.ValidationRules, rule)
}