// Package temporal provides the pathway state machine engine for clinical pathway management
package temporal

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// PathwayEngine manages clinical pathway instances and state transitions
type PathwayEngine struct {
	mu        sync.RWMutex
	instances map[string]*models.PathwayInstance
}

// NewPathwayEngine creates a new pathway engine instance
func NewPathwayEngine() *PathwayEngine {
	return &PathwayEngine{
		instances: make(map[string]*models.PathwayInstance),
	}
}

// StartPathway initiates a new pathway instance for a patient
func (e *PathwayEngine) StartPathway(protocol models.Protocol, patientID string, context map[string]interface{}) (*models.PathwayInstance, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !protocol.Active {
		return nil, fmt.Errorf("protocol %s is not active", protocol.ProtocolID)
	}

	instanceID := uuid.New().String()
	now := time.Now()

	// Determine initial stage
	var initialStage string
	if len(protocol.Stages) > 0 {
		initialStage = protocol.Stages[0].StageID
	}

	// Create actions from protocol stages
	var actions []models.PathwayAction
	for _, stage := range protocol.Stages {
		for _, action := range stage.Actions {
			deadline := now
			if action.Deadline > 0 {
				deadline = now.Add(action.Deadline)
			}

			pa := models.PathwayAction{
				ActionID:    fmt.Sprintf("%s-%s-%s", instanceID[:8], stage.StageID, action.ActionID),
				Name:        action.Name,
				Type:        action.Type,
				Status:      models.ActionPending,
				Deadline:    deadline,
				GracePeriod: 15 * time.Minute, // Default 15-minute grace period
				Required:    action.Required,
				StageID:     stage.StageID,
				Description: action.Description,
			}
			actions = append(actions, pa)
		}
	}

	instance := &models.PathwayInstance{
		InstanceID:   instanceID,
		PathwayID:    protocol.ProtocolID,
		PatientID:    patientID,
		CurrentStage: initialStage,
		Status:       models.PathwayActive,
		StartedAt:    now,
		Context:      context,
		Actions:      actions,
		AuditLog: []models.AuditEntry{
			{
				EntryID:   uuid.New().String(),
				Action:    "pathway_started",
				Timestamp: now,
				Details: map[string]interface{}{
					"protocol_id": protocol.ProtocolID,
					"patient_id":  patientID,
					"context":     context,
				},
			},
		},
	}

	e.instances[instanceID] = instance
	return instance, nil
}

// GetPathwayStatus retrieves the current status of a pathway instance
func (e *PathwayEngine) GetPathwayStatus(instanceID string) (*models.PathwayInstance, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	// Update action statuses based on current time
	e.updateActionStatuses(instance)

	return instance, nil
}

// GetPendingActions returns all pending actions for a pathway instance
func (e *PathwayEngine) GetPendingActions(instanceID string) ([]models.PathwayAction, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	e.updateActionStatuses(instance)

	var pending []models.PathwayAction
	for _, action := range instance.Actions {
		if action.Status == models.ActionPending {
			pending = append(pending, action)
		}
	}
	return pending, nil
}

// GetOverdueActions returns all overdue actions for a pathway instance
func (e *PathwayEngine) GetOverdueActions(instanceID string) ([]models.PathwayAction, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	e.updateActionStatuses(instance)

	var overdue []models.PathwayAction
	for _, action := range instance.Actions {
		if action.Status == models.ActionOverdue || action.Status == models.ActionMissed {
			overdue = append(overdue, action)
		}
	}
	return overdue, nil
}

// EvaluateConstraints evaluates all time constraints for a pathway instance
func (e *PathwayEngine) EvaluateConstraints(instanceID string) ([]models.ConstraintEvaluation, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	var evaluations []models.ConstraintEvaluation
	now := time.Now()

	for i := range instance.Actions {
		action := &instance.Actions[i]

		eval := models.ConstraintEvaluation{
			ActionID:   action.ActionID,
			ActionName: action.Name,
			Deadline:   action.Deadline,
		}

		if action.CompletedAt != nil {
			// Action completed - check if on time
			if action.CompletedAt.Before(action.Deadline) || action.CompletedAt.Equal(action.Deadline) {
				eval.Status = models.StatusMet
				remaining := action.Deadline.Sub(*action.CompletedAt)
				eval.TimeRemaining = &remaining
			} else {
				eval.Status = models.StatusMissed
				overdue := action.CompletedAt.Sub(action.Deadline)
				eval.TimeOverdue = &overdue
			}
			eval.CompletedAt = action.CompletedAt
		} else {
			// Action not completed
			if now.Before(action.Deadline) {
				remaining := action.Deadline.Sub(now)
				eval.TimeRemaining = &remaining

				// Check if approaching (within 20% of deadline)
				totalTime := action.Deadline.Sub(instance.StartedAt)
				threshold := totalTime / 5
				if remaining < threshold {
					eval.Status = models.StatusApproaching
				} else {
					eval.Status = models.StatusPending
				}
			} else {
				overdue := now.Sub(action.Deadline)
				eval.TimeOverdue = &overdue

				// Check if within grace period
				graceEnd := action.Deadline.Add(action.GracePeriod)
				if now.Before(graceEnd) {
					eval.Status = models.StatusOverdue
				} else {
					eval.Status = models.StatusMissed
				}
			}
		}

		evaluations = append(evaluations, eval)
	}

	return evaluations, nil
}

