// Package workflow provides clinical workflow management for order sets and care plans
// Includes time-critical protocol tracking, state management, and KB-3 temporal integration
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kb-12-ordersets-careplans/internal/clients"
	"kb-12-ordersets-careplans/internal/models"
)

// WorkflowEngine manages clinical workflows, order set execution, and care plan activation
type WorkflowEngine struct {
	kb3Client        *clients.KB3Client // Temporal logic
	activeWorkflows  sync.Map           // Active workflow instances
	timeConstraints  sync.Map           // Active time constraints
	eventHandlers    []EventHandler     // Registered event handlers
	tickerInterval   time.Duration
	stopTicker       chan struct{}
	mu               sync.RWMutex
}

// EventHandler is a callback for workflow events
type EventHandler func(event *WorkflowEvent) error

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(kb3Client *clients.KB3Client) *WorkflowEngine {
	return &WorkflowEngine{
		kb3Client:      kb3Client,
		eventHandlers:  make([]EventHandler, 0),
		tickerInterval: 30 * time.Second,
		stopTicker:     make(chan struct{}),
	}
}

// Start begins the workflow engine's background processes
func (e *WorkflowEngine) Start(ctx context.Context) error {
	go e.timeConstraintMonitor(ctx)
	return nil
}

// Stop halts the workflow engine
func (e *WorkflowEngine) Stop() {
	close(e.stopTicker)
}

// RegisterEventHandler adds an event handler
func (e *WorkflowEngine) RegisterEventHandler(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventHandlers = append(e.eventHandlers, handler)
}

// WorkflowInstance represents an active workflow
type WorkflowInstance struct {
	InstanceID       string                 `json:"instance_id"`
	WorkflowType     string                 `json:"workflow_type"` // order_set, care_plan, emergency_protocol
	TemplateID       string                 `json:"template_id"`
	TemplateName     string                 `json:"template_name"`
	PatientID        string                 `json:"patient_id"`
	EncounterID      string                 `json:"encounter_id"`
	InitiatedBy      string                 `json:"initiated_by"`
	Status           string                 `json:"status"` // pending, active, completed, cancelled, expired
	CurrentPhase     string                 `json:"current_phase"`
	Steps            []*WorkflowStep        `json:"steps"`
	TimeConstraints  []*ActiveTimeConstraint `json:"time_constraints"`
	Variables        map[string]interface{} `json:"variables"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	LastUpdated      time.Time              `json:"last_updated"`
	Priority         int                    `json:"priority"` // 1=highest
	Metadata         map[string]interface{} `json:"metadata"`
}

// WorkflowStep represents a step in the workflow
type WorkflowStep struct {
	StepID          string                 `json:"step_id"`
	StepType        string                 `json:"step_type"` // order, assessment, intervention, notification, decision
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"` // pending, in_progress, completed, skipped, failed
	Required        bool                   `json:"required"`
	DependsOn       []string               `json:"depends_on"` // Step IDs this depends on
	OrderID         string                 `json:"order_id,omitempty"`
	TimeConstraint  *TimeConstraintRef     `json:"time_constraint,omitempty"`
	Conditions      []StepCondition        `json:"conditions"`
	Actions         []StepAction           `json:"actions"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	CompletedBy     string                 `json:"completed_by,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Result          map[string]interface{} `json:"result,omitempty"`
}

// TimeConstraintRef references a time constraint
type TimeConstraintRef struct {
	ConstraintID string `json:"constraint_id"`
	Description  string `json:"description"`
}

// StepCondition represents a condition for step execution
type StepCondition struct {
	ConditionID   string `json:"condition_id"`
	Type          string `json:"type"` // equals, greater_than, less_than, contains, exists
	Field         string `json:"field"`
	Value         interface{} `json:"value"`
	Operator      string `json:"operator"` // and, or
}

// StepAction represents an action to take for a step
type StepAction struct {
	ActionID      string                 `json:"action_id"`
	ActionType    string                 `json:"action_type"` // create_order, send_alert, update_status, call_service
	Parameters    map[string]interface{} `json:"parameters"`
	OnSuccess     string                 `json:"on_success,omitempty"` // next step or branch
	OnFailure     string                 `json:"on_failure,omitempty"`
}

