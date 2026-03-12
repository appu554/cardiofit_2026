// Package clients provides HTTP clients for KB services.
//
// KB14HTTPClient implements the KB14Client interface for KB-14 Care Navigator Service.
// It provides workflow orchestration, task management, and care coordination capabilities.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-14 is a RUNTIME KB - called during workflow execution, NOT during snapshot build.
// It orchestrates clinical workflows and manages tasks assigned to care team members.
//
// Workflow Pattern:
// 1. Care gaps identified by KB-9 → trigger workflow initiation
// 2. KB-14 creates workflow instances → assigns tasks to appropriate roles
// 3. Tasks advance through workflow steps → update patient care status
//
// ICU DOMINANCE: KB-14 workflows can be preempted by ICU Intelligence when
// safety conditions require immediate clinical override.
//
// Connects to: http://localhost:8094 (Docker: kb14-care-navigator)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ============================================================================
// WORKFLOW CONTRACT TYPES (KB-14 Specific)
// ============================================================================

// Workflow represents a workflow definition template.
// Defines the structure and steps of a clinical care workflow.
type Workflow struct {
	// WorkflowID unique identifier for the workflow definition
	WorkflowID string `json:"workflowId"`

	// Name human-readable workflow name
	Name string `json:"name"`

	// Description of the workflow purpose
	Description string `json:"description"`

	// WorkflowType categorizes the workflow (e.g., "CARE_GAP_CLOSURE", "TRANSITION_OF_CARE")
	WorkflowType string `json:"workflowType"`

	// Status of the workflow definition (ACTIVE, DRAFT, DEPRECATED)
	Status string `json:"status"`

	// Steps defines the ordered sequence of workflow steps
	Steps []WorkflowStepDefinition `json:"steps"`

	// TriggeredBy conditions that initiate this workflow
	TriggeredBy []WorkflowTrigger `json:"triggeredBy,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"createdAt"`
}

// WorkflowStepDefinition defines a single step within a workflow.
type WorkflowStepDefinition struct {
	// StepID unique identifier within the workflow
	StepID string `json:"stepId"`

	// StepName human-readable step name
	StepName string `json:"stepName"`

	// StepOrder in the sequence (1-based)
	StepOrder int `json:"stepOrder"`

	// StepType (TASK, DECISION, WAIT, NOTIFICATION, SYSTEM)
	StepType string `json:"stepType"`

	// AssigneeRole who should perform this step (e.g., "NURSE", "PHYSICIAN", "CARE_MANAGER")
	AssigneeRole string `json:"assigneeRole,omitempty"`

	// RequiredActions to complete the step
	RequiredActions []string `json:"requiredActions,omitempty"`

	// TimeoutHours before escalation
	TimeoutHours int `json:"timeoutHours,omitempty"`

	// NextSteps possible transitions
	NextSteps []string `json:"nextSteps,omitempty"`
}

// WorkflowTrigger defines conditions that initiate a workflow.
type WorkflowTrigger struct {
	// TriggerType (CARE_GAP, CONDITION, EVENT, SCHEDULE)
	TriggerType string `json:"triggerType"`

	// TriggerCondition the specific condition
	TriggerCondition string `json:"triggerCondition"`

	// Priority for trigger evaluation
	Priority int `json:"priority"`
}

// WorkflowInstance represents an active instance of a workflow for a patient.
type WorkflowInstance struct {
	// InstanceID unique identifier for this workflow instance
	InstanceID string `json:"instanceId"`

	// WorkflowID reference to the workflow definition
	WorkflowID string `json:"workflowId"`

	// WorkflowName for display purposes
	WorkflowName string `json:"workflowName"`

	// PatientID the patient this workflow is for
	PatientID string `json:"patientId"`

	// Status of the instance (ACTIVE, COMPLETED, CANCELLED, PAUSED, ESCALATED)
	Status string `json:"status"`

	// CurrentStepID the current step being executed
	CurrentStepID string `json:"currentStepId"`

	// CurrentStepName for display
	CurrentStepName string `json:"currentStepName"`

	// StartedAt when the workflow was initiated
	StartedAt time.Time `json:"startedAt"`

	// CompletedAt when the workflow finished (if completed)
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// InitiatedBy who or what started this workflow
	InitiatedBy string `json:"initiatedBy"`

	// TriggerReason why this workflow was started
	TriggerReason string `json:"triggerReason,omitempty"`

	// Params additional parameters passed at workflow start
	Params map[string]interface{} `json:"params,omitempty"`

	// CompletedSteps list of completed step IDs
	CompletedSteps []string `json:"completedSteps,omitempty"`

	// ICUPreempted indicates if workflow was preempted by ICU Intelligence
	ICUPreempted bool `json:"icuPreempted,omitempty"`

	// PreemptionReason if ICU preempted
	PreemptionReason string `json:"preemptionReason,omitempty"`
}

