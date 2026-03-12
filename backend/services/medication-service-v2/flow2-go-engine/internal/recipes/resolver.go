package recipes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/models"
)

// RecipeResolver implements the Phase 1 recipe resolution logic
// It resolves both Context Recipes (data requirements) and Clinical Recipes (therapy protocols)
type RecipeResolver struct {
	apolloClient     *clients.ApolloFederationClient
	cache           *RecipeCache
	logger          *logrus.Logger
	
	// In-memory caches for performance
	contextRecipes   map[string]*models.ContextRecipe
	clinicalRecipes  map[string]*models.ClinicalRecipe
}

// RecipeCache provides caching for recipe resolution
type RecipeCache struct {
	contextRecipesTTL  time.Duration
	clinicalRecipesTTL time.Duration
	contextCache       map[string]CacheEntry
	clinicalCache      map[string]CacheEntry
}

// CacheEntry represents a cached entry
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// NewRecipeResolver creates a new recipe resolver
func NewRecipeResolver(apolloClient *clients.ApolloFederationClient, logger *logrus.Logger) *RecipeResolver {
	return &RecipeResolver{
		apolloClient:    apolloClient,
		cache:          NewRecipeCache(),
		logger:         logger,
		contextRecipes: make(map[string]*models.ContextRecipe),
		clinicalRecipes: make(map[string]*models.ClinicalRecipe),
	}
}

// NewRecipeCache creates a new recipe cache
func NewRecipeCache() *RecipeCache {
	return &RecipeCache{
		contextRecipesTTL:  1 * time.Hour,
		clinicalRecipesTTL: 1 * time.Hour,
		contextCache:       make(map[string]CacheEntry),
		clinicalCache:      make(map[string]CacheEntry),
	}
}

// ResolveRecipes performs the core Phase 1 recipe resolution
// This updates the IntentManifest with recipe details and field requirements
func (rr *RecipeResolver) ResolveRecipes(
	ctx context.Context,
	manifest *models.IntentManifest,
	request *models.MedicationRequest,
) error {
	startTime := time.Now()
	
	rr.logger.WithFields(logrus.Fields{
		"manifest_id": manifest.ManifestID,
		"protocol_id": manifest.ProtocolID,
	}).Info("Starting recipe resolution")
	
	// Step 1: Resolve Context Recipe (data requirements)
	contextRecipe, err := rr.resolveContextRecipe(ctx, manifest.ProtocolID)
	if err != nil {
		return fmt.Errorf("failed to resolve context recipe: %w", err)
	}
	
	// Step 2: Apply conditional field requirements
	enhancedContextRecipe := rr.applyConditionalFields(contextRecipe, request)
	
	// Step 3: Resolve Clinical Recipe (therapy protocols)
	clinicalRecipe, err := rr.resolveClinicalRecipe(ctx, manifest.ProtocolID)
	if err != nil {
		return fmt.Errorf("failed to resolve clinical recipe: %w", err)
	}
	
	// Step 4: Merge field requirements
	requiredFields, optionalFields := rr.mergeFieldRequirements(
		enhancedContextRecipe,
		clinicalRecipe,
	)
	
	// Step 5: Update manifest with recipe details
	manifest.ContextRecipeID = enhancedContextRecipe.ID
	manifest.ClinicalRecipeID = clinicalRecipe.ID
	manifest.RequiredFields = requiredFields
	manifest.OptionalFields = optionalFields
	manifest.DataFreshness = rr.determineFreshnessRequirements(
		enhancedContextRecipe,
		clinicalRecipe,
	)
	manifest.SnapshotTTL = rr.calculateSnapshotTTL(manifest.DataFreshness)
	
	rr.logger.WithFields(logrus.Fields{
		"manifest_id":       manifest.ManifestID,
		"context_recipe_id": manifest.ContextRecipeID,
		"clinical_recipe_id": manifest.ClinicalRecipeID,
		"required_fields":   len(manifest.RequiredFields),
		"optional_fields":   len(manifest.OptionalFields),
		"resolution_time_ms": time.Since(startTime).Milliseconds(),
	}).Info("Recipe resolution completed")
	
	return nil
}