// ActiveTimeConstraint tracks a time-critical action
type ActiveTimeConstraint struct {
	ConstraintID   string        `json:"constraint_id"`
	InstanceID     string        `json:"instance_id"`
	PatientID      string        `json:"patient_id"`
	Action         string        `json:"action"`
	Description    string        `json:"description"`
	Deadline       time.Time     `json:"deadline"`
	WarningTime    time.Time     `json:"warning_time"`
	Duration       time.Duration `json:"duration"`
	Severity       string        `json:"severity"` // info, warning, critical
	Status         string        `json:"status"`   // pending, warning_sent, critical, completed, missed
	MetricsCode    string        `json:"metrics_code"` // e.g., SEP-1, STEMI-DTB
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
	AlertsSent     int           `json:"alerts_sent"`
}

// WorkflowEvent represents an event in the workflow lifecycle
type WorkflowEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"` // workflow_started, step_completed, constraint_warning, constraint_critical, workflow_completed
	InstanceID    string                 `json:"instance_id"`
	PatientID     string                 `json:"patient_id"`
	Severity      string                 `json:"severity"`
	Title         string                 `json:"title"`
	Message       string                 `json:"message"`
	Data          map[string]interface{} `json:"data"`
	Timestamp     time.Time              `json:"timestamp"`
	RequiresAction bool                  `json:"requires_action"`
}

// StartOrderSetWorkflow initiates an order set workflow
func (e *WorkflowEngine) StartOrderSetWorkflow(ctx context.Context, req *StartOrderSetRequest) (*WorkflowInstance, error) {
	instanceID := fmt.Sprintf("WF-OS-%d", time.Now().UnixNano())

	instance := &WorkflowInstance{
		InstanceID:      instanceID,
		WorkflowType:    "order_set",
		TemplateID:      req.OrderSetID,
		TemplateName:    req.OrderSetName,
		PatientID:       req.PatientID,
		EncounterID:     req.EncounterID,
		InitiatedBy:     req.InitiatedBy,
		Status:          "active",
		CurrentPhase:    "initial_orders",
		Steps:           make([]*WorkflowStep, 0),
		TimeConstraints: make([]*ActiveTimeConstraint, 0),
		Variables:       req.Variables,
		StartedAt:       time.Now(),
		LastUpdated:     time.Now(),
		Priority:        req.Priority,
		Metadata:        req.Metadata,
	}

	// Create workflow steps from order set
	instance.Steps = e.createStepsFromOrderSet(req.OrderSet)

	// Activate time constraints
	for _, tc := range req.OrderSet.TimeConstraints {
		constraint := e.activateTimeConstraint(instanceID, req.PatientID, tc)
		instance.TimeConstraints = append(instance.TimeConstraints, constraint)
		e.timeConstraints.Store(constraint.ConstraintID, constraint)
	}

	// Store workflow instance
	e.activeWorkflows.Store(instanceID, instance)

	// Emit start event
	e.emitEvent(&WorkflowEvent{
		EventID:    fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		EventType:  "workflow_started",
		InstanceID: instanceID,
		PatientID:  req.PatientID,
		Severity:   "info",
		Title:      "Order Set Initiated",
		Message:    fmt.Sprintf("Order set '%s' started for patient %s", req.OrderSetName, req.PatientID),
		Data:       map[string]interface{}{"template_id": req.OrderSetID, "step_count": len(instance.Steps)},
		Timestamp:  time.Now(),
	})

	return instance, nil
}

// StartOrderSetRequest is the request to start an order set workflow
type StartOrderSetRequest struct {
	OrderSetID   string                   `json:"order_set_id"`
	OrderSetName string                   `json:"order_set_name"`
	OrderSet     *models.OrderSetTemplate `json:"order_set"`
	PatientID    string                   `json:"patient_id"`
	EncounterID  string                   `json:"encounter_id"`
	InitiatedBy  string                   `json:"initiated_by"`
	Variables    map[string]interface{}   `json:"variables"`
	Priority     int                      `json:"priority"`
	Metadata     map[string]interface{}   `json:"metadata"`
}

