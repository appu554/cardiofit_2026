package flow2

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/models"
	"flow2-go-engine/internal/services"
)

// ResponseOptimizer optimizes and formats Flow 2 responses
type ResponseOptimizer struct {
	cacheService   services.CacheService
	metricsService services.MetricsService
	logger         *logrus.Logger
}

// NewResponseOptimizer creates a new response optimizer
func NewResponseOptimizer(
	cacheService services.CacheService,
	metricsService services.MetricsService,
	logger *logrus.Logger,
) *ResponseOptimizer {
	return &ResponseOptimizer{
		cacheService:   cacheService,
		metricsService: metricsService,
		logger:         logger,
	}
}

// OptimizeResponse optimizes and formats a Flow 2 response
func (ro *ResponseOptimizer) OptimizeResponse(
	request *models.Flow2Request,
	clinicalContext *models.ClinicalContext,
	recipeResults []models.RecipeResult,
	startTime time.Time,
) *models.Flow2Response {
	ro.logger.WithFields(logrus.Fields{
		"request_id":       request.RequestID,
		"recipes_executed": len(recipeResults),
	}).Info("Starting response optimization")

	// Determine overall status
	overallStatus := ro.determineOverallStatus(recipeResults)

	// Extract safety alerts
	safetyAlerts := ro.extractSafetyAlerts(recipeResults)

	// Extract recommendations
	recommendations := ro.extractRecommendations(recipeResults)

	// Build clinical decision support
	clinicalDecisionSupport := ro.buildClinicalDecisionSupport(recipeResults)

	// Build execution summary
	executionSummary := ro.buildExecutionSummary(recipeResults)

	// Build analytics
	analytics := ro.buildAnalytics(request, clinicalContext, recipeResults, startTime)

	// Build processing metadata
	processingMetadata := ro.buildProcessingMetadata(request, clinicalContext)

	// Calculate execution time
	executionTime := time.Since(startTime)

	response := &models.Flow2Response{
		RequestID:               request.RequestID,
		PatientID:               request.PatientID,
		OverallStatus:           overallStatus,
		ExecutionSummary:        executionSummary,
		RecipeResults:           recipeResults,
		ClinicalDecisionSupport: clinicalDecisionSupport,
		SafetyAlerts:            safetyAlerts,
		Recommendations:         recommendations,
		Analytics:               analytics,
		ExecutionTimeMs:         executionTime.Milliseconds(),
		EngineUsed:              "go+rust",
		Timestamp:               time.Now(),
		ProcessingMetadata:      processingMetadata,
	}

	ro.logger.WithFields(logrus.Fields{
		"request_id":        request.RequestID,
		"overall_status":    overallStatus,
		"execution_time_ms": executionTime.Milliseconds(),
		"safety_alerts":     len(safetyAlerts),
		"recommendations":   len(recommendations),
	}).Info("Response optimization completed")

	return response
}

// OptimizeMedicationIntelligenceResponse optimizes medication intelligence response
func (ro *ResponseOptimizer) OptimizeMedicationIntelligenceResponse(
	response *models.MedicationIntelligenceResponse,
	request *models.MedicationIntelligenceRequest,
	startTime time.Time,
) *models.MedicationIntelligenceResponse {
	// Add any additional optimization logic here
	// For now, just return the response as-is
	return response
}

// determineOverallStatus determines the overall status from recipe results
func (ro *ResponseOptimizer) determineOverallStatus(recipeResults []models.RecipeResult) string {
	if len(recipeResults) == 0 {
		return "NO_RECIPES"
	}

	hasUnsafe := false
	hasWarning := false

	for _, result := range recipeResults {
		switch result.OverallStatus {
		case "UNSAFE":
			hasUnsafe = true
		case "WARNING":
			hasWarning = true
		case "ERROR":
			return "ERROR" // Return immediately on error
		}
	}

	if hasUnsafe {
		return "UNSAFE"
	}
	if hasWarning {
		return "WARNING"
	}
	return "SAFE"
}

// extractSafetyAlerts extracts safety alerts from recipe results
func (ro *ResponseOptimizer) extractSafetyAlerts(recipeResults []models.RecipeResult) []models.SafetyAlert {
	var safetyAlerts []models.SafetyAlert

	for _, result := range recipeResults {
		for _, validation := range result.Validations {
			if !validation.Passed && (validation.Severity == "CRITICAL" || validation.Severity == "WARNING") {
				safetyAlert := models.SafetyAlert{
					AlertID:        result.RecipeID + "_" + validation.Code,
					Severity:       validation.Severity,
					Type:           "RECIPE_VALIDATION",
					Message:        validation.Message,
					Description:    validation.Explanation,
					ActionRequired: validation.Severity == "CRITICAL",
				}
				safetyAlerts = append(safetyAlerts, safetyAlert)
			}
		}
	}

	return safetyAlerts
}

