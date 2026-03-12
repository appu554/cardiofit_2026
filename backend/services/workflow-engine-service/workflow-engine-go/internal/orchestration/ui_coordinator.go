package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// UICoordinator handles bidirectional communication with Apollo Federation for UI interactions
type UICoordinator struct {
	apolloGatewayURL string
	httpClient       *http.Client
	redis            *redis.Client
	logger           *zap.Logger
	activeWorkflows  map[string]*WorkflowState
}

// WorkflowState represents the current state of an interactive workflow
type WorkflowState struct {
	ID                string                 `json:"id"`
	CorrelationID     string                 `json:"correlation_id"`
	PatientID         string                 `json:"patient_id"`
	CurrentPhase      WorkflowPhase          `json:"current_phase"`
	Status            WorkflowStatus         `json:"status"`
	ValidationResult  *ValidationResult      `json:"validation_result,omitempty"`
	OverrideSession   *OverrideSession       `json:"override_session,omitempty"`
	AwaitingUIInput   bool                   `json:"awaiting_ui_input"`
	CreatedAt         time.Time              `json:"created_at"`
	ExpiresAt         time.Time              `json:"expires_at"`
	Context           map[string]interface{} `json:"context"`
}

// OverrideSession tracks an active clinical override request
type OverrideSession struct {
	ID               string             `json:"id"`
	WorkflowID       string             `json:"workflow_id"`
	ValidationID     string             `json:"validation_id"`
	Verdict          string             `json:"verdict"`
	Findings         []ValidationFinding `json:"findings"`
	RequiredLevel    OverrideLevel      `json:"required_level"`
	RequestedBy      string             `json:"requested_by"`
	RequestedAt      time.Time          `json:"requested_at"`
	ExpiresAt        time.Time          `json:"expires_at"`
	Status           OverrideStatus     `json:"status"`
}

// OverrideRequest represents a request for clinical override
type OverrideRequest struct {
	WorkflowID       string             `json:"workflow_id"`
	ValidationID     string             `json:"validation_id"`
	Verdict          string             `json:"verdict"`
	Findings         []ValidationFinding `json:"findings"`
	OverrideAllowed  bool               `json:"override_allowed"`
	RequiredLevel    OverrideLevel      `json:"required_level"`
	RequestedBy      string             `json:"requested_by"`
	Urgency          ReviewUrgency      `json:"urgency"`
}

// OverrideDecision represents a clinician's override decision
type OverrideDecision struct {
	SessionID            string                 `json:"session_id"`
	WorkflowID           string                 `json:"workflow_id"`
	Decision             string                 `json:"decision"` // OVERRIDE, MODIFY, CANCEL, DEFER
	OverrideLevel        OverrideLevel          `json:"override_level"`
	Reason               OverrideReason         `json:"reason"`
	ClinicalJustification string                `json:"clinical_justification"`
	DecidedBy            string                 `json:"decided_by"`
	CoSignature          *CoSignature           `json:"co_signature,omitempty"`
	AlternativeAction    *AlternativeAction     `json:"alternative_action,omitempty"`
}