// createStepsFromOrderSet converts order set sections to workflow steps
func (e *WorkflowEngine) createStepsFromOrderSet(orderSet *models.OrderSetTemplate) []*WorkflowStep {
	steps := make([]*WorkflowStep, 0)
	stepNum := 0

	for _, section := range orderSet.Sections {
		for _, item := range section.Items {
			stepNum++
			step := &WorkflowStep{
				StepID:      fmt.Sprintf("STEP-%03d", stepNum),
				StepType:    e.mapOrderTypeToStepType(item.Type),
				Name:        item.Name,
				Description: item.Description,
				Status:      "pending",
				Required:    item.Required,
				DependsOn:   item.DependsOn,
			}

			// Add time constraint reference if applicable
			if item.TimeConstraint != nil {
				step.TimeConstraint = &TimeConstraintRef{
					ConstraintID: item.TimeConstraint.ConstraintID,
					Description:  item.TimeConstraint.Description,
				}
			}

			// Create conditions from item conditions
			for _, cond := range item.Conditions {
				step.Conditions = append(step.Conditions, StepCondition{
					ConditionID: cond.ConditionID,
					Field:       cond.Field,
					Type:        cond.Operator,
					Value:       cond.Value,
				})
			}

			steps = append(steps, step)
		}
	}

	return steps
}

// mapOrderTypeToStepType maps order item types to workflow step types
func (e *WorkflowEngine) mapOrderTypeToStepType(orderType string) string {
	switch orderType {
	case "medication":
		return "order"
	case "laboratory":
		return "order"
	case "imaging":
		return "order"
	case "procedure":
		return "order"
	case "nursing":
		return "intervention"
	case "assessment":
		return "assessment"
	case "notification":
		return "notification"
	case "consult":
		return "order"
	default:
		return "order"
	}
}

// activateTimeConstraint creates an active time constraint
func (e *WorkflowEngine) activateTimeConstraint(instanceID, patientID string, tc models.TimeConstraint) *ActiveTimeConstraint {
	now := time.Now()
	deadline := now.Add(tc.Deadline)
	warningTime := now.Add(tc.Deadline / 2) // Warn at 50% of deadline

	return &ActiveTimeConstraint{
		ConstraintID: tc.ConstraintID,
		InstanceID:   instanceID,
		PatientID:    patientID,
		Action:       tc.Action,
		Description:  tc.Description,
		Deadline:     deadline,
		WarningTime:  warningTime,
		Duration:     tc.Deadline,
		Severity:     tc.Severity,
		Status:       "pending",
		MetricsCode:  tc.MetricsCode,
		AlertsSent:   0,
	}
}

// StartCarePlanWorkflow initiates a care plan workflow
func (e *WorkflowEngine) StartCarePlanWorkflow(ctx context.Context, req *StartCarePlanRequest) (*WorkflowInstance, error) {
	instanceID := fmt.Sprintf("WF-CP-%d", time.Now().UnixNano())

	instance := &WorkflowInstance{
		InstanceID:      instanceID,
		WorkflowType:    "care_plan",
		TemplateID:      req.CarePlanID,
		TemplateName:    req.CarePlanName,
		PatientID:       req.PatientID,
		EncounterID:     req.EncounterID,
		InitiatedBy:     req.InitiatedBy,
		Status:          "active",
		CurrentPhase:    "initial_assessment",
		Steps:           make([]*WorkflowStep, 0),
		TimeConstraints: make([]*ActiveTimeConstraint, 0),
		Variables:       req.Variables,
		StartedAt:       time.Now(),
		LastUpdated:     time.Now(),
		Priority:        req.Priority,
		Metadata:        req.Metadata,
	}

	// Create workflow steps from care plan
	instance.Steps = e.createStepsFromCarePlan(req.CarePlan)

	// Store workflow instance
	e.activeWorkflows.Store(instanceID, instance)

	// Emit start event
	e.emitEvent(&WorkflowEvent{
		EventID:    fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		EventType:  "workflow_started",
		InstanceID: instanceID,
		PatientID:  req.PatientID,
		Severity:   "info",
		Title:      "Care Plan Activated",
		Message:    fmt.Sprintf("Care plan '%s' activated for patient %s", req.CarePlanName, req.PatientID),
		Data: map[string]interface{}{
			"template_id":   req.CarePlanID,
			"goal_count":    len(req.CarePlan.Goals),
			"activity_count": len(req.CarePlan.Activities),
		},
		Timestamp: time.Now(),
	})

	return instance, nil
}

