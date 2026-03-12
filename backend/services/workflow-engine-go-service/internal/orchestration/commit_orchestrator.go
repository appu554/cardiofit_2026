package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/pkg/clients"
)

// CommitOrchestrator implements the complete Commit Phase flow from document 13_9.1
type CommitOrchestrator struct {
	medicationClient    clients.MedicationServiceClient
	uiCoordinator      *UICoordinator
	overrideManager    *OverrideManager
	safetyMatrix       *SafetyDecisionMatrix
	kafkaProducer      KafkaProducerInterface
	rollbackManager    *RollbackManager
	batchProcessor     *BatchProcessor
	redisClient        *redis.Client
	logger             *zap.Logger
}

// KafkaProducerInterface defines the interface for Kafka event publishing
type KafkaProducerInterface interface {
	PublishOverrideEvent(ctx context.Context, topic string, event *OverrideEvent) error
	PublishCommitEvent(ctx context.Context, topic string, event *CommitEvent) error
}

// CommitRequest represents the input for commit phase orchestration
type CommitRequest struct {
	ProposalID        string                  `json:"proposal_id"`
	PatientID         string                  `json:"patient_id"`
	WorkflowID        string                  `json:"workflow_id"`
	CorrelationID     string                  `json:"correlation_id"`
	ValidationResult  *CommitValidationResult `json:"validation_result"`
	SelectedProposal  *CommitProposal         `json:"selected_proposal"`
	ProviderContext   *CommitProviderContext  `json:"provider_context"`
	UIInteractionMode string                  `json:"ui_interaction_mode,omitempty"`
	SafetyVerdict     SafetyVerdict           `json:"safety_verdict,omitempty"`
	ClinicalContext   map[string]interface{}  `json:"clinical_context,omitempty"`
	RequestedBy       string                  `json:"requested_by,omitempty"`
	SessionID         string                  `json:"session_id,omitempty"`
	BatchID           string                  `json:"batch_id,omitempty"`
}

// CommitValidationResult contains validation results for commit decision
type CommitValidationResult struct {
	ValidationID     string          `json:"validation_id"`
	Verdict          string          `json:"verdict"`
	RiskScore        float64         `json:"risk_score"`
	Findings         []CommitFinding `json:"findings"`
	Evidence         []EvidenceItem  `json:"evidence"`
	OverrideAllowed  bool            `json:"override_allowed"`
}