// resolveContextRecipe resolves the context recipe for data requirements
func (rr *RecipeResolver) resolveContextRecipe(ctx context.Context, protocolID string) (*models.ContextRecipe, error) {
	// Check cache first
	if cached, ok := rr.cache.GetContextRecipe(protocolID); ok {
		return cached, nil
	}
	
	// Query Apollo Federation for context recipe
	rr.logger.WithField("protocol_id", protocolID).Debug("Loading context recipe from Apollo Federation")
	
	result, err := rr.apolloClient.LoadContextRecipe(ctx, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query context recipe: %w", err)
	}
	
	// Parse the Apollo response into ContextRecipe
	contextRecipe, err := rr.parseContextRecipe(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse context recipe: %w", err)
	}
	
	// Cache for future use
	rr.cache.SetContextRecipe(protocolID, contextRecipe)
	
	return contextRecipe, nil
}

// resolveClinicalRecipe resolves the clinical recipe for therapy protocols
func (rr *RecipeResolver) resolveClinicalRecipe(ctx context.Context, protocolID string) (*models.ClinicalRecipe, error) {
	// Check cache first
	if cached, ok := rr.cache.GetClinicalRecipe(protocolID); ok {
		return cached, nil
	}
	
	// Query Apollo Federation for clinical recipe
	rr.logger.WithField("protocol_id", protocolID).Debug("Loading clinical recipe from Apollo Federation")
	
	result, err := rr.apolloClient.LoadClinicalRecipe(ctx, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query clinical recipe: %w", err)
	}
	
	// Parse the Apollo response into ClinicalRecipe
	clinicalRecipe, err := rr.parseClinicalRecipe(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clinical recipe: %w", err)
	}
	
	// Cache for future use
	rr.cache.SetClinicalRecipe(protocolID, clinicalRecipe)
	
	return clinicalRecipe, nil
}

// applyConditionalFields applies conditional field logic based on patient characteristics
func (rr *RecipeResolver) applyConditionalFields(
	recipe *models.ContextRecipe,
	request *models.MedicationRequest,
) *models.ContextRecipe {
	enhanced := &models.ContextRecipe{
		ID:         recipe.ID,
		ProtocolID: recipe.ProtocolID,
		Version:    recipe.Version,
		CoreFields: make([]models.FieldSpec, len(recipe.CoreFields)),
	}
	copy(enhanced.CoreFields, recipe.CoreFields)
	
	// Evaluate each conditional rule
	for _, rule := range recipe.ConditionalRules {
		if rr.evaluateCondition(rule.Condition, request) {
			rr.logger.WithFields(logrus.Fields{
				"condition": rule.Condition,
				"rationale": rule.Rationale,
				"additional_fields": len(rule.RequiredFields),
			}).Debug("Conditional rule matched, adding fields")
			
			enhanced.CoreFields = append(enhanced.CoreFields, rule.RequiredFields...)
		}
	}
	
	// Deduplicate fields
	enhanced.CoreFields = rr.deduplicateFields(enhanced.CoreFields)
	
	return enhanced
}

