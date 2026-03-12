package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"medication-service-v2/internal/infrastructure/clients"
)

// ClinicalCalculationService orchestrates high-performance clinical calculations via Rust engine
type ClinicalCalculationService struct {
	rustClient         *clients.RustClinicalEngineClient
	performanceMonitor *PerformanceMonitorService
	metricsService     *MetricsService
	logger             *zap.Logger
	config             *ClinicalCalculationConfig
	calculationCache   sync.Map // Cache for frequently calculated results
}

// ClinicalCalculationConfig holds configuration for clinical calculations
type ClinicalCalculationConfig struct {
	EnableCaching         bool          `json:"enable_caching" mapstructure:"enable_caching"`
	CacheExpirationTime   time.Duration `json:"cache_expiration_time" mapstructure:"cache_expiration_time"`
	MaxConcurrentRequests int           `json:"max_concurrent_requests" mapstructure:"max_concurrent_requests"`
	DefaultTimeout        time.Duration `json:"default_timeout" mapstructure:"default_timeout"`
	PerformanceTargets    *PerformanceTargets `json:"performance_targets" mapstructure:"performance_targets"`
}

// PerformanceTargets defines target performance metrics
type PerformanceTargets struct {
	DrugInteractionAnalysis time.Duration `json:"drug_interaction_analysis" mapstructure:"drug_interaction_analysis"`
	DosageCalculation      time.Duration `json:"dosage_calculation" mapstructure:"dosage_calculation"`
	SafetyValidation       time.Duration `json:"safety_validation" mapstructure:"safety_validation"`
	RuleEvaluation         time.Duration `json:"rule_evaluation" mapstructure:"rule_evaluation"`
}

// NewClinicalCalculationService creates a new clinical calculation service
func NewClinicalCalculationService(
	rustClient *clients.RustClinicalEngineClient,
	performanceMonitor *PerformanceMonitorService,
	metricsService *MetricsService,
	config *ClinicalCalculationConfig,
	logger *zap.Logger,
) *ClinicalCalculationService {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 2 * time.Second
	}
	if config.CacheExpirationTime == 0 {
		config.CacheExpirationTime = 10 * time.Minute
	}
	if config.MaxConcurrentRequests == 0 {
		config.MaxConcurrentRequests = 100
	}

	// Set default performance targets if not provided
	if config.PerformanceTargets == nil {
		config.PerformanceTargets = &PerformanceTargets{
			DrugInteractionAnalysis: 50 * time.Millisecond,
			DosageCalculation:      30 * time.Millisecond,
			SafetyValidation:       75 * time.Millisecond,
			RuleEvaluation:         40 * time.Millisecond,
		}
	}

	return &ClinicalCalculationService{
		rustClient:         rustClient,
		performanceMonitor: performanceMonitor,
		metricsService:     metricsService,
		logger:            logger.Named("clinical-calculation-service"),
		config:            config,
	}
}

// Phase3ClinicalIntelligenceRequest represents a request for Phase 3 clinical intelligence
type Phase3ClinicalIntelligenceRequest struct {
	WorkflowID          uuid.UUID                  `json:"workflow_id"`
	PatientID           string                     `json:"patient_id"`
	SnapshotData        *SnapshotBasedContextData  `json:"snapshot_data"`
	RequestedOperations []ClinicalOperation        `json:"requested_operations"`
	Priority            string                     `json:"priority"`
	RequestedBy         string                     `json:"requested_by"`
	RequestedAt         time.Time                  `json:"requested_at"`
}

// ClinicalOperation represents a specific clinical operation to perform
type ClinicalOperation struct {
	OperationType       string                 `json:"operation_type"` // drug_interactions, dosage_calculation, safety_validation, rule_evaluation
	OperationID         string                 `json:"operation_id"`
	Parameters          map[string]interface{} `json:"parameters"`
	PerformanceTarget   time.Duration          `json:"performance_target"`
	RequiredConfidence  float64                `json:"required_confidence"`
}

