// Package services provides recipe management for clinical workflows
package services

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"context-gateway-go/internal/models"
)

// RecipeService manages clinical workflow recipes
type RecipeService struct {
	recipes map[string]*models.WorkflowRecipe
	mu      sync.RWMutex
}

// NewRecipeService creates a new recipe service instance
func NewRecipeService() *RecipeService {
	service := &RecipeService{
		recipes: make(map[string]*models.WorkflowRecipe),
	}
	
	// Load default recipes
	if err := service.LoadDefaultRecipes(); err != nil {
		log.Printf("Warning: Failed to load default recipes: %v", err)
	}
	
	return service
}

// LoadDefaultRecipes loads built-in clinical recipes
func (rs *RecipeService) LoadDefaultRecipes() error {
	// Medication Prescribing Recipe
	medicationRecipe := &models.WorkflowRecipe{
		RecipeID:         "medication_prescribing_v2",
		RecipeName:       "Medication Prescribing Context",
		Version:          "2.0.0",
		ClinicalScenario: "Provider prescribing medication for patient",
		WorkflowCategory: "command_initiated",
		ExecutionPattern: "pessimistic",
		SLAMs:           200,
		RequiredFields: []models.DataPoint{
			{
				Name:                 "patient_demographics",
				SourceType:           models.DataSourcePatientService,
				Fields:               []string{"age", "weight", "height", "gender", "allergies"},
				Required:             true,
				MaxAgeHours:          24,
				QualityThreshold:     0.9,
				TimeoutMs:            2000,
				RetryCount:           2,
				FreshnessRequirement: 60,
			},
			{
				Name:                 "current_medications",
				SourceType:           models.DataSourceMedicationService,
				Fields:               []string{"active_medications", "dosages", "start_dates"},
				Required:             true,
				MaxAgeHours:          6,
				QualityThreshold:     0.85,
				TimeoutMs:            3000,
				RetryCount:           3,
				FreshnessRequirement: 30,
				FallbackSources:      []models.DataSourceType{models.DataSourceFHIRStore},
			},
			{
				Name:                 "lab_results",
				SourceType:           models.DataSourceObservationService,
				Fields:               []string{"creatinine", "liver_function", "recent_labs"},
				Required:             false,
				MaxAgeHours:          72,
				QualityThreshold:     0.8,
				TimeoutMs:            2500,
				RetryCount:           2,
				FreshnessRequirement: 180,
			},
		},
		QualityConstraints: models.QualityConstraints{
			MinimumCompleteness: 0.85,
			MaximumAgeHours:     24,
			RequiredFields:      []string{"patient_demographics", "current_medications"},
			AccuracyThreshold:   0.9,
		},
		SafetyRequirements: models.SafetyRequirements{
			MinimumCompletenessScore:    0.85,
			AbsoluteRequiredEnforcement: "STRICT",
			PreferredDataHandling:       "GRACEFUL_DEGRADE",
			CriticalMissingDataAction:   "FAIL_WORKFLOW",
			StaleDataAction:             "FLAG_FOR_REVIEW",
		},
		CacheStrategy: models.CacheStrategy{
			L1TTLSeconds:       300,  // 5 minutes
			L2TTLSeconds:       900,  // 15 minutes
			L3TTLSeconds:       3600, // 1 hour
			InvalidationEvents: []string{"medication_change", "allergy_update", "weight_change"},
			CacheKeyPattern:    "context:{patient_id}:{recipe_id}",
		},
		AssemblyRules: models.AssemblyRules{
			ParallelExecution:           true,
			TimeoutBudgetMs:             200,
			CircuitBreakerEnabled:       true,
			RetryFailedSources:          true,
			ValidateDataFreshness:       true,
			EnforceQualityConstraints:   true,
		},
		GovernanceMetadata: models.GovernanceMetadata{
			ApprovedBy:              "Clinical Governance Board",
			ApprovalDate:            time.Now().AddDate(0, -1, 0), // 1 month ago
			Version:                 "2.0.0",
			EffectiveDate:           time.Now().AddDate(0, -1, 0),
			ClinicalBoardApprovalID: "CGB-2024-MED-001",
			Tags:                    []string{"medication", "prescribing", "safety"},
			ChangeLog:               []string{"Added lab results requirement", "Updated safety constraints"},
		},
	}
	
	// Safety Context Recipe
	safetyRecipe := &models.WorkflowRecipe{
		RecipeID:         "safety_gateway_context_v1",
		RecipeName:       "Clinical Safety Context",
		Version:          "1.0.0",
		ClinicalScenario: "Safety gateway evaluating clinical decision",
		WorkflowCategory: "event_triggered",
		ExecutionPattern: "digital_reflex_arc",
		SLAMs:           100, // Very fast for safety
		RequiredFields: []models.DataPoint{
			{
				Name:                 "active_medications",
				SourceType:           models.DataSourceMedicationService,
				Fields:               []string{"medications", "dosages", "interactions"},
				Required:             true,
				MaxAgeHours:          1,
				QualityThreshold:     0.95,
				TimeoutMs:            1500,
				RetryCount:           3,
				FreshnessRequirement: 15, // 15 minutes for safety
			},
			{
				Name:                 "allergies",
				SourceType:           models.DataSourcePatientService,
				Fields:               []string{"drug_allergies", "food_allergies", "severity"},
				Required:             true,
				MaxAgeHours:          24,
				QualityThreshold:     0.98,
				TimeoutMs:            1000,
				RetryCount:           2,
				FreshnessRequirement: 60,
			},
			{
				Name:                 "vital_signs",
				SourceType:           models.DataSourceObservationService,
				Fields:               []string{"blood_pressure", "heart_rate", "temperature"},
				Required:             true,
				MaxAgeHours:          2,
				QualityThreshold:     0.9,
				TimeoutMs:            1500,
				RetryCount:           2,
				FreshnessRequirement: 30,
			},
		},
		QualityConstraints: models.QualityConstraints{
			MinimumCompleteness: 0.95,
			MaximumAgeHours:     2,
			RequiredFields:      []string{"active_medications", "allergies", "vital_signs"},
			AccuracyThreshold:   0.98,
		},
		SafetyRequirements: models.SafetyRequirements{
			MinimumCompletenessScore:    0.95,
			AbsoluteRequiredEnforcement: "STRICT",
			PreferredDataHandling:       "FAIL",
			CriticalMissingDataAction:   "FAIL_WORKFLOW",
			StaleDataAction:             "REJECT",
		},
		CacheStrategy: models.CacheStrategy{
			L1TTLSeconds:       60,   // 1 minute for safety data
			L2TTLSeconds:       300,  // 5 minutes
			L3TTLSeconds:       600,  // 10 minutes
			InvalidationEvents: []string{"medication_change", "allergy_update", "vital_signs_change"},
			CacheKeyPattern:    "safety_context:{patient_id}:{recipe_id}",
		},
		AssemblyRules: models.AssemblyRules{
			ParallelExecution:           true,
			TimeoutBudgetMs:             100,
			CircuitBreakerEnabled:       true,
			RetryFailedSources:          true,
			ValidateDataFreshness:       true,
			EnforceQualityConstraints:   true,
		},
		GovernanceMetadata: models.GovernanceMetadata{
			ApprovedBy:              "Clinical Governance Board",
			ApprovalDate:            time.Now().AddDate(0, 0, -15), // 15 days ago
			Version:                 "1.0.0",
			EffectiveDate:           time.Now().AddDate(0, 0, -15),
			ClinicalBoardApprovalID: "CGB-2024-SAFE-001",
			Tags:                    []string{"safety", "alerts", "critical"},
			ChangeLog:               []string{"Initial safety context recipe"},
		},
	}
	
	// Emergency Response Recipe
	emergencyRecipe := &models.WorkflowRecipe{
		RecipeID:         "code_blue_context_v2",
		RecipeName:       "Emergency Response Context",
		Version:          "2.0.0",
		ClinicalScenario: "Code Blue emergency response",
		WorkflowCategory: "event_triggered",
		ExecutionPattern: "digital_reflex_arc",
		SLAMs:           50, // Ultra-fast for emergencies
		RequiredFields: []models.DataPoint{
			{
				Name:                 "patient_vital_status",
				SourceType:           models.DataSourceObservationService,
				Fields:               []string{"consciousness", "breathing", "pulse", "blood_pressure"},
				Required:             true,
				MaxAgeHours:          0, // Real-time data only
				QualityThreshold:     0.98,
				TimeoutMs:            500,
				RetryCount:           1,
				FreshnessRequirement: 1, // 1 minute maximum
			},
			{
				Name:                 "emergency_medications",
				SourceType:           models.DataSourceMedicationService,
				Fields:               []string{"crash_cart_meds", "contraindications"},
				Required:             true,
				MaxAgeHours:          24,
				QualityThreshold:     0.99,
				TimeoutMs:            800,
				RetryCount:           2,
				FreshnessRequirement: 60,
			},
			{
				Name:                 "emergency_contacts",
				SourceType:           models.DataSourcePatientService,
				Fields:               []string{"emergency_contact", "primary_physician", "code_status"},
				Required:             true,
				MaxAgeHours:          168, // 1 week
				QualityThreshold:     0.9,
				TimeoutMs:            1000,
				RetryCount:           1,
				FreshnessRequirement: 1440, // 24 hours
			},
		},
		QualityConstraints: models.QualityConstraints{
			MinimumCompleteness: 0.9,
			MaximumAgeHours:     1,
			RequiredFields:      []string{"patient_vital_status", "emergency_medications"},
			AccuracyThreshold:   0.98,
		},
		SafetyRequirements: models.SafetyRequirements{
			MinimumCompletenessScore:    0.9,
			AbsoluteRequiredEnforcement: "STRICT",
			PreferredDataHandling:       "FAIL",
			CriticalMissingDataAction:   "FAIL_WORKFLOW",
			StaleDataAction:             "REJECT",
		},
		CacheStrategy: models.CacheStrategy{
			L1TTLSeconds:       30,   // 30 seconds for emergency data
			L2TTLSeconds:       60,   // 1 minute
			L3TTLSeconds:       300,  // 5 minutes
			InvalidationEvents: []string{"vital_signs_critical", "code_status_change"},
			CacheKeyPattern:    "emergency:{patient_id}:{recipe_id}",
		},
		AssemblyRules: models.AssemblyRules{
			ParallelExecution:           true,
			TimeoutBudgetMs:             50,
			CircuitBreakerEnabled:       false, // No circuit breaker for emergencies
			RetryFailedSources:          false, // No retries for speed
			ValidateDataFreshness:       true,
			EnforceQualityConstraints:   true,
		},
		GovernanceMetadata: models.GovernanceMetadata{
			ApprovedBy:              "Clinical Governance Board",
			ApprovalDate:            time.Now().AddDate(0, 0, -30), // 30 days ago
			Version:                 "2.0.0",
			EffectiveDate:           time.Now().AddDate(0, 0, -30),
			ClinicalBoardApprovalID: "CGB-2024-EMRG-002",
			Tags:                    []string{"emergency", "code_blue", "critical_care"},
			ChangeLog:               []string{"Reduced timeout for emergency response", "Added vital status priority"},
		},
	}
	
	// Register all recipes
	recipes := []*models.WorkflowRecipe{medicationRecipe, safetyRecipe, emergencyRecipe}
	
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	for _, recipe := range recipes {
		rs.recipes[recipe.RecipeID] = recipe
		log.Printf("Loaded default recipe: %s v%s", recipe.RecipeID, recipe.Version)
	}
	
	return nil
}