// UINotification represents a notification sent to the UI
type UINotification struct {
	WorkflowID string      `json:"workflow_id"`
	Status     string      `json:"status"`
	Title      string      `json:"title"`
	Message    string      `json:"message"`
	Severity   string      `json:"severity"`
	Actions    []UIAction  `json:"actions"`
	Payload    interface{} `json:"payload,omitempty"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
}

// UIAction represents an action the clinician can take
type UIAction struct {
	ID      string      `json:"id"`
	Label   string      `json:"label"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Enums and types
type WorkflowPhase string
type WorkflowStatus string
type OverrideLevel string
type OverrideStatus string
type ReviewUrgency string

const (
	// Workflow Phases
	PhaseCalculate WorkflowPhase = "CALCULATE"
	PhaseValidate  WorkflowPhase = "VALIDATE"
	PhaseCommit    WorkflowPhase = "COMMIT"
	PhaseCompleted WorkflowPhase = "COMPLETED"
	PhaseFailed    WorkflowPhase = "FAILED"

	// Workflow Status
	StatusRunning           WorkflowStatus = "RUNNING"
	StatusAwaitingDecision  WorkflowStatus = "AWAITING_DECISION"
	StatusCompleted         WorkflowStatus = "COMPLETED"
	StatusFailed            WorkflowStatus = "FAILED"
	StatusCancelled         WorkflowStatus = "CANCELLED"

	// Override Levels
	OverrideLevelClinical    OverrideLevel = "CLINICAL_JUDGMENT"
	OverrideLevelPeerReview  OverrideLevel = "PEER_REVIEW"
	OverrideLevelSupervisory OverrideLevel = "SUPERVISORY"
	OverrideLevelEmergency   OverrideLevel = "EMERGENCY"

	// Override Status
	OverrideStatusPending   OverrideStatus = "PENDING"
	OverrideStatusApproved  OverrideStatus = "APPROVED"
	OverrideStatusRejected  OverrideStatus = "REJECTED"
	OverrideStatusExpired   OverrideStatus = "EXPIRED"
	OverrideStatusCancelled OverrideStatus = "CANCELLED"

	// Review Urgency
	UrgencyRoutine   ReviewUrgency = "ROUTINE"
	UrgencyUrgent    ReviewUrgency = "URGENT"
	UrgencyStat      ReviewUrgency = "STAT"
	UrgencyEmergency ReviewUrgency = "EMERGENCY"
)

// ValidationFinding represents a finding from safety validation
type ValidationFinding struct {
	FindingID            string  `json:"finding_id"`
	Severity             string  `json:"severity"`
	Category             string  `json:"category"`
	Description          string  `json:"description"`
	ClinicalSignificance string  `json:"clinical_significance"`
	Recommendation       string  `json:"recommendation"`
	Overridable          bool    `json:"overridable"`
	Evidence             interface{} `json:"evidence,omitempty"`
}

// ValidationResult contains the complete validation result
type ValidationResult struct {
	ValidationID     string              `json:"validation_id"`
	Verdict          string              `json:"verdict"`
	OverallRiskScore float64             `json:"overall_risk_score"`
	Findings         []ValidationFinding `json:"findings"`
	OverrideTokens   []string            `json:"override_tokens"`
}

type OverrideReason struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	FreeText string `json:"free_text"`
}

type CoSignature struct {
	ClinicianID string    `json:"clinician_id"`
	Role        string    `json:"role"`
	Signature   string    `json:"signature"`
	Timestamp   time.Time `json:"timestamp"`
}

type AlternativeAction struct {
	Type             string      `json:"type"`
	ModifiedProposal interface{} `json:"modified_proposal,omitempty"`
	DeferralPeriod   string      `json:"deferral_period,omitempty"`
	MonitoringPlan   interface{} `json:"monitoring_plan,omitempty"`
}

// NewUICoordinator creates a new UI coordinator instance
func NewUICoordinator(apolloGatewayURL string, redis *redis.Client, logger *zap.Logger) *UICoordinator {
	return &UICoordinator{
		apolloGatewayURL: apolloGatewayURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		redis:           redis,
		logger:          logger,
		activeWorkflows: make(map[string]*WorkflowState),
	}
}

// CreateWorkflowState creates and stores a new workflow state
func (u *UICoordinator) CreateWorkflowState(ctx context.Context, workflowID, correlationID, patientID string) (*WorkflowState, error) {
	state := &WorkflowState{
		ID:              workflowID,
		CorrelationID:   correlationID,
		PatientID:       patientID,
		CurrentPhase:    PhaseCalculate,
		Status:          StatusRunning,
		AwaitingUIInput: false,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(2 * time.Hour), // 2-hour default expiration
		Context:         make(map[string]interface{}),
	}

	// Store in Redis
	if err := u.storeWorkflowState(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to store workflow state: %w", err)
	}

	// Store in local cache
	u.activeWorkflows[workflowID] = state

	u.logger.Info("Created workflow state",
		zap.String("workflow_id", workflowID),
		zap.String("correlation_id", correlationID),
		zap.String("patient_id", patientID))

	return state, nil
}

