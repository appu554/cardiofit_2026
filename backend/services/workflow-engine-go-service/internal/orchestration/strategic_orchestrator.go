package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/domain"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/repositories"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/pkg/clients"
)

// StrategicOrchestrator implements the Advanced Calculate > Validate > Commit pattern with UI interaction
type StrategicOrchestrator struct {
	flow2GoClient       clients.Flow2GoClient
	safetyGatewayClient clients.SafetyGatewayClient
	medicationClient    clients.MedicationServiceClient
	snapshotRepo        repositories.SnapshotRepository
	workflowRepo        repositories.WorkflowRepository

	// Advanced pattern components
	uiCoordinator         *UICoordinator
	overrideManager       *OverrideManager
	idempotencyManager    *IdempotencyManager
	recipeResolver        *RecipeResolver
	safetyDecisionMatrix  *SafetyDecisionMatrix
	reworkManager         *ReworkManager
	commitOrchestrator    *CommitOrchestrator
	batchProcessor        *BatchProcessor
	rollbackManager       *RollbackManager
	redisClient           *redis.Client

	logger              *zap.Logger
}

// NewStrategicOrchestrator creates a new advanced orchestrator instance
func NewStrategicOrchestrator(
	flow2GoClient clients.Flow2GoClient,
	safetyGatewayClient clients.SafetyGatewayClient,
	medicationClient clients.MedicationServiceClient,
	snapshotRepo repositories.SnapshotRepository,
	workflowRepo repositories.WorkflowRepository,
	recipeRepo RecipeRepository,
	redisClient *redis.Client,
	logger *zap.Logger,
) *StrategicOrchestrator {
	// Initialize advanced pattern components
	uiCoordinator := NewUICoordinator(redisClient, logger)
	overrideManager := NewOverrideManager(redisClient, logger)
	idempotencyManager := NewIdempotencyManager(redisClient, logger)

	// Initialize recipe resolver
	recipeResolver := NewRecipeResolver(recipeRepo, logger)

	// Initialize safety decision matrix with default configuration
	safetyDecisionMatrix := NewSafetyDecisionMatrix(nil, logger)

	// Initialize rework manager with max 2 attempts and 500ms backoff
	reworkManager := NewReworkManager(2, 500*time.Millisecond, flow2GoClient, logger)

	// Initialize commit orchestrator with all required dependencies
	commitOrchestrator := NewCommitOrchestrator(
		medicationClient,
		redisClient,
		logger,
	)

	// Initialize batch processor for grouped commits
	batchProcessor := NewBatchProcessor(redisClient, logger)

	// Initialize rollback manager for 5-minute rollback capability
	rollbackManager := NewRollbackManager(redisClient, logger)

	return &StrategicOrchestrator{
		flow2GoClient:       flow2GoClient,
		safetyGatewayClient: safetyGatewayClient,
		medicationClient:    medicationClient,
		snapshotRepo:        snapshotRepo,
		workflowRepo:        workflowRepo,

		// Advanced components
		uiCoordinator:         uiCoordinator,
		overrideManager:       overrideManager,
		idempotencyManager:    idempotencyManager,
		recipeResolver:        recipeResolver,
		safetyDecisionMatrix:  safetyDecisionMatrix,
		reworkManager:         reworkManager,
		commitOrchestrator:    commitOrchestrator,
		batchProcessor:        batchProcessor,
		rollbackManager:       rollbackManager,
		redisClient:           redisClient,

		logger:              logger,
	}
}

// OrchestrationRequest represents the input for advanced medication workflow orchestration
type OrchestrationRequest struct {
	PatientID         string                 `json:"patient_id"`
	CorrelationID     string                 `json:"correlation_id"`
	MedicationRequest map[string]interface{} `json:"medication_request"`
	ClinicalIntent    map[string]interface{} `json:"clinical_intent"`
	ProviderContext   map[string]interface{} `json:"provider_context"`
	ExecutionMode     string                 `json:"execution_mode,omitempty"`
	ValidationLevel   string                 `json:"validation_level,omitempty"`
	CommitMode        string                 `json:"commit_mode,omitempty"`

	// Advanced pattern fields
	UIInteractionMode string                 `json:"ui_interaction_mode,omitempty"` // none, notification, interactive, review_required
	OverrideAuthority string                 `json:"override_authority,omitempty"`  // clinical_judgment, peer_review, supervisory, emergency
	IdempotencyToken  string                 `json:"idempotency_token,omitempty"`   // For retry safety
	SessionID         string                 `json:"session_id,omitempty"`          // UI session tracking
}

// OrchestrationResponse represents the advanced workflow execution result
type OrchestrationResponse struct {
	WorkflowInstanceID       string                   `json:"workflow_instance_id"`
	SnapshotID              string                   `json:"snapshot_id"`
	ProposalSetID           string                   `json:"proposal_set_id"`
	ValidationID            string                   `json:"validation_id"`
	MedicationOrderID       string                   `json:"medication_order_id,omitempty"`
	RankedProposals         []map[string]interface{} `json:"ranked_proposals"`
	ValidationResult        *ValidationSummary       `json:"validation_result"`
	CommitResult            *CommitSummary           `json:"commit_result,omitempty"`
	ExecutionMetrics        *ExecutionMetrics        `json:"execution_metrics"`
	Status                  string                   `json:"status"`
	Message                 string                   `json:"message,omitempty"`
	Errors                  []string                 `json:"errors,omitempty"`

	// Advanced pattern fields
	UIState                 *WorkflowUIState         `json:"ui_state,omitempty"`           // Current UI state
	OverrideSession         *OverrideSession         `json:"override_session,omitempty"`   // Active override session
	RequiredActions         []UIAction               `json:"required_actions,omitempty"`   // Actions needed from clinician
	Notifications          []UINotification          `json:"notifications,omitempty"`      // UI notifications
	IdempotencyToken       string                   `json:"idempotency_token,omitempty"`  // Token for retry safety
	WorkflowState          string                   `json:"workflow_state"`               // CALCULATING, VALIDATING, AWAITING_OVERRIDE, COMMITTING, COMPLETED
}

// ValidationSummary contains safety validation results
type ValidationSummary struct {
	Verdict              string                        `json:"verdict"`
	OverallRiskScore     float64                       `json:"overall_risk_score"`
	FindingsCount        int                           `json:"findings_count"`
	CriticalFindings     []clients.ValidationFinding   `json:"critical_findings"`
	ExecutedEngines      []string                      `json:"executed_engines"`
	OverrideTokens       []string                      `json:"override_tokens,omitempty"`
}

// CommitSummary contains medication order commit results
type CommitSummary struct {
	MedicationOrderID        string    `json:"medication_order_id"`
	FHIRResourceID           string    `json:"fhir_resource_id"`
	PersistenceStatus        string    `json:"persistence_status"`
	EventPublicationStatus   string    `json:"event_publication_status"`
	AuditTrailID             string    `json:"audit_trail_id"`
	CommittedAt              time.Time `json:"committed_at"`
}

