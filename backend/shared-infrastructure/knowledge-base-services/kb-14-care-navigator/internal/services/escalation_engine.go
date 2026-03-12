// Package services provides business logic for KB-14 Care Navigator
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
)

// EscalationEngine handles task escalation logic
type EscalationEngine struct {
	taskRepo          *database.TaskRepository
	teamRepo          *database.TeamRepository
	escalationRepo    *database.EscalationRepository
	notificationSvc   *NotificationService
	log               *logrus.Entry
}

// NewEscalationEngine creates a new EscalationEngine
func NewEscalationEngine(
	taskRepo *database.TaskRepository,
	teamRepo *database.TeamRepository,
	escalationRepo *database.EscalationRepository,
	notificationSvc *NotificationService,
	log *logrus.Entry,
) *EscalationEngine {
	return &EscalationEngine{
		taskRepo:        taskRepo,
		teamRepo:        teamRepo,
		escalationRepo:  escalationRepo,
		notificationSvc: notificationSvc,
		log:             log.WithField("service", "escalation-engine"),
	}
}

// CheckAndEscalate checks all active tasks and escalates as needed
func (e *EscalationEngine) CheckAndEscalate(ctx context.Context) (int, error) {
	// Get tasks that might need escalation
	tasks, err := e.taskRepo.FindNeedingEscalation(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find tasks needing escalation: %w", err)
	}

	escalated := 0
	for _, task := range tasks {
		if err := e.checkTask(ctx, &task); err != nil {
			e.log.WithError(err).WithField("task_id", task.ID).Warn("Failed to check task for escalation")
			continue
		}
		escalated++
	}

	e.log.WithField("escalated", escalated).Info("Escalation check completed")
	return escalated, nil
}

// checkTask checks a single task for escalation
func (e *EscalationEngine) checkTask(ctx context.Context, task *models.Task) error {
	if task.DueDate == nil || task.SLAMinutes == 0 {
		return nil // No SLA to check
	}

	// Calculate SLA elapsed percentage
	slaElapsed := e.calculateSLAElapsed(task.CreatedAt, task.SLAMinutes)

	// Determine the appropriate escalation level
	newLevel := models.CalculateEscalationLevel(slaElapsed, task.Priority)

	// Compare with current escalation level
	if int(newLevel) <= task.EscalationLevel {
		return nil // No escalation needed
	}

	// Create escalation record
	return e.createEscalation(ctx, task, newLevel, slaElapsed)
}

// calculateSLAElapsed calculates the percentage of SLA elapsed
func (e *EscalationEngine) calculateSLAElapsed(createdAt time.Time, slaMinutes int) float64 {
	if slaMinutes <= 0 {
		return 0
	}

	elapsedMinutes := time.Since(createdAt).Minutes()
	return elapsedMinutes / float64(slaMinutes)
}

// createEscalation creates an escalation record and sends notifications
func (e *EscalationEngine) createEscalation(ctx context.Context, task *models.Task, level models.EscalationLevel, slaElapsed float64) error {
	// Calculate time overdue
	timeOverdue := 0
	if task.DueDate != nil {
		timeOverdue = int(time.Since(*task.DueDate).Minutes())
	}

	// Generate reason
	reason := models.GetEscalationReason(level, task.Type, slaElapsed)

	// Find escalation target based on level
	escalateTo, escalateToRole := e.findEscalationTarget(ctx, task, level)

	// Create escalation record
	escalation := &models.Escalation{
		TaskID:            task.ID,
		Level:             level,
		Reason:            reason,
		EscalatedTo:       escalateTo,
		EscalatedToRole:   escalateToRole,
		SLAElapsedPercent: slaElapsed * 100,
		TimeOverdue:       timeOverdue,
	}

	if err := e.escalationRepo.Create(ctx, escalation); err != nil {
		return fmt.Errorf("failed to create escalation: %w", err)
	}

	// Update task escalation level
	task.EscalationLevel = int(level)
	if level >= models.EscalationUrgent {
		task.Status = models.TaskStatusEscalated
	}

	if err := e.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task escalation level: %w", err)
	}

	// Send notification
	if escalateTo != nil {
		recipientName := ""
		if member, err := e.teamRepo.GetMemberByID(ctx, *escalateTo); err == nil && member != nil {
			recipientName = member.Name
		}

		notification := models.BuildEscalationNotification(task, escalation, escalateTo.String(), recipientName)
		if result := e.notificationSvc.Send(ctx, &notification); result != nil {
			// Update escalation with notification info
			escalation.NotificationSent = result.Success
			if len(result.Results) > 0 {
				escalation.NotificationChannel = string(result.Results[0].Channel)
			}
			now := time.Now().UTC()
			escalation.NotificationSentAt = &now
			_ = e.escalationRepo.Update(ctx, escalation)
		}
	}

	e.log.WithFields(logrus.Fields{
		"task_id":       task.ID,
		"level":         level,
		"sla_elapsed":   fmt.Sprintf("%.1f%%", slaElapsed*100),
		"escalated_to":  escalateTo,
	}).Info("Task escalated")

	return nil
}

