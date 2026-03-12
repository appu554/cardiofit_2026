package engines

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
	pb "safety-gateway-platform/proto"
)

// GRPCCAEEngine implements the SafetyEngine interface using gRPC to connect to CAE service
type GRPCCAEEngine struct {
	id            string
	name          string
	capabilities  []string
	logger        *logger.Logger
	client        pb.ClinicalReasoningServiceClient
	conn          *grpc.ClientConn
	caeAddress    string
	timeout       time.Duration

}

// NewGRPCCAEEngine creates a new gRPC-based CAE engine
func NewGRPCCAEEngine(logger *logger.Logger, caeAddress string) *GRPCCAEEngine {
	return &GRPCCAEEngine{
		id:            "grpc_cae_engine",
		name:          "gRPC Clinical Assertion Engine",
		capabilities:  []string{"drug_interaction", "contraindication", "dosing", "allergy_check", "duplicate_therapy", "clinical_protocol"},
		logger:        logger,
		caeAddress:    caeAddress,
		timeout:       5 * time.Second, // 5 second timeout for CAE calls

	}
}

// ID returns the engine ID
func (e *GRPCCAEEngine) ID() string {
	return e.id
}

// Name returns the engine name
func (e *GRPCCAEEngine) Name() string {
	return e.name
}

// Capabilities returns the engine capabilities
func (e *GRPCCAEEngine) Capabilities() []string {
	return e.capabilities
}

// Initialize initializes the gRPC connection to CAE service
func (e *GRPCCAEEngine) Initialize(config types.EngineConfig) error {
	e.logger.Info("Initializing gRPC CAE engine", 
		zap.String("engine_id", e.id),
		zap.String("cae_address", e.caeAddress))

	// Create gRPC connection
	conn, err := grpc.Dial(e.caeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to CAE service at %s: %w", e.caeAddress, err)
	}

	e.conn = conn
	e.client = pb.NewClinicalReasoningServiceClient(conn)

	// Test the connection with a health check
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	healthReq := &pb.HealthCheckRequest{
		Service: "clinical-reasoning",
	}

	healthResp, err := e.client.HealthCheck(ctx, healthReq)
	if err != nil {
		e.conn.Close()
		return fmt.Errorf("CAE service health check failed: %w", err)
	}

	e.logger.Info("gRPC CAE engine initialized successfully", 
		zap.String("engine_id", e.id),
		zap.String("health_status", healthResp.Status.String()))

	return nil
}

// Shutdown closes the gRPC connection
func (e *GRPCCAEEngine) Shutdown() error {
	if e.conn != nil {
		err := e.conn.Close()
		e.logger.Info("gRPC CAE engine shutdown", zap.String("engine_id", e.id))
		return err
	}
	return nil
}

