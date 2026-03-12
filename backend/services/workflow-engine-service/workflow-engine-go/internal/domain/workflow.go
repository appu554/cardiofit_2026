package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus represents the status of a workflow instance
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
	WorkflowStatusSuspended WorkflowStatus = "suspended"
)

// TaskStatus represents the status of a workflow task
type TaskStatus string

const (
	TaskStatusCreated    TaskStatus = "created"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusEscalated  TaskStatus = "escalated"
)

// WorkflowDefinition represents a BPMN workflow definition
type WorkflowDefinition struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Version     string                 `json:"version" db:"version"`
	Description string                 `json:"description" db:"description"`
	BPMNData    string                 `json:"bpmn_data" db:"bpmn_data"`
	Variables   map[string]interface{} `json:"variables" db:"variables"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	Active      bool                   `json:"active" db:"active"`
	Tags        []string               `json:"tags,omitempty" db:"tags"`
	Category    string                 `json:"category,omitempty" db:"category"`
}

// NewWorkflowDefinition creates a new workflow definition
func NewWorkflowDefinition(name, version, description, bpmnData string) *WorkflowDefinition {
	now := time.Now().UTC()
	return &WorkflowDefinition{
		ID:          fmt.Sprintf("wf_def_%s", uuid.New().String()),
		Name:        name,
		Version:     version,
		Description: description,
		BPMNData:    bpmnData,
		Variables:   make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		Active:      true,
		Tags:        make([]string, 0),
	}
}

// Update updates the workflow definition
func (w *WorkflowDefinition) Update(description, bpmnData string, variables map[string]interface{}) {
	w.Description = description
	w.BPMNData = bpmnData
	if variables != nil {
		w.Variables = variables
	}
	w.UpdatedAt = time.Now().UTC()
}

// Activate activates the workflow definition
func (w *WorkflowDefinition) Activate() {
	w.Active = true
	w.UpdatedAt = time.Now().UTC()
}

// Deactivate deactivates the workflow definition
func (w *WorkflowDefinition) Deactivate() {
	w.Active = false
	w.UpdatedAt = time.Now().UTC()
}

// WorkflowInstance represents an instance of a workflow
type WorkflowInstance struct {
	ID               string                 `json:"id" db:"id"`
	DefinitionID     string                 `json:"definition_id" db:"definition_id"`
	PatientID        string                 `json:"patient_id" db:"patient_id"`
	Status           WorkflowStatus         `json:"status" db:"status"`
	Variables        map[string]interface{} `json:"variables" db:"variables"`
	StartTime        time.Time              `json:"start_time" db:"start_time"`
	EndTime          *time.Time             `json:"end_time,omitempty" db:"end_time"`
	CorrelationID    string                 `json:"correlation_id" db:"correlation_id"`
	SnapshotChain    *SnapshotChainTracker  `json:"snapshot_chain,omitempty" db:"snapshot_chain"`
	ParentInstanceID *string                `json:"parent_instance_id,omitempty" db:"parent_instance_id"`
	BusinessKey      string                 `json:"business_key,omitempty" db:"business_key"`
	Priority         int                    `json:"priority" db:"priority"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy        string                 `json:"created_by,omitempty" db:"created_by"`
	Tags             []string               `json:"tags,omitempty" db:"tags"`
}

// NewWorkflowInstance creates a new workflow instance
func NewWorkflowInstance(
	definitionID, patientID, correlationID string,
	variables map[string]interface{},
) *WorkflowInstance {
	now := time.Now().UTC()
	return &WorkflowInstance{
		ID:            fmt.Sprintf("wf_inst_%s", uuid.New().String()),
		DefinitionID:  definitionID,
		PatientID:     patientID,
		Status:        WorkflowStatusPending,
		Variables:     variables,
		StartTime:     now,
		CorrelationID: correlationID,
		Priority:      0, // Normal priority
		CreatedAt:     now,
		UpdatedAt:     now,
		Tags:          make([]string, 0),
	}
}

// Start starts the workflow instance
func (w *WorkflowInstance) Start() {
	w.Status = WorkflowStatusRunning
	w.StartTime = time.Now().UTC()
	w.UpdatedAt = time.Now().UTC()
}

// Complete completes the workflow instance
func (w *WorkflowInstance) Complete() {
	w.Status = WorkflowStatusCompleted
	now := time.Now().UTC()
	w.EndTime = &now
	w.UpdatedAt = now
}

// Fail marks the workflow instance as failed
func (w *WorkflowInstance) Fail() {
	w.Status = WorkflowStatusFailed
	now := time.Now().UTC()
	w.EndTime = &now
	w.UpdatedAt = now
}

// Cancel cancels the workflow instance
func (w *WorkflowInstance) Cancel() {
	w.Status = WorkflowStatusCancelled
	now := time.Now().UTC()
	w.EndTime = &now
	w.UpdatedAt = now
}

// Suspend suspends the workflow instance
func (w *WorkflowInstance) Suspend() {
	w.Status = WorkflowStatusSuspended
	w.UpdatedAt = time.Now().UTC()
}