// WorkflowStep represents the current state of a workflow step execution.
type WorkflowStep struct {
	// StepID unique identifier
	StepID string `json:"stepId"`

	// StepName human-readable name
	StepName string `json:"stepName"`

	// StepOrder in workflow sequence
	StepOrder int `json:"stepOrder"`

	// Status of this step (PENDING, IN_PROGRESS, COMPLETED, SKIPPED, FAILED)
	Status string `json:"status"`

	// AssigneeID who is responsible for this step
	AssigneeID string `json:"assigneeId,omitempty"`

	// AssigneeRole role of the assignee
	AssigneeRole string `json:"assigneeRole"`

	// StartedAt when work on this step began
	StartedAt *time.Time `json:"startedAt,omitempty"`

	// CompletedAt when this step was completed
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Output data from step completion
	Output map[string]interface{} `json:"output,omitempty"`

	// NextPossibleSteps available transitions
	NextPossibleSteps []string `json:"nextPossibleSteps,omitempty"`
}

// WorkflowStatus provides detailed status of a workflow instance.
type WorkflowStatus struct {
	// InstanceID of the workflow
	InstanceID string `json:"instanceId"`

	// WorkflowID reference to definition
	WorkflowID string `json:"workflowId"`

	// WorkflowName for display
	WorkflowName string `json:"workflowName"`

	// PatientID the patient this is for
	PatientID string `json:"patientId"`

	// Status current workflow status
	Status string `json:"status"`

	// CurrentStep detailed current step info
	CurrentStep *WorkflowStep `json:"currentStep,omitempty"`

	// Progress percentage complete (0-100)
	Progress int `json:"progress"`

	// TotalSteps in the workflow
	TotalSteps int `json:"totalSteps"`

	// CompletedSteps count
	CompletedSteps int `json:"completedSteps"`

	// StartedAt workflow start time
	StartedAt time.Time `json:"startedAt"`

	// EstimatedCompletionAt based on remaining steps
	EstimatedCompletionAt *time.Time `json:"estimatedCompletionAt,omitempty"`

	// StepHistory completed steps with outcomes
	StepHistory []WorkflowStep `json:"stepHistory,omitempty"`

	// Blockers any blocking issues
	Blockers []string `json:"blockers,omitempty"`

	// ICUStatus indicates if ICU has oversight
	ICUStatus *ICUWorkflowStatus `json:"icuStatus,omitempty"`
}

// ICUWorkflowStatus tracks ICU Intelligence oversight of workflow.
type ICUWorkflowStatus struct {
	// UnderICUOversight if ICU is monitoring
	UnderICUOversight bool `json:"underIcuOversight"`

	// ICUPriority level if under oversight
	ICUPriority string `json:"icuPriority,omitempty"`

	// CanPreempt if ICU can override
	CanPreempt bool `json:"canPreempt"`

	// LastICUReviewAt when ICU last reviewed
	LastICUReviewAt *time.Time `json:"lastIcuReviewAt,omitempty"`
}

