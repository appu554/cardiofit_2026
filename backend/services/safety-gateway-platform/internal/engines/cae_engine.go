package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// CAEEngine implements the SafetyEngine interface for Clinical Assertion Engine
type CAEEngine struct {
	id           string
	name         string
	capabilities []string
	logger       *logger.Logger
	config       CAEConfig
	pythonPath   string
	scriptPath   string
}

// CAEConfig represents configuration for the CAE engine
type CAEConfig struct {
	PythonPath     string `json:"python_path"`
	ScriptPath     string `json:"script_path"`
	TimeoutMs      int    `json:"timeout_ms"`
	CacheEnabled   bool   `json:"cache_enabled"`
	GraphDBURL     string `json:"graphdb_url"`
	RedisURL       string `json:"redis_url"`
}

// CAERequest represents the request structure for CAE
type CAERequest struct {
	PatientID     string                 `json:"patient_id"`
	MedicationIDs []string               `json:"medication_ids"`
	ConditionIDs  []string               `json:"condition_ids"`
	AllergyIDs    []string               `json:"allergy_ids"`
	ActionType    string                 `json:"action_type"`
	Priority      string                 `json:"priority"`
	Context       map[string]interface{} `json:"context"`
	RequestID     string                 `json:"request_id"`
}

// CAEResponse represents the response structure from CAE
type CAEResponse struct {
	Status         string                 `json:"status"`
	RiskScore      float64                `json:"risk_score"`
	Confidence     float64                `json:"confidence"`
	Violations     []string               `json:"violations"`
	Warnings       []string               `json:"warnings"`
	Explanations   []string               `json:"explanations"`
	ProcessingTime int64                  `json:"processing_time_ms"`
	Metadata       map[string]interface{} `json:"metadata"`
	Error          string                 `json:"error,omitempty"`
}

// NewCAEEngine creates a new CAE engine instance
func NewCAEEngine(logger *logger.Logger, config CAEConfig) *CAEEngine {
	return &CAEEngine{
		id:           "cae_engine",
		name:         "Clinical Assertion Engine",
		capabilities: []string{"drug_interaction", "contraindication", "dosing", "allergy_check", "duplicate_therapy", "clinical_protocol"},
		logger:       logger,
		config:       config,
		pythonPath:   config.PythonPath,
		scriptPath:   config.ScriptPath,
	}
}

// ID returns the engine identifier
func (c *CAEEngine) ID() string {
	return c.id
}

// Name returns the engine name
func (c *CAEEngine) Name() string {
	return c.name
}

// Capabilities returns the engine capabilities
func (c *CAEEngine) Capabilities() []string {
	return c.capabilities
}

// Initialize initializes the CAE engine
func (c *CAEEngine) Initialize(config types.EngineConfig) error {
	c.logger.Info("Initializing CAE engine",
		zap.String("engine_id", c.id),
		zap.Duration("timeout", config.Timeout),
		zap.Strings("capabilities", c.capabilities),
	)

	// Validate Python environment
	if err := c.validatePythonEnvironment(); err != nil {
		return fmt.Errorf("python environment validation failed: %w", err)
	}

	// Validate CAE script
	if err := c.validateCAEScript(); err != nil {
		return fmt.Errorf("CAE script validation failed: %w", err)
	}

	c.logger.Info("CAE engine initialized successfully", zap.String("engine_id", c.id))
	return nil
}

// HealthCheck performs a health check on the CAE engine
func (c *CAEEngine) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simple health check by calling CAE with minimal request
	healthRequest := CAERequest{
		PatientID:  "health_check",
		ActionType: "health_check",
		RequestID:  "health_check_" + fmt.Sprintf("%d", time.Now().Unix()),
	}

	_, err := c.callCAEScript(ctx, healthRequest)
	if err != nil {
		return fmt.Errorf("CAE health check failed: %w", err)
	}

	return nil
}

// Evaluate performs safety evaluation using the CAE engine
func (c *CAEEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	startTime := time.Now()

	c.logger.Debug("CAE engine evaluation started",
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("action_type", req.ActionType),
	)

	// Convert to CAE request format
	caeRequest := c.convertToCAERequest(req, clinicalContext)

	// Call CAE script
	caeResponse, err := c.callCAEScript(ctx, caeRequest)
	if err != nil {
		c.logger.Error("CAE script execution failed",
			zap.String("request_id", req.RequestID),
			zap.Error(err),
		)
		return c.createErrorResult(err, time.Since(startTime)), nil
	}

	// Convert CAE response to engine result
	result := c.convertToEngineResult(caeResponse, time.Since(startTime))

	c.logger.Debug("CAE engine evaluation completed",
		zap.String("request_id", req.RequestID),
		zap.String("status", string(result.Status)),
		zap.Float64("risk_score", result.RiskScore),
		zap.Int64("duration_ms", result.Duration.Milliseconds()),
	)

	return result, nil
}