// UpdateWorkflowState updates the workflow state
func (u *UICoordinator) UpdateWorkflowState(ctx context.Context, workflowID string, updates func(*WorkflowState)) error {
	state, err := u.GetWorkflowState(ctx, workflowID)
	if err != nil {
		return err
	}

	// Apply updates
	updates(state)

	// Store updated state
	if err := u.storeWorkflowState(ctx, state); err != nil {
		return fmt.Errorf("failed to update workflow state: %w", err)
	}

	// Update local cache
	u.activeWorkflows[workflowID] = state

	return nil
}

// GetWorkflowState retrieves workflow state
func (u *UICoordinator) GetWorkflowState(ctx context.Context, workflowID string) (*WorkflowState, error) {
	// Try local cache first
	if state, exists := u.activeWorkflows[workflowID]; exists {
		return state, nil
	}

	// Try Redis
	stateJSON, err := u.redis.Get(ctx, fmt.Sprintf("workflow:state:%s", workflowID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("workflow state not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow state: %w", err)
	}

	var state WorkflowState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow state: %w", err)
	}

	// Update local cache
	u.activeWorkflows[workflowID] = &state

	return &state, nil
}

// RequestOverride sends an override request to the UI via Apollo Federation
func (u *UICoordinator) RequestOverride(ctx context.Context, request *OverrideRequest) (*OverrideSession, error) {
	u.logger.Info("Requesting clinical override",
		zap.String("workflow_id", request.WorkflowID),
		zap.String("verdict", request.Verdict),
		zap.String("required_level", string(request.RequiredLevel)))

	// Create override session
	session := &OverrideSession{
		ID:            fmt.Sprintf("override_%d", time.Now().UnixNano()),
		WorkflowID:    request.WorkflowID,
		ValidationID:  request.ValidationID,
		Verdict:       request.Verdict,
		Findings:      request.Findings,
		RequiredLevel: request.RequiredLevel,
		RequestedBy:   request.RequestedBy,
		RequestedAt:   time.Now(),
		ExpiresAt:     calculateExpirationTime(request.Urgency),
		Status:        OverrideStatusPending,
	}

	// Store override session
	if err := u.storeOverrideSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store override session: %w", err)
	}

	// Update workflow state
	err := u.UpdateWorkflowState(ctx, request.WorkflowID, func(state *WorkflowState) {
		state.OverrideSession = session
		state.AwaitingUIInput = true
		state.Status = StatusAwaitingDecision
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update workflow state: %w", err)
	}

	// Send UI notification via Apollo Federation
	notification := &UINotification{
		WorkflowID: request.WorkflowID,
		Status:     "ACTION_REQUIRED",
		Title:      "Clinical Override Required",
		Message:    formatOverrideMessage(request),
		Severity:   determineSeverity(request.Verdict),
		Actions:    createOverrideActions(request),
		Payload: map[string]interface{}{
			"session_id":      session.ID,
			"validation_id":   request.ValidationID,
			"verdict":         request.Verdict,
			"findings":        request.Findings,
			"required_level":  request.RequiredLevel,
			"override_tokens": session.ID, // Simplified for now
		},
		ExpiresAt: &session.ExpiresAt,
	}

	if err := u.sendUINotification(ctx, notification); err != nil {
		u.logger.Error("Failed to send UI notification", zap.Error(err))
		// Continue anyway - the override session is still valid
	}

	// Publish to override required subscription
	if err := u.publishOverrideRequired(ctx, request, session); err != nil {
		u.logger.Error("Failed to publish override required event", zap.Error(err))
	}

	return session, nil
}