// CompleteAction marks an action as completed
func (e *PathwayEngine) CompleteAction(instanceID, actionID, completedBy string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	now := time.Now()
	actionFound := false

	for i := range instance.Actions {
		if instance.Actions[i].ActionID == actionID {
			actionFound = true
			instance.Actions[i].CompletedAt = &now
			instance.Actions[i].CompletedBy = completedBy

			if now.Before(instance.Actions[i].Deadline) || now.Equal(instance.Actions[i].Deadline) {
				instance.Actions[i].Status = models.ActionMet
			} else {
				instance.Actions[i].Status = models.ActionMissed
			}

			// Add audit entry
			instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
				EntryID:   uuid.New().String(),
				Action:    "action_completed",
				UserID:    completedBy,
				Timestamp: now,
				Details: map[string]interface{}{
					"action_id":   actionID,
					"action_name": instance.Actions[i].Name,
					"on_time":     instance.Actions[i].Status == models.ActionMet,
				},
			})
			break
		}
	}

	if !actionFound {
		return fmt.Errorf("action not found: %s", actionID)
	}

	// Check if all required actions in current stage are complete
	e.checkStageCompletion(instance)

	return nil
}

// AdvanceStage moves the pathway to the next stage
func (e *PathwayEngine) AdvanceStage(instanceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	// This is a simplified stage advancement
	// In production, this would validate stage prerequisites
	instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
		EntryID:   uuid.New().String(),
		Action:    "stage_advanced",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"from_stage": instance.CurrentStage,
		},
	})

	return nil
}

// SuspendPathway pauses a pathway instance
func (e *PathwayEngine) SuspendPathway(instanceID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	now := time.Now()
	instance.Status = models.PathwaySuspended
	instance.SuspendedAt = &now

	instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
		EntryID:   uuid.New().String(),
		Action:    "pathway_suspended",
		Timestamp: now,
		Details: map[string]interface{}{
			"reason": reason,
		},
	})

	return nil
}

// ResumePathway resumes a suspended pathway
func (e *PathwayEngine) ResumePathway(instanceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	if instance.Status != models.PathwaySuspended {
		return fmt.Errorf("pathway is not suspended")
	}

	now := time.Now()

	// Adjust deadlines based on suspension duration
	if instance.SuspendedAt != nil {
		suspensionDuration := now.Sub(*instance.SuspendedAt)
		for i := range instance.Actions {
			if instance.Actions[i].CompletedAt == nil {
				instance.Actions[i].Deadline = instance.Actions[i].Deadline.Add(suspensionDuration)
			}
		}
	}

	instance.Status = models.PathwayActive
	instance.SuspendedAt = nil

	instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
		EntryID:   uuid.New().String(),
		Action:    "pathway_resumed",
		Timestamp: now,
	})

	return nil
}

// CancelPathway cancels a pathway instance
func (e *PathwayEngine) CancelPathway(instanceID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	now := time.Now()
	instance.Status = models.PathwayCancelled
	instance.CompletedAt = &now

	instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
		EntryID:   uuid.New().String(),
		Action:    "pathway_cancelled",
		Timestamp: now,
		Details: map[string]interface{}{
			"reason": reason,
		},
	})

	return nil
}

// CompletePathway marks a pathway as completed
func (e *PathwayEngine) CompletePathway(instanceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	now := time.Now()
	instance.Status = models.PathwayCompleted
	instance.CompletedAt = &now

	instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
		EntryID:   uuid.New().String(),
		Action:    "pathway_completed",
		Timestamp: now,
	})

	return nil
}