// WorkflowTask represents a task within a workflow assigned to a care team member.
type WorkflowTask struct {
	// TaskID unique identifier
	TaskID string `json:"taskId"`

	// WorkflowInstanceID the workflow this task belongs to
	WorkflowInstanceID string `json:"workflowInstanceId"`

	// StepID the workflow step this task is for
	StepID string `json:"stepId"`

	// PatientID the patient this task relates to
	PatientID string `json:"patientId"`

	// TaskType categorizes the task (REVIEW, ACTION, APPROVAL, DOCUMENTATION)
	TaskType string `json:"taskType"`

	// TaskName human-readable name
	TaskName string `json:"taskName"`

	// Description detailed task description
	Description string `json:"description"`

	// Priority (HIGH, MEDIUM, LOW, URGENT)
	Priority string `json:"priority"`

	// AssigneeID who the task is assigned to
	AssigneeID string `json:"assigneeId"`

	// AssigneeRole role of the assignee
	AssigneeRole string `json:"assigneeRole"`

	// Status (PENDING, IN_PROGRESS, COMPLETED, CANCELLED, DELEGATED)
	Status string `json:"status"`

	// DueAt when the task should be completed
	DueAt *time.Time `json:"dueAt,omitempty"`

	// CreatedAt when the task was created
	CreatedAt time.Time `json:"createdAt"`

	// CompletedAt when the task was finished
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// RequiredActions to complete the task
	RequiredActions []string `json:"requiredActions,omitempty"`

	// RelatedResources FHIR resource references
	RelatedResources []string `json:"relatedResources,omitempty"`

	// ICUUrgent if flagged by ICU Intelligence
	ICUUrgent bool `json:"icuUrgent,omitempty"`
}

// TaskOutcome represents the result of completing a task.
type TaskOutcome struct {
	// Outcome status (COMPLETED, FAILED, DELEGATED, ESCALATED, DEFERRED)
	Outcome string `json:"outcome"`

	// CompletedBy who completed the task
	CompletedBy string `json:"completedBy"`

	// CompletedAt when completed
	CompletedAt time.Time `json:"completedAt"`

	// Notes any documentation
	Notes string `json:"notes,omitempty"`

	// OutputData structured outcome data
	OutputData map[string]interface{} `json:"outputData,omitempty"`

	// DocumentationReference if documentation was created
	DocumentationReference string `json:"documentationReference,omitempty"`

	// EscalatedTo if escalated to another person
	EscalatedTo string `json:"escalatedTo,omitempty"`

	// EscalationReason why escalated
	EscalationReason string `json:"escalationReason,omitempty"`
}

// ============================================================================
// REQUEST/RESPONSE TYPES (Internal)
// ============================================================================

// Active workflows request
type kb14ActiveWorkflowsRequest struct {
	PatientID  string `json:"patientId"`
	ActiveOnly bool   `json:"activeOnly"`
}

type kb14WorkflowsResponse struct {
	Workflows []Workflow `json:"workflows"`
	Total     int        `json:"total"`
}

// Start workflow request
type kb14StartWorkflowRequest struct {
	PatientID    string                 `json:"patientId"`
	WorkflowType string                 `json:"workflowType"`
	Params       map[string]interface{} `json:"params,omitempty"`
	InitiatedBy  string                 `json:"initiatedBy"`
	TriggerGapID string                 `json:"triggerGapId,omitempty"`
}

type kb14StartWorkflowResponse struct {
	Instance WorkflowInstance `json:"instance"`
	Message  string           `json:"message"`
}

// Advance workflow request
type kb14AdvanceWorkflowRequest struct {
	WorkflowID string                 `json:"workflowId"`
	StepOutput map[string]interface{} `json:"stepOutput,omitempty"`
	AdvancedBy string                 `json:"advancedBy"`
}

type kb14AdvanceWorkflowResponse struct {
	CurrentStep   WorkflowStep `json:"currentStep"`
	WorkflowEnded bool         `json:"workflowEnded"`
	Message       string       `json:"message"`
}

// Workflow status request
type kb14WorkflowStatusRequest struct {
	WorkflowID     string `json:"workflowId"`
	IncludeHistory bool   `json:"includeHistory"`
}

type kb14WorkflowStatusResponse struct {
	Status WorkflowStatus `json:"status"`
}

