// Package clients provides HTTP clients for KB services.
//
// KB18HTTPClient implements the KB18Client interface for KB-18 Governance Engine Service.
// It provides clinical approval workflow management and audit trail functionality.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-18 is a RUNTIME KB - it RECEIVES classifications from CQL and ENFORCES approval workflows.
// CQL classifies (GovernanceClassifier.cql), KB-18 enforces.
//
// CRITICAL WORKFLOW:
// 1. CQL evaluates action → produces GovernanceClassification
// 2. ICU Intelligence veto check (MANDATORY - BEFORE KB-18)
// 3. KB-18 receives classification → routes to appropriate approver
// 4. KB-18 records decision → creates immutable audit trail
//
// Approval Levels:
// - LEVEL_1: Automatic (no human approval)
// - LEVEL_2: Pharmacist approval
// - LEVEL_3: Attending physician approval
// - LEVEL_4: Specialist consultation
// - LEVEL_5: Medical Director approval
// - LEVEL_6: Ethics committee (rare)
//
// Connects to: http://localhost:8098 (Docker: kb18-governance-engine)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// KB18HTTPClient implements KB18Client by calling the KB-18 Governance Engine Service REST API.
type KB18HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB18HTTPClient creates a new KB-18 HTTP client.
func NewKB18HTTPClient(baseURL string) *KB18HTTPClient {
	return &KB18HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB18HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB18HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB18HTTPClient {
	return &KB18HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB18Client Interface Implementation (RUNTIME)
// ============================================================================

// ClassifyApprovalLevel determines approval requirements for a clinical action.
// INPUT: CQL classification output from GovernanceClassifier.cql
//
// This method is called AFTER ICU veto check passes to determine human approval needs.
//
// Parameters:
// - action: The clinical action being performed (e.g., prescribe medication, order test)
// - classification: CQL-generated governance classification
//
// Returns:
// - ApprovalRequirement specifying required approval level and workflow
func (c *KB18HTTPClient) ClassifyApprovalLevel(
	ctx context.Context,
	action contracts.ClinicalAction,
	classification contracts.GovernanceClassification,
) (*contracts.ApprovalRequirement, error) {

	req := kb18ClassifyRequest{
		Action: kb18ClinicalAction{
			ActionID:   action.ActionID,
			ActionType: action.ActionType,
			PatientID:  action.PatientID,
			RequestedBy: action.RequestedBy,
			RequestedAt: action.RequestedAt,
			Details:    action.Details,
		},
		Classification: kb18Classification{
			Level:           classification.Level,
			Reason:          classification.Reason,
			RiskScore:       classification.RiskScore,
			SafetyFlags:     classification.SafetyFlags,
			TriggeredRules:  classification.TriggeredRules,
			EscalationPath:  classification.EscalationPath,
		},
	}

	resp, err := c.callKB18(ctx, "/api/v1/governance/classify", req)
	if err != nil {
		return nil, fmt.Errorf("failed to classify approval level: %w", err)
	}

	var result kb18ClassifyResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse classification response: %w", err)
	}

	return &contracts.ApprovalRequirement{
		ApprovalLevel:    result.Requirement.ApprovalLevel,
		ApproverRole:     result.Requirement.ApproverRole,
		TimeoutMinutes:   result.Requirement.TimeoutMinutes,
		EscalationPath:   result.Requirement.EscalationPath,
		RequiresReason:   result.Requirement.RequiresReason,
		AllowDelegation:  result.Requirement.AllowDelegation,
		PolicyID:         result.Requirement.PolicyID,
		PolicyVersion:    result.Requirement.PolicyVersion,
		AutoApprove:      result.Requirement.AutoApprove,
	}, nil
}

// SubmitForApproval creates an approval request in the governance workflow.
// Returns a submission record that can be tracked for decision status.
//
// CRITICAL: This creates an audit trail entry that CANNOT be modified.
func (c *KB18HTTPClient) SubmitForApproval(
	ctx context.Context,
	request contracts.ApprovalRequest,
) (*contracts.ApprovalSubmission, error) {

	req := kb18SubmitRequest{
		ActionID:       request.ActionID,
		PatientID:      request.PatientID,
		RequestedBy:    request.RequestedBy,
		RequestedAt:    request.RequestedAt,
		ApprovalLevel:  request.ApprovalLevel,
		ApproverRole:   request.ApproverRole,
		ActionDetails:  request.ActionDetails,
		Classification: request.Classification,
		Justification:  request.Justification,
		UrgencyLevel:   request.UrgencyLevel,
	}

	resp, err := c.callKB18(ctx, "/api/v1/governance/submit", req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit for approval: %w", err)
	}

	var result kb18SubmitResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse submission response: %w", err)
	}

	return &contracts.ApprovalSubmission{
		SubmissionID:   result.Submission.SubmissionID,
		ActionID:       result.Submission.ActionID,
		Status:         result.Submission.Status,
		SubmittedAt:    result.Submission.SubmittedAt,
		ExpiresAt:      &result.Submission.ExpiresAt,
		AssignedTo:     result.Submission.AssignedTo,
		EscalationTime: &result.Submission.EscalationTime,
		AuditTrailID:   result.Submission.AuditTrailID,
	}, nil
}