// findEscalationTarget finds the appropriate person to escalate to based on level
func (e *EscalationEngine) findEscalationTarget(ctx context.Context, task *models.Task, level models.EscalationLevel) (*uuid.UUID, string) {
	// If task is assigned, find supervisor chain
	if task.AssignedTo != nil {
		member, err := e.teamRepo.GetMemberByID(ctx, *task.AssignedTo)
		if err == nil && member != nil {
			switch level {
			case models.EscalationWarning:
				// Notify the assignee themselves
				return task.AssignedTo, member.Role
			case models.EscalationUrgent:
				// Escalate to supervisor
				if member.SupervisorID != nil {
					supervisor, err := e.teamRepo.GetMemberByID(ctx, *member.SupervisorID)
					if err == nil && supervisor != nil {
						return member.SupervisorID, supervisor.Role
					}
				}
			case models.EscalationCritical, models.EscalationExecutive:
				// Escalate to team manager
				if team, err := e.teamRepo.GetTeamByID(ctx, member.TeamID); err == nil && team != nil {
					if team.ManagerID != nil {
						return team.ManagerID, "Manager"
					}
				}
			}
		}
	}

	// Fallback: Find someone with the right role
	role := task.Type.GetDefaultRole()
	if members, err := e.teamRepo.GetMembersByRole(ctx, role); err == nil && len(members) > 0 {
		return &members[0].ID, role
	}

	return nil, ""
}

// ManualEscalate manually escalates a task to a specific level
func (e *EscalationEngine) ManualEscalate(ctx context.Context, taskID uuid.UUID, req *models.EscalationRequest) (*models.Escalation, error) {
	task, err := e.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Calculate current SLA
	slaElapsed := 0.0
	if task.SLAMinutes > 0 {
		slaElapsed = e.calculateSLAElapsed(task.CreatedAt, task.SLAMinutes)
	}

	reason := req.Reason
	if reason == "" {
		reason = models.GetEscalationReason(req.Level, task.Type, slaElapsed)
	}

	timeOverdue := 0
	if task.DueDate != nil {
		timeOverdue = int(time.Since(*task.DueDate).Minutes())
	}

	// Determine escalation target
	var escalateTo *uuid.UUID
	escalateToRole := ""
	if req.EscalateTo != nil {
		escalateTo = req.EscalateTo
		if member, err := e.teamRepo.GetMemberByID(ctx, *req.EscalateTo); err == nil && member != nil {
			escalateToRole = member.Role
		}
	} else {
		escalateTo, escalateToRole = e.findEscalationTarget(ctx, task, req.Level)
	}

	// Create escalation
	escalation := &models.Escalation{
		TaskID:            taskID,
		Level:             req.Level,
		Reason:            reason,
		EscalatedTo:       escalateTo,
		EscalatedToRole:   escalateToRole,
		SLAElapsedPercent: slaElapsed * 100,
		TimeOverdue:       timeOverdue,
	}

	if err := e.escalationRepo.Create(ctx, escalation); err != nil {
		return nil, err
	}

	// Update task
	task.EscalationLevel = int(req.Level)
	// Manual escalation always sets status to ESCALATED (indicates urgent attention needed)
	task.Status = models.TaskStatusEscalated

	if err := e.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	e.log.WithFields(logrus.Fields{
		"task_id": taskID,
		"level":   req.Level,
		"reason":  reason,
	}).Info("Task manually escalated")

	return escalation, nil
}

// AcknowledgeEscalation acknowledges an escalation
func (e *EscalationEngine) AcknowledgeEscalation(ctx context.Context, escalationID uuid.UUID, acknowledgedBy uuid.UUID) error {
	if err := e.escalationRepo.Acknowledge(ctx, escalationID, acknowledgedBy); err != nil {
		return err
	}

	e.log.WithFields(logrus.Fields{
		"escalation_id":   escalationID,
		"acknowledged_by": acknowledgedBy,
	}).Info("Escalation acknowledged")

	return nil
}

// GetTaskEscalations retrieves all escalations for a task
func (e *EscalationEngine) GetTaskEscalations(ctx context.Context, taskID uuid.UUID) ([]models.Escalation, error) {
	return e.escalationRepo.FindByTask(ctx, taskID)
}

// GetUnacknowledgedEscalations retrieves all unacknowledged escalations
func (e *EscalationEngine) GetUnacknowledgedEscalations(ctx context.Context) ([]models.Escalation, error) {
	return e.escalationRepo.FindUnacknowledged(ctx)
}

// GetEscalationSummary retrieves escalation statistics
func (e *EscalationEngine) GetEscalationSummary(ctx context.Context) (*models.EscalationSummary, error) {
	return e.escalationRepo.GetEscalationSummary(ctx)
}

// ResolveEscalation resolves an escalation
func (e *EscalationEngine) ResolveEscalation(ctx context.Context, escalationID uuid.UUID, resolvedBy uuid.UUID, resolution string) error {
	if err := e.escalationRepo.Resolve(ctx, escalationID, resolvedBy, resolution); err != nil {
		return err
	}

	e.log.WithFields(logrus.Fields{
		"escalation_id": escalationID,
		"resolved_by":   resolvedBy,
		"resolution":    resolution,
	}).Info("Escalation resolved")

	return nil
}

// EscalateTask manually escalates a task to the next level
func (e *EscalationEngine) EscalateTask(ctx context.Context, task *models.Task, reason string) (*models.Escalation, error) {
	// Determine next escalation level
	currentLevel := models.EscalationLevel(task.EscalationLevel)
	nextLevel := currentLevel + 1
	if nextLevel > models.EscalationExecutive {
		nextLevel = models.EscalationExecutive
	}

	req := &models.EscalationRequest{
		Level:  nextLevel,
		Reason: reason,
	}

	return e.ManualEscalate(ctx, task.ID, req)
}