// Resume resumes a suspended workflow instance
func (w *WorkflowInstance) Resume() {
	if w.Status == WorkflowStatusSuspended {
		w.Status = WorkflowStatusRunning
		w.UpdatedAt = time.Now().UTC()
	}
}

// IsTerminal returns true if the workflow is in a terminal state
func (w *WorkflowInstance) IsTerminal() bool {
	return w.Status == WorkflowStatusCompleted ||
		w.Status == WorkflowStatusFailed ||
		w.Status == WorkflowStatusCancelled
}

// Duration returns the duration of the workflow instance
func (w *WorkflowInstance) Duration() *time.Duration {
	if w.EndTime != nil {
		duration := w.EndTime.Sub(w.StartTime)
		return &duration
	}
	return nil
}

// UpdateVariables updates the workflow variables
func (w *WorkflowInstance) UpdateVariables(variables map[string]interface{}) {
	if w.Variables == nil {
		w.Variables = make(map[string]interface{})
	}
	for key, value := range variables {
		w.Variables[key] = value
	}
	w.UpdatedAt = time.Now().UTC()
}

// SetSnapshotChain sets the snapshot chain for the workflow instance
func (w *WorkflowInstance) SetSnapshotChain(chain *SnapshotChainTracker) {
	w.SnapshotChain = chain
	w.UpdatedAt = time.Now().UTC()
}