// LoadRecipeFromFile loads a recipe from YAML file
func (rs *RecipeService) LoadRecipeFromFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read recipe file %s: %w", filePath, err)
	}
	
	var recipe models.WorkflowRecipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return fmt.Errorf("failed to parse recipe YAML %s: %w", filePath, err)
	}
	
	// Validate the recipe
	valid, errors, warnings := recipe.Validate()
	if !valid {
		return fmt.Errorf("recipe validation failed for %s: %v", filePath, errors)
	}
	
	if len(warnings) > 0 {
		log.Printf("Recipe warnings for %s: %v", filePath, warnings)
	}
	
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	rs.recipes[recipe.RecipeID] = &recipe
	log.Printf("Loaded recipe from file: %s v%s (%s)", recipe.RecipeID, recipe.Version, filepath.Base(filePath))
	
	return nil
}

// LoadRecipesFromDirectory loads all recipe files from a directory
func (rs *RecipeService) LoadRecipesFromDirectory(dirPath string) error {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to find recipe files in %s: %w", dirPath, err)
	}
	
	yamlFiles, err := filepath.Glob(filepath.Join(dirPath, "*.yml"))
	if err == nil {
		files = append(files, yamlFiles...)
	}
	
	var loadErrors []error
	loadedCount := 0
	
	for _, file := range files {
		if err := rs.LoadRecipeFromFile(file); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to load %s: %w", file, err))
		} else {
			loadedCount++
		}
	}
	
	log.Printf("Loaded %d recipes from directory %s", loadedCount, dirPath)
	
	if len(loadErrors) > 0 {
		return fmt.Errorf("errors loading recipes: %v", loadErrors)
	}
	
	return nil
}

