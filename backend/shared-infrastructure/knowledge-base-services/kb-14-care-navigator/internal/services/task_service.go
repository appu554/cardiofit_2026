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

// TaskService handles task business logic
type TaskService struct {
	taskRepo       *database.TaskRepository
	teamRepo       *database.TeamRepository
	escalationRepo *database.EscalationRepository
	governanceSvc  *GovernanceService
	log            *logrus.Entry
}

// NewTaskService creates a new TaskService
func NewTaskService(
	taskRepo *database.TaskRepository,
	teamRepo *database.TeamRepository,
	escalationRepo *database.EscalationRepository,
	governanceSvc *GovernanceService,
	log *logrus.Entry,
) *TaskService {
	return &TaskService{
		taskRepo:       taskRepo,
		teamRepo:       teamRepo,
		escalationRepo: escalationRepo,
		governanceSvc:  governanceSvc,
		log:            log.WithField("service", "task"),
	}
}

// Create creates a new task
func (s *TaskService) Create(ctx context.Context, req *models.CreateTaskRequest) (*models.Task, error) {
	// Validate task type
	if !req.Type.IsValid() {
		return nil, fmt.Errorf("invalid task type: %s", req.Type)
	}

	// Check for duplicate source + sourceID (idempotent creation)
	if req.SourceID != "" {
		existing, err := s.taskRepo.FindBySourceID(ctx, string(req.Source), req.SourceID)
		if err == nil && existing != nil {
			s.log.WithFields(logrus.Fields{
				"existing_task_id": existing.ID,
				"source":           req.Source,
				"source_id":        req.SourceID,
			}).Debug("Returning existing task for duplicate source")
			return existing, nil
		}
	}

	// Generate task ID
	taskNumber := generateTaskNumber(req.Type)

	// Set defaults
	priority := req.Priority
	if priority == "" {
		priority = req.Type.GetDefaultPriority()
	}

	slaMinutes := req.SLAMinutes
	if slaMinutes == 0 {
		slaMinutes = req.Type.GetDefaultSLAMinutes()
	}

	assignedRole := req.AssignedRole
	if assignedRole == "" {
		assignedRole = req.Type.GetDefaultRole()
	}

	// Calculate due date
	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate
	} else {
		due := time.Now().UTC().Add(time.Duration(slaMinutes) * time.Minute)
		dueDate = &due
	}

	// Create task
	task := &models.Task{
		TaskID:       taskNumber,
		Type:         req.Type,
		Status:       models.TaskStatusCreated,
		Priority:     priority,
		Source:       req.Source,
		SourceID:     req.SourceID,
		PatientID:    req.PatientID,
		EncounterID:  req.EncounterID,
		Title:        req.Title,
		Description:  req.Description,
		Instructions: req.Instructions,
		ClinicalNote: req.ClinicalNote,
		TeamID:       req.TeamID,
		AssignedTo:   req.AssignedTo,
		AssignedRole: assignedRole,
		DueDate:      dueDate,
		SLAMinutes:   slaMinutes,
		Actions:      models.ActionSlice(req.Actions),
		Metadata:     models.JSONMap(req.Metadata),
	}

	// If assigned, set assigned timestamp and status
	if task.AssignedTo != nil {
		now := time.Now().UTC()
		task.AssignedAt = &now
		task.Status = models.TaskStatusAssigned
	}

	// Create in database
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Publish audit event (non-blocking)
	if s.governanceSvc != nil {
		go func() {
			auditCtx := SystemAuditContext()
			if err := s.governanceSvc.PublishTaskCreated(context.Background(), task, auditCtx, req.SourceID); err != nil {
				s.log.WithError(err).Warn("Failed to publish task created audit event")
			}
		}()
	}

	s.log.WithFields(logrus.Fields{
		"task_id":    task.ID,
		"task_number": task.TaskID,
		"type":       task.Type,
		"source":     task.Source,
		"patient_id": task.PatientID,
	}).Info("Task created")

	return task, nil
}