// ExecutionMetrics contains performance tracking data
type ExecutionMetrics struct {
	TotalDuration    time.Duration `json:"total_duration_ms"`
	CalculateDuration time.Duration `json:"calculate_duration_ms"`
	ValidateDuration  time.Duration `json:"validate_duration_ms"`
	CommitDuration    time.Duration `json:"commit_duration_ms,omitempty"`
	PhaseBreakdown    map[string]time.Duration `json:"phase_breakdown_ms"`
}

// ExecuteMedicationWorkflow orchestrates the Advanced Calculate > Validate > Commit pattern with UI interactions
func (o *StrategicOrchestrator) ExecuteMedicationWorkflow(ctx context.Context, request *OrchestrationRequest) (*OrchestrationResponse, error) {
	startTime := time.Now()

	// Generate unique identifiers
	workflowInstanceID := uuid.New().String()
	correlationID := request.CorrelationID
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Handle idempotency for retry safety
	if request.IdempotencyToken != "" {
		if cachedResult, err := o.checkIdempotencyCache(ctx, request.IdempotencyToken); err == nil {
			o.logger.Info("Returning cached workflow result", zap.String("token", request.IdempotencyToken))
			return cachedResult, nil
		}
	} else {
		request.IdempotencyToken = uuid.New().String()
	}

	o.logger.Info("Starting advanced medication workflow orchestration",
		zap.String("workflow_instance_id", workflowInstanceID),
		zap.String("correlation_id", correlationID),
		zap.String("patient_id", request.PatientID),
		zap.String("ui_interaction_mode", request.UIInteractionMode),
		zap.String("idempotency_token", request.IdempotencyToken))

	// Initialize workflow state with UI coordination
	workflowState := &WorkflowState{
		WorkflowID:    workflowInstanceID,
		PatientID:     request.PatientID,
		CorrelationID: correlationID,
		Phase:         "INITIALIZING",
		Status:        "IN_PROGRESS",
		StartedAt:     time.Now(),
		UIInteractionMode: request.UIInteractionMode,
		SessionID:     request.SessionID,
		Context: map[string]interface{}{
			"medication_request": request.MedicationRequest,
			"clinical_intent":    request.ClinicalIntent,
			"provider_context":   request.ProviderContext,
			"execution_mode":     request.ExecutionMode,
			"idempotency_token":  request.IdempotencyToken,
		},
	}

	// Register workflow state with UI coordinator
	if err := o.uiCoordinator.RegisterWorkflowState(ctx, workflowState); err != nil {
		o.logger.Error("Failed to register workflow state", zap.Error(err))
		// Continue - UI coordination failure shouldn't block workflow
	}

	// Create workflow instance in repository
	workflowInstance := &domain.WorkflowInstance{
		ID:            workflowInstanceID,
		DefinitionID:  "medication-workflow-advanced-v1",
		PatientID:     request.PatientID,
		Status:        domain.WorkflowStatusRunning,
		StartedAt:     time.Now(),
		CorrelationID: correlationID,
		Context: workflowState.Context,
	}

	if err := o.workflowRepo.Create(ctx, workflowInstance); err != nil {
		return nil, fmt.Errorf("failed to create workflow instance: %w", err)
	}

	response := &OrchestrationResponse{
		WorkflowInstanceID: workflowInstanceID,
		ExecutionMetrics:   &ExecutionMetrics{PhaseBreakdown: make(map[string]time.Duration)},
		Errors:             []string{},
		IdempotencyToken:   request.IdempotencyToken,
		WorkflowState:      "CALCULATING",
		UIState:           o.createInitialUIState(workflowInstanceID, request),
	}

	// Execute workflow with idempotency protection
	executor := func() (interface{}, error) {
		return o.executeAdvancedWorkflowPhases(ctx, request, workflowState, response)
	}

	commitExecutor := NewCommitExecutor(o.idempotencyManager, o.logger)
	result, err := commitExecutor.ExecuteCommit(ctx, request, workflowInstanceID, executor)
	if err != nil {
		o.handleWorkflowFailure(ctx, workflowInstanceID, workflowState, err)
		return response, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Extract final response from idempotent execution
	finalResponse := result.(*OrchestrationResponse)
	finalResponse.ExecutionMetrics.TotalDuration = time.Since(startTime)

	// Cache result for idempotency
	if err := o.cacheIdempotencyResult(ctx, request.IdempotencyToken, finalResponse); err != nil {
		o.logger.Warn("Failed to cache idempotency result", zap.Error(err))
	}

	o.logger.Info("Advanced medication workflow orchestration completed",
		zap.String("workflow_instance_id", workflowInstanceID),
		zap.String("correlation_id", correlationID),
		zap.String("status", finalResponse.Status),
		zap.String("workflow_state", finalResponse.WorkflowState),
		zap.Duration("total_duration", finalResponse.ExecutionMetrics.TotalDuration))

	return finalResponse, nil
}

// executeCalculatePhase calls Flow2 Go Engine for medication intelligence
func (o *StrategicOrchestrator) executeCalculatePhase(ctx context.Context, request *OrchestrationRequest) (*clients.Flow2ExecuteResponse, error) {
	o.logger.Info("Executing calculate phase",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("patient_id", request.PatientID))

	// Set default execution mode if not specified
	executionMode := request.ExecutionMode
	if executionMode == "" {
		executionMode = "advanced"
	}

	flow2Request := &clients.Flow2ExecuteRequest{
		PatientID:         request.PatientID,
		Medication:        request.MedicationRequest,
		ClinicalIntent:    request.ClinicalIntent,
		ProviderContext:   request.ProviderContext,
		ExecutionMode:     executionMode,
		CorrelationID:     request.CorrelationID,
		SnapshotOptimized: true,
		UseCache:          true,
	}

	result, err := o.flow2GoClient.ExecuteAdvanced(ctx, flow2Request)
	if err != nil {
		return nil, fmt.Errorf("Flow2 Go Engine execution failed: %w", err)
	}

	// Create snapshot reference for data consistency
	snapshotRef := &domain.SnapshotReference{
		SnapshotID:     result.SnapshotID,
		Checksum:       o.calculateSnapshotChecksum(result),
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(2 * time.Hour), // 2-hour expiration
		Status:         domain.SnapshotStatusActive,
		PhaseCreated:   domain.WorkflowPhaseCalculate,
		PatientID:      request.PatientID,
		ContextVersion: "v1",
		Metadata: map[string]interface{}{
			"proposal_set_id": result.ProposalSetID,
			"kb_versions":     result.KBVersions,
			"correlation_id":  request.CorrelationID,
		},
	}

	if err := o.snapshotRepo.Create(ctx, snapshotRef); err != nil {
		o.logger.Warn("Failed to create snapshot reference", zap.Error(err))
		// Continue execution - snapshot tracking is not critical for workflow success
	}

	o.logger.Info("Calculate phase completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("snapshot_id", result.SnapshotID),
		zap.String("proposal_set_id", result.ProposalSetID),
		zap.Int("proposal_count", len(result.RankedProposals)))

	return result, nil
}

// executeValidatePhase calls Safety Gateway for comprehensive validation
func (o *StrategicOrchestrator) executeValidatePhase(ctx context.Context, request *OrchestrationRequest, calculateResult *clients.Flow2ExecuteResponse) (*clients.SafetyValidationResponse, error) {
	o.logger.Info("Executing validate phase",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("snapshot_id", calculateResult.SnapshotID))

	// Set default validation level
	validationLevel := request.ValidationLevel
	if validationLevel == "" {
		validationLevel = "comprehensive"
	}

	// Build patient context from available data
	patientContext := map[string]interface{}{
		"patient_id": request.PatientID,
	}
	if request.ProviderContext != nil {
		patientContext["provider_context"] = request.ProviderContext
	}

	validationRequest := &clients.SafetyValidationRequest{
		ProposalSetID:    calculateResult.ProposalSetID,
		SnapshotID:       calculateResult.SnapshotID,
		Proposals:        calculateResult.RankedProposals,
		PatientContext:   patientContext,
		CorrelationID:    request.CorrelationID,
		ValidationScope:  []string{"comprehensive", "drug_interactions", "contraindications", "allergies", "dosing"},
		RiskTolerance:    "standard",
	}

	result, err := o.safetyGatewayClient.ComprehensiveValidation(ctx, validationRequest)
	if err != nil {
		return nil, fmt.Errorf("Safety Gateway validation failed: %w", err)
	}

	o.logger.Info("Validate phase completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("validation_id", result.ValidationID),
		zap.String("verdict", result.Verdict),
		zap.Float64("risk_score", result.OverallRiskScore),
		zap.Int("findings_count", len(result.Findings)))

	return result, nil
}

// executeCommitPhase calls Medication Service to persist the order
func (o *StrategicOrchestrator) executeCommitPhase(ctx context.Context, request *OrchestrationRequest, calculateResult *clients.Flow2ExecuteResponse, validationResult *clients.SafetyValidationResponse) (*clients.MedicationCommitResponse, error) {
	o.logger.Info("Executing commit phase",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("validation_id", validationResult.ValidationID))

	// Select the top-ranked proposal for commit
	if len(calculateResult.RankedProposals) == 0 {
		return nil, fmt.Errorf("no proposals available for commit")
	}

	selectedProposal := calculateResult.RankedProposals[0]

	// Build provider decision context
	providerDecision := map[string]interface{}{
		"selected_proposal_index": 0,
		"validation_override":     false,
		"clinical_justification":  "Top-ranked proposal selected after safety validation",
	}

	// Include override information if validation has warnings but is being committed
	if validationResult.Verdict == "WARNING" && len(validationResult.OverrideTokens) > 0 {
		providerDecision["validation_override"] = true
		providerDecision["override_tokens"] = validationResult.OverrideTokens
		providerDecision["override_justification"] = "Provider override after clinical review"
	}

	commitRequest := &clients.MedicationCommitRequest{
		ProposalSetID:    calculateResult.ProposalSetID,
		ValidationID:     validationResult.ValidationID,
		SelectedProposal: selectedProposal,
		ProviderDecision: providerDecision,
		CorrelationID:    request.CorrelationID,
		PatientID:        request.PatientID,
		CommitMode:       request.CommitMode,
	}

	// Add provider context if available
	if request.ProviderContext != nil {
		if providerID, ok := request.ProviderContext["provider_id"].(string); ok {
			commitRequest.ProviderID = providerID
		}
		if encounterID, ok := request.ProviderContext["encounter_id"].(string); ok {
			commitRequest.EncounterID = encounterID
		}
	}

	result, err := o.medicationClient.Commit(ctx, commitRequest)
	if err != nil {
		return nil, fmt.Errorf("Medication Service commit failed: %w", err)
	}

	o.logger.Info("Commit phase completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("medication_order_id", result.MedicationOrderID),
		zap.String("fhir_resource_id", result.FHIRResourceID),
		zap.String("persistence_status", result.PersistenceStatus))

	return result, nil
}

// Helper methods

func (o *StrategicOrchestrator) shouldCommitBasedOnValidation(validationResult *clients.SafetyValidationResponse, commitMode string) bool {
	switch commitMode {
	case "immediate":
		return true // Always commit regardless of validation
	case "never":
		return false // Never commit, validation only
	case "safe_only":
		return validationResult.Verdict == "SAFE"
	case "conditional":
		fallthrough
	default:
		// Default behavior: commit if SAFE or WARNING (but not UNSAFE or ERROR)
		return validationResult.Verdict == "SAFE" || validationResult.Verdict == "WARNING"
	}
}

func (o *StrategicOrchestrator) summarizeValidation(result *clients.SafetyValidationResponse) *ValidationSummary {
	criticalFindings := []clients.ValidationFinding{}
	for _, finding := range result.Findings {
		if finding.Severity == "CRITICAL" || finding.Severity == "HIGH" {
			criticalFindings = append(criticalFindings, finding)
		}
	}

	return &ValidationSummary{
		Verdict:          result.Verdict,
		OverallRiskScore: result.OverallRiskScore,
		FindingsCount:    len(result.Findings),
		CriticalFindings: criticalFindings,
		ExecutedEngines:  result.ExecutedEngines,
		OverrideTokens:   result.OverrideTokens,
	}
}

func (o *StrategicOrchestrator) calculateSnapshotChecksum(result *clients.Flow2ExecuteResponse) string {
	// Simple checksum based on proposal set ID and snapshot ID
	// In production, this would use a proper hash function
	return fmt.Sprintf("chk_%s_%s", result.ProposalSetID[:8], result.SnapshotID[:8])
}

func (o *StrategicOrchestrator) updateWorkflowStatus(ctx context.Context, workflowID string, status domain.WorkflowStatus, message string) {
	if err := o.workflowRepo.UpdateStatus(ctx, workflowID, status, message); err != nil {
		o.logger.Error("Failed to update workflow status",
			zap.String("workflow_id", workflowID),
			zap.String("status", string(status)),
			zap.Error(err))
	}

	// Also update UI coordinator state
	if err := o.uiCoordinator.UpdateWorkflowPhase(ctx, workflowID, string(status)); err != nil {
		o.logger.Warn("Failed to update UI coordinator state", zap.Error(err))
	}
}

// executeAdvancedWorkflowPhases executes the complete advanced workflow with UI interactions
func (o *StrategicOrchestrator) executeAdvancedWorkflowPhases(ctx context.Context, request *OrchestrationRequest, workflowState *WorkflowState, response *OrchestrationResponse) (*OrchestrationResponse, error) {
	// Phase 1: Advanced Calculate with UI coordination
	if err := o.updateWorkflowPhase(ctx, workflowState, "CALCULATING"); err != nil {
		return response, err
	}
	response.WorkflowState = "CALCULATING"

	calculateStart := time.Now()
	calculateResult, err := o.executeAdvancedCalculatePhase(ctx, request, workflowState)
	calculateDuration := time.Since(calculateStart)
	response.ExecutionMetrics.CalculateDuration = calculateDuration
	response.ExecutionMetrics.PhaseBreakdown["calculate"] = calculateDuration

	if err != nil {
		return response, fmt.Errorf("advanced calculate phase failed: %w", err)
	}

	response.SnapshotID = calculateResult.SnapshotID
	response.ProposalSetID = calculateResult.ProposalSetID
	response.RankedProposals = calculateResult.RankedProposals

	// Phase 2: Advanced Validate with clinical override support
	if err := o.updateWorkflowPhase(ctx, workflowState, "VALIDATING"); err != nil {
		return response, err
	}
	response.WorkflowState = "VALIDATING"

	validateStart := time.Now()
	validationResult, overrideRequired, err := o.executeAdvancedValidatePhase(ctx, request, workflowState, calculateResult)
	validateDuration := time.Since(validateStart)
	response.ExecutionMetrics.ValidateDuration = validateDuration
	response.ExecutionMetrics.PhaseBreakdown["validate"] = validateDuration

	if err != nil {
		return response, fmt.Errorf("advanced validate phase failed: %w", err)
	}

	response.ValidationID = validationResult.ValidationID
	response.ValidationResult = o.summarizeValidation(validationResult)

	// Handle clinical override if required
	if overrideRequired {
		overrideResponse, err := o.handleClinicalOverrideProcess(ctx, request, workflowState, validationResult)
		if err != nil {
			return response, fmt.Errorf("clinical override process failed: %w", err)
		}

		response.OverrideSession = overrideResponse.OverrideSession
		response.RequiredActions = overrideResponse.RequiredActions
		response.Notifications = overrideResponse.Notifications

		if overrideResponse.AwaitingDecision {
			response.WorkflowState = "AWAITING_OVERRIDE"
			return response, nil // Workflow paused, awaiting clinician decision
		}
	}

	// Phase 3: Advanced Commit using CommitOrchestrator (follows document 13_9.1)
	commitMode := request.CommitMode
	if commitMode == "" {
		commitMode = "conditional"
	}

	// The CommitOrchestrator handles the verdict-based branching logic internally
	// following the exact sequence from document 13_9.1: SAFE → immediate commit, UNSAFE/WARNING → UI interaction
	shouldCommit := o.shouldCommitBasedOnValidation(validationResult, commitMode)
	if shouldCommit {
		if err := o.updateWorkflowPhase(ctx, workflowState, "COMMITTING"); err != nil {
			return response, err
		}
		response.WorkflowState = "COMMITTING"

		commitStart := time.Now()
		commitResult, err := o.executeAdvancedCommitPhase(ctx, request, workflowState, calculateResult, validationResult)
		commitDuration := time.Since(commitStart)
		response.ExecutionMetrics.CommitDuration = commitDuration
		response.ExecutionMetrics.PhaseBreakdown["commit"] = commitDuration

		if err != nil {
			return response, fmt.Errorf("advanced commit phase failed: %w", err)
		}

		response.MedicationOrderID = commitResult.MedicationOrderID
		response.CommitResult = &CommitSummary{
			MedicationOrderID:        commitResult.MedicationOrderID,
			FHIRResourceID:          commitResult.FHIRResourceID,
			PersistenceStatus:       commitResult.PersistenceStatus,
			EventPublicationStatus:  commitResult.EventPublicationStatus,
			AuditTrailID:           commitResult.AuditTrailID,
			CommittedAt:            commitResult.CommittedAt,
		}

		response.Status = "completed"
		response.WorkflowState = "COMPLETED"
		response.Message = "Advanced medication workflow completed successfully with order committed"
	} else {
		response.Status = "completed_no_commit"
		response.WorkflowState = "COMPLETED_NO_COMMIT"
		response.Message = fmt.Sprintf("Advanced medication workflow completed but order not committed due to validation verdict: %s", validationResult.Verdict)
	}

	// Final UI state update
	if err := o.updateWorkflowPhase(ctx, workflowState, response.WorkflowState); err != nil {
		o.logger.Warn("Failed to update final workflow phase", zap.Error(err))
	}

	return response, nil
}

// executeAdvancedCalculatePhase performs calculation with Recipe Resolution pattern and enhanced UI coordination
func (o *StrategicOrchestrator) executeAdvancedCalculatePhase(ctx context.Context, request *OrchestrationRequest, workflowState *WorkflowState) (*clients.Flow2ExecuteResponse, error) {
	startTime := time.Now()

	o.logger.Info("Executing advanced calculate phase with Recipe Resolution",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("workflow_id", workflowState.WorkflowID),
		zap.String("patient_id", request.PatientID))

	// Send UI notification if in interactive mode
	if request.UIInteractionMode == "interactive" || request.UIInteractionMode == "notification" {
		notification := &UINotification{
			WorkflowID: workflowState.WorkflowID,
			Status:     "IN_PROGRESS",
			Title:      "Medication Analysis in Progress",
			Message:    "Resolving clinical recipe and creating immutable snapshot...",
			Severity:   "INFO",
			Timestamp:  time.Now(),
		}
		o.uiCoordinator.SendNotification(ctx, notification)
	}

	// Step 1: Recipe Resolution
	protocol := o.extractProtocol(request.MedicationRequest)
	patientData := o.combinePatientData(request)

	recipe, err := o.recipeResolver.ResolveRecipe(ctx, request.PatientID, protocol, patientData)
	if err != nil {
		o.logger.Error("Recipe resolution failed",
			zap.String("patient_id", request.PatientID),
			zap.String("protocol", protocol),
			zap.Error(err))

		// Send failure notification
		if request.UIInteractionMode != "none" {
			notification := &UINotification{
				WorkflowID: workflowState.WorkflowID,
				Status:     "ERROR",
				Title:      "Recipe Resolution Failed",
				Message:    fmt.Sprintf("Failed to resolve clinical recipe: %v", err),
				Severity:   "ERROR",
				Timestamp:  time.Now(),
			}
			o.uiCoordinator.SendNotification(ctx, notification)
		}
		return nil, fmt.Errorf("recipe resolution failed: %w", err)
	}

	// Step 2: Create Immutable Snapshot
	snapshot, err := o.recipeResolver.CreateSnapshot(ctx, recipe, workflowState.WorkflowID, request.PatientID, patientData)
	if err != nil {
		o.logger.Error("Snapshot creation failed",
			zap.String("recipe_id", recipe.RecipeID),
			zap.Error(err))
		return nil, fmt.Errorf("snapshot creation failed: %w", err)
	}

	// Store snapshot ID and original request for rework capability
	workflowState.SnapshotID = snapshot.SnapshotID
	workflowState.OriginalRequest = map[string]interface{}{
		"medication_request": request.MedicationRequest,
		"clinical_intent":    request.ClinicalIntent,
		"provider_context":   request.ProviderContext,
		"patient_data":       patientData,
		"protocol":           protocol,
	}

	// Step 3: Execute Calculations with Recipe
	result, err := o.executeCalculateWithRecipe(ctx, request, snapshot)
	if err != nil {
		// Send failure notification
		if request.UIInteractionMode != "none" {
			notification := &UINotification{
				WorkflowID: workflowState.WorkflowID,
				Status:     "ERROR",
				Title:      "Medication Analysis Failed",
				Message:    fmt.Sprintf("Failed to analyze medication request: %v", err),
				Severity:   "ERROR",
				Timestamp:  time.Now(),
			}
			o.uiCoordinator.SendNotification(ctx, notification)
		}
		return nil, err
	}

	// Update workflow state with calculation results
	workflowState.Phase = "CALCULATED"
	workflowState.Context["snapshot_id"] = result.SnapshotID
	workflowState.Context["proposal_set_id"] = result.ProposalSetID
	workflowState.Context["proposal_count"] = len(result.RankedProposals)

	if err := o.uiCoordinator.UpdateWorkflowState(ctx, workflowState); err != nil {
		o.logger.Warn("Failed to update workflow state after calculation", zap.Error(err))
	}

	// Send success notification with results
	if request.UIInteractionMode != "none" {
		notification := &UINotification{
			WorkflowID: workflowState.WorkflowID,
			Status:     "SUCCESS",
			Title:      "Medication Analysis Complete",
			Message:    fmt.Sprintf("Generated %d clinical proposal(s) for review", len(result.RankedProposals)),
			Severity:   "SUCCESS",
			Timestamp:  time.Now(),
			Actions: []UIAction{
				{
					ID:    "view_proposals",
					Label: "View Proposals",
					Type:  "VIEW",
					Data:  map[string]interface{}{"proposal_set_id": result.ProposalSetID},
				},
			},
		}
		o.uiCoordinator.SendNotification(ctx, notification)
	}

	return result, nil
}

// executeAdvancedValidatePhase performs validation with override detection
func (o *StrategicOrchestrator) executeAdvancedValidatePhase(ctx context.Context, request *OrchestrationRequest, workflowState *WorkflowState, calculateResult *clients.Flow2ExecuteResponse) (*clients.SafetyValidationResponse, bool, error) {
	o.logger.Info("Executing advanced validate phase",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("workflow_id", workflowState.WorkflowID),
		zap.String("snapshot_id", calculateResult.SnapshotID))

	// Send validation start notification
	if request.UIInteractionMode != "none" {
		notification := &UINotification{
			WorkflowID: workflowState.WorkflowID,
			Status:     "IN_PROGRESS",
			Title:      "Safety Validation in Progress",
			Message:    "Performing comprehensive safety validation...",
			Severity:   "INFO",
			Timestamp:  time.Now(),
		}
		o.uiCoordinator.SendNotification(ctx, notification)
	}

	// Execute standard validation
	result, err := o.executeValidatePhase(ctx, request, calculateResult)
	if err != nil {
		return nil, false, err
	}

	// Determine if clinical override is required
	overrideRequired := o.requiresOverride(result, request.UIInteractionMode)

	// Update workflow state with validation results
	workflowState.Phase = "VALIDATED"
	workflowState.Context["validation_id"] = result.ValidationID
	workflowState.Context["validation_verdict"] = result.Verdict
	workflowState.Context["risk_score"] = result.OverallRiskScore
	workflowState.Context["override_required"] = overrideRequired

	if err := o.uiCoordinator.UpdateWorkflowState(ctx, workflowState); err != nil {
		o.logger.Warn("Failed to update workflow state after validation", zap.Error(err))
	}

	return result, overrideRequired, nil
}

// executeAdvancedCommitPhase performs commit using the CommitOrchestrator following document 13_9.1 flow
func (o *StrategicOrchestrator) executeAdvancedCommitPhase(ctx context.Context, request *OrchestrationRequest, workflowState *WorkflowState, calculateResult *clients.Flow2ExecuteResponse, validationResult *clients.SafetyValidationResponse) (*clients.MedicationCommitResponse, error) {
	o.logger.Info("Executing advanced commit phase with CommitOrchestrator",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("workflow_id", workflowState.WorkflowID),
		zap.String("validation_id", validationResult.ValidationID),
		zap.String("verdict", validationResult.Verdict))

	// Build commit request following document 13_9.1 specifications
	commitRequest := &CommitRequest{
		ProposalID:       calculateResult.ProposalSetID,
		PatientID:        request.PatientID,
		WorkflowID:       workflowState.WorkflowID,
		CorrelationID:    request.CorrelationID,
		ValidationResult: &CommitValidationResult{
			ValidationID:     validationResult.ValidationID,
			Verdict:          validationResult.Verdict,
			RiskScore:        validationResult.OverallRiskScore,
			Findings:         o.convertValidationFindings(validationResult.Findings),
			Evidence:         o.buildEvidenceTrail(validationResult),
			OverrideAllowed:  o.isOverrideAllowed(validationResult),
		},
		SelectedProposal: o.convertToCommitProposal(calculateResult.RankedProposals[0]),
		ProviderContext: &CommitProviderContext{
			ProviderID:      o.extractProviderID(request.ProviderContext),
			EncounterID:     o.extractEncounterID(request.ProviderContext),
			SessionID:       request.SessionID,
			Timestamp:       time.Now(),
		},
		UIInteractionMode: request.UIInteractionMode,
	}

	// Execute commit phase using CommitOrchestrator (follows document 13_9.1 sequence)
	commitResponse, err := o.commitOrchestrator.ExecuteCommitPhase(ctx, commitRequest)
	if err != nil {
		o.logger.Error("CommitOrchestrator execution failed",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		return nil, fmt.Errorf("commit phase failed: %w", err)
	}

	// Update workflow state with commit results
	workflowState.Phase = "COMMITTED"
	workflowState.Status = "COMPLETED"
	workflowState.Context["commit_id"] = commitResponse.CommitID
	workflowState.Context["medication_order_id"] = commitResponse.MedicationOrderID
	workflowState.Context["commit_status"] = commitResponse.Status
	workflowState.Context["committed_at"] = commitResponse.CommittedAt
	workflowState.Context["rollback_token"] = commitResponse.RollbackToken
	workflowState.Context["rollback_expires_at"] = commitResponse.RollbackExpiresAt

	if err := o.uiCoordinator.UpdateWorkflowState(ctx, workflowState); err != nil {
		o.logger.Warn("Failed to update workflow state after commit", zap.Error(err))
	}

	// Send appropriate UI notification based on commit outcome
	if request.UIInteractionMode != "none" {
		var notification *UINotification
		switch commitResponse.Status {
		case "COMMITTED":
			notification = &UINotification{
				WorkflowID: workflowState.WorkflowID,
				Status:     "SUCCESS",
				Title:      "Medication Order Committed",
				Message:    fmt.Sprintf("Order %s committed successfully. Rollback available for 5 minutes.", commitResponse.MedicationOrderID),
				Severity:   "SUCCESS",
				Timestamp:  time.Now(),
				Actions: []UIAction{
					{
						ID:    "view_order",
						Label: "View Order",
						Type:  "VIEW",
						Data:  map[string]interface{}{"order_id": commitResponse.MedicationOrderID},
					},
					{
						ID:    "rollback_order",
						Label: "Rollback Order",
						Type:  "ROLLBACK",
						Data:  map[string]interface{}{"rollback_token": commitResponse.RollbackToken},
					},
				},
			}
		case "AWAITING_OVERRIDE":
			notification = &UINotification{
				WorkflowID: workflowState.WorkflowID,
				Status:     "ACTION_REQUIRED",
				Title:      "Clinical Override Required",
				Message:    "Safety concerns detected. Please review and provide override decision.",
				Severity:   "WARNING",
				Timestamp:  time.Now(),
				Actions: []UIAction{
					{
						ID:    "review_override",
						Label: "Review Override",
						Type:  "OVERRIDE_REVIEW",
						Data:  map[string]interface{}{"commit_id": commitResponse.CommitID},
					},
				},
			}
		default:
			notification = &UINotification{
				WorkflowID: workflowState.WorkflowID,
				Status:     "ERROR",
				Title:      "Commit Failed",
				Message:    fmt.Sprintf("Commit failed with status: %s", commitResponse.Status),
				Severity:   "ERROR",
				Timestamp:  time.Now(),
			}
		}
		o.uiCoordinator.SendNotification(ctx, notification)
	}

	// Convert CommitResponse back to MedicationCommitResponse for compatibility
	result := &clients.MedicationCommitResponse{
		MedicationOrderID:        commitResponse.MedicationOrderID,
		FHIRResourceID:          commitResponse.FHIRResourceID,
		PersistenceStatus:       commitResponse.Status,
		EventPublicationStatus:  "PUBLISHED", // CommitOrchestrator handles Kafka publishing
		AuditTrailID:           commitResponse.AuditTrailID,
		CommittedAt:            commitResponse.CommittedAt,
	}

	o.logger.Info("Advanced commit phase completed using CommitOrchestrator",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("commit_id", commitResponse.CommitID),
		zap.String("medication_order_id", commitResponse.MedicationOrderID),
		zap.String("status", commitResponse.Status),
		zap.Bool("rollback_available", commitResponse.RollbackToken != ""))

	return result, nil
}

// Helper methods for advanced pattern support

func (o *StrategicOrchestrator) createInitialUIState(workflowID string, request *OrchestrationRequest) *WorkflowUIState {
	if request.UIInteractionMode == "none" {
		return nil
	}

	return &WorkflowUIState{
		WorkflowID:    workflowID,
		Phase:         "INITIALIZING",
		Status:        "IN_PROGRESS",
		LastUpdate:    time.Now(),
		InteractionMode: request.UIInteractionMode,
		SessionID:     request.SessionID,
	}
}

func (o *StrategicOrchestrator) updateWorkflowPhase(ctx context.Context, workflowState *WorkflowState, phase string) error {
	workflowState.Phase = phase
	workflowState.LastUpdate = time.Now()
	return o.uiCoordinator.UpdateWorkflowState(ctx, workflowState)
}

func (o *StrategicOrchestrator) requiresOverride(validationResult *clients.SafetyValidationResponse, uiMode string) bool {
	if uiMode == "none" {
		return false // No UI interaction, no overrides
	}

	// Require override for unsafe verdicts or high-risk scenarios
	if validationResult.Verdict == "UNSAFE" {
		return true
	}

	// Require override for warnings in review_required mode
	if uiMode == "review_required" && validationResult.Verdict == "WARNING" {
		return true
	}

	// Check for critical findings requiring clinical review
	for _, finding := range validationResult.Findings {
		if finding.Severity == "CRITICAL" && finding.RequiresOverride {
			return true
		}
	}

	return false
}

func (o *StrategicOrchestrator) handleClinicalOverrideProcess(ctx context.Context, request *OrchestrationRequest, workflowState *WorkflowState, validationResult *clients.SafetyValidationResponse) (*ClinicalOverrideResponse, error) {
	// Determine required override level
	overrideLevel, err := o.determineRequiredOverrideLevel(validationResult)
	if err != nil {
		return nil, fmt.Errorf("failed to determine override level: %w", err)
	}

	// Validate clinician authority
	clinicianAuthority := request.OverrideAuthority
	if clinicianAuthority == "" {
		clinicianAuthority = "clinical_judgment" // Default
	}

	canOverride, err := o.overrideManager.ValidateOverrideAuthority(ctx, clinicianAuthority, overrideLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to validate override authority: %w", err)
	}

	if !canOverride {
		// Escalation required
		return o.initiateOverrideEscalation(ctx, workflowState, validationResult, overrideLevel)
	}

	// Direct override possible
	return o.initiateDirectOverride(ctx, workflowState, validationResult, clinicianAuthority)
}

func (o *StrategicOrchestrator) determineRequiredOverrideLevel(validationResult *clients.SafetyValidationResponse) (OverrideLevel, error) {
	if validationResult.Verdict == "UNSAFE" {
		return OverrideLevelSupervisory, nil
	}

	maxSeverity := ""
	for _, finding := range validationResult.Findings {
		if finding.Severity == "CRITICAL" {
			maxSeverity = "CRITICAL"
		} else if finding.Severity == "HIGH" && maxSeverity != "CRITICAL" {
			maxSeverity = "HIGH"
		}
	}

	switch maxSeverity {
	case "CRITICAL":
		return OverrideLevelPeerReview, nil
	case "HIGH":
		return OverrideLevelClinicalJudgment, nil
	default:
		return OverrideLevelClinicalJudgment, nil
	}
}

func (o *StrategicOrchestrator) initiateDirectOverride(ctx context.Context, workflowState *WorkflowState, validationResult *clients.SafetyValidationResponse, authority string) (*ClinicalOverrideResponse, error) {
	overrideSession := &OverrideSession{
		SessionID:        uuid.New().String(),
		WorkflowID:       workflowState.WorkflowID,
		ValidationID:     validationResult.ValidationID,
		RequiredLevel:    OverrideLevelClinicalJudgment,
		ClinicianID:      extractClinicianID(workflowState.Context),
		Status:           "ACTIVE",
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(30 * time.Minute),
		ValidationFindings: validationResult.Findings,
	}

	if err := o.uiCoordinator.CreateOverrideSession(ctx, overrideSession); err != nil {
		return nil, fmt.Errorf("failed to create override session: %w", err)
	}

	return &ClinicalOverrideResponse{
		OverrideSession: overrideSession,
		AwaitingDecision: true,
		RequiredActions: []UIAction{
			{
				ID:    "approve_override",
				Label: "Approve Override",
				Type:  "OVERRIDE_APPROVE",
				Data:  map[string]interface{}{"session_id": overrideSession.SessionID},
			},
			{
				ID:    "reject_override",
				Label: "Reject Override",
				Type:  "OVERRIDE_REJECT",
				Data:  map[string]interface{}{"session_id": overrideSession.SessionID},
			},
		},
		Notifications: []UINotification{
			{
				WorkflowID: workflowState.WorkflowID,
				Status:     "ACTION_REQUIRED",
				Title:      "Clinical Override Required",
				Message:    "Please review validation findings and provide override decision",
				Severity:   "WARNING",
				Timestamp:  time.Now(),
			},
		},
	}, nil
}

func (o *StrategicOrchestrator) initiateOverrideEscalation(ctx context.Context, workflowState *WorkflowState, validationResult *clients.SafetyValidationResponse, requiredLevel OverrideLevel) (*ClinicalOverrideResponse, error) {
	// Implementation for escalation process
	return &ClinicalOverrideResponse{
		AwaitingDecision: true,
		Notifications: []UINotification{
			{
				WorkflowID: workflowState.WorkflowID,
				Status:     "ESCALATION_REQUIRED",
				Title:      "Override Escalation Required",
				Message:    fmt.Sprintf("Override requires %s level authority", requiredLevel),
				Severity:   "ERROR",
				Timestamp:  time.Now(),
			},
		},
	}, nil
}

func (o *StrategicOrchestrator) handleWorkflowFailure(ctx context.Context, workflowID string, workflowState *WorkflowState, err error) {
	workflowState.Status = "FAILED"
	workflowState.Phase = "FAILED"
	o.updateWorkflowStatus(ctx, workflowID, domain.WorkflowStatusFailed, err.Error())

	if workflowState.UIInteractionMode != "none" {
		notification := &UINotification{
			WorkflowID: workflowID,
			Status:     "ERROR",
			Title:      "Workflow Failed",
			Message:    fmt.Sprintf("Workflow execution failed: %v", err),
			Severity:   "ERROR",
			Timestamp:  time.Now(),
		}
		o.uiCoordinator.SendNotification(ctx, notification)
	}
}

// Idempotency support methods
func (o *StrategicOrchestrator) checkIdempotencyCache(ctx context.Context, token string) (*OrchestrationResponse, error) {
	// Check if this request has been processed before
	key := fmt.Sprintf("workflow:idempotency:%s", token)
	result, err := o.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var response OrchestrationResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (o *StrategicOrchestrator) cacheIdempotencyResult(ctx context.Context, token string, response *OrchestrationResponse) error {
	key := fmt.Sprintf("workflow:idempotency:%s", token)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return o.redisClient.Set(ctx, key, data, 24*time.Hour).Err()
}

func extractClinicianID(context map[string]interface{}) string {
	if providerCtx, ok := context["provider_context"].(map[string]interface{}); ok {
		if providerID, ok := providerCtx["provider_id"].(string); ok {
			return providerID
		}
	}
	return "unknown"
}

// Helper methods for CommitOrchestrator integration

// convertValidationFindings converts SafetyGateway findings to CommitOrchestrator format
func (o *StrategicOrchestrator) convertValidationFindings(findings []clients.ValidationFinding) []CommitFinding {
	commitFindings := make([]CommitFinding, len(findings))
	for i, finding := range findings {
		commitFindings[i] = CommitFinding{
			ID:          fmt.Sprintf("finding_%d", i),
			RuleID:      finding.RuleID,
			Severity:    finding.Severity,
			Category:    finding.Category,
			Description: finding.Description,
			Evidence:    finding.Evidence,
			Recommendation: finding.Recommendation,
		}
	}
	return commitFindings
}

// buildEvidenceTrail creates an evidence trail for the commit decision
func (o *StrategicOrchestrator) buildEvidenceTrail(validationResult *clients.SafetyValidationResponse) []EvidenceItem {
	evidence := []EvidenceItem{
		{
			Type:        "VALIDATION_VERDICT",
			Description: fmt.Sprintf("Safety Gateway verdict: %s", validationResult.Verdict),
			Data:        map[string]interface{}{"verdict": validationResult.Verdict, "risk_score": validationResult.OverallRiskScore},
			Timestamp:   time.Now(),
		},
	}

	// Add evidence for each finding
	for i, finding := range validationResult.Findings {
		evidence = append(evidence, EvidenceItem{
			Type:        "VALIDATION_FINDING",
			Description: fmt.Sprintf("Finding %d: %s", i+1, finding.Description),
			Data:        map[string]interface{}{"severity": finding.Severity, "category": finding.Category, "rule_id": finding.RuleID},
			Timestamp:   time.Now(),
		})
	}

	return evidence
}

// isOverrideAllowed determines if clinical override is permitted for this validation result
func (o *StrategicOrchestrator) isOverrideAllowed(validationResult *clients.SafetyValidationResponse) bool {
	// UNSAFE verdicts require override (warnings can be overridden, absolute contraindications cannot)
	if validationResult.Verdict == "UNSAFE" {
		// Check for absolute contraindications
		for _, finding := range validationResult.Findings {
			if finding.Category == "ABSOLUTE_CONTRAINDICATION" || finding.Category == "CRITICAL_ALLERGY" {
				return false // No override allowed for absolute contraindications
			}
		}
		return true // Override allowed for relative contraindications
	}

	// WARNING verdicts can typically be overridden
	if validationResult.Verdict == "WARNING" {
		return true
	}

	// SAFE verdicts don't need override
	return false
}

// convertToCommitProposal converts Flow2 proposal to CommitOrchestrator format
func (o *StrategicOrchestrator) convertToCommitProposal(proposal map[string]interface{}) *CommitProposal {
	return &CommitProposal{
		ProposalID:      o.extractStringFromMap(proposal, "proposal_id"),
		MedicationCode:  o.extractStringFromMap(proposal, "medication_code"),
		MedicationName:  o.extractStringFromMap(proposal, "medication_name"),
		Dosage:          o.extractStringFromMap(proposal, "dosage"),
		Frequency:       o.extractStringFromMap(proposal, "frequency"),
		Route:           o.extractStringFromMap(proposal, "route"),
		Duration:        o.extractStringFromMap(proposal, "duration"),
		Instructions:    o.extractStringFromMap(proposal, "instructions"),
		Ranking:         o.extractFloatFromMap(proposal, "ranking"),
		Confidence:      o.extractFloatFromMap(proposal, "confidence"),
		Rationale:       o.extractStringFromMap(proposal, "rationale"),
		ProposalData:    proposal,
	}
}

// extractProviderID extracts provider ID from provider context
func (o *StrategicOrchestrator) extractProviderID(providerContext map[string]interface{}) string {
	if providerID, ok := providerContext["provider_id"].(string); ok {
		return providerID
	}
	return "unknown_provider"
}

// extractEncounterID extracts encounter ID from provider context
func (o *StrategicOrchestrator) extractEncounterID(providerContext map[string]interface{}) string {
	if encounterID, ok := providerContext["encounter_id"].(string); ok {
		return encounterID
	}
	return ""
}

// extractStringFromMap safely extracts string value from map
func (o *StrategicOrchestrator) extractStringFromMap(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

// extractFloatFromMap safely extracts float64 value from map
func (o *StrategicOrchestrator) extractFloatFromMap(data map[string]interface{}, key string) float64 {
	if value, ok := data[key].(float64); ok {
		return value
	}
	if value, ok := data[key].(int); ok {
		return float64(value)
	}
	return 0.0
}

// Supporting types for advanced pattern
type ClinicalOverrideResponse struct {
	OverrideSession  *OverrideSession    `json:"override_session,omitempty"`
	AwaitingDecision bool                `json:"awaiting_decision"`
	RequiredActions  []UIAction          `json:"required_actions,omitempty"`
	Notifications    []UINotification    `json:"notifications,omitempty"`
}

// HealthCheck verifies connectivity to all external services including advanced components
func (o *StrategicOrchestrator) HealthCheck(ctx context.Context) map[string]string {
	results := make(map[string]string)

	// Check Flow2 Go Engine
	if err := o.flow2GoClient.HealthCheck(ctx); err != nil {
		results["flow2_go"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["flow2_go"] = "healthy"
	}

	// Check Safety Gateway
	if err := o.safetyGatewayClient.HealthCheck(ctx); err != nil {
		results["safety_gateway"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["safety_gateway"] = "healthy"
	}

	// Check Medication Service
	if err := o.medicationClient.HealthCheck(ctx); err != nil {
		results["medication_service"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["medication_service"] = "healthy"
	}

	// Check Redis for advanced pattern components
	if err := o.redisClient.Ping(ctx).Err(); err != nil {
		results["redis"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["redis"] = "healthy"
	}

	// Check UI Coordinator
	if err := o.uiCoordinator.HealthCheck(ctx); err != nil {
		results["ui_coordinator"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["ui_coordinator"] = "healthy"
	}

	// Check Override Manager
	results["override_manager"] = "healthy" // Override manager is always available

	// Check Idempotency Manager
	results["idempotency_manager"] = "healthy" // Idempotency manager is always available

	// Check Commit Orchestrator
	if err := o.commitOrchestrator.HealthCheck(ctx); err != nil {
		results["commit_orchestrator"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["commit_orchestrator"] = "healthy"
	}

	// Check Batch Processor
	results["batch_processor"] = "healthy" // Batch processor is always available

	// Check Rollback Manager
	if err := o.rollbackManager.HealthCheck(ctx); err != nil {
		results["rollback_manager"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		results["rollback_manager"] = "healthy"
	}

	return results
}

// Helper methods for Recipe Resolution pattern

// extractProtocol extracts the clinical protocol from medication request
func (o *StrategicOrchestrator) extractProtocol(medicationRequest map[string]interface{}) string {
	if protocol, exists := medicationRequest["protocol"]; exists {
		if protocolStr, ok := protocol.(string); ok {
			return protocolStr
		}
	}

	// Default protocol based on medication type
	if medType, exists := medicationRequest["medication_type"]; exists {
		if medTypeStr, ok := medType.(string); ok {
			return fmt.Sprintf("standard_%s_protocol", medTypeStr)
		}
	}

	return "standard_medication_protocol"
}

// combinePatientData combines patient data from multiple sources
func (o *StrategicOrchestrator) combinePatientData(request *OrchestrationRequest) map[string]interface{} {
	combined := make(map[string]interface{})

	// Add clinical intent
	for key, value := range request.ClinicalIntent {
		combined[key] = value
	}

	// Add provider context
	for key, value := range request.ProviderContext {
		combined[key] = value
	}

	// Add medication request data
	for key, value := range request.MedicationRequest {
		if key != "protocol" { // Avoid duplicating protocol
			combined[key] = value
		}
	}

	// Add patient ID for reference
	combined["patient_id"] = request.PatientID
	combined["correlation_id"] = request.CorrelationID

	return combined
}

// executeCalculateWithRecipe executes calculation using resolved recipe snapshot
func (o *StrategicOrchestrator) executeCalculateWithRecipe(
	ctx context.Context,
	request *OrchestrationRequest,
	snapshot *RecipeSnapshot,
) (*clients.Flow2ExecuteResponse, error) {
	startTime := time.Now()

	o.logger.Info("Executing calculation with recipe snapshot",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("recipe_id", snapshot.RecipeID),
		zap.Int("resolved_steps", len(snapshot.ResolvedSteps)))

	// Create Flow2 calculation request with resolved recipe
	calcRequest := &clients.Flow2CalculateRequest{
		PatientID:        request.PatientID,
		CorrelationID:    request.CorrelationID,
		MedicationCode:   o.extractMedicationCode(request.MedicationRequest),
		ClinicalContext:  snapshot.PatientData,
		RecipeSnapshot:   snapshot,
		WorkflowID:       snapshot.WorkflowID,
	}

	// Execute calculation via Flow2 engine
	result, err := o.flow2GoClient.Execute(ctx, calcRequest)
	if err != nil {
		o.logger.Error("Flow2 calculation failed with recipe",
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.Error(err))
		return nil, fmt.Errorf("calculation failed: %w", err)
	}

	// Record calculation performance
	calculationDuration := time.Since(startTime)
	o.logger.Info("Calculation completed with recipe",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("proposal_set_id", result.ProposalSetID),
		zap.Int("proposal_count", len(result.RankedProposals)),
		zap.Duration("duration", calculationDuration))

	// Check if calculation meets performance target (110ms)
	targetDuration := 110 * time.Millisecond
	if calculationDuration > targetDuration {
		o.logger.Warn("Calculation exceeded performance target",
			zap.Duration("actual", calculationDuration),
			zap.Duration("target", targetDuration),
			zap.Float64("overage_percent", float64(calculationDuration-targetDuration)/float64(targetDuration)*100))
	}

	return result, nil
}

// extractMedicationCode extracts medication code from request
func (o *StrategicOrchestrator) extractMedicationCode(medicationRequest map[string]interface{}) string {
	if code, exists := medicationRequest["medication_code"]; exists {
		if codeStr, ok := code.(string); ok {
			return codeStr
		}
	}

	// Fallback to medication name or default
	if name, exists := medicationRequest["medication_name"]; exists {
		if nameStr, ok := name.(string); ok {
			return nameStr
		}
	}

	return "unknown_medication"
}