// GetRecipe retrieves a recipe by ID
func (rs *RecipeService) GetRecipe(recipeID string) (*models.WorkflowRecipe, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	recipe, exists := rs.recipes[recipeID]
	if !exists {
		return nil, fmt.Errorf("recipe not found: %s", recipeID)
	}
	
	// Check if recipe is expired
	if recipe.GovernanceMetadata.IsExpired() {
		return nil, fmt.Errorf("recipe %s has expired", recipeID)
	}
	
	// Check if recipe is approved
	if !recipe.GovernanceMetadata.IsApproved() {
		log.Printf("Warning: Recipe %s is not approved by Clinical Governance Board", recipeID)
	}
	
	return recipe, nil
}

// GetRecipeVersion retrieves a specific version of a recipe
func (rs *RecipeService) GetRecipeVersion(recipeID, version string) (*models.WorkflowRecipe, error) {
	// For this implementation, we don't support multiple versions in memory
	// In production, you'd store versions in a database or versioned storage
	recipe, err := rs.GetRecipe(recipeID)
	if err != nil {
		return nil, err
	}
	
	if recipe.Version != version {
		return nil, fmt.Errorf("recipe version %s not found for %s (available: %s)", 
			version, recipeID, recipe.Version)
	}
	
	return recipe, nil
}

