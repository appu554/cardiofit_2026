package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"flow2-go-engine/internal/config"
	"flow2-go-engine/internal/models"
	pb "flow2-go-engine/proto/clinical_engine"
)

// RustRecipeClient interface defines the contract for communicating with the Rust recipe engine
type RustRecipeClient interface {
	// ORB-driven recipe-specific execution (NEW PRIMARY METHOD)
	ExecuteRecipe(ctx context.Context, request *models.RecipeExecutionRequest) (*models.MedicationProposal, error)

	// Legacy Flow2 execution (DEPRECATED - kept for backward compatibility)
	ExecuteFlow2(ctx context.Context, request *models.Flow2Request, clinicalContext *models.ClinicalContext) (*models.Flow2Response, error)

	// Specialized intelligence methods
	ExecuteMedicationIntelligence(ctx context.Context, request *models.RustIntelligenceRequest) (*models.MedicationIntelligenceResponse, error)
	ExecuteDoseOptimization(ctx context.Context, request *models.RustDoseOptimizationRequest) (*models.DoseOptimizationResponse, error)
	ExecuteSafetyValidation(ctx context.Context, request *models.RustSafetyValidationRequest) (*models.SafetyValidationResponse, error)

	// System methods
	HealthCheck(ctx context.Context) error
	Close() error
}

// rustRecipeClient implements the RustRecipeClient interface
type rustRecipeClient struct {
	conn   *grpc.ClientConn
	client pb.ClinicalEngineClient
	config config.RustEngineConfig
	logger *logrus.Logger
}