// Shutdown shuts down the CAE engine
func (c *CAEEngine) Shutdown() error {
	c.logger.Info("Shutting down CAE engine", zap.String("engine_id", c.id))
	// CAE engine is stateless, no cleanup needed
	return nil
}

// EvaluateWithSnapshot performs safety evaluation using snapshot data
func (c *CAEEngine) EvaluateWithSnapshot(ctx context.Context, req *types.SafetyRequest, snapshot *types.ClinicalSnapshot) (*types.EngineResult, error) {
	startTime := time.Now()

	c.logger.Debug("CAE engine snapshot-based evaluation started",
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("action_type", req.ActionType),
	)

	// Validate snapshot data compatibility
	if err := c.validateSnapshotCompatibility(snapshot); err != nil {
		c.logger.Error("Snapshot compatibility validation failed",
			zap.String("request_id", req.RequestID),
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.Error(err),
		)
		return c.createErrorResult(err, time.Since(startTime)), nil
	}

	// Convert to CAE request format with snapshot data
	caeRequest := c.convertToCAERequestWithSnapshot(req, snapshot)

	// Call CAE script with snapshot context
	caeResponse, err := c.callCAEScript(ctx, caeRequest)
	if err != nil {
		c.logger.Error("CAE script execution failed with snapshot",
			zap.String("request_id", req.RequestID),
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.Error(err),
		)
		return c.createErrorResult(err, time.Since(startTime)), nil
	}

	// Convert CAE response to engine result with snapshot metadata
	result := c.convertToEngineResultWithSnapshot(caeResponse, snapshot, time.Since(startTime))

	c.logger.Debug("CAE engine snapshot-based evaluation completed",
		zap.String("request_id", req.RequestID),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("status", string(result.Status)),
		zap.Float64("risk_score", result.RiskScore),
		zap.Int64("duration_ms", result.Duration.Milliseconds()),
	)

	return result, nil
}

// IsSnapshotCompatible returns true if the engine supports snapshot-based evaluation
func (c *CAEEngine) IsSnapshotCompatible() bool {
	return true // CAE engine supports snapshot-based evaluation
}

// GetSnapshotRequirements returns the required snapshot fields for CAE engine
func (c *CAEEngine) GetSnapshotRequirements() []string {
	return []string{
		"demographics",
		"active_medications", 
		"allergies",
		"conditions",
		"recent_vitals",
	}
}

// validateSnapshotCompatibility validates that snapshot contains required data
func (c *CAEEngine) validateSnapshotCompatibility(snapshot *types.ClinicalSnapshot) error {
	if snapshot.Data == nil {
		return fmt.Errorf("snapshot contains no clinical data")
	}

	requirements := c.GetSnapshotRequirements()
	var missing []string

	// Check required fields
	if snapshot.Data.Demographics == nil && contains(requirements, "demographics") {
		missing = append(missing, "demographics")
	}
	if len(snapshot.Data.ActiveMedications) == 0 && contains(requirements, "active_medications") {
		missing = append(missing, "active_medications")
	}
	if len(snapshot.Data.Allergies) == 0 && contains(requirements, "allergies") {
		missing = append(missing, "allergies")
	}
	if len(snapshot.Data.Conditions) == 0 && contains(requirements, "conditions") {
		missing = append(missing, "conditions")
	}

	if len(missing) > 0 && snapshot.DataCompleteness < 0.8 {
		return fmt.Errorf("snapshot missing critical fields for CAE evaluation: %v (data completeness: %.1f%%)", 
			missing, snapshot.DataCompleteness*100)
	}

	return nil
}

// convertToCAERequestWithSnapshot converts SafetyRequest with snapshot to CAE format
func (c *CAEEngine) convertToCAERequestWithSnapshot(req *types.SafetyRequest, snapshot *types.ClinicalSnapshot) CAERequest {
	// Build context map from snapshot data
	context := make(map[string]interface{})
	
	// Add clinical context from snapshot
	if snapshot.Data != nil {
		context["demographics"] = snapshot.Data.Demographics
		context["active_medications"] = snapshot.Data.ActiveMedications
		context["allergies"] = snapshot.Data.Allergies
		context["conditions"] = snapshot.Data.Conditions
		context["recent_vitals"] = snapshot.Data.RecentVitals
		context["lab_results"] = snapshot.Data.LabResults
		context["context_version"] = snapshot.Data.ContextVersion
		
		// Add snapshot metadata
		context["snapshot_id"] = snapshot.SnapshotID
		context["snapshot_checksum"] = snapshot.Checksum
		context["data_completeness"] = snapshot.DataCompleteness
		context["snapshot_created_at"] = snapshot.CreatedAt.Format(time.RFC3339)
		context["processing_mode"] = "snapshot_based"
	}

	// Add request context
	for k, v := range req.Context {
		context[k] = v
	}

	return CAERequest{
		PatientID:     req.PatientID,
		MedicationIDs: req.MedicationIDs,
		ConditionIDs:  req.ConditionIDs,
		AllergyIDs:    req.AllergyIDs,
		ActionType:    req.ActionType,
		Priority:      req.Priority,
		Context:       context,
		RequestID:     req.RequestID,
	}
}