// ListRecipes returns all available recipes
func (rs *RecipeService) ListRecipes() []*models.WorkflowRecipe {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	recipes := make([]*models.WorkflowRecipe, 0, len(rs.recipes))
	for _, recipe := range rs.recipes {
		// Only include valid (approved and not expired) recipes
		if recipe.IsValid() {
			recipes = append(recipes, recipe)
		}
	}
	
	return recipes
}

// ValidateRecipe validates a recipe against governance rules
func (rs *RecipeService) ValidateRecipe(recipe *models.WorkflowRecipe) (*models.RecipeValidationResult, error) {
	startTime := time.Now()
	
	valid, errors, warnings := recipe.Validate()
	
	result := &models.RecipeValidationResult{
		RecipeID:             recipe.RecipeID,
		Valid:                valid,
		Errors:               errors,
		Warnings:             warnings,
		ValidationDurationMs: float64(time.Since(startTime).Milliseconds()),
		ValidatedAt:          time.Now().UTC(),
	}
	
	return result, nil
}

// RegisterRecipe adds a new recipe to the service
func (rs *RecipeService) RegisterRecipe(recipe *models.WorkflowRecipe) error {
	// Validate the recipe first
	valid, errors, _ := recipe.Validate()
	if !valid {
		return fmt.Errorf("recipe validation failed: %v", errors)
	}
	
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	// Check for duplicate recipe ID
	if existing, exists := rs.recipes[recipe.RecipeID]; exists {
		log.Printf("Warning: Overwriting existing recipe %s (v%s -> v%s)", 
			recipe.RecipeID, existing.Version, recipe.Version)
	}
	
	rs.recipes[recipe.RecipeID] = recipe
	log.Printf("Registered recipe: %s v%s", recipe.RecipeID, recipe.Version)
	
	return nil
}

// UnregisterRecipe removes a recipe from the service
func (rs *RecipeService) UnregisterRecipe(recipeID string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	if _, exists := rs.recipes[recipeID]; !exists {
		return fmt.Errorf("recipe not found: %s", recipeID)
	}
	
	delete(rs.recipes, recipeID)
	log.Printf("Unregistered recipe: %s", recipeID)
	
	return nil
}

// GetRecipeStats returns statistics about loaded recipes
func (rs *RecipeService) GetRecipeStats() map[string]interface{} {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	totalRecipes := len(rs.recipes)
	approvedRecipes := 0
	expiredRecipes := 0
	scenarioCount := make(map[string]int)
	categoryCount := make(map[string]int)
	
	for _, recipe := range rs.recipes {
		if recipe.GovernanceMetadata.IsApproved() {
			approvedRecipes++
		}
		if recipe.GovernanceMetadata.IsExpired() {
			expiredRecipes++
		}
		
		scenarioCount[recipe.ClinicalScenario]++
		categoryCount[recipe.WorkflowCategory]++
	}
	
	return map[string]interface{}{
		"total_recipes":    totalRecipes,
		"approved_recipes": approvedRecipes,
		"expired_recipes":  expiredRecipes,
		"valid_recipes":    totalRecipes - expiredRecipes,
		"scenario_distribution": scenarioCount,
		"category_distribution": categoryCount,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}