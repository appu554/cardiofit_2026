package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/domain/repositories"
	"medication-service-v2/internal/infrastructure/redis"
)

// RecipeResolverServiceImpl implements the RecipeResolverService interface
type RecipeResolverServiceImpl struct {
	recipeRepository       repositories.RecipeRepository
	medicationRepository   repositories.MedicationRepository
	cacheClient           redis.Client
	protocolResolvers     map[string]entities.ProtocolResolver
	conditionalRules      map[string][]entities.ConditionalRule
	performanceTarget     time.Duration
	enableParallelProcessing bool
	mu                    sync.RWMutex
}

// NewRecipeResolverService creates a new recipe resolver service
func NewRecipeResolverService(
	recipeRepo repositories.RecipeRepository,
	medicationRepo repositories.MedicationRepository,
	cacheClient redis.Client,
) *RecipeResolverServiceImpl {
	return &RecipeResolverServiceImpl{
		recipeRepository:     recipeRepo,
		medicationRepository: medicationRepo,
		cacheClient:         cacheClient,
		protocolResolvers:   make(map[string]entities.ProtocolResolver),
		conditionalRules:    make(map[string][]entities.ConditionalRule),
		performanceTarget:   10 * time.Millisecond, // <10ms target
		enableParallelProcessing: true,
	}
}

// ResolveRecipe resolves a recipe with patient context
func (r *RecipeResolverServiceImpl) ResolveRecipe(ctx context.Context, request entities.RecipeResolutionRequest) (*entities.RecipeResolution, error) {
	startTime := time.Now()

	// Validate request
	if request.RecipeID == uuid.Nil {
		return nil, errors.New("recipe_id is required")
	}
	if request.PatientContext.PatientID == "" {
		return nil, errors.New("patient_id is required")
	}

	// Check cache if enabled
	var cacheKey string
	if request.Options.UseCache {
		cacheKey = r.generateCacheKey(request.RecipeID, request.PatientContext)
		if cachedResolution, err := r.getCachedResolution(ctx, cacheKey); err == nil && cachedResolution != nil {
			processingTime := time.Since(startTime).Milliseconds()
			cachedResolution.ProcessingTimeMs = processingTime
			return cachedResolution, nil
		}
	}

	// Fetch recipe
	recipe, err := r.recipeRepository.GetByID(ctx, request.RecipeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch recipe")
	}

	if !recipe.IsActive() || recipe.IsExpired() {
		return nil, errors.New("recipe is not active or has expired")
	}

	// Initialize resolution
	resolution := &entities.RecipeResolution{
		RecipeID:         recipe.ID,
		ResolutionTime:   time.Now(),
		ProcessingTimeMs: 0,
		ContextSnapshot:  make(map[string]interface{}),
		CalculatedDoses:  make([]entities.CalculatedDose, 0),
		SafetyViolations: make([]entities.SafetyViolation, 0),
		MonitoringPlan:   make([]entities.MonitoringInstruction, 0),
		Warnings:         make([]string, 0),
	}

	// Resolve fields through multi-phase processing
	resolvedFields, err := r.ResolveFields(ctx, recipe, request.PatientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve fields")
	}

	// Store context snapshot
	resolution.ContextSnapshot = r.createContextSnapshot(request.PatientContext, resolvedFields)

	// Execute calculation rules with resolved fields
	calculatedDoses, err := r.executeCalculationRules(ctx, recipe.CalculationRules, resolvedFields, request.PatientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute calculation rules")
	}
	resolution.CalculatedDoses = calculatedDoses

	// Execute safety rules
	safetyViolations, err := r.executeSafetyRules(ctx, recipe.SafetyRules, resolvedFields, request.PatientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute safety rules")
	}
	resolution.SafetyViolations = safetyViolations

	// Execute monitoring rules
	monitoringInstructions, err := r.executeMonitoringRules(ctx, recipe.MonitoringRules, resolvedFields, request.PatientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute monitoring rules")
	}
	resolution.MonitoringPlan = monitoringInstructions

	// Calculate confidence score
	resolution.ConfidenceScore = r.calculateConfidenceScore(resolvedFields, calculatedDoses, safetyViolations)

	// Set processing time
	processingTime := time.Since(startTime).Milliseconds()
	resolution.ProcessingTimeMs = processingTime

	// Cache result if enabled and within performance target
	if request.Options.UseCache && time.Duration(processingTime)*time.Millisecond <= r.performanceTarget {
		if err := r.cacheResolution(ctx, cacheKey, resolution, request.Options.CacheTTL); err != nil {
			// Log warning but don't fail the request
			resolution.Warnings = append(resolution.Warnings, fmt.Sprintf("Failed to cache resolution: %v", err))
		}
	}

	return resolution, nil
}

