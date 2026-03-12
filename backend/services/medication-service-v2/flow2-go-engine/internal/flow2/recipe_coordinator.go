package flow2

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/models"
	"flow2-go-engine/internal/services"
)

// RecipeCoordinator coordinates recipe execution with the Rust engine
type RecipeCoordinator struct {
	rustRecipeClient clients.RustRecipeClient
	metricsService   services.MetricsService
	logger           *logrus.Logger
}

// NewRecipeCoordinator creates a new recipe coordinator
func NewRecipeCoordinator(
	rustRecipeClient clients.RustRecipeClient,
	metricsService services.MetricsService,
	logger *logrus.Logger,
) *RecipeCoordinator {
	return &RecipeCoordinator{
		rustRecipeClient: rustRecipeClient,
		metricsService:   metricsService,
		logger:           logger,
	}
}

// ExecuteRecipes executes recipes via the Rust engine
func (rc *RecipeCoordinator) ExecuteRecipes(
	ctx context.Context,
	request *models.Flow2Request,
	clinicalContext *models.ClinicalContext,
) ([]models.RecipeResult, error) {
	start := time.Now()

	rc.logger.WithFields(logrus.Fields{
		"request_id": request.RequestID,
		"patient_id": request.PatientID,
		"action_type": request.ActionType,
	}).Info("Starting recipe execution coordination")

	// Execute recipes via Rust engine
	response, err := rc.rustRecipeClient.ExecuteFlow2(ctx, request, clinicalContext)
	if err != nil {
		rc.metricsService.IncrementRustEngineFailures()
		rc.logger.WithFields(logrus.Fields{
			"request_id": request.RequestID,
			"error":      err.Error(),
		}).Error("Rust engine execution failed")
		return nil, err
	}

	// Record Rust engine latency
	executionTime := time.Since(start)
	rc.metricsService.RecordRustEngineLatency(executionTime)

	rc.logger.WithFields(logrus.Fields{
		"request_id":        request.RequestID,
		"execution_time_ms": executionTime.Milliseconds(),
		"recipes_executed":  len(response.RecipeResults),
		"overall_status":    response.OverallStatus,
	}).Info("Recipe execution coordination completed")

	return response.RecipeResults, nil
}

// ExecuteMedicationIntelligence executes medication intelligence via Rust engine
func (rc *RecipeCoordinator) ExecuteMedicationIntelligence(
	ctx context.Context,
	request *models.RustIntelligenceRequest,
) (*models.MedicationIntelligenceResponse, error) {
	start := time.Now()

	rc.logger.WithFields(logrus.Fields{
		"request_id":        request.RequestID,
		"patient_id":        request.PatientID,
		"intelligence_type": request.IntelligenceType,
	}).Info("Starting medication intelligence coordination")

	// Execute via Rust engine
	response, err := rc.rustRecipeClient.ExecuteMedicationIntelligence(ctx, request)
	if err != nil {
		rc.metricsService.IncrementRustEngineFailures()
		rc.logger.WithFields(logrus.Fields{
			"request_id": request.RequestID,
			"error":      err.Error(),
		}).Error("Medication intelligence execution failed")
		return nil, err
	}

	// Record metrics
	executionTime := time.Since(start)
	rc.metricsService.RecordRustEngineLatency(executionTime)

	rc.logger.WithFields(logrus.Fields{
		"request_id":         request.RequestID,
		"execution_time_ms":  executionTime.Milliseconds(),
		"intelligence_score": response.IntelligenceScore,
	}).Info("Medication intelligence coordination completed")

	return response, nil
}

// ExecuteDoseOptimization executes dose optimization via Rust engine
func (rc *RecipeCoordinator) ExecuteDoseOptimization(
	ctx context.Context,
	request *models.RustDoseOptimizationRequest,
) (*models.DoseOptimizationResponse, error) {
	start := time.Now()

	rc.logger.WithFields(logrus.Fields{
		"request_id":      request.RequestID,
		"patient_id":      request.PatientID,
		"medication_code": request.MedicationCode,
	}).Info("Starting dose optimization coordination")

	// Execute via Rust engine
	response, err := rc.rustRecipeClient.ExecuteDoseOptimization(ctx, request)
	if err != nil {
		rc.metricsService.IncrementRustEngineFailures()
		rc.logger.WithFields(logrus.Fields{
			"request_id": request.RequestID,
			"error":      err.Error(),
		}).Error("Dose optimization execution failed")
		return nil, err
	}

	// Record metrics
	executionTime := time.Since(start)
	rc.metricsService.RecordRustEngineLatency(executionTime)

	rc.logger.WithFields(logrus.Fields{
		"request_id":          request.RequestID,
		"execution_time_ms":   executionTime.Milliseconds(),
		"optimization_score":  response.OptimizationScore,
		"optimized_dose":      response.OptimizedDose,
	}).Info("Dose optimization coordination completed")

	return response, nil
}

// ExecuteSafetyValidation executes safety validation via Rust engine
func (rc *RecipeCoordinator) ExecuteSafetyValidation(
	ctx context.Context,
	request *models.RustSafetyValidationRequest,
) (*models.SafetyValidationResponse, error) {
	start := time.Now()

	rc.logger.WithFields(logrus.Fields{
		"request_id":       request.RequestID,
		"patient_id":       request.PatientID,
		"medications_count": len(request.Medications),
		"validation_level": request.ValidationLevel,
	}).Info("Starting safety validation coordination")

	// Execute via Rust engine
	response, err := rc.rustRecipeClient.ExecuteSafetyValidation(ctx, request)
	if err != nil {
		rc.metricsService.IncrementRustEngineFailures()
		rc.logger.WithFields(logrus.Fields{
			"request_id": request.RequestID,
			"error":      err.Error(),
		}).Error("Safety validation execution failed")
		return nil, err
	}

	// Record metrics
	executionTime := time.Since(start)
	rc.metricsService.RecordRustEngineLatency(executionTime)

	rc.logger.WithFields(logrus.Fields{
		"request_id":             request.RequestID,
		"execution_time_ms":      executionTime.Milliseconds(),
		"overall_safety_status":  response.OverallSafetyStatus,
		"safety_score":           response.SafetyScore,
		"drug_interactions":      len(response.DrugInteractions),
		"allergy_alerts":         len(response.AllergyAlerts),
	}).Info("Safety validation coordination completed")

	return response, nil
}
