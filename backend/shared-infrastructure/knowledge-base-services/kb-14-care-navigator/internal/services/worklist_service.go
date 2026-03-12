// Package services provides business logic for KB-14 Care Navigator
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
)

// WorklistService handles worklist generation and filtering
type WorklistService struct {
	taskRepo *database.TaskRepository
	teamRepo *database.TeamRepository
	log      *logrus.Entry
}

// NewWorklistService creates a new WorklistService
func NewWorklistService(
	taskRepo *database.TaskRepository,
	teamRepo *database.TeamRepository,
	log *logrus.Entry,
) *WorklistService {
	return &WorklistService{
		taskRepo: taskRepo,
		teamRepo: teamRepo,
		log:      log.WithField("service", "worklist"),
	}
}

// GetWorklist retrieves a worklist based on filters
func (s *WorklistService) GetWorklist(ctx context.Context, filters models.WorklistFilters) (*models.WorklistResponse, error) {
	// Convert to database filters
	dbFilters := database.TaskFilters{
		UserID:           filters.UserID,
		TeamID:           filters.TeamID,
		PatientID:        filters.PatientID,
		Statuses:         filters.Statuses,
		Priorities:       filters.Priorities,
		Types:            filters.Types,
		Sources:          filters.Sources,
		Overdue:          filters.Overdue,
		Unassigned:       filters.Unassigned,
		DueBefore:        filters.DueBefore,
		DueAfter:         filters.DueAfter,
		CreatedAfter:     filters.CreatedAfter,
		CreatedBefore:    filters.CreatedBefore,
		MinEscalation:    filters.MinEscalationLevel,
		Page:             filters.Page,
		PageSize:         filters.PageSize,
		SortBy:           filters.SortBy,
		SortOrder:        filters.SortOrder,
	}

	// Get tasks
	tasks, total, err := s.taskRepo.FindWithFilters(ctx, dbFilters)
	if err != nil {
		return nil, err
	}

	// Convert to worklist items
	items := make([]models.WorklistItem, 0, len(tasks))
	for _, task := range tasks {
		item := s.taskToWorklistItem(ctx, &task)
		items = append(items, item)
	}

	response := &models.WorklistResponse{
		Success:  true,
		Data:     items,
		Total:    total,
		Page:     filters.Page,
		PageSize: filters.PageSize,
	}

	return response, nil
}

// taskToWorklistItem converts a Task to a WorklistItem
func (s *WorklistService) taskToWorklistItem(ctx context.Context, task *models.Task) models.WorklistItem {
	item := models.WorklistItem{
		TaskID:          task.ID,
		TaskNumber:      task.TaskID,
		Type:            task.Type,
		Status:          task.Status,
		Priority:        task.Priority,
		Title:           task.Title,
		Description:     task.Description,
		PatientID:       task.PatientID,
		AssignedTo:      task.AssignedTo,
		AssignedRole:    task.AssignedRole,
		TeamID:          task.TeamID,
		CreatedAt:       task.CreatedAt,
		DueDate:         task.DueDate,
		SLAMinutes:      task.SLAMinutes,
		EscalationLevel: task.EscalationLevel,
		Source:          task.Source,
		SourceID:        task.SourceID,
	}

	// Calculate overdue status and time remaining
	if task.DueDate != nil {
		now := time.Now().UTC()
		item.IsOverdue = task.DueDate.Before(now)
		item.TimeRemaining = int(task.DueDate.Sub(now).Minutes())
	}

	// Calculate SLA elapsed percentage
	if task.SLAMinutes > 0 {
		elapsed := time.Since(task.CreatedAt).Minutes()
		item.SLAElapsedPercent = (elapsed / float64(task.SLAMinutes)) * 100
		if item.SLAElapsedPercent > 100 {
			item.SLAElapsedPercent = 100
		}
	}

	// Count actions
	item.TotalActions = len(task.Actions)
	for _, action := range task.Actions {
		if action.Completed {
			item.CompletedActions++
		}
	}

	// Get assignee name
	if task.AssignedTo != nil {
		if member, err := s.teamRepo.GetMemberByID(ctx, *task.AssignedTo); err == nil && member != nil {
			item.AssigneeName = member.Name
		}
	}

	// Get team name
	if task.TeamID != nil {
		if team, err := s.teamRepo.GetTeamByID(ctx, *task.TeamID); err == nil && team != nil {
			item.TeamName = team.Name
		}
	}

	return item
}