// evaluateCondition evaluates a conditional rule expression
func (rr *RecipeResolver) evaluateCondition(condition string, request *models.MedicationRequest) bool {
	// Simple condition evaluation logic
	// In production, this would be a more sophisticated expression evaluator
	
	condition = strings.ToLower(strings.TrimSpace(condition))
	
	// Age-based conditions
	if strings.Contains(condition, "age") && request.ClinicalContext.Age > 0 {
		if strings.Contains(condition, ">=65") {
			return request.ClinicalContext.Age >= 65
		}
		if strings.Contains(condition, "<65") {
			return request.ClinicalContext.Age < 65
		}
		if strings.Contains(condition, ">=18") {
			return request.ClinicalContext.Age >= 18
		}
	}
	
	// Comorbidity-based conditions
	if strings.Contains(condition, "ckd") || strings.Contains(condition, "renal") {
		for _, comorbidity := range request.ClinicalContext.Comorbidities {
			if strings.Contains(strings.ToLower(comorbidity), "ckd") ||
			   strings.Contains(strings.ToLower(comorbidity), "renal") {
				return true
			}
		}
	}
	
	if strings.Contains(condition, "diabetes") {
		for _, comorbidity := range request.ClinicalContext.Comorbidities {
			if strings.Contains(strings.ToLower(comorbidity), "diabetes") {
				return true
			}
		}
	}
	
	// Weight-based conditions
	if strings.Contains(condition, "weight") && request.ClinicalContext.Weight > 0 {
		if strings.Contains(condition, ">100kg") {
			return request.ClinicalContext.Weight > 100
		}
	}
	
	// Medication-based conditions
	if strings.Contains(condition, "anticoagulant") {
		for _, med := range request.ClinicalContext.CurrentMeds {
			medName := strings.ToLower(med.MedicationName)
			if strings.Contains(medName, "warfarin") ||
			   strings.Contains(medName, "heparin") ||
			   strings.Contains(medName, "apixaban") {
				return true
			}
		}
	}
	
	// Default: return false for unknown conditions
	rr.logger.WithField("condition", condition).Warn("Unknown condition in conditional rule")
	return false
}

// mergeFieldRequirements merges field requirements from context and clinical recipes
func (rr *RecipeResolver) mergeFieldRequirements(
	contextRecipe *models.ContextRecipe,
	clinicalRecipe *models.ClinicalRecipe,
) ([]models.FieldRequirement, []models.FieldRequirement) {
	
	var required []models.FieldRequirement
	var optional []models.FieldRequirement
	
	// Add context recipe fields
	for _, field := range contextRecipe.CoreFields {
		fieldReq := models.FieldRequirement{
			FieldName:      field.Name,
			FieldType:      field.Type,
			Required:       field.Required,
			MaxAgeHours:    field.MaxAgeHours,
			Source:         "EHR", // Default source
			ClinicalReason: field.ClinicalContext,
		}
		
		if field.Required {
			required = append(required, fieldReq)
		} else {
			optional = append(optional, fieldReq)
		}
	}
	
	// Add clinical recipe monitoring requirements as optional fields
	for _, param := range clinicalRecipe.MonitoringPlan.Required {
		fieldReq := models.FieldRequirement{
			FieldName:      param.Parameter,
			FieldType:      "MONITORING",
			Required:       false, // Monitoring fields are typically optional for Phase 1
			MaxAgeHours:    24,    // Default 24 hours for monitoring
			Source:         "LAB",
			ClinicalReason: fmt.Sprintf("Required monitoring for %s", clinicalRecipe.ID),
		}
		optional = append(optional, fieldReq)
	}
	
	return required, optional
}

// determineFreshnessRequirements determines data freshness requirements
func (rr *RecipeResolver) determineFreshnessRequirements(
	contextRecipe *models.ContextRecipe,
	clinicalRecipe *models.ClinicalRecipe,
) models.FreshnessRequirements {
	
	// Find the most restrictive freshness requirement
	maxAge := 24 * time.Hour // Default 24 hours
	
	for _, rule := range contextRecipe.FreshnessRules {
		if rule.MaxAge < maxAge {
			maxAge = rule.MaxAge
		}
	}
	
	// Identify critical fields
	var criticalFields []string
	for _, field := range contextRecipe.CoreFields {
		if field.Required && field.MaxAgeHours <= 6 {
			criticalFields = append(criticalFields, field.Name)
		}
	}
	
	return models.FreshnessRequirements{
		MaxAge:         maxAge,
		CriticalFields: criticalFields,
		PreferredSources: []string{"EHR", "LAB", "DEVICE"}, // Default preferred sources
	}
}