// StartCarePlanRequest is the request to start a care plan workflow
type StartCarePlanRequest struct {
	CarePlanID   string                    `json:"care_plan_id"`
	CarePlanName string                    `json:"care_plan_name"`
	CarePlan     *models.CarePlanTemplate  `json:"care_plan"`
	PatientID    string                    `json:"patient_id"`
	EncounterID  string                    `json:"encounter_id"`
	InitiatedBy  string                    `json:"initiated_by"`
	Variables    map[string]interface{}    `json:"variables"`
	Priority     int                       `json:"priority"`
	Metadata     map[string]interface{}    `json:"metadata"`
}

// createStepsFromCarePlan converts care plan activities to workflow steps
func (e *WorkflowEngine) createStepsFromCarePlan(carePlan *models.CarePlanTemplate) []*WorkflowStep {
	steps := make([]*WorkflowStep, 0)
	stepNum := 0

	// Create initial assessment step
	stepNum++
	steps = append(steps, &WorkflowStep{
		StepID:      fmt.Sprintf("STEP-%03d", stepNum),
		StepType:    "assessment",
		Name:        "Initial Care Plan Assessment",
		Description: fmt.Sprintf("Initial assessment for %s care plan", carePlan.Name),
		Status:      "pending",
		Required:    true,
	})

	// Create steps for each goal
	for _, goal := range carePlan.Goals {
		stepNum++
		steps = append(steps, &WorkflowStep{
			StepID:      fmt.Sprintf("STEP-%03d", stepNum),
			StepType:    "assessment",
			Name:        fmt.Sprintf("Establish Goal: %s", goal.Description),
			Description: goal.Description,
			Status:      "pending",
			Required:    true,
			Result: map[string]interface{}{
				"goal_id":  goal.GoalID,
				"targets":  goal.Targets,
				"priority": goal.Priority,
			},
		})
	}

	// Create steps for each activity
	for _, activity := range carePlan.Activities {
		if activity.Status == "conditional" {
			continue // Skip conditional activities until triggered
		}

		stepNum++
		step := &WorkflowStep{
			StepID:      fmt.Sprintf("STEP-%03d", stepNum),
			StepType:    e.mapActivityTypeToStepType(activity.Type),
			Name:        activity.Description,
			Description: fmt.Sprintf("%s - %s", activity.Type, activity.Frequency),
			Status:      "pending",
			Required:    true,
			Result: map[string]interface{}{
				"activity_id": activity.ActivityID,
				"type":        activity.Type,
				"details":     activity.Details,
			},
		}

		if activity.Condition != "" {
			step.Conditions = append(step.Conditions, StepCondition{
				Type:  "expression",
				Value: activity.Condition,
			})
		}

		steps = append(steps, step)
	}

	// Create monitoring steps
	for _, monitoring := range carePlan.Monitoring {
		stepNum++
		steps = append(steps, &WorkflowStep{
			StepID:      fmt.Sprintf("STEP-%03d", stepNum),
			StepType:    "assessment",
			Name:        fmt.Sprintf("Monitor: %s", monitoring.Parameter),
			Description: fmt.Sprintf("Target: %s, Alert if: %s", monitoring.Target, monitoring.AlertThreshold),
			Status:      "pending",
			Required:    true,
			Result: map[string]interface{}{
				"item_id":   monitoring.ItemID,
				"parameter": monitoring.Parameter,
				"target":    monitoring.Target,
				"threshold": monitoring.AlertThreshold,
			},
		})
	}

	return steps
}

// mapActivityTypeToStepType maps activity types to step types
func (e *WorkflowEngine) mapActivityTypeToStepType(activityType string) string {
	switch activityType {
	case "medication":
		return "order"
	case "lifestyle":
		return "intervention"
	case "education":
		return "intervention"
	case "appointment":
		return "order"
	case "therapy":
		return "order"
	case "safety":
		return "assessment"
	default:
		return "intervention"
	}
}