// GetPendingApprovals returns all approvals awaiting decision for a specific approver.
// Used for approval queues and dashboards.
//
// Parameters:
// - approverID: User ID of the approver checking their queue
func (c *KB18HTTPClient) GetPendingApprovals(
	ctx context.Context,
	approverID string,
) ([]contracts.PendingApproval, error) {

	endpoint := fmt.Sprintf("/api/v1/governance/pending/%s", approverID)
	resp, err := c.callKB18Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}

	var result kb18PendingResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pending approvals response: %w", err)
	}

	approvals := make([]contracts.PendingApproval, 0, len(result.Approvals))
	for _, a := range result.Approvals {
		expiresAt := a.ExpiresAt // Copy to take address
		approvals = append(approvals, contracts.PendingApproval{
			SubmissionID:   a.SubmissionID,
			ActionID:       a.ActionID,
			PatientID:      a.PatientID,
			PatientName:    a.PatientName,
			ActionType:     a.ActionType,
			ActionSummary:  a.ActionSummary,
			RequestedBy:    a.RequestedBy,
			RequestedAt:    a.RequestedAt,
			ApprovalLevel:  a.ApprovalLevel,
			UrgencyLevel:   a.UrgencyLevel,
			Classification: a.Classification,
			ExpiresAt:      &expiresAt,
			RiskScore:      a.RiskScore,
			SafetyFlags:    a.SafetyFlags,
		})
	}

	return approvals, nil
}

// GetPendingApprovalsByRole returns all approvals awaiting decision for a role.
// Used when approvals are assigned to roles rather than specific users.
func (c *KB18HTTPClient) GetPendingApprovalsByRole(
	ctx context.Context,
	role string,
) ([]contracts.PendingApproval, error) {

	req := kb18RolePendingRequest{
		Role: role,
	}

	resp, err := c.callKB18(ctx, "/api/v1/governance/pending-by-role", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals by role: %w", err)
	}

	var result kb18PendingResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pending approvals response: %w", err)
	}

	approvals := make([]contracts.PendingApproval, 0, len(result.Approvals))
	for _, a := range result.Approvals {
		expiresAt := a.ExpiresAt // Copy to take address
		approvals = append(approvals, contracts.PendingApproval{
			SubmissionID:   a.SubmissionID,
			ActionID:       a.ActionID,
			PatientID:      a.PatientID,
			PatientName:    a.PatientName,
			ActionType:     a.ActionType,
			ActionSummary:  a.ActionSummary,
			RequestedBy:    a.RequestedBy,
			RequestedAt:    a.RequestedAt,
			ApprovalLevel:  a.ApprovalLevel,
			UrgencyLevel:   a.UrgencyLevel,
			Classification: a.Classification,
			ExpiresAt:      &expiresAt,
			RiskScore:      a.RiskScore,
			SafetyFlags:    a.SafetyFlags,
		})
	}

	return approvals, nil
}