// Pending tasks request
type kb14PendingTasksRequest struct {
	AssigneeID   string   `json:"assigneeId"`
	AssigneeRole string   `json:"assigneeRole,omitempty"`
	TaskTypes    []string `json:"taskTypes,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	Limit        int      `json:"limit,omitempty"`
}

type kb14TasksResponse struct {
	Tasks []WorkflowTask `json:"tasks"`
	Total int            `json:"total"`
}

// Complete task request
type kb14CompleteTaskRequest struct {
	TaskID  string      `json:"taskId"`
	Outcome TaskOutcome `json:"outcome"`
}

type kb14CompleteTaskResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Get workflow definitions request
type kb14WorkflowDefinitionsRequest struct {
	WorkflowType string `json:"workflowType,omitempty"`
	ActiveOnly   bool   `json:"activeOnly"`
}

type kb14WorkflowDefinitionsResponse struct {
	Workflows []Workflow `json:"workflows"`
	Total     int        `json:"total"`
}

// ============================================================================
// KB14 HTTP CLIENT IMPLEMENTATION
// ============================================================================

// KB14HTTPClient implements KB14Client by calling the KB-14 Care Navigator REST API.
type KB14HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB14HTTPClient creates a new KB-14 HTTP client.
func NewKB14HTTPClient(baseURL string) *KB14HTTPClient {
	return &KB14HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB14HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB14HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB14HTTPClient {
	return &KB14HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB14Client Interface Implementation (RUNTIME)
// ============================================================================

// GetActiveWorkflows returns active workflow instances for a patient.
// These are workflows that are currently in progress.
//
// ARCHITECTURE NOTE: Returns workflow instances, not definitions.
// Use GetWorkflowDefinitions to retrieve available workflow templates.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
//
// Returns:
// - List of active workflow instances with current step information
func (c *KB14HTTPClient) GetActiveWorkflows(
	ctx context.Context,
	patientID string,
) ([]Workflow, error) {

	req := kb14ActiveWorkflowsRequest{
		PatientID:  patientID,
		ActiveOnly: true,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/active", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get active workflows: %w", err)
	}

	var result kb14WorkflowsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse workflows response: %w", err)
	}

	return result.Workflows, nil
}

// StartWorkflow initiates a new care workflow for a patient.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// Workflow start may be triggered by:
// - Care gap closure (from KB-9)
// - Protocol recommendation (from KB-19)
// - Manual initiation by care team
// - ICU Intelligence override
//
// ICU DOMINANCE: If safety conditions exist, ICU Intelligence may preempt
// or modify the workflow immediately after creation.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
// - workflowType: Type of workflow to start (e.g., "CARE_GAP_CLOSURE")
// - params: Additional parameters for workflow initialization
//
// Returns:
// - WorkflowInstance with initial step and assigned tasks
func (c *KB14HTTPClient) StartWorkflow(
	ctx context.Context,
	patientID string,
	workflowType string,
	params map[string]interface{},
) (*WorkflowInstance, error) {

	// Extract initiator if provided, otherwise use system
	initiatedBy := "SYSTEM"
	if v, ok := params["initiatedBy"].(string); ok {
		initiatedBy = v
		delete(params, "initiatedBy")
	}

	// Extract trigger gap ID if this is gap-triggered
	triggerGapID := ""
	if v, ok := params["triggerGapId"].(string); ok {
		triggerGapID = v
		delete(params, "triggerGapId")
	}

	req := kb14StartWorkflowRequest{
		PatientID:    patientID,
		WorkflowType: workflowType,
		Params:       params,
		InitiatedBy:  initiatedBy,
		TriggerGapID: triggerGapID,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/start", req)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	var result kb14StartWorkflowResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse start workflow response: %w", err)
	}

	return &result.Instance, nil
}

// AdvanceWorkflow moves a workflow to its next step.
//
// ARCHITECTURE NOTE: Step advancement may:
// - Create new tasks for the next step
// - Complete the workflow if this was the last step
// - Trigger notifications to relevant care team members
// - Update care gap status if applicable
//
// ICU DOMINANCE: If safety conditions arise during advancement,
// ICU Intelligence may halt or redirect the workflow.
//
// Parameters:
// - workflowID: Unique workflow instance identifier
// - stepOutput: Output data from the completing step
//
// Returns:
// - WorkflowStep representing the new current step (or final state)
func (c *KB14HTTPClient) AdvanceWorkflow(
	ctx context.Context,
	workflowID string,
	stepOutput map[string]interface{},
) (*WorkflowStep, error) {

	// Extract advancer if provided
	advancedBy := "SYSTEM"
	if v, ok := stepOutput["advancedBy"].(string); ok {
		advancedBy = v
		delete(stepOutput, "advancedBy")
	}

	req := kb14AdvanceWorkflowRequest{
		WorkflowID: workflowID,
		StepOutput: stepOutput,
		AdvancedBy: advancedBy,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/advance", req)
	if err != nil {
		return nil, fmt.Errorf("failed to advance workflow: %w", err)
	}

	var result kb14AdvanceWorkflowResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse advance workflow response: %w", err)
	}

	return &result.CurrentStep, nil
}

// GetWorkflowStatus returns detailed status of a workflow instance.
// Includes current step, progress, and step history.
//
// Parameters:
// - workflowID: Unique workflow instance identifier
//
// Returns:
// - WorkflowStatus with detailed progress information
func (c *KB14HTTPClient) GetWorkflowStatus(
	ctx context.Context,
	workflowID string,
) (*WorkflowStatus, error) {

	req := kb14WorkflowStatusRequest{
		WorkflowID:     workflowID,
		IncludeHistory: true,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/status", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow status: %w", err)
	}

	var result kb14WorkflowStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse workflow status response: %w", err)
	}

	return &result.Status, nil
}

// GetPendingTasks returns tasks awaiting action for an assignee.
// Tasks are filtered by the assignee's ID and optionally by role and priority.
//
// Parameters:
// - assigneeID: User ID of the care team member
//
// Returns:
// - List of pending WorkflowTasks sorted by priority and due date
func (c *KB14HTTPClient) GetPendingTasks(
	ctx context.Context,
	assigneeID string,
) ([]WorkflowTask, error) {

	req := kb14PendingTasksRequest{
		AssigneeID: assigneeID,
		Limit:      100, // Default limit
	}

	resp, err := c.callKB14(ctx, "/api/v1/tasks/pending", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending tasks: %w", err)
	}

	var result kb14TasksResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tasks response: %w", err)
	}

	return result.Tasks, nil
}

// CompleteTask marks a task as completed with the specified outcome.
//
// ARCHITECTURE NOTE: Task completion may:
// - Advance the associated workflow to the next step
// - Update care gap status if this was a gap-closure task
// - Trigger downstream tasks or notifications
// - Record audit trail for compliance
//
// Parameters:
// - taskID: Unique task identifier
// - outcome: TaskOutcome with completion details
//
// Returns:
// - Error if task completion failed
func (c *KB14HTTPClient) CompleteTask(
	ctx context.Context,
	taskID string,
	outcome TaskOutcome,
) error {

	req := kb14CompleteTaskRequest{
		TaskID:  taskID,
		Outcome: outcome,
	}

	resp, err := c.callKB14(ctx, "/api/v1/tasks/complete", req)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	var result kb14CompleteTaskResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse complete task response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("task completion failed: %s", result.Message)
	}

	return nil
}

// ============================================================================
// EXTENDED OPERATIONS (Beyond Core Interface)
// ============================================================================

// GetPendingTasksByRole returns tasks for a specific role, regardless of assignee.
// Useful for workload balancing and task reassignment.
//
// Parameters:
// - role: Care team role (e.g., "NURSE", "PHYSICIAN", "CARE_MANAGER")
// - priority: Optional priority filter (empty for all)
// - limit: Maximum number of tasks to return
//
// Returns:
// - List of pending tasks for the role
func (c *KB14HTTPClient) GetPendingTasksByRole(
	ctx context.Context,
	role string,
	priority string,
	limit int,
) ([]WorkflowTask, error) {

	if limit <= 0 {
		limit = 50
	}

	req := kb14PendingTasksRequest{
		AssigneeRole: role,
		Priority:     priority,
		Limit:        limit,
	}

	resp, err := c.callKB14(ctx, "/api/v1/tasks/by-role", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by role: %w", err)
	}

	var result kb14TasksResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tasks response: %w", err)
	}

	return result.Tasks, nil
}

// GetWorkflowDefinitions returns available workflow definitions.
// These are templates that can be instantiated for patients.
//
// Parameters:
// - workflowType: Optional filter by type (empty for all)
//
// Returns:
// - List of workflow definitions
func (c *KB14HTTPClient) GetWorkflowDefinitions(
	ctx context.Context,
	workflowType string,
) ([]Workflow, error) {

	req := kb14WorkflowDefinitionsRequest{
		WorkflowType: workflowType,
		ActiveOnly:   true,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/definitions", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow definitions: %w", err)
	}

	var result kb14WorkflowDefinitionsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse definitions response: %w", err)
	}

	return result.Workflows, nil
}

// PauseWorkflow temporarily halts a workflow instance.
// Workflow can be resumed later with ResumeWorkflow.
//
// Parameters:
// - workflowID: Workflow instance to pause
// - reason: Reason for pausing
//
// Returns:
// - Updated WorkflowStatus
func (c *KB14HTTPClient) PauseWorkflow(
	ctx context.Context,
	workflowID string,
	reason string,
) (*WorkflowStatus, error) {

	req := struct {
		WorkflowID string `json:"workflowId"`
		Reason     string `json:"reason"`
	}{
		WorkflowID: workflowID,
		Reason:     reason,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/pause", req)
	if err != nil {
		return nil, fmt.Errorf("failed to pause workflow: %w", err)
	}

	var result kb14WorkflowStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pause response: %w", err)
	}

	return &result.Status, nil
}

// ResumeWorkflow resumes a paused workflow.
//
// Parameters:
// - workflowID: Workflow instance to resume
//
// Returns:
// - Updated WorkflowStatus
func (c *KB14HTTPClient) ResumeWorkflow(
	ctx context.Context,
	workflowID string,
) (*WorkflowStatus, error) {

	req := struct {
		WorkflowID string `json:"workflowId"`
	}{
		WorkflowID: workflowID,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/resume", req)
	if err != nil {
		return nil, fmt.Errorf("failed to resume workflow: %w", err)
	}

	var result kb14WorkflowStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resume response: %w", err)
	}

	return &result.Status, nil
}

// CancelWorkflow terminates a workflow instance.
//
// Parameters:
// - workflowID: Workflow instance to cancel
// - reason: Reason for cancellation
//
// Returns:
// - Error if cancellation failed
func (c *KB14HTTPClient) CancelWorkflow(
	ctx context.Context,
	workflowID string,
	reason string,
) error {

	req := struct {
		WorkflowID string `json:"workflowId"`
		Reason     string `json:"reason"`
	}{
		WorkflowID: workflowID,
		Reason:     reason,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/cancel", req)
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse cancel response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("workflow cancellation failed: %s", result.Message)
	}

	return nil
}

// GetPatientWorkflowHistory returns all workflow instances for a patient.
// Includes completed and cancelled workflows.
//
// Parameters:
// - patientID: FHIR Patient.id
// - limit: Maximum number of results
//
// Returns:
// - List of workflow instances (historical)
func (c *KB14HTTPClient) GetPatientWorkflowHistory(
	ctx context.Context,
	patientID string,
	limit int,
) ([]WorkflowInstance, error) {

	if limit <= 0 {
		limit = 50
	}

	req := struct {
		PatientID string `json:"patientId"`
		Limit     int    `json:"limit"`
	}{
		PatientID: patientID,
		Limit:     limit,
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/history", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow history: %w", err)
	}

	var result struct {
		Instances []WorkflowInstance `json:"instances"`
		Total     int                `json:"total"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse history response: %w", err)
	}

	return result.Instances, nil
}