// StartEmergencyProtocol initiates an emergency protocol workflow with time tracking
func (e *WorkflowEngine) StartEmergencyProtocol(ctx context.Context, req *StartEmergencyProtocolRequest) (*WorkflowInstance, error) {
	instanceID := fmt.Sprintf("WF-EP-%d", time.Now().UnixNano())

	instance := &WorkflowInstance{
		InstanceID:      instanceID,
		WorkflowType:    "emergency_protocol",
		TemplateID:      req.ProtocolID,
		TemplateName:    req.ProtocolName,
		PatientID:       req.PatientID,
		EncounterID:     req.EncounterID,
		InitiatedBy:     req.InitiatedBy,
		Status:          "active",
		CurrentPhase:    "immediate",
		Steps:           make([]*WorkflowStep, 0),
		TimeConstraints: make([]*ActiveTimeConstraint, 0),
		Variables:       req.Variables,
		StartedAt:       time.Now(),
		LastUpdated:     time.Now(),
		Priority:        1, // Emergency protocols are always highest priority
		Metadata:        req.Metadata,
	}

	// Create workflow steps from emergency protocol
	instance.Steps = e.createStepsFromOrderSet(req.Protocol)

	// Activate time constraints immediately
	for _, tc := range req.Protocol.TimeConstraints {
		constraint := e.activateTimeConstraint(instanceID, req.PatientID, tc)
		instance.TimeConstraints = append(instance.TimeConstraints, constraint)
		e.timeConstraints.Store(constraint.ConstraintID, constraint)
	}

	// Store workflow instance
	e.activeWorkflows.Store(instanceID, instance)

	// Emit high-priority start event
	e.emitEvent(&WorkflowEvent{
		EventID:        fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		EventType:      "emergency_protocol_started",
		InstanceID:     instanceID,
		PatientID:      req.PatientID,
		Severity:       "critical",
		Title:          "Emergency Protocol Activated",
		Message:        fmt.Sprintf("EMERGENCY: %s protocol activated for patient %s", req.ProtocolName, req.PatientID),
		Data:           map[string]interface{}{"protocol_id": req.ProtocolID, "constraint_count": len(instance.TimeConstraints)},
		Timestamp:      time.Now(),
		RequiresAction: true,
	})

	// Send to KB-3 for temporal tracking
	if e.kb3Client != nil {
		e.registerTemporalEvent(ctx, instance)
	}

	return instance, nil
}

// StartEmergencyProtocolRequest is the request to start an emergency protocol
type StartEmergencyProtocolRequest struct {
	ProtocolID   string                   `json:"protocol_id"`
	ProtocolName string                   `json:"protocol_name"`
	Protocol     *models.OrderSetTemplate `json:"protocol"`
	PatientID    string                   `json:"patient_id"`
	EncounterID  string                   `json:"encounter_id"`
	InitiatedBy  string                   `json:"initiated_by"`
	Variables    map[string]interface{}   `json:"variables"`
	Metadata     map[string]interface{}   `json:"metadata"`
}

// registerTemporalEvent registers an event with KB-3 temporal logic service
func (e *WorkflowEngine) registerTemporalEvent(ctx context.Context, instance *WorkflowInstance) {
	for _, tc := range instance.TimeConstraints {
		eventData := map[string]interface{}{
			"event_type":    "time_constraint_activated",
			"constraint_id": tc.ConstraintID,
			"patient_id":    tc.PatientID,
			"action":        tc.Action,
			"deadline":      tc.Deadline,
			"severity":      tc.Severity,
			"metrics_code":  tc.MetricsCode,
		}

		eventBytes, _ := json.Marshal(eventData)
		e.kb3Client.RegisterTemporalEvent(ctx, eventBytes)
	}
}