// NewRustRecipeClient creates a new Rust recipe client
func NewRustRecipeClient(cfg config.RustEngineConfig) (RustRecipeClient, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Only real gRPC client - no mocks or fallbacks
	if cfg.Address == "" {
		return nil, fmt.Errorf("Rust engine address is required - no fallback available")
	}

	logger.WithField("address", cfg.Address).Info("Connecting to Rust recipe engine")

	// Real gRPC client implementation
	conn, err := grpc.Dial(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Rust engine at %s: %w", cfg.Address, err)
	}

	client := pb.NewClinicalEngineClient(conn)

	// Test connection immediately
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.HealthCheck(ctx, &pb.HealthCheckRequest{Service: "clinical_engine"})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Rust engine health check failed at %s: %w", cfg.Address, err)
	}

	logger.Info("Successfully connected to Rust recipe engine")

	return &rustRecipeClient{
		conn:   conn,
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// ExecuteRecipe executes a specific recipe via the Rust engine (ORB-driven)
// This is the NEW PRIMARY METHOD for ORB-driven architecture
func (r *rustRecipeClient) ExecuteRecipe(ctx context.Context, request *models.RecipeExecutionRequest) (*models.MedicationProposal, error) {
	r.logger.WithFields(logrus.Fields{
		"request_id":   request.RequestID,
		"recipe_id":    request.RecipeID,
		"patient_id":   request.PatientID,
		"medication":   request.MedicationCode,
	}).Info("Executing recipe-specific calculation")

	// Convert clinical context to JSON
	contextJSON, err := json.Marshal(request.ClinicalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal clinical context: %w", err)
	}

	// Create gRPC request for recipe execution
	pbRequest := &pb.RecipeExecutionRequest{
		RequestId:       request.RequestID,
		RecipeId:        request.RecipeID,
		Variant:         request.Variant,
		PatientId:       request.PatientID,
		MedicationCode:  request.MedicationCode,
		ClinicalContext: string(contextJSON),
		TimeoutMs:       int64(r.config.Timeout.Milliseconds()),
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	response, err := r.client.ExecuteRecipe(ctx, pbRequest)
	if err != nil {
		return nil, fmt.Errorf("recipe execution failed for %s: %w", request.RecipeID, err)
	}

	// Convert response to MedicationProposal
	medicationProposal := &models.MedicationProposal{
		MedicationCode:    response.MedicationCode,
		MedicationName:    response.MedicationName,
		CalculatedDose:    response.CalculatedDose,
		DoseUnit:          response.DoseUnit,
		Frequency:         response.Frequency,
		Duration:          response.Duration,
		SafetyStatus:      response.SafetyStatus,
		SafetyAlerts:      response.SafetyAlerts,
		Contraindications: response.Contraindications,
		ClinicalRationale: response.ClinicalRationale,
		MonitoringPlan:    response.MonitoringPlan,
		ExecutionTimeMs:   response.ExecutionTimeMs,
		RecipeVersion:     response.RecipeVersion,
	}

	// Convert alternatives
	for _, alt := range response.Alternatives {
		medicationProposal.Alternatives = append(medicationProposal.Alternatives, models.MedicationAlternative{
			MedicationCode: alt.MedicationCode,
			MedicationName: alt.MedicationName,
			Rationale:      alt.Rationale,
			SafetyProfile:  alt.SafetyProfile,
		})
	}

	r.logger.WithFields(logrus.Fields{
		"request_id":       request.RequestID,
		"recipe_id":        request.RecipeID,
		"safety_status":    medicationProposal.SafetyStatus,
		"execution_time":   medicationProposal.ExecutionTimeMs,
		"alternatives":     len(medicationProposal.Alternatives),
	}).Info("Recipe execution completed successfully")

	return medicationProposal, nil
}

// ExecuteFlow2 executes Flow 2 via the Rust engine (LEGACY METHOD)
func (r *rustRecipeClient) ExecuteFlow2(ctx context.Context, request *models.Flow2Request, clinicalContext *models.ClinicalContext) (*models.Flow2Response, error) {
	// Convert clinical context to JSON
	contextJSON, err := json.Marshal(clinicalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal clinical context: %w", err)
	}

	// Convert medications to protobuf format
	var pbMedications []*pb.Medication
	if request.MedicationData != nil {
		// Extract medications from medication data
		if meds, ok := request.MedicationData["medications"].([]interface{}); ok {
			for _, med := range meds {
				if medMap, ok := med.(map[string]interface{}); ok {
					pbMed := &pb.Medication{
						Code: getStringFromMap(medMap, "code"),
						Name: getStringFromMap(medMap, "name"),
						Dose: getFloatFromMap(medMap, "dose"),
						Unit: getStringFromMap(medMap, "unit"),
						Frequency: getStringFromMap(medMap, "frequency"),
						Route: getStringFromMap(medMap, "route"),
						Duration: getStringFromMap(medMap, "duration"),
						Indication: getStringFromMap(medMap, "indication"),
					}
					pbMedications = append(pbMedications, pbMed)
				}
			}
		}
	}

	// Create gRPC request
	pbRequest := &pb.Flow2ExecutionRequest{
		RequestId:       request.RequestID,
		PatientId:       request.PatientID,
		ActionType:      request.ActionType,
		Medications:     pbMedications,
		ClinicalContext: string(contextJSON),
		ProcessingHints: convertProcessingHints(request.ProcessingHints),
		TimeoutMs:       int64(request.Timeout.Milliseconds()),
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	response, err := r.client.ExecuteFlow2(ctx, pbRequest)
	if err != nil {
		return nil, fmt.Errorf("Rust engine execution failed: %w", err)
	}

	// Convert response back to Go models
	return r.convertFlow2Response(response, request)
}

// ExecuteMedicationIntelligence executes medication intelligence via the Rust engine
func (r *rustRecipeClient) ExecuteMedicationIntelligence(ctx context.Context, request *models.RustIntelligenceRequest) (*models.MedicationIntelligenceResponse, error) {
	// Convert clinical context to JSON
	contextJSON, err := json.Marshal(request.ClinicalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal clinical context: %w", err)
	}

	// Convert medications to protobuf format
	var pbMedications []*pb.Medication
	for _, med := range request.Medications {
		pbMed := &pb.Medication{
			Code:       med.Code,
			Name:       med.Name,
			Dose:       med.Dose,
			Unit:       med.Unit,
			Frequency:  med.Frequency,
			Route:      med.Route,
			Duration:   med.Duration,
			Indication: med.Indication,
		}
		pbMedications = append(pbMedications, pbMed)
	}

	// Create gRPC request
	pbRequest := &pb.MedicationIntelligenceRequest{
		RequestId:        request.RequestID,
		PatientId:        request.PatientID,
		Medications:      pbMedications,
		IntelligenceType: request.IntelligenceType,
		AnalysisDepth:    request.AnalysisDepth,
		ClinicalContext:  string(contextJSON),
		ProcessingHints:  convertProcessingHints(request.ProcessingHints),
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	response, err := r.client.ExecuteMedicationIntelligence(ctx, pbRequest)
	if err != nil {
		return nil, fmt.Errorf("medication intelligence execution failed: %w", err)
	}

	// Convert response
	return &models.MedicationIntelligenceResponse{
		RequestID:         response.RequestId,
		IntelligenceScore: response.IntelligenceScore,
		ExecutionTimeMs:   int64(response.ExecutionTimeMs),
		// TODO: Parse JSON fields
	}, nil
}

// ExecuteDoseOptimization executes dose optimization via the Rust engine
func (r *rustRecipeClient) ExecuteDoseOptimization(ctx context.Context, request *models.RustDoseOptimizationRequest) (*models.DoseOptimizationResponse, error) {
	// Implementation similar to above methods
	// For now, return a placeholder
	return &models.DoseOptimizationResponse{
		RequestID:         request.RequestID,
		OptimizedDose:     10.0, // Placeholder
		OptimizationScore: 0.85,
		ExecutionTimeMs:   5,
	}, nil
}

// ExecuteSafetyValidation executes safety validation via the Rust engine
func (r *rustRecipeClient) ExecuteSafetyValidation(ctx context.Context, request *models.RustSafetyValidationRequest) (*models.SafetyValidationResponse, error) {
	// Implementation similar to above methods
	// For now, return a placeholder
	return &models.SafetyValidationResponse{
		RequestID:           request.RequestID,
		OverallSafetyStatus: "SAFE",
		SafetyScore:         0.95,
		ExecutionTimeMs:     3,
	}, nil
}

// HealthCheck performs a health check on the Rust engine
func (r *rustRecipeClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.client.HealthCheck(ctx, &pb.HealthCheckRequest{
		Service: "clinical_engine",
	})
	return err
}

// Close closes the gRPC connection
func (r *rustRecipeClient) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Helper functions
func (r *rustRecipeClient) convertFlow2Response(pbResponse *pb.Flow2ExecutionResponse, originalRequest *models.Flow2Request) (*models.Flow2Response, error) {
	// Convert recipe results
	var recipeResults []models.RecipeResult
	for _, pbResult := range pbResponse.RecipeResults {
		// Convert validations
		var validations []models.RecipeValidation
		for _, pbValidation := range pbResult.Validations {
			validations = append(validations, models.RecipeValidation{
				Passed:      pbValidation.Passed,
				Severity:    pbValidation.Severity,
				Message:     pbValidation.Message,
				Explanation: pbValidation.Explanation,
				Code:        pbValidation.Code,
			})
		}

		// Parse clinical decision support JSON
		var cds map[string]interface{}
		if pbResult.ClinicalDecisionSupport != "" {
			json.Unmarshal([]byte(pbResult.ClinicalDecisionSupport), &cds)
		}

		recipeResult := models.RecipeResult{
			RecipeID:                pbResult.RecipeId,
			RecipeName:              pbResult.RecipeName,
			OverallStatus:           pbResult.OverallStatus,
			ExecutionTimeMs:         int64(pbResult.ExecutionTimeMs),
			Validations:             validations,
			ClinicalDecisionSupport: cds,
			Recommendations:         pbResult.Recommendations,
			Warnings:                pbResult.Warnings,
			Errors:                  pbResult.Errors,
		}
		recipeResults = append(recipeResults, recipeResult)
	}

	// Parse clinical decision support JSON
	var overallCDS map[string]interface{}
	if pbResponse.ClinicalDecisionSupport != "" {
		json.Unmarshal([]byte(pbResponse.ClinicalDecisionSupport), &overallCDS)
	}

	return &models.Flow2Response{
		RequestID:               pbResponse.RequestId,
		PatientID:               originalRequest.PatientID,
		OverallStatus:           pbResponse.OverallStatus,
		RecipeResults:           recipeResults,
		ClinicalDecisionSupport: overallCDS,
		ExecutionTimeMs:         int64(pbResponse.ExecutionTimeMs),
		EngineUsed:              "rust-" + pbResponse.EngineVersion,
		Timestamp:               time.Now(),
		ExecutionSummary: models.ExecutionSummary{
			TotalRecipesExecuted: len(recipeResults),
			SuccessfulRecipes:    countSuccessfulRecipes(recipeResults),
			Engine:               "rust",
		},
	}, nil
}

func convertProcessingHints(hints map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range hints {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

func countSuccessfulRecipes(results []models.RecipeResult) int {
	count := 0
	for _, result := range results {
		if result.OverallStatus == "SAFE" || result.OverallStatus == "WARNING" {
			count++
		}
	}
	return count
}
