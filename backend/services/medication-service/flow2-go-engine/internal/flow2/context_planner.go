package flow2

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/models"
	"flow2-go-engine/internal/orb"
)

// ContextPlanner converts Intent Manifests into Context Service requests
// This replaces the generic context assembly with intelligent, targeted data gathering
type ContextPlanner struct {
	contextRecipes *orb.ContextServiceRecipeBook
	logger         *logrus.Logger
}

// NewContextPlanner creates a new context planner
func NewContextPlanner(contextRecipes *orb.ContextServiceRecipeBook, logger *logrus.Logger) *ContextPlanner {
	return &ContextPlanner{
		contextRecipes: contextRecipes,
		logger:         logger,
	}
}

// PlanDataRequirements converts an Intent Manifest into a Context Service request
// This is the bridge between ORB decisions and Context Service execution
func (cp *ContextPlanner) PlanDataRequirements(intentManifest *orb.IntentManifest) *models.ContextRequest {
	// Create base context request from Intent Manifest
	contextRequest := &models.ContextRequest{
		PatientID:        intentManifest.PatientID,
		DataRequirements: intentManifest.DataRequirements,
		Priority:         intentManifest.Priority,
		RequestID:        intentManifest.RequestID,
		TimeoutMs:        cp.calculateTimeout(intentManifest.Priority),
	}

	// Enhance with recipe-specific requirements if available
	if recipe := cp.getContextRecipe(intentManifest.MedicationCode, intentManifest.RecipeID); recipe != nil {
		contextRequest = cp.enhanceWithRecipeRequirements(contextRequest, recipe, intentManifest)
	}

	cp.logger.WithFields(logrus.Fields{
		"request_id":        intentManifest.RequestID,
		"recipe_id":         intentManifest.RecipeID,
		"data_requirements": len(contextRequest.DataRequirements),
		"priority":          contextRequest.Priority,
		"timeout_ms":        contextRequest.TimeoutMs,
	}).Info("Context Planner created targeted data request")

	return contextRequest
}

// getContextRecipe retrieves the appropriate context recipe
func (cp *ContextPlanner) getContextRecipe(medicationCode, recipeID string) *orb.ContextRecipe {
	// First try medication-specific recipe
	if recipe, exists := cp.contextRecipes.Recipes[medicationCode]; exists {
		return recipe
	}

	// Fall back to standard recipe
	if recipe, exists := cp.contextRecipes.Recipes["standard"]; exists {
		return recipe
	}

	cp.logger.WithFields(logrus.Fields{
		"medication_code": medicationCode,
		"recipe_id":       recipeID,
	}).Warn("No context recipe found, using base requirements only")

	return nil
}

// enhanceWithRecipeRequirements adds recipe-specific data requirements
func (cp *ContextPlanner) enhanceWithRecipeRequirements(
	baseRequest *models.ContextRequest,
	recipe *orb.ContextRecipe,
	intentManifest *orb.IntentManifest,
) *models.ContextRequest {

	// Start with base requirements from Intent Manifest
	allRequirements := make(map[string]bool)
	for _, req := range baseRequest.DataRequirements {
		allRequirements[req] = true
	}

	// Add base requirements from recipe
	for _, req := range recipe.BaseRequirements {
		if req.Required {
			allRequirements[req.Field] = true
		}
	}

	// Add recipe-specific requirements
	if recipeSpecific, exists := recipe.RecipeSpecificRequirements[intentManifest.RecipeID]; exists {
		for _, req := range recipeSpecific.AdditionalRequirements {
			if req.Required {
				allRequirements[req.Field] = true
			}
		}
	}

	// Add medication-specific requirements
	if medicationSpecific, exists := recipe.MedicationSpecificRequirements[intentManifest.MedicationCode]; exists {
		for _, req := range medicationSpecific.AdditionalRequirements {
			if req.Required {
				allRequirements[req.Field] = true
			}
		}
	}

	// Convert back to slice
	var finalRequirements []string
	for req := range allRequirements {
		finalRequirements = append(finalRequirements, req)
	}

	baseRequest.DataRequirements = finalRequirements

	cp.logger.WithFields(logrus.Fields{
		"request_id":           intentManifest.RequestID,
		"base_requirements":    len(baseRequest.DataRequirements),
		"enhanced_requirements": len(finalRequirements),
		"recipe_used":          recipe.RecipeID,
	}).Info("Enhanced context request with recipe requirements")

	return baseRequest
}