// CompleteStep marks a workflow step as completed
func (e *WorkflowEngine) CompleteStep(ctx context.Context, req *CompleteStepRequest) (*WorkflowStep, error) {
	instanceVal, ok := e.activeWorkflows.Load(req.InstanceID)
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", req.InstanceID)
	}
	instance := instanceVal.(*WorkflowInstance)

	// Find the step
	var step *WorkflowStep
	for _, s := range instance.Steps {
		if s.StepID == req.StepID {
			step = s
			break
		}
	}

	if step == nil {
		return nil, fmt.Errorf("step not found: %s", req.StepID)
	}

	// Update step status
	now := time.Now()
	step.Status = "completed"
	step.CompletedAt = &now
	step.CompletedBy = req.CompletedBy
	step.Notes = req.Notes
	if req.Result != nil {
		step.Result = req.Result
	}

	// Check if this completes a time constraint
	if step.TimeConstraint != nil {
		e.completeTimeConstraint(step.TimeConstraint.ConstraintID, &now)
	}

	// Update instance
	instance.LastUpdated = now
	e.activeWorkflows.Store(req.InstanceID, instance)

	// Check if all required steps are complete
	allComplete := true
	for _, s := range instance.Steps {
		if s.Required && s.Status != "completed" {
			allComplete = false
			break
		}
	}

	if allComplete {
		instance.Status = "completed"
		instance.CompletedAt = &now

		e.emitEvent(&WorkflowEvent{
			EventID:    fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
			EventType:  "workflow_completed",
			InstanceID: req.InstanceID,
			PatientID:  instance.PatientID,
			Severity:   "info",
			Title:      "Workflow Completed",
			Message:    fmt.Sprintf("Workflow '%s' completed for patient %s", instance.TemplateName, instance.PatientID),
			Timestamp:  now,
		})
	}

	// Emit step completed event
	e.emitEvent(&WorkflowEvent{
		EventID:    fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		EventType:  "step_completed",
		InstanceID: req.InstanceID,
		PatientID:  instance.PatientID,
		Severity:   "info",
		Title:      "Step Completed",
		Message:    fmt.Sprintf("Step '%s' completed", step.Name),
		Data:       map[string]interface{}{"step_id": step.StepID, "step_name": step.Name},
		Timestamp:  now,
	})

	return step, nil
}

// CompleteStepRequest is the request to complete a workflow step
type CompleteStepRequest struct {
	InstanceID  string                 `json:"instance_id"`
	StepID      string                 `json:"step_id"`
	CompletedBy string                 `json:"completed_by"`
	Notes       string                 `json:"notes"`
	Result      map[string]interface{} `json:"result"`
}

// completeTimeConstraint marks a time constraint as completed
func (e *WorkflowEngine) completeTimeConstraint(constraintID string, completedAt *time.Time) {
	constraintVal, ok := e.timeConstraints.Load(constraintID)
	if !ok {
		return
	}

	constraint := constraintVal.(*ActiveTimeConstraint)
	constraint.Status = "completed"
	constraint.CompletedAt = completedAt

	e.timeConstraints.Store(constraintID, constraint)

	// Calculate metrics
	timeToComplete := completedAt.Sub(constraint.Deadline.Add(-constraint.Duration))
	withinDeadline := completedAt.Before(constraint.Deadline)

	e.emitEvent(&WorkflowEvent{
		EventID:    fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		EventType:  "constraint_completed",
		InstanceID: constraint.InstanceID,
		PatientID:  constraint.PatientID,
		Severity:   "info",
		Title:      "Time Constraint Met",
		Message:    fmt.Sprintf("Action '%s' completed in %v", constraint.Action, timeToComplete),
		Data: map[string]interface{}{
			"constraint_id":    constraintID,
			"metrics_code":     constraint.MetricsCode,
			"time_to_complete": timeToComplete.String(),
			"within_deadline":  withinDeadline,
		},
		Timestamp: *completedAt,
	})
}

// timeConstraintMonitor runs in the background to monitor time constraints
func (e *WorkflowEngine) timeConstraintMonitor(ctx context.Context) {
	ticker := time.NewTicker(e.tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopTicker:
			return
		case <-ticker.C:
			e.checkTimeConstraints()
		}
	}
}