// CreateWithAudit creates a new task with explicit audit context
func (s *TaskService) CreateWithAudit(ctx context.Context, req *models.CreateTaskRequest, auditCtx *AuditContext) (*models.Task, error) {
	task, err := s.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Publish audit event with explicit context
	if s.governanceSvc != nil && auditCtx != nil {
		go func() {
			if err := s.governanceSvc.PublishTaskCreated(context.Background(), task, auditCtx, req.SourceID); err != nil {
				s.log.WithError(err).Warn("Failed to publish task created audit event")
			}
		}()
	}

	return task, nil
}

// GetByID retrieves a task by ID
func (s *TaskService) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	return s.taskRepo.GetByID(ctx, id)
}

// GetByTaskID retrieves a task by task_id string
func (s *TaskService) GetByTaskID(ctx context.Context, taskID string) (*models.Task, error) {
	return s.taskRepo.GetByTaskID(ctx, taskID)
}

// Update updates a task
func (s *TaskService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Instructions != nil {
		task.Instructions = *req.Instructions
	}
	if req.ClinicalNote != nil {
		task.ClinicalNote = *req.ClinicalNote
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.Metadata != nil {
		task.Metadata = models.JSONMap(req.Metadata)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// Assign assigns a task to a user
func (s *TaskService) Assign(ctx context.Context, taskID uuid.UUID, req *models.AssignTaskRequest) (*models.Task, error) {
	return s.AssignWithAudit(ctx, taskID, req, SystemAuditContext())
}

// AssignWithAudit assigns a task to a user with audit context
func (s *TaskService) AssignWithAudit(ctx context.Context, taskID uuid.UUID, req *models.AssignTaskRequest, auditCtx *AuditContext) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Validate assignment
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusCancelled {
		return nil, fmt.Errorf("cannot assign task in %s status", task.Status)
	}

	// Validate assignee exists and is active
	// Try to find member by ID first (if AssigneeID is member's primary key)
	member, err := s.teamRepo.GetMemberByID(ctx, req.AssigneeID)
	s.log.WithFields(logrus.Fields{
		"assignee_id": req.AssigneeID,
		"found_by_id": err == nil,
		"error":       err,
	}).Debug("Looking up member by ID")

	if err != nil {
		// If not found by ID, try by UserID (AssigneeID might be the user's identity)
		member, err = s.teamRepo.GetMemberByUserID(ctx, req.AssigneeID.String())
		s.log.WithFields(logrus.Fields{
			"user_id":       req.AssigneeID.String(),
			"found_by_user": err == nil,
			"error":         err,
		}).Debug("Looking up member by UserID")

		if err != nil {
			return nil, fmt.Errorf("assignee not found: %w", err)
		}
	}

	s.log.WithFields(logrus.Fields{
		"member_id":   member.ID,
		"member_name": member.Name,
		"active":      member.Active,
		"user_id":     member.UserID,
	}).Debug("Member found for assignment validation")

	// Check if member is active
	if !member.Active {
		s.log.WithField("member_id", member.ID).Warn("Rejecting assignment to inactive member")
		return nil, fmt.Errorf("cannot assign to inactive member: %w", database.ErrInactiveMember)
	}

	// Check if member has capacity
	if member.CurrentTasks >= member.MaxTasks {
		return nil, fmt.Errorf("member has no capacity: %w", database.ErrNoCapacity)
	}

	// Store previous assignee for audit
	previousAssignee := task.AssignedTo

	// Update assignment
	now := time.Now().UTC()
	task.AssignedTo = &req.AssigneeID
	task.AssignedAt = &now
	task.Status = models.TaskStatusAssigned

	if req.Role != "" {
		task.AssignedRole = req.Role
	}

	// Apply team override if specified
	if req.TeamID != nil {
		task.TeamID = req.TeamID
		s.log.WithFields(logrus.Fields{
			"task_id":     task.ID,
			"new_team_id": *req.TeamID,
		}).Debug("Applied team override")
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish audit event
	if s.governanceSvc != nil {
		go func() {
			if err := s.governanceSvc.PublishTaskAssigned(context.Background(), task, previousAssignee, auditCtx); err != nil {
				s.log.WithError(err).Warn("Failed to publish task assigned audit event")
			}
		}()
	}

	// Increment assignee's task count
	if member, err := s.teamRepo.GetMemberByID(ctx, req.AssigneeID); err == nil && member != nil {
		_ = s.teamRepo.IncrementMemberTaskCount(ctx, member.ID)
	}

	s.log.WithFields(logrus.Fields{
		"task_id":     task.ID,
		"assigned_to": req.AssigneeID,
	}).Info("Task assigned")

	return task, nil
}

// Start starts a task
func (s *TaskService) Start(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Store previous status for audit
	previousStatus := task.Status

	// Validate transition - task must be ASSIGNED or BLOCKED to start
	// CREATED tasks must first be assigned
	if task.Status != models.TaskStatusAssigned && task.Status != models.TaskStatusBlocked {
		return nil, fmt.Errorf("cannot start task in %s status", task.Status)
	}

	// Update status
	now := time.Now().UTC()
	task.Status = models.TaskStatusInProgress
	task.StartedAt = &now

	// Auto-assign if not already assigned
	if task.AssignedTo == nil {
		task.AssignedTo = &userID
		task.AssignedAt = &now
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish audit event for STARTED status change
	if s.governanceSvc != nil {
		go func() {
			auditCtx := &AuditContext{
				ActorID:   &userID,
				ActorType: models.ActorTypeUser,
			}
			if err := s.governanceSvc.PublishStatusChange(context.Background(), task, previousStatus, auditCtx, "", ""); err != nil {
				s.log.WithError(err).Warn("Failed to publish task started audit event")
			}
		}()
	}

	s.log.WithField("task_id", task.ID).Info("Task started")
	return task, nil
}

// Complete completes a task
func (s *TaskService) Complete(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, req *models.CompleteTaskRequest) (*models.Task, error) {
	auditCtx := &AuditContext{
		ActorID:   &userID,
		ActorType: models.ActorTypeUser,
	}
	return s.CompleteWithAudit(ctx, taskID, userID, req, auditCtx)
}

// CompleteWithAudit completes a task with audit context
func (s *TaskService) CompleteWithAudit(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, req *models.CompleteTaskRequest, auditCtx *AuditContext) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Store previous status for audit
	previousStatus := task.Status

	// Validate transition - only certain statuses can transition to COMPLETED
	validFromStatuses := []models.TaskStatus{
		models.TaskStatusInProgress,
		models.TaskStatusAssigned,
		models.TaskStatusEscalated,
		models.TaskStatusBlocked,
	}
	isValidTransition := false
	for _, validStatus := range validFromStatuses {
		if task.Status == validStatus {
			isValidTransition = true
			break
		}
	}
	if !isValidTransition {
		if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusCancelled {
			return nil, fmt.Errorf("task already in %s status", task.Status)
		}
		return nil, fmt.Errorf("invalid status transition: cannot complete task in %s status", task.Status)
	}

	// Validate reason code if provided (governance requirement)
	if s.governanceSvc != nil && req.ReasonCode != "" {
		valid, requiresJustification, _, err := s.governanceSvc.ValidateReasonCode(ctx, req.ReasonCode)
		if err != nil {
			return nil, fmt.Errorf("failed to validate reason code: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("invalid reason code: %s", req.ReasonCode)
		}
		if requiresJustification && req.ClinicalJustification == "" {
			return nil, fmt.Errorf("reason code %s requires clinical justification", req.ReasonCode)
		}
	}

	// Check if all required actions are completed
	for _, action := range task.Actions {
		if action.Required && !action.Completed {
			return nil, fmt.Errorf("required action '%s' not completed", action.ActionID)
		}
	}

	// Update status
	now := time.Now().UTC()
	task.Status = models.TaskStatusCompleted
	task.CompletedAt = &now
	task.CompletedBy = &userID
	task.Outcome = req.Outcome

	// Add governance fields
	task.ReasonCode = req.ReasonCode
	task.ReasonText = req.ReasonText
	task.ClinicalJustification = req.ClinicalJustification
	task.LastAuditAt = &now

	// Add completion note if provided
	if req.Notes != "" {
		note := models.TaskNote{
			NoteID:    uuid.NewString(),
			Author:    "System",
			AuthorID:  userID.String(),
			Content:   "Task completed: " + req.Notes,
			CreatedAt: now,
		}
		task.Notes = append(task.Notes, note)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish audit event
	if s.governanceSvc != nil {
		go func() {
			if err := s.governanceSvc.PublishStatusChange(context.Background(), task, previousStatus, auditCtx, req.ReasonCode, req.ReasonText); err != nil {
				s.log.WithError(err).Warn("Failed to publish task completed audit event")
			}
		}()
	}

	// Decrement assignee's task count
	if task.AssignedTo != nil {
		_ = s.teamRepo.DecrementMemberTaskCount(ctx, *task.AssignedTo)
	}

	s.log.WithFields(logrus.Fields{
		"task_id":      task.ID,
		"completed_by": userID,
		"outcome":      req.Outcome,
		"reason_code":  req.ReasonCode,
	}).Info("Task completed")

	return task, nil
}

// Cancel cancels a task
func (s *TaskService) Cancel(ctx context.Context, taskID uuid.UUID, reason string) (*models.Task, error) {
	req := &models.CancelTaskRequest{
		ReasonCode: "NO_LONGER_APPLICABLE",
		ReasonText: reason,
	}
	return s.CancelWithAudit(ctx, taskID, req, SystemAuditContext())
}

// CancelWithAudit cancels a task with governance compliance
func (s *TaskService) CancelWithAudit(ctx context.Context, taskID uuid.UUID, req *models.CancelTaskRequest, auditCtx *AuditContext) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Store previous status for audit
	previousStatus := task.Status

	// Cannot cancel completed/verified tasks
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusVerified {
		return nil, fmt.Errorf("cannot cancel task in %s status", task.Status)
	}

	// Update status with governance fields
	now := time.Now().UTC()
	task.Status = models.TaskStatusCancelled
	task.Outcome = "CANCELLED"
	task.ReasonCode = req.ReasonCode
	task.ReasonText = req.ReasonText
	task.ClinicalJustification = req.ClinicalJustification
	task.LastAuditAt = &now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish audit event
	if s.governanceSvc != nil {
		go func() {
			if err := s.governanceSvc.PublishStatusChange(context.Background(), task, previousStatus, auditCtx, req.ReasonCode, req.ReasonText); err != nil {
				s.log.WithError(err).Warn("Failed to publish task cancelled audit event")
			}
		}()
	}

	// Decrement assignee's task count
	if task.AssignedTo != nil {
		_ = s.teamRepo.DecrementMemberTaskCount(ctx, *task.AssignedTo)
	}

	s.log.WithFields(logrus.Fields{
		"task_id":     task.ID,
		"reason_code": req.ReasonCode,
		"reason_text": req.ReasonText,
	}).Info("Task cancelled")

	return task, nil
}

// Decline declines a task with required reason code
func (s *TaskService) Decline(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, req *models.DeclineTaskRequest) (*models.Task, error) {
	auditCtx := &AuditContext{
		ActorID:   &userID,
		ActorType: models.ActorTypeUser,
	}
	return s.DeclineWithAudit(ctx, taskID, userID, req, auditCtx)
}

// DeclineWithAudit declines a task with governance compliance
func (s *TaskService) DeclineWithAudit(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, req *models.DeclineTaskRequest, auditCtx *AuditContext) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Store previous status for audit
	previousStatus := task.Status

	// Validate reason code if governance service is available
	if s.governanceSvc != nil && req.ReasonCode != "" {
		valid, requiresJustification, _, err := s.governanceSvc.ValidateReasonCode(ctx, req.ReasonCode)
		if err != nil {
			return nil, fmt.Errorf("failed to validate reason code: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("invalid reason code: %s", req.ReasonCode)
		}
		if requiresJustification && req.ClinicalJustification == "" {
			return nil, fmt.Errorf("reason code %s requires clinical justification", req.ReasonCode)
		}
	}

	// Update status with governance fields
	now := time.Now().UTC()
	task.Status = models.TaskStatusDeclined
	task.Outcome = "DECLINED"
	task.ReasonCode = req.ReasonCode
	task.ReasonText = req.ReasonText
	task.ClinicalJustification = req.ClinicalJustification
	task.LastAuditAt = &now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Publish audit event
	if s.governanceSvc != nil {
		go func() {
			if err := s.governanceSvc.PublishStatusChange(context.Background(), task, previousStatus, auditCtx, req.ReasonCode, req.ReasonText); err != nil {
				s.log.WithError(err).Warn("Failed to publish task declined audit event")
			}
		}()
	}

	// Decrement assignee's task count
	if task.AssignedTo != nil {
		_ = s.teamRepo.DecrementMemberTaskCount(ctx, *task.AssignedTo)
	}

	s.log.WithFields(logrus.Fields{
		"task_id":     task.ID,
		"declined_by": userID,
		"reason_code": req.ReasonCode,
	}).Info("Task declined")

	return task, nil
}

// AddNote adds a note to a task
func (s *TaskService) AddNote(ctx context.Context, taskID uuid.UUID, req *models.AddNoteRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Add note
	note := models.TaskNote{
		NoteID:    uuid.NewString(),
		Author:    req.Author,
		AuthorID:  req.AuthorID,
		Content:   req.Content,
		CreatedAt: time.Now().UTC(),
	}
	task.Notes = append(task.Notes, note)

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// CompleteAction marks an action as completed
func (s *TaskService) CompleteAction(ctx context.Context, taskID uuid.UUID, actionID string, completedBy string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Find and update the action
	found := false
	for i := range task.Actions {
		if task.Actions[i].ActionID == actionID {
			now := time.Now().UTC()
			task.Actions[i].Completed = true
			task.Actions[i].CompletedAt = &now
			task.Actions[i].CompletedBy = completedBy
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("action %s not found", actionID)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetTasksByPatient retrieves all tasks for a patient
func (s *TaskService) GetTasksByPatient(ctx context.Context, patientID string) ([]models.Task, error) {
	return s.taskRepo.FindByPatient(ctx, patientID)
}

// GetTasksByAssignee retrieves all tasks assigned to a user
func (s *TaskService) GetTasksByAssignee(ctx context.Context, assigneeID uuid.UUID) ([]models.Task, error) {
	return s.taskRepo.FindByAssignee(ctx, assigneeID)
}

// GetTasksByTeam retrieves all tasks for a team
func (s *TaskService) GetTasksByTeam(ctx context.Context, teamID uuid.UUID) ([]models.Task, error) {
	return s.taskRepo.FindByTeam(ctx, teamID)
}

// GetOverdueTasks retrieves all overdue tasks
func (s *TaskService) GetOverdueTasks(ctx context.Context) ([]models.Task, error) {
	return s.taskRepo.FindOverdue(ctx)
}

// GetUnassignedTasks retrieves all unassigned tasks
func (s *TaskService) GetUnassignedTasks(ctx context.Context) ([]models.Task, error) {
	return s.taskRepo.FindUnassigned(ctx)
}

// UpdateEscalationLevel updates the escalation level of a task
func (s *TaskService) UpdateEscalationLevel(ctx context.Context, taskID uuid.UUID, level int) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	task.EscalationLevel = level
	if level >= int(models.EscalationUrgent) {
		task.Status = models.TaskStatusEscalated
	}

	return s.taskRepo.Update(ctx, task)
}

// generateTaskNumber generates a unique task number
func generateTaskNumber(taskType models.TaskType) string {
	prefix := "TASK"
	switch taskType {
	case models.TaskTypeCriticalLabReview, models.TaskTypeMedicationReview:
		prefix = "CLN"
	case models.TaskTypeCareGapClosure, models.TaskTypeScreeningOutreach:
		prefix = "GAP"
	case models.TaskTypeMonitoringOverdue, models.TaskTypeAcuteProtocolDeadline:
		prefix = "TMP"
	case models.TaskTypeAppointmentRemind, models.TaskTypeMissedAppointment:
		prefix = "OUT"
	case models.TaskTypePriorAuthNeeded, models.TaskTypeReferralProcessing:
		prefix = "ADM"
	}

	return fmt.Sprintf("%s-%s", prefix, uuid.NewString()[:8])
}