// ResolveOverride processes a clinician's override decision
func (u *UICoordinator) ResolveOverride(ctx context.Context, decision *OverrideDecision) error {
	u.logger.Info("Resolving clinical override",
		zap.String("session_id", decision.SessionID),
		zap.String("decision", decision.Decision),
		zap.String("decided_by", decision.DecidedBy))

	// Get override session
	session, err := u.getOverrideSession(ctx, decision.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get override session: %w", err)
	}

	// Validate decision authority
	if err := u.validateOverrideAuthority(decision); err != nil {
		return fmt.Errorf("invalid override authority: %w", err)
	}

	// Update session status
	session.Status = OverrideStatusApproved
	if err := u.storeOverrideSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update override session: %w", err)
	}

	// Update workflow state
	err = u.UpdateWorkflowState(ctx, decision.WorkflowID, func(state *WorkflowState) {
		state.AwaitingUIInput = false
		state.Status = StatusRunning
		state.CurrentPhase = PhaseCommit
		state.Context["override_decision"] = decision
	})
	if err != nil {
		return fmt.Errorf("failed to update workflow state: %w", err)
	}

	// Create audit entry
	if err := u.createOverrideAuditEntry(ctx, decision, session); err != nil {
		u.logger.Error("Failed to create audit entry", zap.Error(err))
	}

	// Send completion notification to UI
	completionNotification := &UINotification{
		WorkflowID: decision.WorkflowID,
		Status:     "COMPLETED",
		Title:      "Override Decision Processed",
		Message:    fmt.Sprintf("Override decision '%s' has been processed successfully", decision.Decision),
		Severity:   "INFO",
		Actions:    []UIAction{},
	}

	if err := u.sendUINotification(ctx, completionNotification); err != nil {
		u.logger.Error("Failed to send completion notification", zap.Error(err))
	}

	return nil
}

// Helper methods

func (u *UICoordinator) storeWorkflowState(ctx context.Context, state *WorkflowState) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("workflow:state:%s", state.ID)
	expiration := state.ExpiresAt.Sub(time.Now())
	return u.redis.Set(ctx, key, stateJSON, expiration).Err()
}

func (u *UICoordinator) storeOverrideSession(ctx context.Context, session *OverrideSession) error {
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("override:session:%s", session.ID)
	expiration := session.ExpiresAt.Sub(time.Now())
	return u.redis.Set(ctx, key, sessionJSON, expiration).Err()
}

func (u *UICoordinator) getOverrideSession(ctx context.Context, sessionID string) (*OverrideSession, error) {
	sessionJSON, err := u.redis.Get(ctx, fmt.Sprintf("override:session:%s", sessionID)).Result()
	if err != nil {
		return nil, err
	}

	var session OverrideSession
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (u *UICoordinator) sendUINotification(ctx context.Context, notification *UINotification) error {
	mutation := `
		mutation UpdateUINotification($workflowId: ID!, $notification: UINotificationInput!) {
			updateUINotification(workflowId: $workflowId, notification: $notification) {
				id
				status
			}
		}
	`

	variables := map[string]interface{}{
		"workflowId": notification.WorkflowID,
		"notification": map[string]interface{}{
			"status":   notification.Status,
			"title":    notification.Title,
			"message":  notification.Message,
			"severity": notification.Severity,
			"actions":  notification.Actions,
			"payload":  notification.Payload,
		},
	}

	return u.sendGraphQLMutation(ctx, mutation, variables)
}

func (u *UICoordinator) publishOverrideRequired(ctx context.Context, request *OverrideRequest, session *OverrideSession) error {
	// This would publish to a GraphQL subscription
	// For now, we'll use Redis pub/sub as a bridge
	overrideData := map[string]interface{}{
		"workflowId":       request.WorkflowID,
		"validationId":     request.ValidationID,
		"verdict":          request.Verdict,
		"criticalFindings": filterCriticalFindings(request.Findings),
		"overrideOptions":  getOverrideOptions(request.RequiredLevel),
		"timeoutAt":        session.ExpiresAt.Format(time.RFC3339),
		"sessionId":        session.ID,
	}

	overrideJSON, err := json.Marshal(overrideData)
	if err != nil {
		return err
	}

	return u.redis.Publish(ctx, "override-required", overrideJSON).Err()
}

func (u *UICoordinator) sendGraphQLMutation(ctx context.Context, mutation string, variables map[string]interface{}) error {
	requestBody := map[string]interface{}{
		"query":     mutation,
		"variables": variables,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.apolloGatewayURL+"/graphql", bytes.NewBuffer(bodyJSON))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d", resp.StatusCode)
	}

	return nil
}