// RecordDecision records an approval decision with complete audit trail.
// This creates an IMMUTABLE audit record for regulatory compliance.
//
// Parameters:
// - approvalID: The submission ID being decided
// - decision: The decision (approved, denied, escalated) with rationale
//
// Returns:
// - AuditRecord containing the immutable record of the decision
func (c *KB18HTTPClient) RecordDecision(
	ctx context.Context,
	approvalID string,
	decision contracts.ApprovalDecision,
) (*contracts.AuditRecord, error) {

	req := kb18DecisionRequest{
		SubmissionID:  approvalID,
		Decision:      decision.Decision,
		DecisionBy:    decision.DecisionBy,
		DecisionAt:    time.Now().UTC(),
		Rationale:     decision.Rationale,
		Conditions:    decision.Conditions,
		DelegatedFrom: decision.DelegatedFrom,
	}

	resp, err := c.callKB18(ctx, "/api/v1/governance/decision", req)
	if err != nil {
		return nil, fmt.Errorf("failed to record decision: %w", err)
	}

	var result kb18DecisionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse decision response: %w", err)
	}

	return &contracts.AuditRecord{
		AuditID:       result.Audit.AuditID,
		SubmissionID:  result.Audit.SubmissionID,
		ActionID:      result.Audit.ActionID,
		PatientID:     result.Audit.PatientID,
		EventType:     result.Audit.EventType,
		EventTime:     result.Audit.EventTime,
		ActorID:       result.Audit.ActorID,
		ActorRole:     result.Audit.ActorRole,
		Decision:      result.Audit.Decision,
		Rationale:     result.Audit.Rationale,
		Hash:          result.Audit.Hash,
		PreviousHash:  result.Audit.PreviousHash,
		Immutable:     true,
	}, nil
}

// GetGovernancePolicy returns current governance policy definition.
// Policies define approval levels, timeout rules, and escalation paths.
func (c *KB18HTTPClient) GetGovernancePolicy(
	ctx context.Context,
	policyID string,
) (*contracts.GovernancePolicy, error) {

	endpoint := fmt.Sprintf("/api/v1/governance/policy/%s", policyID)
	resp, err := c.callKB18Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance policy: %w", err)
	}

	var result kb18PolicyResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse policy response: %w", err)
	}

	levels := make([]contracts.ApprovalLevelDefinition, 0, len(result.Policy.Levels))
	for _, l := range result.Policy.Levels {
		levels = append(levels, contracts.ApprovalLevelDefinition{
			Level:         l.Level,
			Name:          l.Name,
			Description:   l.Description,
			ApproverRoles: l.ApproverRoles,
			TimeoutMins:   l.TimeoutMins,
			EscalatesTo:   l.EscalatesTo,
		})
	}

	return &contracts.GovernancePolicy{
		PolicyID:          result.Policy.PolicyID,
		Version:           result.Policy.Version,
		Name:              result.Policy.Name,
		Description:       result.Policy.Description,
		Levels:            levels,
		DefaultTimeout:    result.Policy.DefaultTimeout,
		EscalationEnabled: result.Policy.EscalationEnabled,
		AuditRetention:    result.Policy.AuditRetention,
		EffectiveFrom:     &result.Policy.EffectiveFrom,
		EffectiveTo:       &result.Policy.EffectiveTo,
	}, nil
}

// GetAuditTrail retrieves the complete audit trail for an action.
// Used for compliance review and incident investigation.
func (c *KB18HTTPClient) GetAuditTrail(
	ctx context.Context,
	actionID string,
) ([]contracts.AuditRecord, error) {

	endpoint := fmt.Sprintf("/api/v1/governance/audit/%s", actionID)
	resp, err := c.callKB18Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit trail: %w", err)
	}

	var result kb18AuditTrailResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse audit trail response: %w", err)
	}

	records := make([]contracts.AuditRecord, 0, len(result.Records))
	for _, r := range result.Records {
		records = append(records, contracts.AuditRecord{
			AuditID:       r.AuditID,
			SubmissionID:  r.SubmissionID,
			ActionID:      r.ActionID,
			PatientID:     r.PatientID,
			EventType:     r.EventType,
			EventTime:     r.EventTime,
			ActorID:       r.ActorID,
			ActorRole:     r.ActorRole,
			Decision:      r.Decision,
			Rationale:     r.Rationale,
			Hash:          r.Hash,
			PreviousHash:  r.PreviousHash,
			Immutable:     true,
		})
	}

	return records, nil
}