// CommitFinding represents a single validation finding
type CommitFinding struct {
	ID             string                 `json:"id"`
	RuleID         string                 `json:"rule_id"`
	Severity       string                 `json:"severity"`
	Category       string                 `json:"category"`
	Description    string                 `json:"description"`
	Evidence       map[string]interface{} `json:"evidence,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
}

// EvidenceItem represents a piece of evidence in the audit trail
type EvidenceItem struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

// CommitProposal represents the selected medication proposal
type CommitProposal struct {
	ProposalID      string                 `json:"proposal_id"`
	MedicationCode  string                 `json:"medication_code"`
	MedicationName  string                 `json:"medication_name"`
	Dosage          string                 `json:"dosage"`
	Frequency       string                 `json:"frequency"`
	Route           string                 `json:"route"`
	Duration        string                 `json:"duration"`
	Instructions    string                 `json:"instructions"`
	Ranking         float64                `json:"ranking"`
	Confidence      float64                `json:"confidence"`
	Rationale       string                 `json:"rationale"`
	ProposalData    map[string]interface{} `json:"proposal_data,omitempty"`
}

// CommitProviderContext contains provider and encounter information
type CommitProviderContext struct {
	ProviderID   string    `json:"provider_id"`
	EncounterID  string    `json:"encounter_id,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// CommitResponse represents the result of commit phase orchestration
type CommitResponse struct {
	CommitID           string                 `json:"commit_id"`
	Status             string                 `json:"status"`
	Result             CommitResult           `json:"result"`
	MedicationOrderID  string                 `json:"medication_order_id,omitempty"`
	FHIRResourceID     string                 `json:"fhir_resource_id,omitempty"`
	AuditTrailID       string                 `json:"audit_trail_id,omitempty"`
	CommittedAt        time.Time              `json:"committed_at"`
	UINotification     *UINotification        `json:"ui_notification,omitempty"`
	OverrideSession    *OverrideSession       `json:"override_session,omitempty"`
	RollbackToken      string                 `json:"rollback_token,omitempty"`
	RollbackExpiresAt  time.Time              `json:"rollback_expires_at,omitempty"`
	AuditTrail         []AuditEntry           `json:"audit_trail"`
	ExecutionMetrics   *CommitMetrics         `json:"execution_metrics"`
}

// Enums and types
type SafetyVerdict string

const (
	SafetyVerdictSafe    SafetyVerdict = "SAFE"
	SafetyVerdictUnsafe  SafetyVerdict = "UNSAFE"
	SafetyVerdictWarning SafetyVerdict = "WARNING"
)

type CommitStatus string

const (
	CommitStatusPending         CommitStatus = "PENDING"
	CommitStatusCommitted       CommitStatus = "COMMITTED"
	CommitStatusAwaitingOverride CommitStatus = "AWAITING_OVERRIDE"
	CommitStatusOverridden      CommitStatus = "OVERRIDDEN"
	CommitStatusCancelled       CommitStatus = "CANCELLED"
	CommitStatusFailed          CommitStatus = "FAILED"
	CommitStatusRolledBack      CommitStatus = "ROLLED_BACK"
)

type CommitResult string

const (
	CommitResultSuccess           CommitResult = "SUCCESS"
	CommitResultUserActionRequired CommitResult = "USER_ACTION_REQUIRED"
	CommitResultFailed            CommitResult = "FAILED"
	CommitResultRolledBack        CommitResult = "ROLLED_BACK"
)

// Event structures for Kafka
type OverrideEvent struct {
	EventID           string                 `json:"event_id"`
	WorkflowID        string                 `json:"workflow_id"`
	ProposalID        string                 `json:"proposal_id"`
	OverriddenBy      string                 `json:"overridden_by"`
	OverrideLevel     OverrideLevel          `json:"override_level"`
	OverrideReason    string                 `json:"override_reason"`
	OriginalVerdict   SafetyVerdict          `json:"original_verdict"`
	SafetyFindings    []Finding              `json:"safety_findings"`
	ClinicalContext   map[string]interface{} `json:"clinical_context"`
	Timestamp         time.Time              `json:"timestamp"`
	LearningLoopData  map[string]interface{} `json:"learning_loop_data"`
}

type CommitEvent struct {
	EventID         string                 `json:"event_id"`
	WorkflowID      string                 `json:"workflow_id"`
	ProposalID      string                 `json:"proposal_id"`
	CommitType      string                 `json:"commit_type"` // DIRECT, OVERRIDE, BATCH
	Status          CommitStatus           `json:"status"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	Timestamp       time.Time              `json:"timestamp"`
}

type AuditEntry struct {
	EntryID     string                 `json:"entry_id"`
	Action      string                 `json:"action"`
	Actor       string                 `json:"actor"`
	Context     map[string]interface{} `json:"context"`
	Timestamp   time.Time              `json:"timestamp"`
	Signature   string                 `json:"signature,omitempty"`
}

type CommitMetrics struct {
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	TotalDuration    time.Duration `json:"total_duration"`
	VerdictTime      time.Duration `json:"verdict_processing_time"`
	CommitTime       time.Duration `json:"commit_operation_time"`
	UIInteractionTime time.Duration `json:"ui_interaction_time,omitempty"`
	PhaseBreakdown   map[string]time.Duration `json:"phase_breakdown"`
}

// NewCommitOrchestrator creates a new commit orchestrator following document 13_9.1
func NewCommitOrchestrator(
	medicationClient clients.MedicationServiceClient,
	uiCoordinator *UICoordinator,
	overrideManager *OverrideManager,
	safetyMatrix *SafetyDecisionMatrix,
	kafkaProducer KafkaProducerInterface,
	redisClient *redis.Client,
	logger *zap.Logger,
) *CommitOrchestrator {
	return &CommitOrchestrator{
		medicationClient: medicationClient,
		uiCoordinator:    uiCoordinator,
		overrideManager:  overrideManager,
		safetyMatrix:     safetyMatrix,
		kafkaProducer:    kafkaProducer,
		rollbackManager:  NewRollbackManager(redisClient, logger),
		batchProcessor:   NewBatchProcessor(logger),
		redisClient:      redisClient,
		logger:           logger,
	}
}

// ExecuteCommitPhase orchestrates the complete commit phase following document 13_9.1 sequence
func (c *CommitOrchestrator) ExecuteCommitPhase(ctx context.Context, request *CommitRequest) (*CommitResponse, error) {
	startTime := time.Now()
	commitID := c.generateCommitID(request.WorkflowID, request.ProposalID)

	c.logger.Info("Starting commit phase orchestration",
		zap.String("commit_id", commitID),
		zap.String("workflow_id", request.WorkflowID),
		zap.String("proposal_id", request.ProposalID),
		zap.String("verdict", string(request.SafetyVerdict)))

	// Create audit trail
	auditTrail := []AuditEntry{{
		EntryID:   fmt.Sprintf("%s_start", commitID),
		Action:    "COMMIT_PHASE_STARTED",
		Actor:     request.RequestedBy,
		Context:   map[string]interface{}{"verdict": request.SafetyVerdict},
		Timestamp: startTime,
	}}

	// Step 2: Handle Verdict (Branching Logic) - Following document 13_9.1 sequence
	switch request.SafetyVerdict {
	case SafetyVerdictSafe:
		return c.handleSafeVerdict(ctx, request, commitID, startTime, auditTrail)
	case SafetyVerdictUnsafe, SafetyVerdictWarning:
		return c.handleUnsafeWarningVerdict(ctx, request, commitID, startTime, auditTrail)
	default:
		return c.handleUnknownVerdict(ctx, request, commitID, startTime, auditTrail)
	}
}

// handleSafeVerdict implements the SAFE path: immediate commit (Steps 3a-6a from document)
func (c *CommitOrchestrator) handleSafeVerdict(
	ctx context.Context,
	request *CommitRequest,
	commitID string,
	startTime time.Time,
	auditTrail []AuditEntry,
) (*CommitResponse, error) {
	c.logger.Info("Handling SAFE verdict - proceeding with immediate commit",
		zap.String("commit_id", commitID))

	// Step 3a: Commit(proposal_id) - Call Medication Service
	commitStartTime := time.Now()
	commitResult, err := c.medicationClient.Commit(ctx, &clients.MedicationCommitRequest{
		ProposalSetID:    request.ProposalID,
		ValidationID:     request.ValidationResult.ValidationID,
		SelectedProposal: request.ClinicalContext,
		ProviderDecision: map[string]interface{}{
			"commit_id":    commitID,
			"requested_by": request.RequestedBy,
		},
		CorrelationID: request.CorrelationID,
		PatientID:     request.PatientID,
		CommitMode:    "immediate",
		ProviderID:    request.ProviderContext.ProviderID,
		EncounterID:   request.ProviderContext.EncounterID,
	})

	commitDuration := time.Since(commitStartTime)

	// Step 4a: Handle commit response
	if err != nil {
		c.logger.Error("Medication service commit failed", zap.Error(err))
		return c.buildFailureResponse(commitID, startTime, auditTrail, err)
	}

	// Add audit entry for successful commit
	auditTrail = append(auditTrail, AuditEntry{
		EntryID:   fmt.Sprintf("%s_commit", commitID),
		Action:    "PROPOSAL_COMMITTED",
		Actor:     request.RequestedBy,
		Context:   map[string]interface{}{"commit_result": commitResult},
		Timestamp: time.Now(),
	})

	// Step 5a: GraphQL Mutation: updateUINotification (SUCCESS)
	notification := &UINotification{
		WorkflowID: request.WorkflowID,
		Status:     "SAVED",
		Title:      "Medication Proposal Committed",
		Message:    "Proposal has been successfully saved and committed to the medication system.",
		Severity:   "SUCCESS",
		Actions:    []UIAction{},
	}

	// Send notification to UI via Apollo Federation
	// TODO: Implement SendNotification method
	if false { // c.uiCoordinator.SendNotification(ctx, notification); err != nil {
		c.logger.Warn("Failed to send success notification to UI", zap.Error(err))
		// Don't fail the entire operation for notification failure
	}

	// Create rollback token (5-minute window as per document 13_9.2)
	rollbackToken, rollbackExpiry := c.rollbackManager.CreateRollbackToken(ctx, commitID, request.ProposalID)

	// Publish commit event to Kafka for analytics
	commitEvent := &CommitEvent{
		EventID:         fmt.Sprintf("%s_commit", commitID),
		WorkflowID:      request.WorkflowID,
		ProposalID:      request.ProposalID,
		CommitType:      "DIRECT",
		Status:          CommitStatusCommitted,
		ExecutionTime:   commitDuration,
		ClinicalContext: request.ClinicalContext,
		Timestamp:       time.Now(),
	}

	if err := c.kafkaProducer.PublishCommitEvent(ctx, "medication-commits", commitEvent); err != nil {
		c.logger.Warn("Failed to publish commit event to Kafka", zap.Error(err))
		// Don't fail for Kafka publishing issues
	}

	// Build successful response
	totalDuration := time.Since(startTime)
	return &CommitResponse{
		CommitID:          commitID,
		Status:            string(CommitStatusCommitted),
		Result:            CommitResultSuccess,
		UINotification:    notification,
		RollbackToken:     rollbackToken,
		RollbackExpiresAt: rollbackExpiry,
		AuditTrail:        auditTrail,
		ExecutionMetrics: &CommitMetrics{
			StartTime:     startTime,
			EndTime:       time.Now(),
			TotalDuration: totalDuration,
			CommitTime:    commitDuration,
			PhaseBreakdown: map[string]time.Duration{
				"verdict_processing": 0, // No processing needed for SAFE
				"commit_operation":   commitDuration,
				"notification":       totalDuration - commitDuration,
			},
		},
	}, nil
}

// handleUnsafeWarningVerdict implements UNSAFE/WARNING path: UI interaction (Steps 3b-13b from document)
func (c *CommitOrchestrator) handleUnsafeWarningVerdict(
	ctx context.Context,
	request *CommitRequest,
	commitID string,
	startTime time.Time,
	auditTrail []AuditEntry,
) (*CommitResponse, error) {
	c.logger.Info("Handling UNSAFE/WARNING verdict - initiating UI interaction flow",
		zap.String("commit_id", commitID),
		zap.String("verdict", string(request.SafetyVerdict)))

	// Determine if override is allowed based on safety findings
	overrideAllowed := c.determineOverrideEligibility(request.ValidationResult, request.SafetyVerdict)

	// Step 3b: GraphQL Mutation: updateUINotification (ACTION_REQUIRED)
	notification := &UINotification{
		WorkflowID: request.WorkflowID,
		Status:     "ACTION_REQUIRED",
		Title:      c.buildNotificationTitle(request.SafetyVerdict),
		Message:    c.buildNotificationMessage(request.SafetyVerdict, request.ValidationResult),
		Severity:   c.mapVerdictToSeverity(request.SafetyVerdict),
		Actions:    c.buildUIActions(overrideAllowed),
		Payload: map[string]interface{}{
			"verdict":          request.SafetyVerdict,
			"evidence":         request.ValidationResult.Findings,
			"override_allowed": overrideAllowed,
		},
	}

	// Send notification to UI
	// TODO: Implement SendNotification method
	if false { // c.uiCoordinator.SendNotification(ctx, notification); err != nil {
		c.logger.Error("Failed to send action required notification")
		return c.buildFailureResponse(commitID, startTime, auditTrail, fmt.Errorf("notification failed"))
	}

	// Create override session for tracking
	overrideSession := &OverrideSession{
		SessionID:        fmt.Sprintf("%s_override", commitID),
		WorkflowID:       request.WorkflowID,
		ValidationID:     request.ValidationResult.ValidationID,
		RequiredLevel:    c.determineRequiredOverrideLevel(request.ValidationResult),
		ClinicianID:      request.RequestedBy,
		Status:           "PENDING",
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(30 * time.Minute), // 30-minute override decision timeout
		ValidationFindings: convertCommitFindingsToInterface(request.ValidationResult.Findings),
	}

	// Store override session
	if err := c.overrideManager.CreateOverrideSession(ctx, overrideSession); err != nil {
		c.logger.Error("Failed to create override session", zap.Error(err))
		return c.buildFailureResponse(commitID, startTime, auditTrail, err)
	}

	// Add audit entry for override session creation
	auditTrail = append(auditTrail, AuditEntry{
		EntryID: fmt.Sprintf("%s_override_session", commitID),
		Action:  "OVERRIDE_SESSION_CREATED",
		Actor:   request.RequestedBy,
		Context: map[string]interface{}{
			"session_id":       overrideSession.SessionID,
			"override_allowed": overrideAllowed,
			"required_level":   overrideSession.RequiredLevel,
		},
		Timestamp: time.Now(),
	})

	// Return response indicating user action is required
	return &CommitResponse{
		CommitID:        commitID,
		Status:          string(CommitStatusAwaitingOverride),
		Result:          CommitResultUserActionRequired,
		UINotification:  notification,
		OverrideSession: overrideSession,
		AuditTrail:      auditTrail,
		ExecutionMetrics: &CommitMetrics{
			StartTime:     startTime,
			EndTime:       time.Now(),
			TotalDuration: time.Since(startTime),
			PhaseBreakdown: map[string]time.Duration{
				"verdict_processing": time.Since(startTime),
				"ui_notification":    0, // Async operation
			},
		},
	}, nil
}

// HandleOverrideDecision processes clinician's override decision (Steps 6b-13b from document)
func (c *CommitOrchestrator) HandleOverrideDecision(ctx context.Context, decision *OverrideDecision) (*CommitResponse, error) {
	startTime := time.Now()

	c.logger.Info("Processing override decision",
		zap.String("session_id", decision.SessionID),
		zap.String("decision", decision.Decision),
		zap.String("decided_by", decision.DecidedBy))

	// Retrieve override session
	session, err := c.overrideManager.GetOverrideSession(ctx, decision.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve override session: %w", err)
	}

	auditTrail := []AuditEntry{{
		EntryID: fmt.Sprintf("%s_decision", decision.SessionID),
		Action:  "OVERRIDE_DECISION_RECEIVED",
		Actor:   decision.DecidedBy,
		Context: map[string]interface{}{
			"decision": decision.Decision,
			"reason":   decision.ClinicalJustification,
		},
		Timestamp: startTime,
	}}

	switch decision.Decision {
	case "OVERRIDE":
		return c.handleOverrideCommit(ctx, session, decision, startTime, auditTrail)
	case "CANCEL":
		return c.handleOverrideCancel(ctx, session, decision, startTime, auditTrail)
	default:
		return nil, fmt.Errorf("unknown override decision: %s", decision.Decision)
	}
}

// Helper methods

func (c *CommitOrchestrator) handleOverrideCommit(
	ctx context.Context,
	session *OverrideSession,
	decision *OverrideDecision,
	startTime time.Time,
	auditTrail []AuditEntry,
) (*CommitResponse, error) {
	c.logger.Info("Processing override commit", zap.String("session_id", session.SessionID))

	// Step 9b: Commit(proposal_id, override_details)
	commitStartTime := time.Now()
	_, err := c.medicationClient.CommitWithOverride(ctx, &clients.CommitWithOverrideRequest{
		ProposalID: session.ValidationID, // Using validation ID as proposal reference
		WorkflowID: session.WorkflowID,
		OverrideDetails: &clients.OverrideDetails{
			OverriddenBy:         decision.DecidedBy,
			OverrideReason:       decision.ClinicalJustification,
			OverrideLevel:        string(decision.OverrideLevel),
			OriginalVerdict:      "UNKNOWN", // We'll add this field later if needed
			CoSignature:          convertToClientCoSignature(decision.CoSignature),
			AlternativeAction:    convertAlternativeActionToString(decision.AlternativeAction),
		},
	})

	commitDuration := time.Since(commitStartTime)

	if err != nil {
		c.logger.Error("Override commit failed", zap.Error(err))
		return c.buildFailureResponse(session.SessionID, startTime, auditTrail, err)
	}

	// Step 11b: GraphQL Mutation: updateUINotification (SUCCESS)
	notification := &UINotification{
		WorkflowID: session.WorkflowID,
		Status:     "SAVED",
		Title:      "Medication Proposal Overridden & Committed",
		Message:    fmt.Sprintf("Proposal has been successfully overridden and committed. Override reason: %s", decision.ClinicalJustification),
		Severity:   "SUCCESS",
	}

	// TODO: Implement SendNotification method
	if false { // c.uiCoordinator.SendNotification(ctx, notification); err != nil {
		c.logger.Warn("Failed to send override success notification", zap.Error(err))
	}

	// Step 13b: Publish OverrideEvent to 'clinical-overrides' topic (Learning Loop)
	overrideEvent := &OverrideEvent{
		EventID:         fmt.Sprintf("%s_override", session.SessionID),
		WorkflowID:      session.WorkflowID,
		ProposalID:      session.ValidationID,
		OverriddenBy:    decision.DecidedBy,
		OverrideLevel:   decision.OverrideLevel,
		OverrideReason:  decision.ClinicalJustification,
		OriginalVerdict: "UNKNOWN", // Will be populated from validation findings
		SafetyFindings:  []Finding{}, // Will convert ValidationFindings later if needed
		Timestamp:       time.Now(),
		LearningLoopData: map[string]interface{}{
			"override_justification": decision.ClinicalJustification,
			"clinical_context":      decision.AlternativeAction,
			"co_signature_required": decision.CoSignature != nil,
		},
	}

	if err := c.kafkaProducer.PublishOverrideEvent(ctx, "clinical-overrides", overrideEvent); err != nil {
		c.logger.Error("Failed to publish override event to learning loop", zap.Error(err))
		// Continue execution - learning loop failure shouldn't fail the commit
	}

	// Update override session status
	session.Status = "APPROVED"
	// TODO: Implement UpdateOverrideSession method
	// c.overrideManager.UpdateOverrideSession(ctx, session)

	// Create rollback token
	rollbackToken, rollbackExpiry := c.rollbackManager.CreateRollbackToken(ctx, session.SessionID, session.ValidationID)

	return &CommitResponse{
		CommitID:          session.SessionID,
		Status:            string(CommitStatusOverridden),
		Result:            CommitResultSuccess,
		UINotification:    notification,
		RollbackToken:     rollbackToken,
		RollbackExpiresAt: rollbackExpiry,
		AuditTrail:        auditTrail,
		ExecutionMetrics: &CommitMetrics{
			StartTime:     startTime,
			EndTime:       time.Now(),
			TotalDuration: time.Since(startTime),
			CommitTime:    commitDuration,
		},
	}, nil
}

func (c *CommitOrchestrator) handleOverrideCancel(
	ctx context.Context,
	session *OverrideSession,
	decision *OverrideDecision,
	startTime time.Time,
	auditTrail []AuditEntry,
) (*CommitResponse, error) {
	c.logger.Info("Processing override cancellation", zap.String("session_id", session.SessionID))

	// TODO: Implement Cancel method and full cancellation logic
	return c.buildFailureResponse(session.SessionID, startTime, auditTrail, fmt.Errorf("override cancellation not implemented"))
}

// Utility methods

func (c *CommitOrchestrator) generateCommitID(workflowID, proposalID string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("commit_%s_%s_%d", workflowID[:8], proposalID[:8], timestamp)
}

func (c *CommitOrchestrator) determineOverrideEligibility(validationResult *CommitValidationResult, verdict SafetyVerdict) bool {
	// Override allowed for warnings, but not for absolute contraindications
	if verdict == SafetyVerdictWarning {
		return true
	}

	// Check for absolute contraindications in findings
	for _, finding := range validationResult.Findings {
		if finding.Category == "CONTRAINDICATION" || finding.Category == "LIFE_THREATENING" {
			return false
		}
	}

	return true
}

func (c *CommitOrchestrator) determineRequiredOverrideLevel(validationResult *CommitValidationResult) OverrideLevel {
	highSeverityCount := 0
	for _, finding := range validationResult.Findings {
		if finding.Severity == "HIGH" || finding.Severity == "CRITICAL" {
			highSeverityCount++
		}
	}

	// Escalate override level based on finding severity
	if highSeverityCount >= 3 {
		return OverrideLevelSupervisory
	} else if highSeverityCount >= 1 {
		return OverrideLevelPeerReview
	}
	return OverrideLevelClinicalJudgment
}

func (c *CommitOrchestrator) buildNotificationTitle(verdict SafetyVerdict) string {
	switch verdict {
	case SafetyVerdictUnsafe:
		return "Safety Alert: Unsafe Medication Proposal"
	case SafetyVerdictWarning:
		return "Safety Warning: Review Required"
	default:
		return "Safety Review Required"
	}
}

func (c *CommitOrchestrator) buildNotificationMessage(verdict SafetyVerdict, validation *CommitValidationResult) string {
	findingCount := len(validation.Findings)
	switch verdict {
	case SafetyVerdictUnsafe:
		return fmt.Sprintf("Unsafe medication proposal detected with %d safety findings. Clinical review and possible override required.", findingCount)
	case SafetyVerdictWarning:
		return fmt.Sprintf("Medication proposal has %d warning(s). Please review and decide whether to proceed.", findingCount)
	default:
		return "Safety review required before proceeding."
	}
}

func (c *CommitOrchestrator) mapVerdictToSeverity(verdict SafetyVerdict) string {
	switch verdict {
	case SafetyVerdictUnsafe:
		return "ERROR"
	case SafetyVerdictWarning:
		return "WARNING"
	default:
		return "INFO"
	}
}

func (c *CommitOrchestrator) buildUIActions(overrideAllowed bool) []UIAction {
	actions := []UIAction{{
		ID:    "cancel",
		Label: "Cancel",
		Type:  "button",
	}}

	if overrideAllowed {
		actions = append(actions, UIAction{
			ID:    "override",
			Label: "Override & Proceed",
			Type:  "button",
		})
	}

	return actions
}

func (c *CommitOrchestrator) buildFailureResponse(commitID string, startTime time.Time, auditTrail []AuditEntry, err error) (*CommitResponse, error) {
	auditTrail = append(auditTrail, AuditEntry{
		EntryID:   fmt.Sprintf("%s_failure", commitID),
		Action:    "COMMIT_FAILED",
		Context:   map[string]interface{}{"error": err.Error()},
		Timestamp: time.Now(),
	})

	return &CommitResponse{
		CommitID: commitID,
		Status:   string(CommitStatusFailed),
		Result:   CommitResultFailed,
		AuditTrail: auditTrail,
		ExecutionMetrics: &CommitMetrics{
			StartTime:     startTime,
			EndTime:       time.Now(),
			TotalDuration: time.Since(startTime),
		},
	}, err
}

func (c *CommitOrchestrator) handleUnknownVerdict(ctx context.Context, request *CommitRequest, commitID string, startTime time.Time, auditTrail []AuditEntry) (*CommitResponse, error) {
	err := fmt.Errorf("unknown safety verdict: %s", request.SafetyVerdict)
	c.logger.Error("Unknown safety verdict received", zap.String("verdict", string(request.SafetyVerdict)))
	return c.buildFailureResponse(commitID, startTime, auditTrail, err)
}

// convertCommitFindingsToInterface converts CommitFindings to interface{} slice
func convertCommitFindingsToInterface(findings []CommitFinding) []interface{} {
	result := make([]interface{}, len(findings))
	for i, finding := range findings {
		result[i] = finding
	}
	return result
}

// convertToClientCoSignature converts CoSignature to clients.CoSignatureDetails
func convertToClientCoSignature(coSig *CoSignature) *clients.CoSignatureDetails {
	if coSig == nil {
		return nil
	}
	return &clients.CoSignatureDetails{
		CoSignedBy:    "unknown", // TODO: Map actual CoSignature fields
		CoSignedAt:    time.Now(),
		CoSignerLevel: "unknown",
	}
}

// convertAlternativeActionToString converts AlternativeAction to string
func convertAlternativeActionToString(altAction *AlternativeAction) string {
	if altAction == nil {
		return ""
	}
	return "alternative_action" // TODO: Map actual AlternativeAction fields
}