// DelegateTask reassigns a task to another care team member.
//
// Parameters:
// - taskID: Task to delegate
// - newAssigneeID: User ID of new assignee
// - reason: Reason for delegation
//
// Returns:
// - Updated WorkflowTask
func (c *KB14HTTPClient) DelegateTask(
	ctx context.Context,
	taskID string,
	newAssigneeID string,
	reason string,
) (*WorkflowTask, error) {

	req := struct {
		TaskID        string `json:"taskId"`
		NewAssigneeID string `json:"newAssigneeId"`
		Reason        string `json:"reason"`
	}{
		TaskID:        taskID,
		NewAssigneeID: newAssigneeID,
		Reason:        reason,
	}

	resp, err := c.callKB14(ctx, "/api/v1/tasks/delegate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to delegate task: %w", err)
	}

	var result struct {
		Task    WorkflowTask `json:"task"`
		Message string       `json:"message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delegate response: %w", err)
	}

	return &result.Task, nil
}

// EscalateTask escalates a task to a higher-level care team member.
//
// Parameters:
// - taskID: Task to escalate
// - reason: Reason for escalation
//
// Returns:
// - Updated WorkflowTask
func (c *KB14HTTPClient) EscalateTask(
	ctx context.Context,
	taskID string,
	reason string,
) (*WorkflowTask, error) {

	req := struct {
		TaskID string `json:"taskId"`
		Reason string `json:"reason"`
	}{
		TaskID: taskID,
		Reason: reason,
	}

	resp, err := c.callKB14(ctx, "/api/v1/tasks/escalate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to escalate task: %w", err)
	}

	var result struct {
		Task    WorkflowTask `json:"task"`
		Message string       `json:"message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse escalate response: %w", err)
	}

	return &result.Task, nil
}