// WorkflowTask represents a human or service task in a workflow
type WorkflowTask struct {
	ID                 string                 `json:"id" db:"id"`
	WorkflowInstanceID string                 `json:"workflow_instance_id" db:"workflow_instance_id"`
	TaskDefinitionID   string                 `json:"task_definition_id" db:"task_definition_id"`
	Name               string                 `json:"name" db:"name"`
	AssigneeID         *string                `json:"assignee_id,omitempty" db:"assignee_id"`
	CandidateGroups    []string               `json:"candidate_groups,omitempty" db:"candidate_groups"`
	Status             TaskStatus             `json:"status" db:"status"`
	Variables          map[string]interface{} `json:"variables" db:"variables"`
	FormKey            string                 `json:"form_key,omitempty" db:"form_key"`
	DueDate            *time.Time             `json:"due_date,omitempty" db:"due_date"`
	FollowUpDate       *time.Time             `json:"follow_up_date,omitempty" db:"follow_up_date"`
	Priority           int                    `json:"priority" db:"priority"`
	Description        string                 `json:"description,omitempty" db:"description"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CompletedBy        *string                `json:"completed_by,omitempty" db:"completed_by"`
}

// NewWorkflowTask creates a new workflow task
func NewWorkflowTask(
	workflowInstanceID, taskDefinitionID, name string,
	candidateGroups []string,
) *WorkflowTask {
	now := time.Now().UTC()
	return &WorkflowTask{
		ID:                 fmt.Sprintf("task_%s", uuid.New().String()),
		WorkflowInstanceID: workflowInstanceID,
		TaskDefinitionID:   taskDefinitionID,
		Name:               name,
		CandidateGroups:    candidateGroups,
		Status:             TaskStatusCreated,
		Variables:          make(map[string]interface{}),
		Priority:           50, // Normal priority
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Claim assigns the task to a user
func (t *WorkflowTask) Claim(userID string) {
	t.AssigneeID = &userID
	t.Status = TaskStatusAssigned
	t.UpdatedAt = time.Now().UTC()
}

// Start marks the task as in progress
func (t *WorkflowTask) Start() {
	if t.Status == TaskStatusAssigned || t.Status == TaskStatusCreated {
		t.Status = TaskStatusInProgress
		t.UpdatedAt = time.Now().UTC()
	}
}

// Complete completes the task
func (t *WorkflowTask) Complete(completedBy string, variables map[string]interface{}) {
	t.Status = TaskStatusCompleted
	now := time.Now().UTC()
	t.CompletedAt = &now
	t.CompletedBy = &completedBy
	t.UpdatedAt = now

	if variables != nil {
		t.UpdateVariables(variables)
	}
}

// Cancel cancels the task
func (t *WorkflowTask) Cancel() {
	t.Status = TaskStatusCancelled
	t.UpdatedAt = time.Now().UTC()
}

// Escalate escalates the task
func (t *WorkflowTask) Escalate() {
	t.Status = TaskStatusEscalated
	t.UpdatedAt = time.Now().UTC()
}

// Delegate delegates the task to another user
func (t *WorkflowTask) Delegate(newAssigneeID string) {
	t.AssigneeID = &newAssigneeID
	t.Status = TaskStatusAssigned
	t.UpdatedAt = time.Now().UTC()
}

// SetDueDate sets the due date for the task
func (t *WorkflowTask) SetDueDate(dueDate time.Time) {
	t.DueDate = &dueDate
	t.UpdatedAt = time.Now().UTC()
}

// IsOverdue returns true if the task is overdue
func (t *WorkflowTask) IsOverdue() bool {
	return t.DueDate != nil && time.Now().UTC().After(*t.DueDate)
}

// UpdateVariables updates the task variables
func (t *WorkflowTask) UpdateVariables(variables map[string]interface{}) {
	if t.Variables == nil {
		t.Variables = make(map[string]interface{})
	}
	for key, value := range variables {
		t.Variables[key] = value
	}
	t.UpdatedAt = time.Now().UTC()
}

// WorkflowEvent represents an event in the workflow execution
type WorkflowEvent struct {
	ID                 string                 `json:"id" db:"id"`
	WorkflowInstanceID string                 `json:"workflow_instance_id" db:"workflow_instance_id"`
	TaskID             *string                `json:"task_id,omitempty" db:"task_id"`
	EventType          string                 `json:"event_type" db:"event_type"`
	EventData          map[string]interface{} `json:"event_data" db:"event_data"`
	UserID             *string                `json:"user_id,omitempty" db:"user_id"`
	Timestamp          time.Time              `json:"timestamp" db:"timestamp"`
	CorrelationID      string                 `json:"correlation_id" db:"correlation_id"`
}

// NewWorkflowEvent creates a new workflow event
func NewWorkflowEvent(
	workflowInstanceID, eventType, correlationID string,
	eventData map[string]interface{},
	userID *string,
) *WorkflowEvent {
	return &WorkflowEvent{
		ID:                 fmt.Sprintf("event_%s", uuid.New().String()),
		WorkflowInstanceID: workflowInstanceID,
		EventType:          eventType,
		EventData:          eventData,
		UserID:             userID,
		Timestamp:          time.Now().UTC(),
		CorrelationID:      correlationID,
	}
}

// WorkflowTimer represents a timer event in the workflow
type WorkflowTimer struct {
	ID                 string                 `json:"id" db:"id"`
	WorkflowInstanceID string                 `json:"workflow_instance_id" db:"workflow_instance_id"`
	TimerName          string                 `json:"timer_name" db:"timer_name"`
	ScheduledAt        time.Time              `json:"scheduled_at" db:"scheduled_at"`
	ExecutedAt         *time.Time             `json:"executed_at,omitempty" db:"executed_at"`
	Configuration      map[string]interface{} `json:"configuration" db:"configuration"`
	Status             string                 `json:"status" db:"status"` // scheduled, executed, cancelled
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
}

// NewWorkflowTimer creates a new workflow timer
func NewWorkflowTimer(
	workflowInstanceID, timerName string,
	scheduledAt time.Time,
	configuration map[string]interface{},
) *WorkflowTimer {
	return &WorkflowTimer{
		ID:                 fmt.Sprintf("timer_%s", uuid.New().String()),
		WorkflowInstanceID: workflowInstanceID,
		TimerName:          timerName,
		ScheduledAt:        scheduledAt,
		Configuration:      configuration,
		Status:             "scheduled",
		CreatedAt:          time.Now().UTC(),
	}
}

// Execute marks the timer as executed
func (t *WorkflowTimer) Execute() {
	t.Status = "executed"
	now := time.Now().UTC()
	t.ExecutedAt = &now
}

// Cancel marks the timer as cancelled
func (t *WorkflowTimer) Cancel() {
	t.Status = "cancelled"
}

// IsReady returns true if the timer is ready to execute
func (t *WorkflowTimer) IsReady() bool {
	return t.Status == "scheduled" && time.Now().UTC().After(t.ScheduledAt)
}

// Value implements the driver.Valuer interface for JSON fields
func (w WorkflowDefinition) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// Scan implements the sql.Scanner interface for JSON fields
func (w *WorkflowDefinition) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into WorkflowDefinition", value)
	}
	return json.Unmarshal(bytes, w)
}

// Value implements the driver.Valuer interface for JSON fields
func (w WorkflowInstance) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// Scan implements the sql.Scanner interface for JSON fields
func (w *WorkflowInstance) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into WorkflowInstance", value)
	}
	return json.Unmarshal(bytes, w)
}

// WorkflowMetrics contains performance metrics for workflow execution
type WorkflowMetrics struct {
	WorkflowInstanceID string    `json:"workflow_instance_id" db:"workflow_instance_id"`
	MetricName         string    `json:"metric_name" db:"metric_name"`
	MetricValue        float64   `json:"metric_value" db:"metric_value"`
	RecordedAt         time.Time `json:"recorded_at" db:"recorded_at"`
	CorrelationID      string    `json:"correlation_id" db:"correlation_id"`
}

// NewWorkflowMetric creates a new workflow metric
func NewWorkflowMetric(
	workflowInstanceID, metricName string,
	metricValue float64,
	correlationID string,
) *WorkflowMetrics {
	return &WorkflowMetrics{
		WorkflowInstanceID: workflowInstanceID,
		MetricName:         metricName,
		MetricValue:        metricValue,
		RecordedAt:         time.Now().UTC(),
		CorrelationID:      correlationID,
	}
}