// GetUserWorklist retrieves a worklist for a specific user
func (s *WorklistService) GetUserWorklist(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.WorklistResponse, error) {
	filters := models.WorklistFilters{
		UserID:   &userID,
		Statuses: []models.TaskStatus{
			models.TaskStatusAssigned,
			models.TaskStatusInProgress,
			models.TaskStatusBlocked,
			models.TaskStatusEscalated,
		},
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "due_date",
		SortOrder: "asc",
	}

	return s.GetWorklist(ctx, filters)
}

// GetTeamWorklist retrieves a worklist for a team
func (s *WorklistService) GetTeamWorklist(ctx context.Context, teamID uuid.UUID, page, pageSize int) (*models.WorklistResponse, error) {
	filters := models.WorklistFilters{
		TeamID:   &teamID,
		Statuses: []models.TaskStatus{
			models.TaskStatusCreated,
			models.TaskStatusAssigned,
			models.TaskStatusInProgress,
			models.TaskStatusBlocked,
			models.TaskStatusEscalated,
		},
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "due_date",
		SortOrder: "asc",
	}

	return s.GetWorklist(ctx, filters)
}

// GetPatientWorklist retrieves all tasks for a patient
func (s *WorklistService) GetPatientWorklist(ctx context.Context, patientID string, page, pageSize int) (*models.WorklistResponse, error) {
	filters := models.WorklistFilters{
		PatientID: patientID,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	return s.GetWorklist(ctx, filters)
}

// GetOverdueWorklist retrieves all overdue tasks
func (s *WorklistService) GetOverdueWorklist(ctx context.Context, page, pageSize int) (*models.WorklistResponse, error) {
	filters := models.WorklistFilters{
		Overdue:   true,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "due_date",
		SortOrder: "asc",
	}

	return s.GetWorklist(ctx, filters)
}

// GetUrgentWorklist retrieves urgent tasks (CRITICAL and HIGH priority)
func (s *WorklistService) GetUrgentWorklist(ctx context.Context, page, pageSize int) (*models.WorklistResponse, error) {
	filters := models.WorklistFilters{
		Priorities: []models.TaskPriority{
			models.TaskPriorityCritical,
			models.TaskPriorityHigh,
		},
		Statuses: []models.TaskStatus{
			models.TaskStatusCreated,
			models.TaskStatusAssigned,
			models.TaskStatusInProgress,
			models.TaskStatusBlocked,
			models.TaskStatusEscalated,
		},
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "priority",
		SortOrder: "desc",
	}

	return s.GetWorklist(ctx, filters)
}

// GetUnassignedWorklist retrieves unassigned tasks
func (s *WorklistService) GetUnassignedWorklist(ctx context.Context, page, pageSize int) (*models.WorklistResponse, error) {
	filters := models.WorklistFilters{
		Unassigned: true,
		Statuses:   []models.TaskStatus{models.TaskStatusCreated},
		Page:       page,
		PageSize:   pageSize,
		SortBy:     "priority",
		SortOrder:  "desc",
	}

	return s.GetWorklist(ctx, filters)
}

// GetWorklistSummary retrieves a summary of the worklist
func (s *WorklistService) GetWorklistSummary(ctx context.Context, filters models.WorklistFilters) (*models.WorklistSummary, error) {
	dbFilters := database.TaskFilters{
		UserID: filters.UserID,
		TeamID: filters.TeamID,
	}

	return s.taskRepo.GetTaskSummary(ctx, dbFilters)
}

// GetUserWorklistSummary retrieves worklist summary for a user
func (s *WorklistService) GetUserWorklistSummary(ctx context.Context, userID uuid.UUID) (*models.WorklistSummary, error) {
	filters := models.WorklistFilters{UserID: &userID}
	return s.GetWorklistSummary(ctx, filters)
}

// GetTeamWorklistSummary retrieves worklist summary for a team
func (s *WorklistService) GetTeamWorklistSummary(ctx context.Context, teamID uuid.UUID) (*models.WorklistSummary, error) {
	filters := models.WorklistFilters{TeamID: &teamID}
	return s.GetWorklistSummary(ctx, filters)
}