// ============================================================================
// ICU DOMINANCE OPERATIONS
// ============================================================================

// RequestICUPreemption allows ICU Intelligence to preempt a workflow.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// ICU Intelligence has VETO AUTHORITY over all workflow operations.
// This method is called when safety conditions require workflow override.
//
// Parameters:
// - workflowID: Workflow to preempt
// - reason: Safety reason for preemption
// - urgency: Preemption urgency level (IMMEDIATE, HIGH, NORMAL)
//
// Returns:
// - Updated WorkflowStatus with preemption details
func (c *KB14HTTPClient) RequestICUPreemption(
	ctx context.Context,
	workflowID string,
	reason string,
	urgency string,
) (*WorkflowStatus, error) {

	req := struct {
		WorkflowID string `json:"workflowId"`
		Reason     string `json:"reason"`
		Urgency    string `json:"urgency"`
		RequestedBy string `json:"requestedBy"`
	}{
		WorkflowID: workflowID,
		Reason:     reason,
		Urgency:    urgency,
		RequestedBy: "ICU_INTELLIGENCE",
	}

	resp, err := c.callKB14(ctx, "/api/v1/workflows/icu-preempt", req)
	if err != nil {
		return nil, fmt.Errorf("failed to request ICU preemption: %w", err)
	}

	var result kb14WorkflowStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse preemption response: %w", err)
	}

	return &result.Status, nil
}