// calculateTimeout determines appropriate timeout based on priority
func (cp *ContextPlanner) calculateTimeout(priority string) int {
	switch priority {
	case "critical":
		return 1000 // 1 second for critical
	case "high":
		return 2000 // 2 seconds for high
	case "medium":
		return 3000 // 3 seconds for medium
	case "low":
		return 5000 // 5 seconds for low
	default:
		return 3000 // Default 3 seconds
	}
}

// ValidateContextResponse validates the response from Context Service
func (cp *ContextPlanner) ValidateContextResponse(
	contextRequest *models.ContextRequest,
	contextResponse *models.ClinicalContext,
) error {

	// Check if we got the minimum required data
	requiredFields := make(map[string]bool)
	for _, req := range contextRequest.DataRequirements {
		requiredFields[req] = true
	}

	missingFields := []string{}
	for field := range requiredFields {
		if _, exists := contextResponse.Fields[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		cp.logger.WithFields(logrus.Fields{
			"request_id":      contextRequest.RequestID,
			"missing_fields":  missingFields,
			"total_missing":   len(missingFields),
			"total_required":  len(contextRequest.DataRequirements),
		}).Warn("Context Service response missing required fields")

		// For now, we'll proceed with partial data but log the issue
		// In production, you might want to fail or retry based on criticality
	}

	// Calculate completeness
	completeness := float64(len(contextResponse.Fields)) / float64(len(contextRequest.DataRequirements))
	contextResponse.Completeness = completeness

	cp.logger.WithFields(logrus.Fields{
		"request_id":    contextRequest.RequestID,
		"completeness":  completeness,
		"fields_received": len(contextResponse.Fields),
		"fields_requested": len(contextRequest.DataRequirements),
	}).Info("Context response validation completed")

	return nil
}

// OptimizeDataRequirements removes duplicate or redundant requirements
func (cp *ContextPlanner) OptimizeDataRequirements(requirements []string) []string {
	// Remove duplicates
	seen := make(map[string]bool)
	optimized := []string{}

	for _, req := range requirements {
		if !seen[req] {
			seen[req] = true
			optimized = append(optimized, req)
		}
	}

	// Apply optimization rules (e.g., if we have weight_kg, we don't need weight_lbs)
	optimized = cp.applyOptimizationRules(optimized)

	return optimized
}

// applyOptimizationRules applies intelligent optimization to data requirements
func (cp *ContextPlanner) applyOptimizationRules(requirements []string) []string {
	// Example optimization rules
	optimizationRules := map[string][]string{
		"weight_kg": {"weight_lbs", "weight_pounds"}, // If we have kg, remove lbs
		"age_years": {"age_months", "age_days"},      // If we have years, remove months/days
		"creatinine_clearance": {"estimated_gfr"},    // Prefer calculated clearance
	}

	optimized := []string{}
	toRemove := make(map[string]bool)

	// Mark items for removal based on optimization rules
	for _, req := range requirements {
		if redundant, exists := optimizationRules[req]; exists {
			for _, remove := range redundant {
				toRemove[remove] = true
			}
		}
	}

	// Keep only non-redundant requirements
	for _, req := range requirements {
		if !toRemove[req] {
			optimized = append(optimized, req)
		}
	}

	if len(optimized) < len(requirements) {
		cp.logger.WithFields(logrus.Fields{
			"original_count":  len(requirements),
			"optimized_count": len(optimized),
			"removed_count":   len(requirements) - len(optimized),
		}).Info("Applied data requirement optimizations")
	}

	return optimized
}

// GetContextPlanningMetrics returns metrics about context planning performance
func (cp *ContextPlanner) GetContextPlanningMetrics() map[string]interface{} {
	return map[string]interface{}{
		"available_recipes": len(cp.contextRecipes.Recipes),
		"planning_version":  "orb_driven_v1",
		"optimization_enabled": true,
	}
}
