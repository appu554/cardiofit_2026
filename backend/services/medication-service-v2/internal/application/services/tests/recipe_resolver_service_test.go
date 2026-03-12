package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/tests/helpers/mocks"
	"medication-service-v2/tests/helpers/fixtures"
)

func TestRecipeResolverService_ResolveRecipe(t *testing.T) {
	// Setup
	mockRecipeRepo := mocks.NewMockRecipeRepository(t)
	mockMedicationRepo := mocks.NewMockMedicationRepository(t)
	mockCacheClient := mocks.NewMockRedisClient(t)

	resolverService := services.NewRecipeResolverService(
		mockRecipeRepo,
		mockMedicationRepo,
		mockCacheClient,
	)

	ctx := context.Background()
	testRecipeID := uuid.New()
	patientContext := fixtures.ValidPatientContext()

	t.Run("successful_recipe_resolution", func(t *testing.T) {
		// Given
		request := entities.RecipeResolutionRequest{
			RecipeID:       testRecipeID,
			PatientContext: patientContext,
			Options: entities.ResolutionOptions{
				UseCache:    true,
				CacheTTL:    5 * time.Minute,
				MaxAge:      1 * time.Hour,
				EnableDebug: false,
			},
		}

		// Mock no cached result initially
		mockCacheClient.On("GetRecipeResolution", ctx, mock.AnythingOfType("string")).
			Return(nil, assert.AnError)

		// Mock recipe retrieval
		expectedRecipe := fixtures.ValidRecipeWithRules()
		mockRecipeRepo.On("GetByID", ctx, testRecipeID).
			Return(expectedRecipe, nil)

		// Mock caching success
		mockCacheClient.On("SetRecipeResolution", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*entities.RecipeResolution"), mock.AnythingOfType("time.Duration")).
			Return(nil)

		// When
		resolution, err := resolverService.ResolveRecipe(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, resolution)
		assert.Equal(t, testRecipeID, resolution.RecipeID)
		assert.NotNil(t, resolution.ResolutionTime)
		assert.True(t, resolution.ProcessingTimeMs > 0)
		assert.NotEmpty(t, resolution.ContextSnapshot)
		assert.True(t, resolution.ConfidenceScore > 0)

		// Verify performance target (<10ms as per requirements)
		assert.True(t, resolution.ProcessingTimeMs < 10, 
			"Resolution took %dms, expected <10ms", resolution.ProcessingTimeMs)

		mockRecipeRepo.AssertExpectations(t)
		mockCacheClient.AssertExpectations(t)
	})

	t.Run("cached_result_retrieval", func(t *testing.T) {
		// Given
		request := entities.RecipeResolutionRequest{
			RecipeID:       testRecipeID,
			PatientContext: patientContext,
			Options: entities.ResolutionOptions{
				UseCache: true,
				CacheTTL: 5 * time.Minute,
			},
		}

		// Mock cached result exists
		cachedResolution := &entities.RecipeResolution{
			RecipeID:         testRecipeID,
			ResolutionTime:   time.Now().Add(-1 * time.Minute),
			ProcessingTimeMs: 2,
			ContextSnapshot:  map[string]interface{}{"cached": true},
			ConfidenceScore:  0.95,
		}
		mockCacheClient.On("GetRecipeResolution", ctx, mock.AnythingOfType("string")).
			Return(cachedResolution, nil)

		// When
		resolution, err := resolverService.ResolveRecipe(ctx, request)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, resolution)
		assert.Equal(t, testRecipeID, resolution.RecipeID)
		assert.True(t, resolution.ContextSnapshot["cached"].(bool))
		assert.True(t, resolution.ProcessingTimeMs < 10) // Should be very fast from cache

		mockCacheClient.AssertExpectations(t)
	})

	t.Run("invalid_recipe_id", func(t *testing.T) {
		// Given
		request := entities.RecipeResolutionRequest{
			RecipeID:       uuid.Nil,
			PatientContext: patientContext,
		}

		// When
		resolution, err := resolverService.ResolveRecipe(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, resolution)
		assert.Contains(t, err.Error(), "recipe_id is required")
	})

	t.Run("inactive_recipe", func(t *testing.T) {
		// Given
		request := entities.RecipeResolutionRequest{
			RecipeID:       testRecipeID,
			PatientContext: patientContext,
			Options: entities.ResolutionOptions{
				UseCache: false,
			},
		}

		// Mock recipe retrieval - inactive recipe
		inactiveRecipe := fixtures.ValidRecipeWithRules()
		inactiveRecipe.Status = entities.RecipeStatusDeprecated
		mockRecipeRepo.On("GetByID", ctx, testRecipeID).
			Return(inactiveRecipe, nil)

		// When
		resolution, err := resolverService.ResolveRecipe(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, resolution)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("expired_recipe", func(t *testing.T) {
		// Given
		request := entities.RecipeResolutionRequest{
			RecipeID:       testRecipeID,
			PatientContext: patientContext,
		}

		// Mock recipe retrieval - expired recipe
		expiredRecipe := fixtures.ValidRecipeWithRules()
		expiredRecipe.UpdatedAt = time.Now().Add(-2 * time.Hour)
		expiredRecipe.TTL = 1 * time.Hour
		mockRecipeRepo.On("GetByID", ctx, testRecipeID).
			Return(expiredRecipe, nil)

		// When
		resolution, err := resolverService.ResolveRecipe(ctx, request)

		// Then
		assert.Error(t, err)
		assert.Nil(t, resolution)
		assert.Contains(t, err.Error(), "expired")
	})
}

func TestRecipeResolverService_ResolveFields(t *testing.T) {
	// Setup
	mockRecipeRepo := mocks.NewMockRecipeRepository(t)
	mockMedicationRepo := mocks.NewMockMedicationRepository(t)
	mockCacheClient := mocks.NewMockRedisClient(t)

	resolverService := services.NewRecipeResolverService(
		mockRecipeRepo,
		mockMedicationRepo,
		mockCacheClient,
	)

	ctx := context.Background()
	recipe := fixtures.ValidRecipeWithRules()
	patientContext := fixtures.ValidPatientContext()

	t.Run("successful_multi_phase_field_resolution", func(t *testing.T) {
		// When
		resolvedFields, err := resolverService.ResolveFields(ctx, recipe, patientContext)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, resolvedFields)

		// Verify calculation fields
		assert.Contains(t, resolvedFields, "age")
		assert.Contains(t, resolvedFields, "weight")
		assert.Equal(t, patientContext.Age, resolvedFields["age"].Value)
		assert.Equal(t, patientContext.Weight, resolvedFields["weight"].Value)

		// Verify safety fields
		assert.Contains(t, resolvedFields, "gender")
		assert.Equal(t, patientContext.Gender, resolvedFields["gender"].Value)

		// Verify audit fields if encounter context exists
		if patientContext.EncounterContext.ProviderID != "" {
			assert.Contains(t, resolvedFields, "provider_id")
			assert.Equal(t, patientContext.EncounterContext.ProviderID, resolvedFields["provider_id"].Value)
		}

		// Verify conditional fields for pediatric patients
		if patientContext.Age < 18 {
			assert.Contains(t, resolvedFields, "pediatric_dosing")
			assert.True(t, resolvedFields["pediatric_dosing"].Value.(bool))
		}

		// Verify field phases are correct
		for _, field := range resolvedFields {
			assert.NotEmpty(t, field.Phase)
			assert.True(t, field.Confidence > 0)
			assert.True(t, field.Priority > 0)
		}
	})

	t.Run("pediatric_patient_conditional_fields", func(t *testing.T) {
		// Given - pediatric patient
		pediatricContext := fixtures.ValidPatientContext()
		pediatricContext.Age = 8

		// When
		resolvedFields, err := resolverService.ResolveFields(ctx, recipe, pediatricContext)

		// Then
		require.NoError(t, err)
		assert.Contains(t, resolvedFields, "pediatric_dosing")
		assert.True(t, resolvedFields["pediatric_dosing"].Value.(bool))
		assert.Equal(t, entities.PhaseConditional, resolvedFields["pediatric_dosing"].Phase)
	})

	t.Run("pregnant_patient_conditional_fields", func(t *testing.T) {
		// Given - pregnant patient
		pregnantContext := fixtures.ValidPatientContext()
		pregnantContext.PregnancyStatus = true

		// When
		resolvedFields, err := resolverService.ResolveFields(ctx, recipe, pregnantContext)

		// Then
		require.NoError(t, err)
		assert.Contains(t, resolvedFields, "pregnancy_considerations")
		assert.True(t, resolvedFields["pregnancy_considerations"].Value.(bool))
	})

	t.Run("renal_impairment_conditional_fields", func(t *testing.T) {
		// Given - patient with renal impairment
		renalContext := fixtures.ValidPatientContext()
		renalContext.RenalFunction = &entities.RenalFunction{
			eGFR: 45.0, // < 60, indicating impairment
		}

		// When
		resolvedFields, err := resolverService.ResolveFields(ctx, recipe, renalContext)

		// Then
		require.NoError(t, err)
		assert.Contains(t, resolvedFields, "renal_adjustment")
		assert.True(t, resolvedFields["renal_adjustment"].Value.(bool))
	})

	t.Run("missing_required_field", func(t *testing.T) {
		// Given - recipe with required fields
		recipeWithRequiredFields := fixtures.ValidRecipeWithRules()
		recipeWithRequiredFields.ContextRequirements.CalculationFields = append(
			recipeWithRequiredFields.ContextRequirements.CalculationFields,
			entities.ContextField{
				Name:     "creatinine",
				Type:     entities.FieldTypeNumber,
				Required: true,
				Unit:     "mg/dL",
			},
		)

		// Patient context without required creatinine
		incompleteContext := fixtures.ValidPatientContext()
		incompleteContext.LabResults = make(map[string]entities.LabValue)

		// When
		resolvedFields, err := resolverService.ResolveFields(ctx, recipeWithRequiredFields, incompleteContext)

		// Then
		assert.Error(t, err)
		assert.Nil(t, resolvedFields)
		assert.Contains(t, err.Error(), "required calculation field creatinine not available")
	})
}

func TestRecipeResolverService_ValidateFreshness(t *testing.T) {
	// Setup
	mockRecipeRepo := mocks.NewMockRecipeRepository(t)
	mockMedicationRepo := mocks.NewMockMedicationRepository(t)
	mockCacheClient := mocks.NewMockRedisClient(t)

	resolverService := services.NewRecipeResolverService(
		mockRecipeRepo,
		mockMedicationRepo,
		mockCacheClient,
	)

	ctx := context.Background()
	now := time.Now()

	t.Run("fresh_fields_pass_validation", func(t *testing.T) {
		// Given
		fields := map[string]*entities.ResolvedField{
			"recent_lab": {
				Name:        "recent_lab",
				Value:       85.0,
				LastUpdated: now.Add(-10 * time.Minute),
			},
		}
		requirements := map[string]time.Duration{
			"recent_lab": 30 * time.Minute,
		}

		// When
		err := resolverService.ValidateFreshness(ctx, fields, requirements)

		// Then
		assert.NoError(t, err)
	})

	t.Run("stale_fields_fail_validation", func(t *testing.T) {
		// Given
		fields := map[string]*entities.ResolvedField{
			"old_lab": {
				Name:        "old_lab",
				Value:       85.0,
				LastUpdated: now.Add(-2 * time.Hour),
			},
		}
		requirements := map[string]time.Duration{
			"old_lab": 30 * time.Minute,
		}

		// When
		err := resolverService.ValidateFreshness(ctx, fields, requirements)

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stale")
		assert.Contains(t, err.Error(), "old_lab")
	})

	t.Run("missing_required_field", func(t *testing.T) {
		// Given
		fields := map[string]*entities.ResolvedField{
			"available_field": {
				Name:        "available_field",
				Value:       100.0,
				LastUpdated: now,
			},
		}
		requirements := map[string]time.Duration{
			"missing_field": 1 * time.Hour,
		}

		// When
		err := resolverService.ValidateFreshness(ctx, fields, requirements)

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
		assert.Contains(t, err.Error(), "missing_field")
	})

	t.Run("no_freshness_requirement", func(t *testing.T) {
		// Given
		fields := map[string]*entities.ResolvedField{
			"any_age_field": {
				Name:        "any_age_field",
				Value:       "any_value",
				LastUpdated: now.Add(-24 * time.Hour), // Very old
			},
		}
		requirements := map[string]time.Duration{
			"any_age_field": 0, // No freshness requirement
		}

		// When
		err := resolverService.ValidateFreshness(ctx, fields, requirements)

		// Then
		assert.NoError(t, err)
	})
}

func TestRecipeResolverService_MergeFields(t *testing.T) {
	// Setup
	mockRecipeRepo := mocks.NewMockRecipeRepository(t)
	mockMedicationRepo := mocks.NewMockMedicationRepository(t)
	mockCacheClient := mocks.NewMockRedisClient(t)

	resolverService := services.NewRecipeResolverService(
		mockRecipeRepo,
		mockMedicationRepo,
		mockCacheClient,
	)

	ctx := context.Background()

	t.Run("merge_without_conflicts", func(t *testing.T) {
		// Given
		phaseFields := map[entities.FieldResolutionPhase]map[string]*entities.ResolvedField{
			entities.PhaseCalculation: {
				"weight": {
					Name:     "weight",
					Value:    70.0,
					Priority: 1,
				},
			},
			entities.PhaseSafety: {
				"allergies": {
					Name:     "allergies",
					Value:    []string{"penicillin"},
					Priority: 2,
				},
			},
		}

		// When
		merged, err := resolverService.MergeFields(ctx, phaseFields)

		// Then
		require.NoError(t, err)
		assert.Len(t, merged, 2)
		assert.Contains(t, merged, "weight")
		assert.Contains(t, merged, "allergies")
		assert.Equal(t, 70.0, merged["weight"].Value)
		assert.Equal(t, []string{"penicillin"}, merged["allergies"].Value)
	})

	t.Run("merge_with_priority_resolution", func(t *testing.T) {
		// Given - same field in different phases with different priorities
		phaseFields := map[entities.FieldResolutionPhase]map[string]*entities.ResolvedField{
			entities.PhaseCalculation: {
				"age": {
					Name:          "age",
					Value:         45,
					Priority:      1,
					MergeStrategy: entities.MergeStrategyPrioritize,
				},
			},
			entities.PhaseSafety: {
				"age": {
					Name:          "age",
					Value:         46, // Slightly different value
					Priority:      2,
					MergeStrategy: entities.MergeStrategyPrioritize,
				},
			},
		}

		// When
		merged, err := resolverService.MergeFields(ctx, phaseFields)

		// Then
		require.NoError(t, err)
		assert.Len(t, merged, 1)
		assert.Contains(t, merged, "age")
		// Higher priority (2) should win
		assert.Equal(t, 46, merged["age"].Value)
	})

	t.Run("merge_with_replace_strategy", func(t *testing.T) {
		// Given - replace strategy always takes new value
		phaseFields := map[entities.FieldResolutionPhase]map[string]*entities.ResolvedField{
			entities.PhaseCalculation: {
				"status": {
					Name:          "status",
					Value:         "old_status",
					Priority:      3,
					MergeStrategy: entities.MergeStrategyReplace,
				},
			},
			entities.PhaseConditional: {
				"status": {
					Name:          "status",
					Value:         "new_status",
					Priority:      1, // Lower priority
					MergeStrategy: entities.MergeStrategyReplace,
				},
			},
		}

		// When
		merged, err := resolverService.MergeFields(ctx, phaseFields)

		// Then
		require.NoError(t, err)
		assert.Len(t, merged, 1)
		assert.Contains(t, merged, "status")
		// Replace strategy should use the new value regardless of priority
		assert.Equal(t, "new_status", merged["status"].Value)
	})

	t.Run("merge_with_validation_strategy", func(t *testing.T) {
		// Given - validation strategy uses higher confidence
		phaseFields := map[entities.FieldResolutionPhase]map[string]*entities.ResolvedField{
			entities.PhaseCalculation: {
				"measurement": {
					Name:          "measurement",
					Value:         100.0,
					Confidence:    0.8,
					MergeStrategy: entities.MergeStrategyValidate,
				},
			},
			entities.PhaseSafety: {
				"measurement": {
					Name:          "measurement",
					Value:         101.0,
					Confidence:    0.95, // Higher confidence
					MergeStrategy: entities.MergeStrategyValidate,
				},
			},
		}

		// When
		merged, err := resolverService.MergeFields(ctx, phaseFields)

		// Then
		require.NoError(t, err)
		assert.Len(t, merged, 1)
		assert.Contains(t, merged, "measurement")
		// Higher confidence should win
		assert.Equal(t, 101.0, merged["measurement"].Value)
		assert.Equal(t, 0.95, merged["measurement"].Confidence)
	})
}

func BenchmarkRecipeResolverService_ResolveRecipe(b *testing.B) {
	// Setup
	mockRecipeRepo := mocks.NewMockRecipeRepository(b)
	mockMedicationRepo := mocks.NewMockMedicationRepository(b)
	mockCacheClient := mocks.NewMockRedisClient(b)

	resolverService := services.NewRecipeResolverService(
		mockRecipeRepo,
		mockMedicationRepo,
		mockCacheClient,
	)

	// Setup mocks
	recipe := fixtures.ValidRecipeWithRules()
	mockRecipeRepo.On("GetByID", mock.Anything, mock.Anything).Return(recipe, nil)
	mockCacheClient.On("GetRecipeResolution", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	mockCacheClient.On("SetRecipeResolution", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx := context.Background()
	request := entities.RecipeResolutionRequest{
		RecipeID:       uuid.New(),
		PatientContext: fixtures.ValidPatientContext(),
		Options: entities.ResolutionOptions{
			UseCache: true,
			CacheTTL: 5 * time.Minute,
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := resolverService.ResolveRecipe(ctx, request)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestRecipeResolverService_Performance_Target(t *testing.T) {
	// Setup for performance testing
	mockRecipeRepo := mocks.NewMockRecipeRepository(t)
	mockMedicationRepo := mocks.NewMockMedicationRepository(t)
	mockCacheClient := mocks.NewMockRedisClient(t)

	resolverService := services.NewRecipeResolverService(
		mockRecipeRepo,
		mockMedicationRepo,
		mockCacheClient,
	)

	// Setup fast mocks
	recipe := fixtures.ValidRecipeWithRules()
	mockRecipeRepo.On("GetByID", mock.Anything, mock.Anything).Return(recipe, nil)
	mockCacheClient.On("GetRecipeResolution", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	mockCacheClient.On("SetRecipeResolution", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ctx := context.Background()

	t.Run("recipe_resolution_under_10ms_target", func(t *testing.T) {
		// Given
		request := entities.RecipeResolutionRequest{
			RecipeID:       uuid.New(),
			PatientContext: fixtures.ValidPatientContext(),
			Options: entities.ResolutionOptions{
				UseCache: false, // Force full resolution
			},
		}

		// When - measure multiple calls to get average
		totalDuration := time.Duration(0)
		iterations := 10

		for i := 0; i < iterations; i++ {
			start := time.Now()
			resolution, err := resolverService.ResolveRecipe(ctx, request)
			duration := time.Since(start)
			
			require.NoError(t, err)
			assert.NotNil(t, resolution)
			totalDuration += duration
		}

		averageDuration := totalDuration / time.Duration(iterations)

		// Then
		assert.True(t, averageDuration < 10*time.Millisecond,
			"Average resolution time %v exceeds 10ms target", averageDuration)

		// Log performance metrics
		t.Logf("Recipe resolution performance: average %v over %d iterations", averageDuration, iterations)
	})
}