// MarkTasksICUUrgent marks tasks as ICU-urgent for priority handling.
//
// Parameters:
// - patientID: Patient whose tasks should be marked urgent
// - reason: Reason for urgency
//
// Returns:
// - Number of tasks marked urgent
func (c *KB14HTTPClient) MarkTasksICUUrgent(
	ctx context.Context,
	patientID string,
	reason string,
) (int, error) {

	req := struct {
		PatientID string `json:"patientId"`
		Reason    string `json:"reason"`
	}{
		PatientID: patientID,
		Reason:    reason,
	}

	resp, err := c.callKB14(ctx, "/api/v1/tasks/mark-icu-urgent", req)
	if err != nil {
		return 0, fmt.Errorf("failed to mark tasks ICU urgent: %w", err)
	}

	var result struct {
		MarkedCount int    `json:"markedCount"`
		Message     string `json:"message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to parse mark urgent response: %w", err)
	}

	return result.MarkedCount, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// callKB14 makes an HTTP POST request to the KB-14 service.
func (c *KB14HTTPClient) callKB14(
	ctx context.Context,
	endpoint string,
	request interface{},
) ([]byte, error) {

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-14 returned error status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// HealthCheck verifies KB-14 service is healthy.
func (c *KB14HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-14 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// INTERFACE COMPLIANCE NOTE
// ============================================================================

// KB14HTTPClient implements the KB14Client interface as defined in the
// VAIDSHALA_IMPLEMENTATION_PLAN.md specification. The interface is not
// added to the FROZEN contracts file to maintain governance compliance.
//
// Interface Methods:
// - GetActiveWorkflows(ctx, patientID) → []Workflow
// - StartWorkflow(ctx, patientID, workflowType, params) → *WorkflowInstance
// - AdvanceWorkflow(ctx, workflowID, stepOutput) → *WorkflowStep
// - GetWorkflowStatus(ctx, workflowID) → *WorkflowStatus
// - GetPendingTasks(ctx, assigneeID) → []WorkflowTask
// - CompleteTask(ctx, taskID, outcome) → error