// checkTimeConstraints evaluates all active time constraints
func (e *WorkflowEngine) checkTimeConstraints() {
	now := time.Now()

	e.timeConstraints.Range(func(key, value interface{}) bool {
		constraint := value.(*ActiveTimeConstraint)

		if constraint.Status == "completed" || constraint.Status == "missed" {
			return true // Skip completed constraints
		}

		// Check if deadline passed
		if now.After(constraint.Deadline) {
			constraint.Status = "missed"
			e.timeConstraints.Store(key, constraint)

			e.emitEvent(&WorkflowEvent{
				EventID:        fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
				EventType:      "constraint_missed",
				InstanceID:     constraint.InstanceID,
				PatientID:      constraint.PatientID,
				Severity:       "critical",
				Title:          "Time Constraint MISSED",
				Message:        fmt.Sprintf("CRITICAL: %s deadline exceeded by %v", constraint.Action, now.Sub(constraint.Deadline)),
				Data:           map[string]interface{}{"constraint_id": constraint.ConstraintID, "metrics_code": constraint.MetricsCode},
				Timestamp:      now,
				RequiresAction: true,
			})

			return true
		}

		// Check warning time
		if now.After(constraint.WarningTime) && constraint.Status == "pending" {
			constraint.Status = "warning_sent"
			constraint.AlertsSent++
			e.timeConstraints.Store(key, constraint)

			remaining := constraint.Deadline.Sub(now)
			severity := "warning"
			if remaining < 5*time.Minute {
				severity = "critical"
			}

			e.emitEvent(&WorkflowEvent{
				EventID:        fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
				EventType:      "constraint_warning",
				InstanceID:     constraint.InstanceID,
				PatientID:      constraint.PatientID,
				Severity:       severity,
				Title:          "Time Constraint Warning",
				Message:        fmt.Sprintf("WARNING: %s due in %v", constraint.Action, remaining.Round(time.Second)),
				Data:           map[string]interface{}{"constraint_id": constraint.ConstraintID, "time_remaining": remaining.String()},
				Timestamp:      now,
				RequiresAction: true,
			})
		}

		return true
	})
}

// GetWorkflowInstance retrieves a workflow instance
func (e *WorkflowEngine) GetWorkflowInstance(instanceID string) (*WorkflowInstance, error) {
	instanceVal, ok := e.activeWorkflows.Load(instanceID)
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", instanceID)
	}
	return instanceVal.(*WorkflowInstance), nil
}

// GetPatientWorkflows retrieves all workflows for a patient
func (e *WorkflowEngine) GetPatientWorkflows(patientID string) []*WorkflowInstance {
	workflows := make([]*WorkflowInstance, 0)

	e.activeWorkflows.Range(func(key, value interface{}) bool {
		instance := value.(*WorkflowInstance)
		if instance.PatientID == patientID {
			workflows = append(workflows, instance)
		}
		return true
	})

	return workflows
}

// GetActiveTimeConstraints retrieves all active time constraints
func (e *WorkflowEngine) GetActiveTimeConstraints() []*ActiveTimeConstraint {
	constraints := make([]*ActiveTimeConstraint, 0)

	e.timeConstraints.Range(func(key, value interface{}) bool {
		constraint := value.(*ActiveTimeConstraint)
		if constraint.Status != "completed" && constraint.Status != "missed" {
			constraints = append(constraints, constraint)
		}
		return true
	})

	return constraints
}

// GetConstraintsByPatient retrieves time constraints for a patient
func (e *WorkflowEngine) GetConstraintsByPatient(patientID string) []*ActiveTimeConstraint {
	constraints := make([]*ActiveTimeConstraint, 0)

	e.timeConstraints.Range(func(key, value interface{}) bool {
		constraint := value.(*ActiveTimeConstraint)
		if constraint.PatientID == patientID {
			constraints = append(constraints, constraint)
		}
		return true
	})

	return constraints
}

// CancelWorkflow cancels an active workflow
func (e *WorkflowEngine) CancelWorkflow(instanceID string, reason string, cancelledBy string) error {
	instanceVal, ok := e.activeWorkflows.Load(instanceID)
	if !ok {
		return fmt.Errorf("workflow not found: %s", instanceID)
	}

	instance := instanceVal.(*WorkflowInstance)
	instance.Status = "cancelled"
	now := time.Now()
	instance.CompletedAt = &now
	instance.LastUpdated = now
	instance.Metadata["cancellation_reason"] = reason
	instance.Metadata["cancelled_by"] = cancelledBy

	e.activeWorkflows.Store(instanceID, instance)

	// Cancel associated time constraints
	for _, tc := range instance.TimeConstraints {
		if constraintVal, ok := e.timeConstraints.Load(tc.ConstraintID); ok {
			constraint := constraintVal.(*ActiveTimeConstraint)
			constraint.Status = "cancelled"
			e.timeConstraints.Store(tc.ConstraintID, constraint)
		}
	}

	e.emitEvent(&WorkflowEvent{
		EventID:    fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		EventType:  "workflow_cancelled",
		InstanceID: instanceID,
		PatientID:  instance.PatientID,
		Severity:   "warning",
		Title:      "Workflow Cancelled",
		Message:    fmt.Sprintf("Workflow '%s' cancelled: %s", instance.TemplateName, reason),
		Data:       map[string]interface{}{"reason": reason, "cancelled_by": cancelledBy},
		Timestamp:  now,
	})

	return nil
}