// Phase3ClinicalIntelligenceResponse represents the response from Phase 3 processing
type Phase3ClinicalIntelligenceResponse struct {
	WorkflowID               uuid.UUID                    `json:"workflow_id"`
	PatientID                string                       `json:"patient_id"`
	OperationResults         []ClinicalOperationResult    `json:"operation_results"`
	DrugInteractionAnalysis  *clients.DrugInteractionResponse    `json:"drug_interaction_analysis,omitempty"`
	DosageCalculations       []clients.DosageCalculationResponse `json:"dosage_calculations,omitempty"`
	SafetyValidationResults  []clients.SafetyValidationResponse  `json:"safety_validation_results,omitempty"`
	RuleEvaluationResults    []clients.ClinicalRuleEvaluationResponse `json:"rule_evaluation_results,omitempty"`
	OverallRiskAssessment    *OverallRiskAssessment       `json:"overall_risk_assessment"`
	PerformanceMetrics       *ClinicalCalculationMetrics  `json:"performance_metrics"`
	QualityScore             float64                       `json:"quality_score"`
	ProcessingTime           time.Duration                 `json:"processing_time"`
	Success                  bool                          `json:"success"`
	Errors                   []ClinicalCalculationError    `json:"errors,omitempty"`
	Warnings                 []ClinicalCalculationWarning  `json:"warnings,omitempty"`
}

// ClinicalOperationResult represents the result of a specific clinical operation
type ClinicalOperationResult struct {
	OperationID         string                 `json:"operation_id"`
	OperationType       string                 `json:"operation_type"`
	Status              string                 `json:"status"` // completed, failed, partial, timeout
	Result              interface{}            `json:"result"`
	ProcessingTime      time.Duration          `json:"processing_time"`
	PerformanceTarget   time.Duration          `json:"performance_target"`
	PerformanceMet      bool                   `json:"performance_met"`
	ConfidenceScore     float64                `json:"confidence_score"`
	Error               string                 `json:"error,omitempty"`
}

// OverallRiskAssessment represents the overall risk assessment from all operations
type OverallRiskAssessment struct {
	OverallRiskScore     float64                    `json:"overall_risk_score"`
	RiskLevel           string                     `json:"risk_level"` // low, moderate, high, critical
	PrimaryRiskFactors  []RiskFactor              `json:"primary_risk_factors"`
	Recommendations     []ClinicalRecommendation  `json:"recommendations"`
	RequiredActions     []RequiredAction          `json:"required_actions"`
	MonitoringRequirements []MonitoringRequirement `json:"monitoring_requirements"`
}

// RiskFactor represents a clinical risk factor
type RiskFactor struct {
	RiskID          string  `json:"risk_id"`
	Category        string  `json:"category"`
	Description     string  `json:"description"`
	Severity        string  `json:"severity"`
	Impact          string  `json:"impact"`
	Mitigation      string  `json:"mitigation"`
	Evidence        string  `json:"evidence"`
	Score           float64 `json:"score"`
}