// ResolveFields resolves all required fields through multi-phase processing
func (r *RecipeResolverServiceImpl) ResolveFields(ctx context.Context, recipe *entities.Recipe, patientContext entities.PatientContext) (map[string]*entities.ResolvedField, error) {
	phaseResults := make(map[entities.FieldResolutionPhase]map[string]*entities.ResolvedField)

	// Phase 1: Calculation fields
	calculationFields, err := r.resolveCalculationFields(ctx, recipe.ContextRequirements.CalculationFields, patientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve calculation fields")
	}
	phaseResults[entities.PhaseCalculation] = calculationFields

	// Phase 2: Safety fields
	safetyFields, err := r.resolveSafetyFields(ctx, recipe.ContextRequirements.SafetyFields, patientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve safety fields")
	}
	phaseResults[entities.PhaseSafety] = safetyFields

	// Phase 3: Audit fields
	auditFields, err := r.resolveAuditFields(ctx, patientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve audit fields")
	}
	phaseResults[entities.PhaseAudit] = auditFields

	// Phase 4: Conditional fields based on protocol
	conditionalFields, err := r.resolveConditionalFields(ctx, recipe, patientContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve conditional fields")
	}
	phaseResults[entities.PhaseConditional] = conditionalFields

	// Merge all phases
	mergedFields, err := r.MergeFields(ctx, phaseResults)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge fields")
	}

	// Validate freshness requirements
	if err := r.ValidateFreshness(ctx, mergedFields, recipe.ContextRequirements.FreshnessRequirements); err != nil {
		return nil, errors.Wrap(err, "freshness validation failed")
	}

	return mergedFields, nil
}

// MergeFields merges fields from different phases with conflict resolution
func (r *RecipeResolverServiceImpl) MergeFields(ctx context.Context, phaseFields map[entities.FieldResolutionPhase]map[string]*entities.ResolvedField) (map[string]*entities.ResolvedField, error) {
	result := make(map[string]*entities.ResolvedField)

	// Define phase priority order
	phasePriority := []entities.FieldResolutionPhase{
		entities.PhaseCalculation,
		entities.PhaseSafety,
		entities.PhaseAudit,
		entities.PhaseConditional,
	}

	// Process phases in priority order
	for _, phase := range phasePriority {
		fields, exists := phaseFields[phase]
		if !exists {
			continue
		}

		for fieldName, field := range fields {
			if existingField, exists := result[fieldName]; exists {
				// Merge conflict resolution based on strategy
				mergedField, err := r.mergeFieldConflict(existingField, field)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to merge field conflict for %s", fieldName)
				}
				result[fieldName] = mergedField
			} else {
				result[fieldName] = field
			}
		}
	}

	return result, nil
}

// ValidateFreshness validates the freshness of resolved fields
func (r *RecipeResolverServiceImpl) ValidateFreshness(ctx context.Context, fields map[string]*entities.ResolvedField, requirements map[string]time.Duration) error {
	now := time.Now()
	
	for fieldName, maxAge := range requirements {
		field, exists := fields[fieldName]
		if !exists {
			return fmt.Errorf("required field %s is missing", fieldName)
		}

		if maxAge > 0 {
			age := now.Sub(field.LastUpdated)
			if age > maxAge {
				return fmt.Errorf("field %s is stale (age: %v, max: %v)", fieldName, age, maxAge)
			}
		}
	}

	return nil
}

// GetProtocolResolver gets the resolver for a specific protocol
func (r *RecipeResolverServiceImpl) GetProtocolResolver(protocolID string) (entities.ProtocolResolver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resolver, exists := r.protocolResolvers[protocolID]
	if !exists {
		return nil, fmt.Errorf("protocol resolver not found for protocol: %s", protocolID)
	}

	return resolver, nil
}