// extractRecommendations extracts recommendations from recipe results
func (ro *ResponseOptimizer) extractRecommendations(recipeResults []models.RecipeResult) []models.Recommendation {
	var recommendations []models.Recommendation

	for _, result := range recipeResults {
		for i, rec := range result.Recommendations {
			recommendation := models.Recommendation{
				RecommendationID: result.RecipeID + "_rec_" + string(rune(i)),
				Type:             "CLINICAL",
				Priority:         "MEDIUM",
				Title:            "Clinical Recommendation",
				Description:      rec,
				Rationale:        "Generated by " + result.RecipeName,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

// buildClinicalDecisionSupport builds comprehensive clinical decision support
func (ro *ResponseOptimizer) buildClinicalDecisionSupport(recipeResults []models.RecipeResult) map[string]interface{} {
	cds := make(map[string]interface{})

	// Aggregate clinical decision support from all recipes
	for _, result := range recipeResults {
		if result.ClinicalDecisionSupport != nil {
			for key, value := range result.ClinicalDecisionSupport {
				cds[result.RecipeID+"_"+key] = value
			}
		}
	}

	// Add summary information
	cds["total_recipes_executed"] = len(recipeResults)
	cds["processing_timestamp"] = time.Now().UTC()

	return cds
}

// buildExecutionSummary builds execution summary
func (ro *ResponseOptimizer) buildExecutionSummary(recipeResults []models.RecipeResult) models.ExecutionSummary {
	totalRecipes := len(recipeResults)
	successfulRecipes := 0
	failedRecipes := 0
	warnings := 0
	errors := 0

	for _, result := range recipeResults {
		switch result.OverallStatus {
		case "SAFE":
			successfulRecipes++
		case "WARNING":
			successfulRecipes++
			warnings++
		case "UNSAFE":
			failedRecipes++
		case "ERROR":
			failedRecipes++
			errors++
		}
	}

	return models.ExecutionSummary{
		TotalRecipesExecuted: totalRecipes,
		SuccessfulRecipes:    successfulRecipes,
		FailedRecipes:        failedRecipes,
		Warnings:             warnings,
		Errors:               errors,
		Engine:               "go+rust",
		CacheHitRate:         0.0, // TODO: Calculate actual cache hit rate
	}
}

// buildAnalytics builds analytics information
func (ro *ResponseOptimizer) buildAnalytics(
	request *models.Flow2Request,
	clinicalContext *models.ClinicalContext,
	recipeResults []models.RecipeResult,
	startTime time.Time,
) map[string]interface{} {
	analytics := make(map[string]interface{})

	// Performance analytics
	analytics["execution_time_ms"] = time.Since(startTime).Milliseconds()
	analytics["recipes_executed"] = len(recipeResults)

	// Clinical analytics
	if clinicalContext.PatientDemographics != nil {
		analytics["patient_age"] = clinicalContext.PatientDemographics.Age
		analytics["patient_weight"] = clinicalContext.PatientDemographics.Weight
	}

	// Recipe analytics
	recipeAnalytics := make(map[string]interface{})
	for _, result := range recipeResults {
		recipeAnalytics[result.RecipeID] = map[string]interface{}{
			"status":           result.OverallStatus,
			"execution_time":   result.ExecutionTimeMs,
			"validations":      len(result.Validations),
			"recommendations":  len(result.Recommendations),
		}
	}
	analytics["recipe_details"] = recipeAnalytics

	return analytics
}

// buildProcessingMetadata builds processing metadata
func (ro *ResponseOptimizer) buildProcessingMetadata(
	request *models.Flow2Request,
	clinicalContext *models.ClinicalContext,
) models.ProcessingMetadata {
	// Determine context sources
	contextSources := []string{}
	if clinicalContext.PatientDemographics != nil {
		contextSources = append(contextSources, "demographics")
	}
	if len(clinicalContext.CurrentMedications) > 0 {
		contextSources = append(contextSources, "medications")
	}
	if len(clinicalContext.Allergies) > 0 {
		contextSources = append(contextSources, "allergies")
	}
	if len(clinicalContext.Conditions) > 0 {
		contextSources = append(contextSources, "conditions")
	}

	// Build processing stages
	processingStages := []models.ProcessingStage{
		{
			StageName:       "context_assembly",
			ExecutionTimeMs: 15, // TODO: Track actual time
			Status:          "completed",
			Details: map[string]interface{}{
				"sources": contextSources,
			},
		},
		{
			StageName:       "recipe_execution",
			ExecutionTimeMs: 5, // TODO: Track actual time
			Status:          "completed",
			Details: map[string]interface{}{
				"engine": "rust",
			},
		},
		{
			StageName:       "response_optimization",
			ExecutionTimeMs: 2, // TODO: Track actual time
			Status:          "completed",
			Details: map[string]interface{}{
				"optimizer": "go",
			},
		},
	}

	return models.ProcessingMetadata{
		FallbackUsed:     false,
		CacheUsed:        false, // TODO: Track actual cache usage
		ContextSources:   contextSources,
		ProcessingStages: processingStages,
	}
}