// calculateSnapshotTTL calculates the snapshot TTL based on freshness requirements
func (rr *RecipeResolver) calculateSnapshotTTL(freshness models.FreshnessRequirements) int {
	// Convert freshness max age to seconds, with minimum of 300 seconds (5 minutes)
	ttlSeconds := int(freshness.MaxAge.Seconds())
	if ttlSeconds < 300 {
		ttlSeconds = 300
	}
	
	// Maximum TTL of 1 hour for Phase 1 performance
	if ttlSeconds > 3600 {
		ttlSeconds = 3600
	}
	
	return ttlSeconds
}

// deduplicateFields removes duplicate field specifications
func (rr *RecipeResolver) deduplicateFields(fields []models.FieldSpec) []models.FieldSpec {
	seen := make(map[string]bool)
	var result []models.FieldSpec
	
	for _, field := range fields {
		key := field.Name + ":" + field.Type
		if !seen[key] {
			seen[key] = true
			result = append(result, field)
		}
	}
	
	return result
}

// Parsing methods for Apollo Federation responses

// parseContextRecipe parses Apollo Federation response into ContextRecipe
func (rr *RecipeResolver) parseContextRecipe(data interface{}) (*models.ContextRecipe, error) {
	// This would parse the actual GraphQL response structure
	// For now, return a basic structure
	return &models.ContextRecipe{
		ID:         "default_context_recipe",
		ProtocolID: "default_protocol",
		Version:    "1.0.0",
		CoreFields: []models.FieldSpec{
			{
				Name:            "patient_age",
				Type:            "DEMOGRAPHIC",
				Required:        true,
				MaxAgeHours:     24,
				ClinicalContext: "Age required for dose calculation",
			},
			{
				Name:            "current_medications",
				Type:            "MEDICATION",
				Required:        true,
				MaxAgeHours:     6,
				ClinicalContext: "Current medications for interaction checking",
			},
		},
		ConditionalRules: []models.ConditionalFieldRule{},
		FreshnessRules:   map[string]models.FreshnessRule{},
	}, nil
}

// parseClinicalRecipe parses Apollo Federation response into ClinicalRecipe
func (rr *RecipeResolver) parseClinicalRecipe(data interface{}) (*models.ClinicalRecipe, error) {
	// This would parse the actual GraphQL response structure
	// For now, return a basic structure
	return &models.ClinicalRecipe{
		ID:         "default_clinical_recipe",
		ProtocolID: "default_protocol",
		Version:    "1.0.0",
		TherapySelectionRules: []models.TherapyRule{},
		DosingStrategy: models.DosingStrategy{
			Approach:          "STANDARD",
			AdjustmentFactors: []string{"age", "weight", "renal_function"},
		},
		SafetyChecks: []models.SafetyCheckRequirement{},
		MonitoringPlan: models.MonitoringRequirements{
			Required: []models.MonitoringParameter{},
			Optional: []models.MonitoringParameter{},
			Duration: "ongoing",
		},
	}, nil
}

// Cache methods

// GetContextRecipe retrieves a context recipe from cache
func (c *RecipeCache) GetContextRecipe(protocolID string) (*models.ContextRecipe, bool) {
	entry, exists := c.contextCache[protocolID]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	recipe, ok := entry.Data.(*models.ContextRecipe)
	return recipe, ok
}

// SetContextRecipe stores a context recipe in cache
func (c *RecipeCache) SetContextRecipe(protocolID string, recipe *models.ContextRecipe) {
	c.contextCache[protocolID] = CacheEntry{
		Data:      recipe,
		ExpiresAt: time.Now().Add(c.contextRecipesTTL),
	}
}

// GetClinicalRecipe retrieves a clinical recipe from cache
func (c *RecipeCache) GetClinicalRecipe(protocolID string) (*models.ClinicalRecipe, bool) {
	entry, exists := c.clinicalCache[protocolID]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	recipe, ok := entry.Data.(*models.ClinicalRecipe)
	return recipe, ok
}

// SetClinicalRecipe stores a clinical recipe in cache
func (c *RecipeCache) SetClinicalRecipe(protocolID string, recipe *models.ClinicalRecipe) {
	c.clinicalCache[protocolID] = CacheEntry{
		Data:      recipe,
		ExpiresAt: time.Now().Add(c.clinicalRecipesTTL),
	}
}