// ClearCache clears the cached resolution
func (r *RecipeResolverServiceImpl) ClearCache(ctx context.Context, cacheKey string) error {
	return r.cacheClient.Del(ctx, cacheKey)
}

// RegisterProtocolResolver registers a protocol-specific resolver
func (r *RecipeResolverServiceImpl) RegisterProtocolResolver(protocolID string, resolver entities.ProtocolResolver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.protocolResolvers[protocolID] = resolver
}

// resolveCalculationFields resolves fields needed for dose calculations
func (r *RecipeResolverServiceImpl) resolveCalculationFields(ctx context.Context, fields []entities.ContextField, patientContext entities.PatientContext) (map[string]*entities.ResolvedField, error) {
	result := make(map[string]*entities.ResolvedField)

	for _, field := range fields {
		value, err := r.getFieldValue(field.Name, patientContext)
		if err != nil && field.Required {
			return nil, errors.Wrapf(err, "required calculation field %s not available", field.Name)
		}

		if value != nil {
			resolvedField := &entities.ResolvedField{
				Name:          field.Name,
				Value:         value,
				Source:        "patient_context",
				Phase:         entities.PhaseCalculation,
				MergeStrategy: entities.MergeStrategyReplace,
				Priority:      1,
				LastUpdated:   time.Now(),
				Confidence:    1.0,
				ValidationStatus: "valid",
			}

			result[field.Name] = resolvedField
		}
	}

	return result, nil
}

// resolveSafetyFields resolves fields needed for safety checks
func (r *RecipeResolverServiceImpl) resolveSafetyFields(ctx context.Context, fields []entities.ContextField, patientContext entities.PatientContext) (map[string]*entities.ResolvedField, error) {
	result := make(map[string]*entities.ResolvedField)

	for _, field := range fields {
		value, err := r.getFieldValue(field.Name, patientContext)
		if err != nil && field.Required {
			return nil, errors.Wrapf(err, "required safety field %s not available", field.Name)
		}

		if value != nil {
			resolvedField := &entities.ResolvedField{
				Name:          field.Name,
				Value:         value,
				Source:        "patient_context",
				Phase:         entities.PhaseSafety,
				MergeStrategy: entities.MergeStrategyValidate,
				Priority:      2,
				LastUpdated:   time.Now(),
				Confidence:    1.0,
				ValidationStatus: "valid",
			}

			result[field.Name] = resolvedField
		}
	}

	return result, nil
}

// resolveAuditFields resolves fields needed for audit trail
func (r *RecipeResolverServiceImpl) resolveAuditFields(ctx context.Context, patientContext entities.PatientContext) (map[string]*entities.ResolvedField, error) {
	result := make(map[string]*entities.ResolvedField)

	// Provider information
	if patientContext.EncounterContext.ProviderID != "" {
		result["provider_id"] = &entities.ResolvedField{
			Name:          "provider_id",
			Value:         patientContext.EncounterContext.ProviderID,
			Source:        "encounter_context",
			Phase:         entities.PhaseAudit,
			MergeStrategy: entities.MergeStrategyReplace,
			Priority:      3,
			LastUpdated:   time.Now(),
			Confidence:    1.0,
			ValidationStatus: "valid",
		}
	}

	// Encounter information
	if patientContext.EncounterContext.EncounterID != "" {
		result["encounter_id"] = &entities.ResolvedField{
			Name:          "encounter_id",
			Value:         patientContext.EncounterContext.EncounterID,
			Source:        "encounter_context",
			Phase:         entities.PhaseAudit,
			MergeStrategy: entities.MergeStrategyReplace,
			Priority:      3,
			LastUpdated:   time.Now(),
			Confidence:    1.0,
			ValidationStatus: "valid",
		}
	}

	// Facility information
	if patientContext.EncounterContext.FacilityID != "" {
		result["facility_id"] = &entities.ResolvedField{
			Name:          "facility_id",
			Value:         patientContext.EncounterContext.FacilityID,
			Source:        "encounter_context",
			Phase:         entities.PhaseAudit,
			MergeStrategy: entities.MergeStrategyReplace,
			Priority:      3,
			LastUpdated:   time.Now(),
			Confidence:    1.0,
			ValidationStatus: "valid",
		}
	}

	return result, nil
}