// GetPatientPathways retrieves all pathway instances for a patient
func (e *PathwayEngine) GetPatientPathways(patientID string) []*models.PathwayInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var pathways []*models.PathwayInstance
	for _, instance := range e.instances {
		if instance.PatientID == patientID {
			e.updateActionStatuses(instance)
			pathways = append(pathways, instance)
		}
	}
	return pathways
}

// GetActivePathways retrieves all active pathway instances
func (e *PathwayEngine) GetActivePathways() []*models.PathwayInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var pathways []*models.PathwayInstance
	for _, instance := range e.instances {
		if instance.Status == models.PathwayActive {
			e.updateActionStatuses(instance)
			pathways = append(pathways, instance)
		}
	}
	return pathways
}

// GetPathwayAudit retrieves the audit log for a pathway instance
func (e *PathwayEngine) GetPathwayAudit(instanceID string) ([]models.AuditEntry, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("pathway instance not found: %s", instanceID)
	}

	return instance.AuditLog, nil
}

// updateActionStatuses updates action statuses based on current time
func (e *PathwayEngine) updateActionStatuses(instance *models.PathwayInstance) {
	now := time.Now()

	for i := range instance.Actions {
		action := &instance.Actions[i]

		// Skip completed actions
		if action.CompletedAt != nil {
			continue
		}

		if now.Before(action.Deadline) {
			// Check if approaching
			totalTime := action.Deadline.Sub(instance.StartedAt)
			remaining := action.Deadline.Sub(now)
			threshold := totalTime / 5

			if remaining < threshold {
				action.Status = models.ActionApproaching
			} else {
				action.Status = models.ActionPending
			}
		} else {
			// Past deadline
			graceEnd := action.Deadline.Add(action.GracePeriod)
			if now.Before(graceEnd) {
				action.Status = models.ActionOverdue
			} else {
				action.Status = models.ActionMissed
			}
		}
	}
}

// checkStageCompletion checks if all required actions in current stage are complete
func (e *PathwayEngine) checkStageCompletion(instance *models.PathwayInstance) {
	allRequiredComplete := true

	for _, action := range instance.Actions {
		if action.StageID == instance.CurrentStage && action.Required {
			if action.CompletedAt == nil {
				allRequiredComplete = false
				break
			}
		}
	}

	if allRequiredComplete {
		instance.AuditLog = append(instance.AuditLog, models.AuditEntry{
			EntryID:   uuid.New().String(),
			Action:    "stage_requirements_met",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"stage": instance.CurrentStage,
			},
		})
	}
}

// GetOverdueAlerts returns all overdue actions across all active pathways
func (e *PathwayEngine) GetOverdueAlerts() []OverdueAlert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var alerts []OverdueAlert
	now := time.Now()

	for _, instance := range e.instances {
		if instance.Status != models.PathwayActive {
			continue
		}

		for _, action := range instance.Actions {
			if action.CompletedAt != nil {
				continue
			}

			if now.After(action.Deadline) {
				overdue := now.Sub(action.Deadline)
				severity := "warning"

				// Determine severity based on how overdue
				if overdue > 2*time.Hour {
					severity = "critical"
				} else if overdue > 30*time.Minute {
					severity = "major"
				}

				alerts = append(alerts, OverdueAlert{
					InstanceID:  instance.InstanceID,
					PatientID:   instance.PatientID,
					PathwayID:   instance.PathwayID,
					ActionID:    action.ActionID,
					ActionName:  action.Name,
					Deadline:    action.Deadline,
					OverdueBy:   overdue,
					Severity:    severity,
					CurrentStage: instance.CurrentStage,
				})
			}
		}
	}

	return alerts
}

// OverdueAlert represents an alert for an overdue action
type OverdueAlert struct {
	InstanceID   string        `json:"instance_id"`
	PatientID    string        `json:"patient_id"`
	PathwayID    string        `json:"pathway_id"`
	ActionID     string        `json:"action_id"`
	ActionName   string        `json:"action_name"`
	Deadline     time.Time     `json:"deadline"`
	OverdueBy    time.Duration `json:"overdue_by"`
	Severity     string        `json:"severity"`
	CurrentStage string        `json:"current_stage"`
}

// Global pathway engine instance
var globalPathwayEngine *PathwayEngine
var pathwayEngineOnce sync.Once

// GetPathwayEngine returns the global pathway engine instance
func GetPathwayEngine() *PathwayEngine {
	pathwayEngineOnce.Do(func() {
		globalPathwayEngine = NewPathwayEngine()
	})
	return globalPathwayEngine
}