// convertToEngineResultWithSnapshot converts CAE response to EngineResult with snapshot metadata
func (c *CAEEngine) convertToEngineResultWithSnapshot(response *CAEResponse, snapshot *types.ClinicalSnapshot, duration time.Duration) *types.EngineResult {
	// Use existing conversion logic
	result := c.convertToEngineResult(response, duration)

	// Add snapshot-specific metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	
	result.Metadata["snapshot_id"] = snapshot.SnapshotID
	result.Metadata["snapshot_checksum"] = snapshot.Checksum
	result.Metadata["data_completeness"] = snapshot.DataCompleteness
	result.Metadata["snapshot_created_at"] = snapshot.CreatedAt.Format(time.RFC3339)
	result.Metadata["processing_mode"] = "snapshot_based"
	result.Metadata["snapshot_version"] = snapshot.Version

	return result
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// validatePythonEnvironment validates the Python environment
func (c *CAEEngine) validatePythonEnvironment() error {
	cmd := exec.Command(c.pythonPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("python not found at %s: %w", c.pythonPath, err)
	}

	version := strings.TrimSpace(string(output))
	c.logger.Debug("Python environment validated", zap.String("version", version))
	return nil
}

// validateCAEScript validates the CAE script exists and is executable
func (c *CAEEngine) validateCAEScript() error {
	cmd := exec.Command(c.pythonPath, c.scriptPath, "--validate")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("CAE script validation failed: %w", err)
	}

	c.logger.Debug("CAE script validated", zap.String("script_path", c.scriptPath))
	return nil
}

// convertToCAERequest converts SafetyRequest to CAE format
func (c *CAEEngine) convertToCAERequest(req *types.SafetyRequest, clinicalContext *types.ClinicalContext) CAERequest {
	// Build context map
	context := make(map[string]interface{})
	
	// Add clinical context if available
	if clinicalContext != nil {
		context["demographics"] = clinicalContext.Demographics
		context["active_medications"] = clinicalContext.ActiveMedications
		context["allergies"] = clinicalContext.Allergies
		context["conditions"] = clinicalContext.Conditions
		context["recent_vitals"] = clinicalContext.RecentVitals
		context["lab_results"] = clinicalContext.LabResults
		context["context_version"] = clinicalContext.ContextVersion
	}

	// Add request context
	for k, v := range req.Context {
		context[k] = v
	}

	return CAERequest{
		PatientID:     req.PatientID,
		MedicationIDs: req.MedicationIDs,
		ConditionIDs:  req.ConditionIDs,
		AllergyIDs:    req.AllergyIDs,
		ActionType:    req.ActionType,
		Priority:      req.Priority,
		Context:       context,
		RequestID:     req.RequestID,
	}
}

// callCAEScript executes the CAE Python script
func (c *CAEEngine) callCAEScript(ctx context.Context, request CAERequest) (*CAEResponse, error) {
	// Serialize request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize CAE request: %w", err)
	}

	// Create command with timeout
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "--json-input")
	cmd.Stdin = strings.NewReader(string(requestJSON))

	// Execute command
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("CAE script execution failed: %w", err)
	}

	// Parse response
	var response CAEResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse CAE response: %w", err)
	}

	// Check for CAE-level errors
	if response.Error != "" {
		return nil, fmt.Errorf("CAE returned error: %s", response.Error)
	}

	return &response, nil
}

// convertToEngineResult converts CAE response to EngineResult
func (c *CAEEngine) convertToEngineResult(response *CAEResponse, duration time.Duration) *types.EngineResult {
	// Convert status
	var status types.SafetyStatus
	switch strings.ToLower(response.Status) {
	case "safe":
		status = types.SafetyStatusSafe
	case "unsafe":
		status = types.SafetyStatusUnsafe
	case "warning":
		status = types.SafetyStatusWarning
	default:
		status = types.SafetyStatusManualReview
	}

	return &types.EngineResult{
		EngineID:   c.id,
		EngineName: c.name,
		Status:     status,
		RiskScore:  response.RiskScore,
		Violations: response.Violations,
		Warnings:   response.Warnings,
		Confidence: response.Confidence,
		Duration:   duration,
		Tier:       types.TierVetoCritical, // CAE is Tier 1 (critical)
		Metadata:   response.Metadata,
	}
}

// createErrorResult creates an error result for failed CAE execution
func (c *CAEEngine) createErrorResult(err error, duration time.Duration) *types.EngineResult {
	return &types.EngineResult{
		EngineID:   c.id,
		EngineName: c.name,
		Status:     types.SafetyStatusUnsafe, // Fail closed for Tier 1 engine
		RiskScore:  1.0,                      // Maximum risk for errors
		Violations: []string{fmt.Sprintf("CAE engine execution failed: %s", err.Error())},
		Confidence: 0.0,
		Duration:   duration,
		Tier:       types.TierVetoCritical,
		Error:      err.Error(),
	}
}