// resolveConditionalFields resolves fields based on conditional rules
func (r *RecipeResolverServiceImpl) resolveConditionalFields(ctx context.Context, recipe *entities.Recipe, patientContext entities.PatientContext) (map[string]*entities.ResolvedField, error) {
	result := make(map[string]*entities.ResolvedField)

	// Get protocol-specific resolver
	protocolResolver, err := r.GetProtocolResolver(recipe.ProtocolID)
	if err != nil {
		// If no protocol-specific resolver, use default logic
		return r.resolveDefaultConditionalFields(ctx, recipe, patientContext)
	}

	// Use protocol-specific resolution
	protocolFields, err := protocolResolver.ResolveFields(ctx, patientContext, recipe)
	if err != nil {
		return nil, errors.Wrap(err, "protocol-specific field resolution failed")
	}

	// Convert to resolved fields
	for fieldName, value := range protocolFields.ContextSnapshot {
		result[fieldName] = &entities.ResolvedField{
			Name:          fieldName,
			Value:         value,
			Source:        fmt.Sprintf("protocol_%s", recipe.ProtocolID),
			Phase:         entities.PhaseConditional,
			MergeStrategy: entities.MergeStrategyPrioritize,
			Priority:      4,
			LastUpdated:   time.Now(),
			Confidence:    0.9,
			ValidationStatus: "valid",
		}
	}

	return result, nil
}

// resolveDefaultConditionalFields provides default conditional field resolution
func (r *RecipeResolverServiceImpl) resolveDefaultConditionalFields(ctx context.Context, recipe *entities.Recipe, patientContext entities.PatientContext) (map[string]*entities.ResolvedField, error) {
	result := make(map[string]*entities.ResolvedField)

	// Age-based conditionals
	if patientContext.Age < 18 {
		result["pediatric_dosing"] = &entities.ResolvedField{
			Name:          "pediatric_dosing",
			Value:         true,
			Source:        "age_conditional",
			Phase:         entities.PhaseConditional,
			MergeStrategy: entities.MergeStrategyReplace,
			Priority:      4,
			LastUpdated:   time.Now(),
			Confidence:    1.0,
			ValidationStatus: "valid",
		}
	}

	// Pregnancy-based conditionals
	if patientContext.PregnancyStatus {
		result["pregnancy_considerations"] = &entities.ResolvedField{
			Name:          "pregnancy_considerations",
			Value:         true,
			Source:        "pregnancy_conditional",
			Phase:         entities.PhaseConditional,
			MergeStrategy: entities.MergeStrategyReplace,
			Priority:      4,
			LastUpdated:   time.Now(),
			Confidence:    1.0,
			ValidationStatus: "valid",
		}
	}

	// Renal function conditionals
	if patientContext.RenalFunction != nil && patientContext.RenalFunction.eGFR < 60 {
		result["renal_adjustment"] = &entities.ResolvedField{
			Name:          "renal_adjustment",
			Value:         true,
			Source:        "renal_function_conditional",
			Phase:         entities.PhaseConditional,
			MergeStrategy: entities.MergeStrategyReplace,
			Priority:      4,
			LastUpdated:   time.Now(),
			Confidence:    0.95,
			ValidationStatus: "valid",
		}
	}

	return result, nil
}

// Helper methods

func (r *RecipeResolverServiceImpl) generateCacheKey(recipeID uuid.UUID, patientContext entities.PatientContext) string {
	return fmt.Sprintf("recipe_resolution:%s:%s", recipeID.String(), patientContext.PatientID)
}

func (r *RecipeResolverServiceImpl) getCachedResolution(ctx context.Context, cacheKey string) (*entities.RecipeResolution, error) {
	return r.cacheClient.GetRecipeResolution(ctx, cacheKey)
}

func (r *RecipeResolverServiceImpl) cacheResolution(ctx context.Context, cacheKey string, resolution *entities.RecipeResolution, ttl time.Duration) error {
	return r.cacheClient.SetRecipeResolution(ctx, cacheKey, resolution, ttl)
}