// PauseWorkflow pauses a workflow
func (e *WorkflowEngine) PauseWorkflow(instanceID string, reason string) error {
	instanceVal, ok := e.activeWorkflows.Load(instanceID)
	if !ok {
		return fmt.Errorf("workflow not found: %s", instanceID)
	}

	instance := instanceVal.(*WorkflowInstance)
	instance.Status = "paused"
	instance.LastUpdated = time.Now()
	instance.Metadata["pause_reason"] = reason

	e.activeWorkflows.Store(instanceID, instance)

	return nil
}

// ResumeWorkflow resumes a paused workflow
func (e *WorkflowEngine) ResumeWorkflow(instanceID string) error {
	instanceVal, ok := e.activeWorkflows.Load(instanceID)
	if !ok {
		return fmt.Errorf("workflow not found: %s", instanceID)
	}

	instance := instanceVal.(*WorkflowInstance)
	if instance.Status != "paused" {
		return fmt.Errorf("workflow is not paused")
	}

	instance.Status = "active"
	instance.LastUpdated = time.Now()
	delete(instance.Metadata, "pause_reason")

	e.activeWorkflows.Store(instanceID, instance)

	return nil
}

// emitEvent sends an event to all registered handlers
func (e *WorkflowEngine) emitEvent(event *WorkflowEvent) {
	e.mu.RLock()
	handlers := make([]EventHandler, len(e.eventHandlers))
	copy(handlers, e.eventHandlers)
	e.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(event); err != nil {
				// Log error but don't stop other handlers
			}
		}(handler)
	}
}

// WorkflowMetrics provides metrics about workflows and time constraints
type WorkflowMetrics struct {
	ActiveWorkflows        int            `json:"active_workflows"`
	CompletedWorkflows     int            `json:"completed_workflows"`
	CancelledWorkflows     int            `json:"cancelled_workflows"`
	ActiveConstraints      int            `json:"active_constraints"`
	MissedConstraints      int            `json:"missed_constraints"`
	CompletedConstraints   int            `json:"completed_constraints"`
	WorkflowsByType        map[string]int `json:"workflows_by_type"`
	ConstraintsByMetric    map[string]int `json:"constraints_by_metric"`
	AverageCompletionTime  time.Duration  `json:"average_completion_time"`
}

// GetMetrics returns workflow engine metrics
func (e *WorkflowEngine) GetMetrics() *WorkflowMetrics {
	metrics := &WorkflowMetrics{
		WorkflowsByType:     make(map[string]int),
		ConstraintsByMetric: make(map[string]int),
	}

	var totalCompletionTime time.Duration
	completedCount := 0

	e.activeWorkflows.Range(func(key, value interface{}) bool {
		instance := value.(*WorkflowInstance)
		metrics.WorkflowsByType[instance.WorkflowType]++

		switch instance.Status {
		case "active":
			metrics.ActiveWorkflows++
		case "completed":
			metrics.CompletedWorkflows++
			if instance.CompletedAt != nil {
				totalCompletionTime += instance.CompletedAt.Sub(instance.StartedAt)
				completedCount++
			}
		case "cancelled":
			metrics.CancelledWorkflows++
		}

		return true
	})

	e.timeConstraints.Range(func(key, value interface{}) bool {
		constraint := value.(*ActiveTimeConstraint)
		if constraint.MetricsCode != "" {
			metrics.ConstraintsByMetric[constraint.MetricsCode]++
		}

		switch constraint.Status {
		case "pending", "warning_sent":
			metrics.ActiveConstraints++
		case "missed":
			metrics.MissedConstraints++
		case "completed":
			metrics.CompletedConstraints++
		}

		return true
	})

	if completedCount > 0 {
		metrics.AverageCompletionTime = totalCompletionTime / time.Duration(completedCount)
	}

	return metrics
}