// EscalateApproval manually escalates an approval to the next level.
// Used when an approver wants to defer to higher authority.
func (c *KB18HTTPClient) EscalateApproval(
	ctx context.Context,
	submissionID string,
	escalatedBy string,
	reason string,
) error {

	req := kb18EscalateRequest{
		SubmissionID: submissionID,
		EscalatedBy:  escalatedBy,
		Reason:       reason,
		EscalatedAt:  time.Now().UTC(),
	}

	_, err := c.callKB18(ctx, "/api/v1/governance/escalate", req)
	if err != nil {
		return fmt.Errorf("failed to escalate approval: %w", err)
	}

	return nil
}

// OverrideApproval allows emergency override of approval requirements.
// CRITICAL: Creates special audit trail entry for override actions.
// Use only in genuine emergencies with proper justification.
func (c *KB18HTTPClient) OverrideApproval(
	ctx context.Context,
	submissionID string,
	overrideBy string,
	reason string,
	emergencyCode string,
) (*contracts.AuditRecord, error) {

	req := kb18OverrideRequest{
		SubmissionID:  submissionID,
		OverrideBy:    overrideBy,
		Reason:        reason,
		EmergencyCode: emergencyCode,
		OverrideAt:    time.Now().UTC(),
	}

	resp, err := c.callKB18(ctx, "/api/v1/governance/override", req)
	if err != nil {
		return nil, fmt.Errorf("failed to override approval: %w", err)
	}

	var result kb18DecisionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse override response: %w", err)
	}

	return &contracts.AuditRecord{
		AuditID:      result.Audit.AuditID,
		SubmissionID: result.Audit.SubmissionID,
		ActionID:     result.Audit.ActionID,
		PatientID:    result.Audit.PatientID,
		EventType:    "EMERGENCY_OVERRIDE",
		EventTime:    result.Audit.EventTime,
		ActorID:      result.Audit.ActorID,
		ActorRole:    result.Audit.ActorRole,
		Decision:     "OVERRIDE_APPROVED",
		Rationale:    reason,
		Hash:         result.Audit.Hash,
		PreviousHash: result.Audit.PreviousHash,
		Immutable:    true,
	}, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB18HTTPClient) callKB18(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-18 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *KB18HTTPClient) callKB18Get(ctx context.Context, endpoint string) ([]byte, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-18 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-18 service is healthy.
func (c *KB18HTTPClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-18 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-18 Request/Response Types (internal)
// ============================================================================

type kb18ClassifyRequest struct {
	Action         kb18ClinicalAction `json:"action"`
	Classification kb18Classification `json:"classification"`
}

type kb18ClinicalAction struct {
	ActionID    string                 `json:"action_id"`
	ActionType  string                 `json:"action_type"`
	PatientID   string                 `json:"patient_id"`
	RequestedBy string                 `json:"requested_by"`
	RequestedAt time.Time              `json:"requested_at"`
	Details     map[string]interface{} `json:"details"`
}

type kb18Classification struct {
	Level          string   `json:"level"`
	Reason         string   `json:"reason"`
	RiskScore      float64  `json:"risk_score"`
	SafetyFlags    []string `json:"safety_flags"`
	TriggeredRules []string `json:"triggered_rules"`
	EscalationPath []string `json:"escalation_path"`
}

type kb18ClassifyResponse struct {
	Requirement kb18ApprovalRequirement `json:"requirement"`
}

type kb18ApprovalRequirement struct {
	ApprovalLevel   string   `json:"approval_level"`
	ApproverRole    string   `json:"approver_role"`
	TimeoutMinutes  int      `json:"timeout_minutes"`
	EscalationPath  []string `json:"escalation_path"`
	RequiresReason  bool     `json:"requires_reason"`
	AllowDelegation bool     `json:"allow_delegation"`
	PolicyID        string   `json:"policy_id"`
	PolicyVersion   string   `json:"policy_version"`
	AutoApprove     bool     `json:"auto_approve"`
}

type kb18SubmitRequest struct {
	ActionID       string                 `json:"action_id"`
	PatientID      string                 `json:"patient_id"`
	RequestedBy    string                 `json:"requested_by"`
	RequestedAt    time.Time              `json:"requested_at"`
	ApprovalLevel  string                 `json:"approval_level"`
	ApproverRole   string                 `json:"approver_role"`
	ActionDetails  map[string]interface{} `json:"action_details"`
	Classification string                 `json:"classification"`
	Justification  string                 `json:"justification"`
	UrgencyLevel   string                 `json:"urgency_level"`
}

type kb18SubmitResponse struct {
	Submission kb18Submission `json:"submission"`
}

type kb18Submission struct {
	SubmissionID   string    `json:"submission_id"`
	ActionID       string    `json:"action_id"`
	Status         string    `json:"status"`
	SubmittedAt    time.Time `json:"submitted_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	AssignedTo     string    `json:"assigned_to"`
	EscalationTime time.Time `json:"escalation_time"`
	AuditTrailID   string    `json:"audit_trail_id"`
}

type kb18RolePendingRequest struct {
	Role string `json:"role"`
}

type kb18PendingResponse struct {
	Approvals []kb18PendingApproval `json:"approvals"`
}

type kb18PendingApproval struct {
	SubmissionID   string    `json:"submission_id"`
	ActionID       string    `json:"action_id"`
	PatientID      string    `json:"patient_id"`
	PatientName    string    `json:"patient_name"`
	ActionType     string    `json:"action_type"`
	ActionSummary  string    `json:"action_summary"`
	RequestedBy    string    `json:"requested_by"`
	RequestedAt    time.Time `json:"requested_at"`
	ApprovalLevel  string    `json:"approval_level"`
	UrgencyLevel   string    `json:"urgency_level"`
	Classification string    `json:"classification"`
	ExpiresAt      time.Time `json:"expires_at"`
	RiskScore      float64   `json:"risk_score"`
	SafetyFlags    []string  `json:"safety_flags"`
}

type kb18DecisionRequest struct {
	SubmissionID  string    `json:"submission_id"`
	Decision      string    `json:"decision"` // approved, denied, escalated
	DecisionBy    string    `json:"decision_by"`
	DecisionAt    time.Time `json:"decision_at"`
	Rationale     string    `json:"rationale"`
	Conditions    []string  `json:"conditions"`
	DelegatedFrom string    `json:"delegated_from,omitempty"`
}

type kb18DecisionResponse struct {
	Audit kb18AuditRecord `json:"audit"`
}

type kb18AuditRecord struct {
	AuditID      string    `json:"audit_id"`
	SubmissionID string    `json:"submission_id"`
	ActionID     string    `json:"action_id"`
	PatientID    string    `json:"patient_id"`
	EventType    string    `json:"event_type"`
	EventTime    time.Time `json:"event_time"`
	ActorID      string    `json:"actor_id"`
	ActorRole    string    `json:"actor_role"`
	Decision     string    `json:"decision"`
	Rationale    string    `json:"rationale"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previous_hash"`
}

type kb18PolicyResponse struct {
	Policy kb18Policy `json:"policy"`
}

type kb18Policy struct {
	PolicyID          string                  `json:"policy_id"`
	Version           string                  `json:"version"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	Levels            []kb18ApprovalLevel     `json:"levels"`
	DefaultTimeout    int                     `json:"default_timeout"`
	EscalationEnabled bool                    `json:"escalation_enabled"`
	AuditRetention    int                     `json:"audit_retention_days"`
	EffectiveFrom     time.Time               `json:"effective_from"`
	EffectiveTo       time.Time               `json:"effective_to"`
}

type kb18ApprovalLevel struct {
	Level         string   `json:"level"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ApproverRoles []string `json:"approver_roles"`
	TimeoutMins   int      `json:"timeout_minutes"`
	EscalatesTo   string   `json:"escalates_to"`
}

type kb18AuditTrailResponse struct {
	Records []kb18AuditRecord `json:"records"`
}

type kb18EscalateRequest struct {
	SubmissionID string    `json:"submission_id"`
	EscalatedBy  string    `json:"escalated_by"`
	Reason       string    `json:"reason"`
	EscalatedAt  time.Time `json:"escalated_at"`
}

type kb18OverrideRequest struct {
	SubmissionID  string    `json:"submission_id"`
	OverrideBy    string    `json:"override_by"`
	Reason        string    `json:"reason"`
	EmergencyCode string    `json:"emergency_code"`
	OverrideAt    time.Time `json:"override_at"`
}