func (r *RecipeResolverServiceImpl) createContextSnapshot(patientContext entities.PatientContext, resolvedFields map[string]*entities.ResolvedField) map[string]interface{} {
	snapshot := make(map[string]interface{})
	
	// Add patient demographics
	snapshot["patient_id"] = patientContext.PatientID
	snapshot["age"] = patientContext.Age
	snapshot["weight"] = patientContext.Weight
	snapshot["height"] = patientContext.Height
	snapshot["gender"] = patientContext.Gender
	snapshot["pregnancy_status"] = patientContext.PregnancyStatus
	
	// Add resolved fields
	for name, field := range resolvedFields {
		snapshot[name] = field.Value
	}
	
	return snapshot
}

func (r *RecipeResolverServiceImpl) getFieldValue(fieldName string, patientContext entities.PatientContext) (interface{}, error) {
	switch fieldName {
	case "age":
		return patientContext.Age, nil
	case "weight":
		return patientContext.Weight, nil
	case "height":
		return patientContext.Height, nil
	case "gender":
		return patientContext.Gender, nil
	case "pregnancy_status":
		return patientContext.PregnancyStatus, nil
	case "renal_function.egfr":
		if patientContext.RenalFunction != nil {
			return patientContext.RenalFunction.eGFR, nil
		}
		return nil, fmt.Errorf("renal function not available")
	case "hepatic_function.child_pugh":
		if patientContext.HepaticFunction != nil {
			return patientContext.HepaticFunction.ChildPugh, nil
		}
		return nil, fmt.Errorf("hepatic function not available")
	default:
		// Check lab results
		if labValue, exists := patientContext.LabResults[fieldName]; exists {
			return labValue.Value, nil
		}
		return nil, fmt.Errorf("field not found: %s", fieldName)
	}
}

func (r *RecipeResolverServiceImpl) mergeFieldConflict(existing, new *entities.ResolvedField) (*entities.ResolvedField, error) {
	switch new.MergeStrategy {
	case entities.MergeStrategyReplace:
		return new, nil
	case entities.MergeStrategyPrioritize:
		if new.Priority >= existing.Priority {
			return new, nil
		}
		return existing, nil
	case entities.MergeStrategyValidate:
		// For validation strategy, keep the field with higher confidence
		if new.Confidence > existing.Confidence {
			return new, nil
		}
		return existing, nil
	default:
		return existing, nil
	}
}

func (r *RecipeResolverServiceImpl) calculateConfidenceScore(fields map[string]*entities.ResolvedField, doses []entities.CalculatedDose, violations []entities.SafetyViolation) float64 {
	if len(fields) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, field := range fields {
		totalConfidence += field.Confidence
	}

	baseScore := totalConfidence / float64(len(fields))

	// Reduce confidence based on safety violations
	if len(violations) > 0 {
		violationPenalty := 0.1 * float64(len(violations))
		baseScore -= violationPenalty
	}

	if baseScore < 0 {
		return 0.0
	}
	if baseScore > 1.0 {
		return 1.0
	}

	return baseScore
}

// Placeholder methods for rule execution (to be implemented based on business logic)
func (r *RecipeResolverServiceImpl) executeCalculationRules(ctx context.Context, rules []entities.CalculationRule, fields map[string]*entities.ResolvedField, patientContext entities.PatientContext) ([]entities.CalculatedDose, error) {
	// Implementation will depend on specific calculation engine
	return []entities.CalculatedDose{}, nil
}

func (r *RecipeResolverServiceImpl) executeSafetyRules(ctx context.Context, rules []entities.SafetyRule, fields map[string]*entities.ResolvedField, patientContext entities.PatientContext) ([]entities.SafetyViolation, error) {
	// Implementation will depend on specific safety engine
	return []entities.SafetyViolation{}, nil
}

func (r *RecipeResolverServiceImpl) executeMonitoringRules(ctx context.Context, rules []entities.MonitoringRule, fields map[string]*entities.ResolvedField, patientContext entities.PatientContext) ([]entities.MonitoringInstruction, error) {
	// Implementation will depend on specific monitoring engine
	return []entities.MonitoringInstruction{}, nil
}