// HealthCheck performs a health check on the CAE service
func (e *GRPCCAEEngine) HealthCheck() error {
	if e.client == nil {
		return fmt.Errorf("CAE client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	healthReq := &pb.HealthCheckRequest{
		Service: "clinical-reasoning",
	}

	_, err := e.client.HealthCheck(ctx, healthReq)
	return err
}

// Evaluate performs safety evaluation using the CAE service
func (e *GRPCCAEEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	startTime := time.Now()

	e.logger.Debug("Starting CAE evaluation", 
		zap.String("engine_id", e.id),
		zap.String("patient_id", req.PatientID),
		zap.String("request_id", req.RequestID))

	// Create context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Convert clinical context to protobuf struct
	contextStruct, err := e.convertClinicalContextToStruct(clinicalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to convert clinical context: %w", err)
	}

	// Create CAE request
	caeReq := &pb.ClinicalAssertionRequest{
		PatientId:      req.PatientID,
		CorrelationId:  req.RequestID,
		MedicationIds:  req.MedicationIDs,  // Medication names are passed directly
		ConditionIds:   req.ConditionIDs,
		PatientContext: contextStruct,
		Priority:       e.convertPriority(req.Priority),
		ReasonerTypes:  []string{"interaction", "dosing", "contraindication", "duplicate_therapy"},
	}

	// Call CAE service
	caeResp, err := e.client.GenerateAssertions(evalCtx, caeReq)
	if err != nil {
		// CAE service call failed - return UNSAFE status
		return &types.EngineResult{
			EngineID:   e.id,
			EngineName: e.name,
			Status:     types.SafetyStatusUnsafe,
			RiskScore:  1.0,
			Violations: []string{fmt.Sprintf("CAE service call failed: %v", err)},
			Warnings:   []string{},
			Confidence: 0.0,
			Duration:   time.Since(startTime),
			Tier:       types.TierVetoCritical,
			Error:      fmt.Sprintf("CAE service call failed: %v", err),
			Metadata: map[string]interface{}{
				"error_type": "grpc_call_failed",
				"error_details": err.Error(),
			},
		}, nil
	}

	// Convert CAE response to engine result
	result := e.convertCAEResponseToEngineResult(caeResp, startTime)

	e.logger.Debug("CAE evaluation completed", 
		zap.String("engine_id", e.id),
		zap.String("patient_id", req.PatientID),
		zap.String("request_id", req.RequestID),
		zap.Int("assertions_count", len(caeResp.Assertions)),
		zap.Float64("processing_time_ms", float64(time.Since(startTime).Nanoseconds())/1e6))

	return result, nil
}

// Helper methods

// convertClinicalContextToStruct converts ClinicalContext to protobuf Struct
func (e *GRPCCAEEngine) convertClinicalContextToStruct(context *types.ClinicalContext) (*structpb.Struct, error) {
	if context == nil {
		return &structpb.Struct{}, nil
	}

	// Convert complex types to basic types for protobuf compatibility
	contextMap := map[string]interface{}{
		"patient_id":         context.PatientID,
		"context_version":    context.ContextVersion,
		"assembly_time":      context.AssemblyTime.Format(time.RFC3339),
		"metadata":           context.Metadata,
	}

	// Convert DataSources []string to []interface{} for protobuf compatibility
	if len(context.DataSources) > 0 {
		dataSources := make([]interface{}, len(context.DataSources))
		for i, source := range context.DataSources {
			dataSources[i] = source
		}
		contextMap["data_sources"] = dataSources
	}

	// Convert Demographics to basic types
	if context.Demographics != nil {
		contextMap["demographics"] = map[string]interface{}{
			"age":              context.Demographics.Age,
			"gender":           context.Demographics.Gender,
			"weight_kg":        context.Demographics.Weight,
			"height_cm":        context.Demographics.Height,
			"bmi":              context.Demographics.BMI,
			"pregnancy_status": context.Demographics.PregnancyStatus,
		}
	}

	// Convert ActiveMedications to basic types
	if len(context.ActiveMedications) > 0 {
		medications := make([]interface{}, len(context.ActiveMedications))
		for i, med := range context.ActiveMedications {
			medications[i] = map[string]interface{}{
				"id":           med.ID,
				"name":         med.Name,
				"generic_name": med.GenericName,
				"dosage":       med.Dosage,
				"route":        med.Route,
				"frequency":    med.Frequency,
				"start_date":   med.StartDate.Format(time.RFC3339),
				"end_date":     med.EndDate.Format(time.RFC3339),
				"status":       med.Status,
				"prescriber":   med.Prescriber,
			}
		}
		contextMap["active_medications"] = medications
	}

	// Convert Allergies to basic types
	if len(context.Allergies) > 0 {
		allergies := make([]interface{}, len(context.Allergies))
		for i, allergy := range context.Allergies {
			allergies[i] = map[string]interface{}{
				"id":         allergy.ID,
				"allergen":   allergy.Allergen,
				"reaction":   allergy.Reaction,
				"severity":   allergy.Severity,
				"onset_date": allergy.OnsetDate.Format(time.RFC3339),
				"status":     allergy.Status,
			}
		}
		contextMap["allergies"] = allergies
	}

	// Convert Conditions to basic types
	if len(context.Conditions) > 0 {
		conditions := make([]interface{}, len(context.Conditions))
		for i, condition := range context.Conditions {
			conditions[i] = map[string]interface{}{
				"id":           condition.ID,
				"code":         condition.Code,
				"display":      condition.Display,
				"severity":     condition.Severity,
				"onset_date":   condition.OnsetDate.Format(time.RFC3339),
				"status":       condition.Status,
				"diagnosed_by": condition.DiagnosedBy,
			}
		}
		contextMap["conditions"] = conditions
	}

	// Convert other complex fields to basic types if they exist
	if len(context.RecentVitals) > 0 {
		contextMap["recent_vitals"] = context.RecentVitals // Assuming this is already basic types
	}

	if len(context.LabResults) > 0 {
		contextMap["lab_results"] = context.LabResults // Assuming this is already basic types
	}

	if len(context.RecentEncounters) > 0 {
		contextMap["recent_encounters"] = context.RecentEncounters // Assuming this is already basic types
	}

	return structpb.NewStruct(contextMap)
}

// convertPriority converts string priority to protobuf enum
func (e *GRPCCAEEngine) convertPriority(priority string) pb.AssertionPriority {
	switch priority {
	case "low":
		return pb.AssertionPriority_PRIORITY_LOW
	case "normal":
		return pb.AssertionPriority_PRIORITY_NORMAL
	case "high":
		return pb.AssertionPriority_PRIORITY_HIGH
	case "critical":
		return pb.AssertionPriority_PRIORITY_CRITICAL
	default:
		return pb.AssertionPriority_PRIORITY_NORMAL
	}
}

// convertCAEResponseToEngineResult converts CAE response to EngineResult
func (e *GRPCCAEEngine) convertCAEResponseToEngineResult(resp *pb.ClinicalAssertionResponse, startTime time.Time) *types.EngineResult {
	result := &types.EngineResult{
		EngineID:   e.id,
		EngineName: e.name,
		Status:     types.SafetyStatusSafe,
		RiskScore:  0.0,
		Violations: []string{},
		Warnings:   []string{},
		Confidence: 1.0,
		Duration:   time.Since(startTime),
		Tier:       types.TierVetoCritical,
		Metadata: map[string]interface{}{
			"cae_request_id":     resp.RequestId,
			"cae_correlation_id": resp.CorrelationId,
			"total_assertions":   len(resp.Assertions),
			"cae_metadata":       resp.Metadata,
		},
	}

	// Check for CAE internal failures and extract specific errors
	// If we get 0 assertions, this likely indicates reasoner failures
	if len(resp.Assertions) == 0 {
		e.logger.Error("CAE returned 0 assertions - likely reasoner failures detected",
			zap.String("request_id", resp.RequestId),
			zap.String("correlation_id", resp.CorrelationId))

		// Extract specific errors from CAE metadata warnings
		var specificErrors []string
		if resp.Metadata != nil && len(resp.Metadata.Warnings) > 0 {
			specificErrors = resp.Metadata.Warnings
			e.logger.Error("CAE reasoner errors detected",
				zap.Strings("errors", specificErrors))
		} else {
			specificErrors = []string{
				"CAE reasoners failed to generate clinical assertions",
				"Unable to perform safety validation - manual review required",
			}
		}

		// Treat 0 assertions as UNSAFE condition for patient safety
		result.Status = types.SafetyStatusUnsafe
		result.RiskScore = 1.0
		result.Confidence = 0.0
		result.Violations = specificErrors
		result.Error = "CAE internal reasoner failures detected"
		result.Metadata["error_type"] = "cae_reasoner_failures"
		result.Metadata["cae_specific_errors"] = specificErrors
		result.Metadata["cae_warning"] = "Reasoner failures - requires immediate attention"

		return result
	}

	// Check for suspicious conditions that might indicate CAE reasoner failures
	// If we get 0 assertions, this might indicate internal CAE failures
	if len(resp.Assertions) == 0 {
		e.logger.Warn("CAE returned 0 assertions - potential reasoner failures",
			zap.String("request_id", resp.RequestId),
			zap.String("correlation_id", resp.CorrelationId))

		// Add warning about potential CAE issues
		result.Warnings = append(result.Warnings, "CAE returned no clinical assertions - verify reasoner functionality")
		result.Status = types.SafetyStatusWarning
		result.Confidence = 0.5 // Reduced confidence due to lack of assertions
		result.Metadata["cae_warning"] = "No assertions generated - potential reasoner issues"
	}

	// Convert assertions to violations and warnings
	var totalRiskScore float64
	for _, assertion := range resp.Assertions {
		severity := e.convertSeverity(assertion.Severity)

		// Add to violations or warnings based on severity
		if severity == pb.AssertionSeverity_SEVERITY_CRITICAL {
			result.Violations = append(result.Violations, assertion.Message)
			result.Status = types.SafetyStatusUnsafe
			totalRiskScore += float64(assertion.ConfidenceScore) * 0.8 // High weight for critical
		} else if severity == pb.AssertionSeverity_SEVERITY_MODERATE {
			result.Warnings = append(result.Warnings, assertion.Message)
			if result.Status == types.SafetyStatusSafe {
				result.Status = types.SafetyStatusWarning
			}
			totalRiskScore += float64(assertion.ConfidenceScore) * 0.3 // Lower weight for warnings
		}

		// Store detailed assertion info in metadata
		assertionKey := fmt.Sprintf("assertion_%s", assertion.AssertionId)
		result.Metadata[assertionKey] = map[string]interface{}{
			"type":                assertion.ReasonerType,
			"severity":            severity,
			"description":         assertion.Description,
			"confidence":          assertion.ConfidenceScore,
			"recommendations":     assertion.Recommendations,
			"affected_medications": assertion.AffectedMedications,
			"affected_conditions":  assertion.AffectedConditions,
			"evidence":            assertion.Evidence,
		}
	}

	// Calculate overall risk score and confidence
	if len(resp.Assertions) > 0 {
		result.RiskScore = totalRiskScore / float64(len(resp.Assertions))
		result.Confidence = result.RiskScore
	}

	return result
}

// convertSeverity converts protobuf severity to protobuf enum value
func (e *GRPCCAEEngine) convertSeverity(severity pb.AssertionSeverity) pb.AssertionSeverity {
	return severity
}

// EvaluateWithSnapshot performs safety evaluation using snapshot data via gRPC
func (e *GRPCCAEEngine) EvaluateWithSnapshot(ctx context.Context, req *types.SafetyRequest, snapshot *types.ClinicalSnapshot) (*types.EngineResult, error) {
	startTime := time.Now()

	e.logger.Debug("Starting gRPC CAE snapshot-based evaluation", 
		zap.String("engine_id", e.id),
		zap.String("patient_id", req.PatientID),
		zap.String("request_id", req.RequestID),
		zap.String("snapshot_id", snapshot.SnapshotID))

	// Validate snapshot data compatibility
	if err := e.validateSnapshotCompatibility(snapshot); err != nil {
		e.logger.Error("Snapshot compatibility validation failed",
			zap.String("request_id", req.RequestID),
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.Error(err),
		)
		return e.createSnapshotErrorResult(err, snapshot, time.Since(startTime)), nil
	}

	// Create context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Convert snapshot clinical context to protobuf struct
	contextStruct, err := e.convertSnapshotToStruct(snapshot)
	if err != nil {
		return e.createSnapshotErrorResult(fmt.Errorf("failed to convert snapshot context: %w", err), snapshot, time.Since(startTime)), nil
	}

	// Create CAE request with snapshot context
	caeReq := &pb.ClinicalAssertionRequest{
		PatientId:      req.PatientID,
		CorrelationId:  req.RequestID,
		MedicationIds:  req.MedicationIDs,
		ConditionIds:   req.ConditionIDs,
		PatientContext: contextStruct,
		Priority:       e.convertPriority(req.Priority),
		ReasonerTypes:  []string{"interaction", "dosing", "contraindication", "duplicate_therapy"},
	}

	// Call CAE service with snapshot context
	caeResp, err := e.client.GenerateAssertions(evalCtx, caeReq)
	if err != nil {
		// CAE service call failed - return UNSAFE status with snapshot metadata
		return e.createSnapshotErrorResult(fmt.Errorf("CAE service call failed: %v", err), snapshot, time.Since(startTime)), nil
	}

	// Convert CAE response to engine result with snapshot metadata
	result := e.convertCAEResponseToEngineResultWithSnapshot(caeResp, snapshot, startTime)

	e.logger.Debug("gRPC CAE snapshot-based evaluation completed", 
		zap.String("engine_id", e.id),
		zap.String("patient_id", req.PatientID),
		zap.String("request_id", req.RequestID),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.Int("assertions_count", len(caeResp.Assertions)),
		zap.Float64("processing_time_ms", float64(time.Since(startTime).Nanoseconds())/1e6))

	return result, nil
}

// IsSnapshotCompatible returns true if the gRPC engine supports snapshot-based evaluation
func (e *GRPCCAEEngine) IsSnapshotCompatible() bool {
	return true // gRPC CAE engine supports snapshot-based evaluation
}

// GetSnapshotRequirements returns the required snapshot fields for gRPC CAE engine
func (e *GRPCCAEEngine) GetSnapshotRequirements() []string {
	return []string{
		"demographics",
		"active_medications",
		"allergies", 
		"conditions",
		"recent_vitals",
		"lab_results",
	}
}

// validateSnapshotCompatibility validates that snapshot contains required data for gRPC CAE
func (e *GRPCCAEEngine) validateSnapshotCompatibility(snapshot *types.ClinicalSnapshot) error {
	if snapshot.Data == nil {
		return fmt.Errorf("snapshot contains no clinical data")
	}

	requirements := e.GetSnapshotRequirements()
	var missing []string

	// Check required fields
	if snapshot.Data.Demographics == nil && e.containsRequirement(requirements, "demographics") {
		missing = append(missing, "demographics")
	}
	if len(snapshot.Data.ActiveMedications) == 0 && e.containsRequirement(requirements, "active_medications") {
		missing = append(missing, "active_medications")
	}
	if len(snapshot.Data.Allergies) == 0 && e.containsRequirement(requirements, "allergies") {
		missing = append(missing, "allergies")
	}
	if len(snapshot.Data.Conditions) == 0 && e.containsRequirement(requirements, "conditions") {
		missing = append(missing, "conditions")
	}

	if len(missing) > 0 && snapshot.DataCompleteness < 0.75 {
		return fmt.Errorf("snapshot missing critical fields for gRPC CAE evaluation: %v (data completeness: %.1f%%)", 
			missing, snapshot.DataCompleteness*100)
	}

	return nil
}

// convertSnapshotToStruct converts ClinicalSnapshot to protobuf Struct for gRPC
func (e *GRPCCAEEngine) convertSnapshotToStruct(snapshot *types.ClinicalSnapshot) (*structpb.Struct, error) {
	if snapshot.Data == nil {
		return &structpb.Struct{}, nil
	}

	// Use existing conversion logic but with snapshot data
	contextStruct, err := e.convertClinicalContextToStruct(snapshot.Data)
	if err != nil {
		return nil, err
	}

	// Enhance with snapshot metadata
	contextMap := contextStruct.AsMap()
	contextMap["snapshot_id"] = snapshot.SnapshotID
	contextMap["snapshot_checksum"] = snapshot.Checksum
	contextMap["data_completeness"] = snapshot.DataCompleteness
	contextMap["snapshot_created_at"] = snapshot.CreatedAt.Format(time.RFC3339)
	contextMap["snapshot_version"] = snapshot.Version
	contextMap["processing_mode"] = "snapshot_based"

	return structpb.NewStruct(contextMap)
}

// convertCAEResponseToEngineResultWithSnapshot converts CAE response to EngineResult with snapshot metadata
func (e *GRPCCAEEngine) convertCAEResponseToEngineResultWithSnapshot(resp *pb.ClinicalAssertionResponse, snapshot *types.ClinicalSnapshot, startTime time.Time) *types.EngineResult {
	// Use existing conversion logic
	result := e.convertCAEResponseToEngineResult(resp, startTime)

	// Add snapshot-specific metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}

	result.Metadata["snapshot_id"] = snapshot.SnapshotID
	result.Metadata["snapshot_checksum"] = snapshot.Checksum
	result.Metadata["data_completeness"] = snapshot.DataCompleteness
	result.Metadata["snapshot_created_at"] = snapshot.CreatedAt.Format(time.RFC3339)
	result.Metadata["snapshot_version"] = snapshot.Version
	result.Metadata["processing_mode"] = "snapshot_based"
	result.Metadata["snapshot_expires_at"] = snapshot.ExpiresAt.Format(time.RFC3339)

	return result
}

// createSnapshotErrorResult creates an error result for failed snapshot-based execution
func (e *GRPCCAEEngine) createSnapshotErrorResult(err error, snapshot *types.ClinicalSnapshot, duration time.Duration) *types.EngineResult {
	return &types.EngineResult{
		EngineID:   e.id,
		EngineName: e.name,
		Status:     types.SafetyStatusUnsafe,
		RiskScore:  1.0,
		Violations: []string{fmt.Sprintf("gRPC CAE snapshot evaluation failed: %v", err)},
		Warnings:   []string{},
		Confidence: 0.0,
		Duration:   duration,
		Tier:       types.TierVetoCritical,
		Error:      fmt.Sprintf("gRPC CAE snapshot evaluation failed: %v", err),
		Metadata: map[string]interface{}{
			"error_type":              "grpc_snapshot_call_failed",
			"error_details":           err.Error(),
			"processing_mode":         "snapshot_based",
			"snapshot_id":             snapshot.SnapshotID,
			"snapshot_checksum":       snapshot.Checksum,
			"snapshot_data_completeness": snapshot.DataCompleteness,
		},
	}
}

// containsRequirement checks if requirements slice contains a specific requirement
func (e *GRPCCAEEngine) containsRequirement(requirements []string, requirement string) bool {
	for _, req := range requirements {
		if req == requirement {
			return true
		}
	}
	return false
}