func (u *UICoordinator) validateOverrideAuthority(decision *OverrideDecision) error {
	// Implement authority validation logic based on override level
	// This would check the clinician's role against required authority level
	return nil
}

func (u *UICoordinator) createOverrideAuditEntry(ctx context.Context, decision *OverrideDecision, session *OverrideSession) error {
	auditEntry := map[string]interface{}{
		"action":       "OVERRIDE_APPROVED",
		"workflow_id":  decision.WorkflowID,
		"session_id":   decision.SessionID,
		"decided_by":   decision.DecidedBy,
		"decision":     decision.Decision,
		"reason":       decision.Reason,
		"justification": decision.ClinicalJustification,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	auditJSON, err := json.Marshal(auditEntry)
	if err != nil {
		return err
	}

	// Store in audit trail
	auditKey := fmt.Sprintf("audit:%s", decision.WorkflowID)
	return u.redis.LPush(ctx, auditKey, auditJSON).Err()
}

// Utility functions

func calculateExpirationTime(urgency ReviewUrgency) time.Time {
	var duration time.Duration
	switch urgency {
	case UrgencyEmergency:
		duration = 5 * time.Minute
	case UrgencyStat:
		duration = 15 * time.Minute
	case UrgencyUrgent:
		duration = 1 * time.Hour
	case UrgencyRoutine:
		duration = 4 * time.Hour
	default:
		duration = 4 * time.Hour
	}
	return time.Now().Add(duration)
}

func formatOverrideMessage(request *OverrideRequest) string {
	criticalCount := 0
	for _, finding := range request.Findings {
		if finding.Severity == "CRITICAL" {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		return fmt.Sprintf("Validation found %d critical findings requiring clinical review before proceeding", criticalCount)
	}

	return fmt.Sprintf("Validation verdict '%s' requires clinical override decision", request.Verdict)
}

func determineSeverity(verdict string) string {
	switch verdict {
	case "UNSAFE":
		return "ERROR"
	case "WARNING":
		return "WARNING"
	default:
		return "INFO"
	}
}

func createOverrideActions(request *OverrideRequest) []UIAction {
	actions := []UIAction{
		{
			ID:    "cancel",
			Label: "Cancel",
			Type:  "CANCEL",
		},
	}

	if request.OverrideAllowed {
		actions = append(actions, UIAction{
			ID:    "override",
			Label: "Override",
			Type:  "OVERRIDE",
			Payload: map[string]interface{}{
				"required_level": request.RequiredLevel,
			},
		})
	}

	return actions
}

func filterCriticalFindings(findings []ValidationFinding) []ValidationFinding {
	var critical []ValidationFinding
	for _, finding := range findings {
		if finding.Severity == "CRITICAL" {
			critical = append(critical, finding)
		}
	}
	return critical
}

func getOverrideOptions(requiredLevel OverrideLevel) []map[string]interface{} {
	// Return available override options based on required level
	return []map[string]interface{}{
		{
			"level":       string(requiredLevel),
			"available":   true,
			"requirements": []string{"Clinical justification required"},
		},
	}
}