// RequiredAction represents an action that must be taken
type RequiredAction struct {
	ActionID        string `json:"action_id"`
	ActionType      string `json:"action_type"`
	Description     string `json:"description"`
	Urgency         string `json:"urgency"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	ResponsibleRole string `json:"responsible_role"`
}

// MonitoringRequirement represents a monitoring requirement
type MonitoringRequirement struct {
	MonitoringID    string        `json:"monitoring_id"`
	Parameter       string        `json:"parameter"`
	Frequency       time.Duration `json:"frequency"`
	Thresholds      map[string]float64 `json:"thresholds"`
	AlertConditions []string      `json:"alert_conditions"`
}

// ClinicalCalculationMetrics represents performance metrics for clinical calculations
type ClinicalCalculationMetrics struct {
	TotalOperations         int           `json:"total_operations"`
	SuccessfulOperations    int           `json:"successful_operations"`
	FailedOperations        int           `json:"failed_operations"`
	AverageProcessingTime   time.Duration `json:"average_processing_time"`
	FastestOperation        time.Duration `json:"fastest_operation"`
	SlowestOperation        time.Duration `json:"slowest_operation"`
	TargetsMet              int           `json:"targets_met"`
	TargetsMissed           int           `json:"targets_missed"`
	CacheHitRate            float64       `json:"cache_hit_rate"`
	RustEngineResponseTimes map[string]time.Duration `json:"rust_engine_response_times"`
}

// ClinicalCalculationError represents an error during clinical calculations
type ClinicalCalculationError struct {
	ErrorID         string `json:"error_id"`
	OperationID     string `json:"operation_id"`
	OperationType   string `json:"operation_type"`
	ErrorType       string `json:"error_type"`
	ErrorMessage    string `json:"error_message"`
	Severity        string `json:"severity"`
	Recoverable     bool   `json:"recoverable"`
	Timestamp       time.Time `json:"timestamp"`
}

// ClinicalCalculationWarning represents a warning during clinical calculations
type ClinicalCalculationWarning struct {
	WarningID       string `json:"warning_id"`
	OperationID     string `json:"operation_id"`
	OperationType   string `json:"operation_type"`
	WarningMessage  string `json:"warning_message"`
	Severity        string `json:"severity"`
	Recommendation  string `json:"recommendation"`
	Timestamp       time.Time `json:"timestamp"`
}

// Main service method for Phase 3 clinical intelligence processing
func (s *ClinicalCalculationService) ProcessPhase3Intelligence(ctx context.Context, request *Phase3ClinicalIntelligenceRequest) (*Phase3ClinicalIntelligenceResponse, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	s.logger.Info("Starting Phase 3 clinical intelligence processing",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("patient_id", request.PatientID),
		zap.String("request_id", requestID),
		zap.Int("operations_count", len(request.RequestedOperations)))

	// Initialize response
	response := &Phase3ClinicalIntelligenceResponse{
		WorkflowID:        request.WorkflowID,
		PatientID:         request.PatientID,
		OperationResults:  make([]ClinicalOperationResult, 0, len(request.RequestedOperations)),
		Success:           true,
	}

	// Process operations in parallel for maximum performance
	operationChan := make(chan ClinicalOperationResult, len(request.RequestedOperations))
	var wg sync.WaitGroup

	for _, operation := range request.RequestedOperations {
		wg.Add(1)
		go func(op ClinicalOperation) {
			defer wg.Done()
			result := s.processOperation(ctx, requestID, request.PatientID, request.SnapshotData, &op)
			operationChan <- result
		}(operation)
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(operationChan)
	}()

	// Collect results
	var errors []ClinicalCalculationError
	var warnings []ClinicalCalculationWarning
	metrics := &ClinicalCalculationMetrics{
		RustEngineResponseTimes: make(map[string]time.Duration),
	}

	for result := range operationChan {
		response.OperationResults = append(response.OperationResults, result)
		
		if result.Status == "failed" {
			response.Success = false
			errors = append(errors, ClinicalCalculationError{
				ErrorID:       uuid.New().String(),
				OperationID:   result.OperationID,
				OperationType: result.OperationType,
				ErrorType:     "operation_failed",
				ErrorMessage:  result.Error,
				Severity:      "high",
				Recoverable:   true,
				Timestamp:     time.Now(),
			})
		}

		if !result.PerformanceMet {
			warnings = append(warnings, ClinicalCalculationWarning{
				WarningID:      uuid.New().String(),
				OperationID:    result.OperationID,
				OperationType:  result.OperationType,
				WarningMessage: fmt.Sprintf("Performance target missed: %v > %v", result.ProcessingTime, result.PerformanceTarget),
				Severity:       "medium",
				Recommendation: "Consider optimization or increasing timeout",
				Timestamp:      time.Now(),
			})
		}

		// Update metrics
		metrics.TotalOperations++
		if result.Status == "completed" {
			metrics.SuccessfulOperations++
		} else {
			metrics.FailedOperations++
		}

		if result.PerformanceMet {
			metrics.TargetsMet++
		} else {
			metrics.TargetsMissed++
		}

		metrics.RustEngineResponseTimes[result.OperationType] = result.ProcessingTime
	}

	// Process specialized results based on operation types
	s.processSpecializedResults(response, request.SnapshotData)

	// Calculate overall risk assessment
	response.OverallRiskAssessment = s.calculateOverallRiskAssessment(response)

	// Calculate quality score
	response.QualityScore = s.calculateQualityScore(response)

	// Finalize metrics
	response.ProcessingTime = time.Since(startTime)
	s.calculateFinalMetrics(metrics, response.OperationResults, response.ProcessingTime)
	response.PerformanceMetrics = metrics
	response.Errors = errors
	response.Warnings = warnings

	s.logger.Info("Phase 3 clinical intelligence processing completed",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("request_id", requestID),
		zap.Duration("processing_time", response.ProcessingTime),
		zap.Bool("success", response.Success),
		zap.Float64("quality_score", response.QualityScore),
		zap.Int("operations_completed", metrics.SuccessfulOperations),
		zap.Int("operations_failed", metrics.FailedOperations))

	// Record performance metrics
	if s.metricsService != nil {
		s.metricsService.RecordPhase3Performance(response.ProcessingTime, response.Success)
	}

	return response, nil
}

// Process individual clinical operation
func (s *ClinicalCalculationService) processOperation(ctx context.Context, requestID, patientID string, snapshotData *SnapshotBasedContextData, operation *ClinicalOperation) ClinicalOperationResult {
	operationStartTime := time.Now()
	operationRequestID := fmt.Sprintf("%s-%s", requestID, operation.OperationID)

	s.logger.Debug("Processing clinical operation",
		zap.String("operation_id", operation.OperationID),
		zap.String("operation_type", operation.OperationType))

	result := ClinicalOperationResult{
		OperationID:       operation.OperationID,
		OperationType:     operation.OperationType,
		PerformanceTarget: operation.PerformanceTarget,
		Status:           "processing",
	}

	// Check cache first if caching is enabled
	if s.config.EnableCaching {
		if cachedResult := s.getCachedResult(operation, snapshotData); cachedResult != nil {
			result.Result = cachedResult
			result.Status = "completed"
			result.ProcessingTime = time.Since(operationStartTime)
			result.PerformanceMet = result.ProcessingTime <= operation.PerformanceTarget
			result.ConfidenceScore = 1.0
			s.logger.Debug("Using cached result for operation", zap.String("operation_id", operation.OperationID))
			return result
		}
	}

	// Set timeout based on performance target
	operationCtx, cancel := context.WithTimeout(ctx, operation.PerformanceTarget*2)
	defer cancel()

	// Process based on operation type
	switch operation.OperationType {
	case "drug_interactions":
		drugResult, err := s.processDrugInteractions(operationCtx, operationRequestID, patientID, snapshotData, operation.Parameters)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "completed"
			result.Result = drugResult
			result.ConfidenceScore = s.calculateConfidenceScore(drugResult, operation.RequiredConfidence)
		}

	case "dosage_calculation":
		dosageResult, err := s.processDosageCalculation(operationCtx, operationRequestID, patientID, snapshotData, operation.Parameters)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "completed"
			result.Result = dosageResult
			result.ConfidenceScore = s.calculateConfidenceScore(dosageResult, operation.RequiredConfidence)
		}

	case "safety_validation":
		safetyResult, err := s.processSafetyValidation(operationCtx, operationRequestID, patientID, snapshotData, operation.Parameters)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "completed"
			result.Result = safetyResult
			result.ConfidenceScore = s.calculateConfidenceScore(safetyResult, operation.RequiredConfidence)
		}

	case "rule_evaluation":
		ruleResult, err := s.processRuleEvaluation(operationCtx, operationRequestID, patientID, snapshotData, operation.Parameters)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "completed"
			result.Result = ruleResult
			result.ConfidenceScore = s.calculateConfidenceScore(ruleResult, operation.RequiredConfidence)
		}

	default:
		result.Status = "failed"
		result.Error = fmt.Sprintf("unsupported operation type: %s", operation.OperationType)
	}

	result.ProcessingTime = time.Since(operationStartTime)
	result.PerformanceMet = result.ProcessingTime <= operation.PerformanceTarget

	// Cache successful results if caching is enabled
	if s.config.EnableCaching && result.Status == "completed" && result.Result != nil {
		s.cacheResult(operation, snapshotData, result.Result)
	}

	s.logger.Debug("Clinical operation processed",
		zap.String("operation_id", operation.OperationID),
		zap.String("status", result.Status),
		zap.Duration("processing_time", result.ProcessingTime),
		zap.Bool("performance_met", result.PerformanceMet))

	return result
}

// Process drug interactions
func (s *ClinicalCalculationService) processDrugInteractions(ctx context.Context, requestID, patientID string, snapshotData *SnapshotBasedContextData, parameters map[string]interface{}) (*clients.DrugInteractionResponse, error) {
	// Extract medications from snapshot data
	medications := s.extractMedicationsFromSnapshot(snapshotData)
	
	request := &clients.DrugInteractionRequest{
		RequestID:       requestID,
		PatientID:       patientID,
		Medications:     medications,
		ClinicalContext: s.buildClinicalContext(snapshotData),
		AnalysisDepth:   s.getStringParameter(parameters, "analysis_depth", "comprehensive"),
		Priority:        s.getStringParameter(parameters, "priority", "high"),
	}

	return s.rustClient.AnalyzeDrugInteractions(ctx, request)
}

// Process dosage calculation
func (s *ClinicalCalculationService) processDosageCalculation(ctx context.Context, requestID, patientID string, snapshotData *SnapshotBasedContextData, parameters map[string]interface{}) (*clients.DosageCalculationResponse, error) {
	medicationCode := s.getStringParameter(parameters, "medication_code", "")
	if medicationCode == "" {
		return nil, fmt.Errorf("medication_code is required for dosage calculation")
	}

	request := &clients.DosageCalculationRequest{
		RequestID:       requestID,
		PatientID:       patientID,
		MedicationCode:  medicationCode,
		PatientWeight:   s.getFloatParameter(parameters, "patient_weight", 70.0),
		PatientAge:      s.getIntParameter(parameters, "patient_age", 35),
		KidneyFunction:  s.extractKidneyFunction(snapshotData),
		LiverFunction:   s.extractLiverFunction(snapshotData),
		ClinicalContext: s.buildClinicalContext(snapshotData),
		CalculationType: s.getStringParameter(parameters, "calculation_type", "standard"),
	}

	return s.rustClient.CalculateDosage(ctx, request)
}

// Process safety validation
func (s *ClinicalCalculationService) processSafetyValidation(ctx context.Context, requestID, patientID string, snapshotData *SnapshotBasedContextData, parameters map[string]interface{}) (*clients.SafetyValidationResponse, error) {
	proposedMed := s.extractProposedMedication(parameters)
	if proposedMed == nil {
		return nil, fmt.Errorf("proposed_medication is required for safety validation")
	}

	request := &clients.SafetyValidationRequest{
		RequestID:          requestID,
		PatientID:          patientID,
		ProposedMedication: proposedMed,
		CurrentMedications: s.extractMedicationsFromSnapshot(snapshotData),
		Allergies:          s.extractAllergies(snapshotData),
		Conditions:         s.extractConditions(snapshotData),
		ClinicalContext:    s.buildClinicalContext(snapshotData),
		ValidationLevel:    s.getStringParameter(parameters, "validation_level", "comprehensive"),
	}

	return s.rustClient.ValidateSafety(ctx, request)
}

// Process rule evaluation
func (s *ClinicalCalculationService) processRuleEvaluation(ctx context.Context, requestID, patientID string, snapshotData *SnapshotBasedContextData, parameters map[string]interface{}) (*clients.ClinicalRuleEvaluationResponse, error) {
	ruleSet := s.getStringParameter(parameters, "rule_set", "drug_rules")

	request := &clients.ClinicalRuleEvaluationRequest{
		RequestID:         requestID,
		PatientID:         patientID,
		RuleSet:           ruleSet,
		EvaluationContext: s.buildClinicalContext(snapshotData),
		RuleFilters:       s.getStringSliceParameter(parameters, "rule_filters", nil),
		Priority:          s.getStringParameter(parameters, "priority", "high"),
	}

	return s.rustClient.EvaluateRules(ctx, request)
}

// Helper methods for data extraction and processing

func (s *ClinicalCalculationService) extractMedicationsFromSnapshot(snapshotData *SnapshotBasedContextData) []clients.MedicationForAnalysis {
	// This would extract medications from the snapshot data structure
	// Implementation depends on the actual snapshot data format
	var medications []clients.MedicationForAnalysis
	
	// Placeholder implementation - should be replaced with actual extraction logic
	if snapshotData != nil && snapshotData.MedicationData != nil {
		for _, medData := range snapshotData.MedicationData {
			medication := clients.MedicationForAnalysis{
				MedicationCode: s.getStringFromInterface(medData, "medication_code"),
				Name:          s.getStringFromInterface(medData, "name"),
				Dose:          s.getStringFromInterface(medData, "dose"),
				Route:         s.getStringFromInterface(medData, "route"),
				Frequency:     s.getStringFromInterface(medData, "frequency"),
				IsActive:      s.getBoolFromInterface(medData, "is_active"),
			}
			medications = append(medications, medication)
		}
	}
	
	return medications
}

func (s *ClinicalCalculationService) buildClinicalContext(snapshotData *SnapshotBasedContextData) map[string]interface{} {
	context := make(map[string]interface{})
	
	if snapshotData != nil {
		context["patient_data"] = snapshotData.PatientData
		context["medication_data"] = snapshotData.MedicationData
		context["clinical_data"] = snapshotData.ClinicalData
		context["recipe_data"] = snapshotData.RecipeData
		context["metadata"] = snapshotData.MetaData
	}
	
	return context
}

func (s *ClinicalCalculationService) extractKidneyFunction(snapshotData *SnapshotBasedContextData) *clients.KidneyFunctionData {
	// Extract kidney function data from snapshot
	// Placeholder implementation
	return &clients.KidneyFunctionData{
		CreatinineLevel:     1.0,
		CreatinineClearance: 90.0,
		GFR:                90.0,
		Stage:              "normal",
	}
}

func (s *ClinicalCalculationService) extractLiverFunction(snapshotData *SnapshotBasedContextData) *clients.LiverFunctionData {
	// Extract liver function data from snapshot
	// Placeholder implementation
	return &clients.LiverFunctionData{
		ALTLevel:       20.0,
		ASTLevel:       18.0,
		BilirubinLevel: 1.0,
		AlbuminLevel:   4.0,
		ChildPughScore: 5,
	}
}

func (s *ClinicalCalculationService) extractProposedMedication(parameters map[string]interface{}) *clients.MedicationForAnalysis {
	proposedMedData, ok := parameters["proposed_medication"]
	if !ok {
		return nil
	}

	// Convert to MedicationForAnalysis
	medData, ok := proposedMedData.(map[string]interface{})
	if !ok {
		return nil
	}

	return &clients.MedicationForAnalysis{
		MedicationCode: s.getStringFromInterface(medData, "medication_code"),
		Name:          s.getStringFromInterface(medData, "name"),
		Dose:          s.getStringFromInterface(medData, "dose"),
		Route:         s.getStringFromInterface(medData, "route"),
		Frequency:     s.getStringFromInterface(medData, "frequency"),
		IsActive:      s.getBoolFromInterface(medData, "is_active"),
	}
}

func (s *ClinicalCalculationService) extractAllergies(snapshotData *SnapshotBasedContextData) []clients.AllergyInfo {
	var allergies []clients.AllergyInfo
	
	// Extract from snapshot data - placeholder implementation
	if snapshotData != nil && snapshotData.ClinicalData != nil {
		if allergyData, ok := snapshotData.ClinicalData["allergies"]; ok {
			if allergyList, ok := allergyData.([]interface{}); ok {
				for _, allergyInterface := range allergyList {
					if allergy, ok := allergyInterface.(map[string]interface{}); ok {
						allergies = append(allergies, clients.AllergyInfo{
							AllergenCode: s.getStringFromInterface(allergy, "allergen_code"),
							AllergenName: s.getStringFromInterface(allergy, "allergen_name"),
							Severity:     s.getStringFromInterface(allergy, "severity"),
							Reaction:     s.getStringFromInterface(allergy, "reaction"),
						})
					}
				}
			}
		}
	}
	
	return allergies
}

func (s *ClinicalCalculationService) extractConditions(snapshotData *SnapshotBasedContextData) []clients.ConditionInfo {
	var conditions []clients.ConditionInfo
	
	// Extract from snapshot data - placeholder implementation
	if snapshotData != nil && snapshotData.ClinicalData != nil {
		if conditionData, ok := snapshotData.ClinicalData["conditions"]; ok {
			if conditionList, ok := conditionData.([]interface{}); ok {
				for _, conditionInterface := range conditionList {
					if condition, ok := conditionInterface.(map[string]interface{}); ok {
						conditions = append(conditions, clients.ConditionInfo{
							ConditionCode: s.getStringFromInterface(condition, "condition_code"),
							ConditionName: s.getStringFromInterface(condition, "condition_name"),
							Severity:      s.getStringFromInterface(condition, "severity"),
							Status:        s.getStringFromInterface(condition, "status"),
						})
					}
				}
			}
		}
	}
	
	return conditions
}

// Cache management methods

func (s *ClinicalCalculationService) getCachedResult(operation *ClinicalOperation, snapshotData *SnapshotBasedContextData) interface{} {
	cacheKey := s.buildCacheKey(operation, snapshotData)
	if result, ok := s.calculationCache.Load(cacheKey); ok {
		if cachedEntry, ok := result.(*CachedCalculationResult); ok {
			if time.Since(cachedEntry.CachedAt) < s.config.CacheExpirationTime {
				return cachedEntry.Result
			}
			s.calculationCache.Delete(cacheKey)
		}
	}
	return nil
}

func (s *ClinicalCalculationService) cacheResult(operation *ClinicalOperation, snapshotData *SnapshotBasedContextData, result interface{}) {
	cacheKey := s.buildCacheKey(operation, snapshotData)
	cachedEntry := &CachedCalculationResult{
		Result:   result,
		CachedAt: time.Now(),
	}
	s.calculationCache.Store(cacheKey, cachedEntry)
}

func (s *ClinicalCalculationService) buildCacheKey(operation *ClinicalOperation, snapshotData *SnapshotBasedContextData) string {
	// Build a cache key based on operation and snapshot data
	keyData := map[string]interface{}{
		"operation_type": operation.OperationType,
		"parameters":    operation.Parameters,
		"snapshot_id":   snapshotData.SnapshotID,
	}
	
	keyBytes, _ := json.Marshal(keyData)
	return fmt.Sprintf("calc_%x", keyBytes)
}

// CachedCalculationResult represents a cached calculation result
type CachedCalculationResult struct {
	Result   interface{} `json:"result"`
	CachedAt time.Time   `json:"cached_at"`
}

// Process specialized results based on operation types
func (s *ClinicalCalculationService) processSpecializedResults(response *Phase3ClinicalIntelligenceResponse, snapshotData *SnapshotBasedContextData) {
	for _, result := range response.OperationResults {
		if result.Status != "completed" || result.Result == nil {
			continue
		}

		switch result.OperationType {
		case "drug_interactions":
			if drugResult, ok := result.Result.(*clients.DrugInteractionResponse); ok {
				response.DrugInteractionAnalysis = drugResult
			}
		case "dosage_calculation":
			if dosageResult, ok := result.Result.(*clients.DosageCalculationResponse); ok {
				response.DosageCalculations = append(response.DosageCalculations, *dosageResult)
			}
		case "safety_validation":
			if safetyResult, ok := result.Result.(*clients.SafetyValidationResponse); ok {
				response.SafetyValidationResults = append(response.SafetyValidationResults, *safetyResult)
			}
		case "rule_evaluation":
			if ruleResult, ok := result.Result.(*clients.ClinicalRuleEvaluationResponse); ok {
				response.RuleEvaluationResults = append(response.RuleEvaluationResults, *ruleResult)
			}
		}
	}
}

// Calculate overall risk assessment
func (s *ClinicalCalculationService) calculateOverallRiskAssessment(response *Phase3ClinicalIntelligenceResponse) *OverallRiskAssessment {
	assessment := &OverallRiskAssessment{
		OverallRiskScore:      0.0,
		RiskLevel:            "low",
		PrimaryRiskFactors:   []RiskFactor{},
		Recommendations:      []ClinicalRecommendation{},
		RequiredActions:      []RequiredAction{},
		MonitoringRequirements: []MonitoringRequirement{},
	}

	// Analyze drug interactions
	if response.DrugInteractionAnalysis != nil {
		for _, interaction := range response.DrugInteractionAnalysis.Interactions {
			riskScore := s.calculateInteractionRiskScore(interaction)
			assessment.OverallRiskScore += riskScore

			if interaction.Severity == "major" || interaction.Severity == "contraindicated" {
				assessment.PrimaryRiskFactors = append(assessment.PrimaryRiskFactors, RiskFactor{
					RiskID:      fmt.Sprintf("interaction_%s", interaction.InteractionID),
					Category:    "drug_interaction",
					Description: interaction.Effect,
					Severity:    interaction.Severity,
					Impact:      "high",
					Evidence:    interaction.Evidence,
					Score:       riskScore,
				})
			}
		}
	}

	// Analyze safety validation results
	for _, safetyResult := range response.SafetyValidationResults {
		assessment.OverallRiskScore += safetyResult.OverallRiskScore

		for _, alert := range safetyResult.SafetyAlerts {
			if alert.Severity == "high" || alert.Severity == "critical" {
				assessment.PrimaryRiskFactors = append(assessment.PrimaryRiskFactors, RiskFactor{
					RiskID:      alert.AlertID,
					Category:    alert.Category,
					Description: alert.Description,
					Severity:    alert.Severity,
					Impact:      "high",
					Evidence:    alert.Reference,
					Score:       s.calculateAlertRiskScore(alert),
				})
			}
		}
	}

	// Determine overall risk level
	if assessment.OverallRiskScore >= 8.0 {
		assessment.RiskLevel = "critical"
	} else if assessment.OverallRiskScore >= 5.0 {
		assessment.RiskLevel = "high"
	} else if assessment.OverallRiskScore >= 2.0 {
		assessment.RiskLevel = "moderate"
	} else {
		assessment.RiskLevel = "low"
	}

	return assessment
}

// Calculate quality score based on all results
func (s *ClinicalCalculationService) calculateQualityScore(response *Phase3ClinicalIntelligenceResponse) float64 {
	if len(response.OperationResults) == 0 {
		return 0.0
	}

	var totalScore float64
	var validScores int

	for _, result := range response.OperationResults {
		if result.Status == "completed" && result.ConfidenceScore > 0 {
			totalScore += result.ConfidenceScore
			validScores++

			// Penalty for performance misses
			if !result.PerformanceMet {
				totalScore *= 0.9
			}
		}
	}

	if validScores == 0 {
		return 0.0
	}

	return totalScore / float64(validScores)
}

// Utility methods for parameter extraction

func (s *ClinicalCalculationService) getStringParameter(params map[string]interface{}, key, defaultValue string) string {
	if value, ok := params[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func (s *ClinicalCalculationService) getFloatParameter(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key]; ok {
		if f, ok := value.(float64); ok {
			return f
		}
		if i, ok := value.(int); ok {
			return float64(i)
		}
	}
	return defaultValue
}

func (s *ClinicalCalculationService) getIntParameter(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key]; ok {
		if i, ok := value.(int); ok {
			return i
		}
		if f, ok := value.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

func (s *ClinicalCalculationService) getStringSliceParameter(params map[string]interface{}, key string, defaultValue []string) []string {
	if value, ok := params[key]; ok {
		if slice, ok := value.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return defaultValue
}

func (s *ClinicalCalculationService) getStringFromInterface(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (s *ClinicalCalculationService) getBoolFromInterface(data map[string]interface{}, key string) bool {
	if value, ok := data[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func (s *ClinicalCalculationService) calculateConfidenceScore(result interface{}, requiredConfidence float64) float64 {
	// Calculate confidence score based on result type and content
	// This is a placeholder implementation
	return 0.85 // Default confidence score
}

func (s *ClinicalCalculationService) calculateInteractionRiskScore(interaction clients.DrugInteraction) float64 {
	switch interaction.Severity {
	case "contraindicated":
		return 3.0
	case "major":
		return 2.0
	case "moderate":
		return 1.0
	case "minor":
		return 0.5
	default:
		return 0.0
	}
}

func (s *ClinicalCalculationService) calculateAlertRiskScore(alert clients.SafetyAlert) float64 {
	switch alert.Severity {
	case "critical":
		return 3.0
	case "high":
		return 2.0
	case "medium":
		return 1.0
	case "low":
		return 0.5
	default:
		return 0.0
	}
}

func (s *ClinicalCalculationService) calculateFinalMetrics(metrics *ClinicalCalculationMetrics, results []ClinicalOperationResult, totalTime time.Duration) {
	if len(results) == 0 {
		return
	}

	var totalOperationTime time.Duration
	var fastestTime, slowestTime time.Duration
	first := true

	for _, result := range results {
		totalOperationTime += result.ProcessingTime
		
		if first {
			fastestTime = result.ProcessingTime
			slowestTime = result.ProcessingTime
			first = false
		} else {
			if result.ProcessingTime < fastestTime {
				fastestTime = result.ProcessingTime
			}
			if result.ProcessingTime > slowestTime {
				slowestTime = result.ProcessingTime
			}
		}
	}

	metrics.AverageProcessingTime = totalOperationTime / time.Duration(len(results))
	metrics.FastestOperation = fastestTime
	metrics.SlowestOperation = slowestTime

	// Calculate cache hit rate (placeholder)
	metrics.CacheHitRate = 0.15 // 15% cache hit rate as example
}