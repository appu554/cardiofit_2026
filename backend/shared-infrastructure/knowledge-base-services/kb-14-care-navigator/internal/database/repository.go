// Package database provides PostgreSQL database connectivity and repositories for KB-14
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-14-care-navigator/internal/models"
)

// Common errors
var (
	ErrNotFound       = errors.New("resource not found")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrInactiveMember = errors.New("team member is inactive")
	ErrNoCapacity     = errors.New("team member has no capacity")
)

// ================================================================================
// TASK REPOSITORY
// ================================================================================

// TaskRepository handles task database operations
type TaskRepository struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(db *Database, log *logrus.Entry) *TaskRepository {
	return &TaskRepository{
		db:     db.DB,
		logger: log.WithField("repository", "task"),
	}
}

// Ping checks if the database connection is healthy
func (r *TaskRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to create task: %w", err)
	}
	r.logger.WithField("task_id", task.ID).Debug("Task created")
	return nil
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, nil
}

// GetByTaskID retrieves a task by task_id string
func (r *TaskRepository) GetByTaskID(ctx context.Context, taskID string) (*models.Task, error) {
	var task models.Task
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, nil
}

// Update updates a task
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	result := r.db.WithContext(ctx).Save(task)
	if result.Error != nil {
		return fmt.Errorf("failed to update task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithField("task_id", task.ID).Debug("Task updated")
	return nil
}

// Delete deletes a task
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Task{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithField("task_id", id).Debug("Task deleted")
	return nil
}

// FindByPatient retrieves all tasks for a patient
func (r *TaskRepository) FindByPatient(ctx context.Context, patientID string) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.WithContext(ctx).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks by patient: %w", err)
	}
	return tasks, nil
}

// FindByAssignee retrieves all tasks assigned to a user
func (r *TaskRepository) FindByAssignee(ctx context.Context, assigneeID uuid.UUID) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.WithContext(ctx).
		Where("assigned_to = ?", assigneeID).
		Order("due_date ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks by assignee: %w", err)
	}
	return tasks, nil
}

// FindByTeam retrieves all tasks for a team
func (r *TaskRepository) FindByTeam(ctx context.Context, teamID uuid.UUID) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("due_date ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks by team: %w", err)
	}
	return tasks, nil
}

// FindByStatus retrieves all tasks with a specific status
func (r *TaskRepository) FindByStatus(ctx context.Context, status models.TaskStatus) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("due_date ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks by status: %w", err)
	}
	return tasks, nil
}

// FindOverdue retrieves all overdue tasks
func (r *TaskRepository) FindOverdue(ctx context.Context) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.WithContext(ctx).
		Where("due_date < ? AND status NOT IN ?", time.Now().UTC(),
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusVerified, models.TaskStatusCancelled}).
		Order("due_date ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find overdue tasks: %w", err)
	}
	return tasks, nil
}

// FindDueSoon retrieves tasks due within the specified duration
func (r *TaskRepository) FindDueSoon(ctx context.Context, duration time.Duration) ([]models.Task, error) {
	var tasks []models.Task
	now := time.Now().UTC()
	deadline := now.Add(duration)

	if err := r.db.WithContext(ctx).
		Where("due_date BETWEEN ? AND ? AND status NOT IN ?", now, deadline,
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusVerified, models.TaskStatusCancelled}).
		Order("due_date ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks due soon: %w", err)
	}
	return tasks, nil
}

// FindUnassigned retrieves all unassigned tasks
func (r *TaskRepository) FindUnassigned(ctx context.Context) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.WithContext(ctx).
		Where("assigned_to IS NULL AND status = ?", models.TaskStatusCreated).
		Order("priority DESC, created_at ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find unassigned tasks: %w", err)
	}
	return tasks, nil
}

// FindBySource retrieves tasks from a specific source
func (r *TaskRepository) FindBySource(ctx context.Context, source models.TaskSource, sourceID string) ([]models.Task, error) {
	var tasks []models.Task
	query := r.db.WithContext(ctx).Where("source = ?", source)
	if sourceID != "" {
		query = query.Where("source_id = ?", sourceID)
	}
	if err := query.Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks by source: %w", err)
	}
	return tasks, nil
}

// FindBySourceID retrieves a single task by source and source ID
func (r *TaskRepository) FindBySourceID(ctx context.Context, source string, sourceID string) (*models.Task, error) {
	var task models.Task
	if err := r.db.WithContext(ctx).
		Where("source = ? AND source_id = ?", source, sourceID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error for this use case
		}
		return nil, fmt.Errorf("failed to find task by source ID: %w", err)
	}
	return &task, nil
}

// FindNeedingEscalation finds tasks that need escalation check
func (r *TaskRepository) FindNeedingEscalation(ctx context.Context) ([]models.Task, error) {
	var tasks []models.Task
	// Find active tasks with due dates
	if err := r.db.WithContext(ctx).
		Where("status IN ? AND due_date IS NOT NULL",
			[]models.TaskStatus{
				models.TaskStatusCreated,
				models.TaskStatusAssigned,
				models.TaskStatusInProgress,
				models.TaskStatusBlocked,
			}).
		Order("due_date ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks needing escalation: %w", err)
	}
	return tasks, nil
}

// CountByAssignee counts tasks for an assignee
func (r *TaskRepository) CountByAssignee(ctx context.Context, assigneeID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Task{}).
		Where("assigned_to = ? AND status NOT IN ?", assigneeID,
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusVerified, models.TaskStatusCancelled}).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}
	return count, nil
}

// TaskFilters for worklist queries
type TaskFilters struct {
	UserID           *uuid.UUID
	TeamID           *uuid.UUID
	PatientID        string
	Statuses         []models.TaskStatus
	Priorities       []models.TaskPriority
	Types            []models.TaskType
	Sources          []models.TaskSource
	Overdue          bool
	Unassigned       bool
	DueBefore        *time.Time
	DueAfter         *time.Time
	CreatedAfter     *time.Time
	CreatedBefore    *time.Time
	MinEscalation    *int
	Page             int
	PageSize         int
	SortBy           string
	SortOrder        string
}

// FindWithFilters retrieves tasks matching the given filters
func (r *TaskRepository) FindWithFilters(ctx context.Context, filters TaskFilters) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Task{})

	// Apply filters
	if filters.UserID != nil {
		query = query.Where("assigned_to = ?", *filters.UserID)
	}
	if filters.TeamID != nil {
		query = query.Where("team_id = ?", *filters.TeamID)
	}
	if filters.PatientID != "" {
		query = query.Where("patient_id = ?", filters.PatientID)
	}
	if len(filters.Statuses) > 0 {
		query = query.Where("status IN ?", filters.Statuses)
	}
	if len(filters.Priorities) > 0 {
		query = query.Where("priority IN ?", filters.Priorities)
	}
	if len(filters.Types) > 0 {
		query = query.Where("type IN ?", filters.Types)
	}
	if len(filters.Sources) > 0 {
		query = query.Where("source IN ?", filters.Sources)
	}
	if filters.Overdue {
		query = query.Where("due_date < ? AND status NOT IN ?", time.Now().UTC(),
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusVerified, models.TaskStatusCancelled})
	}
	if filters.Unassigned {
		query = query.Where("assigned_to IS NULL")
	}
	if filters.DueBefore != nil {
		query = query.Where("due_date < ?", *filters.DueBefore)
	}
	if filters.DueAfter != nil {
		query = query.Where("due_date > ?", *filters.DueAfter)
	}
	if filters.CreatedAfter != nil {
		query = query.Where("created_at > ?", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		query = query.Where("created_at < ?", *filters.CreatedBefore)
	}
	if filters.MinEscalation != nil {
		query = query.Where("escalation_level >= ?", *filters.MinEscalation)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Apply sorting
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "due_date"
	}
	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	page := filters.Page
	if page < 1 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	if err := query.Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find tasks: %w", err)
	}

	return tasks, total, nil
}

// GetTaskSummary retrieves task counts grouped by status, priority, and type
func (r *TaskRepository) GetTaskSummary(ctx context.Context, filters TaskFilters) (*models.WorklistSummary, error) {
	var summary models.WorklistSummary
	summary.TasksByStatus = make(map[models.TaskStatus]int64)
	summary.TasksByPriority = make(map[models.TaskPriority]int64)
	summary.TasksByType = make(map[models.TaskType]int64)

	baseQuery := r.db.WithContext(ctx).Model(&models.Task{})

	// Apply base filters
	if filters.UserID != nil {
		baseQuery = baseQuery.Where("assigned_to = ?", *filters.UserID)
	}
	if filters.TeamID != nil {
		baseQuery = baseQuery.Where("team_id = ?", *filters.TeamID)
	}

	// Exclude completed/cancelled for counts
	activeStatuses := []models.TaskStatus{
		models.TaskStatusCreated, models.TaskStatusAssigned, models.TaskStatusInProgress,
		models.TaskStatusBlocked, models.TaskStatusEscalated,
	}

	// Total active tasks
	baseQuery.Where("status IN ?", activeStatuses).Count(&summary.TotalTasks)

	// Overdue tasks
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ? AND due_date < ?", activeStatuses, time.Now().UTC()).
		Count(&summary.OverdueTasks)

	// Urgent tasks (CRITICAL or HIGH priority)
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ? AND priority IN ?", activeStatuses,
			[]models.TaskPriority{models.TaskPriorityCritical, models.TaskPriorityHigh}).
		Count(&summary.UrgentTasks)

	// Due today
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ? AND due_date BETWEEN ? AND ?", activeStatuses, startOfDay, endOfDay).
		Count(&summary.DueTodayTasks)

	// Due this week
	endOfWeek := startOfDay.Add(7 * 24 * time.Hour)
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ? AND due_date BETWEEN ? AND ?", activeStatuses, time.Now().UTC(), endOfWeek).
		Count(&summary.DueThisWeekTasks)

	// Unassigned tasks
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ? AND assigned_to IS NULL", activeStatuses).
		Count(&summary.UnassignedTasks)

	// Tasks by status
	var statusCounts []struct {
		Status models.TaskStatus
		Count  int64
	}
	r.db.WithContext(ctx).Model(&models.Task{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts)
	for _, sc := range statusCounts {
		summary.TasksByStatus[sc.Status] = sc.Count
	}

	// Tasks by priority
	var priorityCounts []struct {
		Priority models.TaskPriority
		Count    int64
	}
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ?", activeStatuses).
		Select("priority, count(*) as count").
		Group("priority").
		Scan(&priorityCounts)
	for _, pc := range priorityCounts {
		summary.TasksByPriority[pc.Priority] = pc.Count
	}

	// Tasks by type
	var typeCounts []struct {
		Type  models.TaskType
		Count int64
	}
	r.db.WithContext(ctx).Model(&models.Task{}).
		Where("status IN ?", activeStatuses).
		Select("type, count(*) as count").
		Group("type").
		Scan(&typeCounts)
	for _, tc := range typeCounts {
		summary.TasksByType[tc.Type] = tc.Count
	}

	return &summary, nil
}

// ================================================================================
// TEAM REPOSITORY
// ================================================================================

// TeamRepository handles team database operations
type TeamRepository struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewTeamRepository creates a new TeamRepository
func NewTeamRepository(db *Database, log *logrus.Entry) *TeamRepository {
	return &TeamRepository{
		db:     db.DB,
		logger: log.WithField("repository", "team"),
	}
}

// CreateTeam creates a new team
func (r *TeamRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	if err := r.db.WithContext(ctx).Create(team).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to create team: %w", err)
	}
	r.logger.WithField("team_id", team.ID).Debug("Team created")
	return nil
}

// GetTeamByID retrieves a team by ID
func (r *TeamRepository) GetTeamByID(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	var team models.Team
	if err := r.db.WithContext(ctx).
		Preload("Members").
		Where("id = ?", id).
		First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return &team, nil
}

// GetTeamByTeamID retrieves a team by team_id string
func (r *TeamRepository) GetTeamByTeamID(ctx context.Context, teamID string) (*models.Team, error) {
	var team models.Team
	if err := r.db.WithContext(ctx).
		Preload("Members").
		Where("team_id = ?", teamID).
		First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return &team, nil
}

// UpdateTeam updates a team
func (r *TeamRepository) UpdateTeam(ctx context.Context, team *models.Team) error {
	result := r.db.WithContext(ctx).Save(team)
	if result.Error != nil {
		return fmt.Errorf("failed to update team: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithField("team_id", team.ID).Debug("Team updated")
	return nil
}

// DeleteTeam deletes a team
func (r *TeamRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Team{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete team: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithField("team_id", id).Debug("Team deleted")
	return nil
}

// ListTeams lists all active teams
func (r *TeamRepository) ListTeams(ctx context.Context) ([]models.Team, error) {
	var teams []models.Team
	if err := r.db.WithContext(ctx).
		Preload("Members").
		Where("active = ?", true).
		Order("name ASC").
		Find(&teams).Error; err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	return teams, nil
}

// FindTeamsByType finds teams by type
func (r *TeamRepository) FindTeamsByType(ctx context.Context, teamType string) ([]models.Team, error) {
	var teams []models.Team
	if err := r.db.WithContext(ctx).
		Preload("Members").
		Where("type = ? AND active = ?", teamType, true).
		Find(&teams).Error; err != nil {
		return nil, fmt.Errorf("failed to find teams by type: %w", err)
	}
	return teams, nil
}

// ================================================================================
// TEAM MEMBER REPOSITORY
// ================================================================================

// CreateMember creates a new team member
func (r *TeamRepository) CreateMember(ctx context.Context, member *models.TeamMember) error {
	if err := r.db.WithContext(ctx).Create(member).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to create team member: %w", err)
	}
	r.logger.WithField("member_id", member.ID).Debug("Team member created")
	return nil
}

// GetMemberByID retrieves a team member by ID
func (r *TeamRepository) GetMemberByID(ctx context.Context, id uuid.UUID) (*models.TeamMember, error) {
	var member models.TeamMember
	if err := r.db.WithContext(ctx).
		Preload("Team").
		Where("id = ?", id).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}
	return &member, nil
}

// GetMemberByUserID retrieves a team member by user_id
func (r *TeamRepository) GetMemberByUserID(ctx context.Context, userID string) (*models.TeamMember, error) {
	var member models.TeamMember
	if err := r.db.WithContext(ctx).
		Preload("Team").
		Where("user_id = ?", userID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}
	return &member, nil
}

// UpdateMember updates a team member
func (r *TeamRepository) UpdateMember(ctx context.Context, member *models.TeamMember) error {
	result := r.db.WithContext(ctx).Save(member)
	if result.Error != nil {
		return fmt.Errorf("failed to update team member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithField("member_id", member.ID).Debug("Team member updated")
	return nil
}

// DeleteMember deletes a team member
func (r *TeamRepository) DeleteMember(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.TeamMember{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete team member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithField("member_id", id).Debug("Team member deleted")
	return nil
}

// GetMembersByTeam retrieves all members of a team
func (r *TeamRepository) GetMembersByTeam(ctx context.Context, teamID uuid.UUID) ([]models.TeamMember, error) {
	var members []models.TeamMember
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND active = ?", teamID, true).
		Order("role ASC, name ASC").
		Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	return members, nil
}

// GetMembersByRole retrieves members by role
func (r *TeamRepository) GetMembersByRole(ctx context.Context, role string) ([]models.TeamMember, error) {
	var members []models.TeamMember
	if err := r.db.WithContext(ctx).
		Preload("Team").
		Where("role = ? AND active = ?", role, true).
		Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get members by role: %w", err)
	}
	return members, nil
}

// GetAvailableMembers retrieves available members with capacity
func (r *TeamRepository) GetAvailableMembers(ctx context.Context, teamID *uuid.UUID, role string) ([]models.TeamMember, error) {
	var members []models.TeamMember
	query := r.db.WithContext(ctx).
		Where("active = ? AND current_tasks < max_tasks", true)

	if teamID != nil {
		query = query.Where("team_id = ?", *teamID)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	// Check availability window
	now := time.Now().UTC()
	query = query.Where("(available_from IS NULL OR available_from <= ?) AND (available_to IS NULL OR available_to >= ?)", now, now)

	if err := query.Order("current_tasks ASC").Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get available members: %w", err)
	}
	return members, nil
}

// IncrementMemberTaskCount increments a member's task count
func (r *TeamRepository) IncrementMemberTaskCount(ctx context.Context, memberID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.TeamMember{}).
		Where("id = ?", memberID).
		UpdateColumn("current_tasks", gorm.Expr("current_tasks + 1"))
	if result.Error != nil {
		return fmt.Errorf("failed to increment task count: %w", result.Error)
	}
	return nil
}

// DecrementMemberTaskCount decrements a member's task count
func (r *TeamRepository) DecrementMemberTaskCount(ctx context.Context, memberID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.TeamMember{}).
		Where("id = ? AND current_tasks > 0", memberID).
		UpdateColumn("current_tasks", gorm.Expr("current_tasks - 1"))
	if result.Error != nil {
		return fmt.Errorf("failed to decrement task count: %w", result.Error)
	}
	return nil
}

// ================================================================================
// ESCALATION REPOSITORY
// ================================================================================

// EscalationRepository handles escalation database operations
type EscalationRepository struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewEscalationRepository creates a new EscalationRepository
func NewEscalationRepository(db *Database, log *logrus.Entry) *EscalationRepository {
	return &EscalationRepository{
		db:     db.DB,
		logger: log.WithField("repository", "escalation"),
	}
}

// Create creates a new escalation
func (r *EscalationRepository) Create(ctx context.Context, escalation *models.Escalation) error {
	if err := r.db.WithContext(ctx).Create(escalation).Error; err != nil {
		return fmt.Errorf("failed to create escalation: %w", err)
	}
	r.logger.WithFields(logrus.Fields{
		"escalation_id": escalation.ID,
		"task_id":       escalation.TaskID,
		"level":         escalation.Level,
	}).Debug("Escalation created")
	return nil
}

// GetByID retrieves an escalation by ID
func (r *EscalationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Escalation, error) {
	var escalation models.Escalation
	if err := r.db.WithContext(ctx).
		Preload("Task").
		Where("id = ?", id).
		First(&escalation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get escalation: %w", err)
	}
	return &escalation, nil
}

// Update updates an escalation
func (r *EscalationRepository) Update(ctx context.Context, escalation *models.Escalation) error {
	result := r.db.WithContext(ctx).Save(escalation)
	if result.Error != nil {
		return fmt.Errorf("failed to update escalation: %w", result.Error)
	}
	return nil
}

// FindByTask retrieves all escalations for a task
func (r *EscalationRepository) FindByTask(ctx context.Context, taskID uuid.UUID) ([]models.Escalation, error) {
	var escalations []models.Escalation
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Find(&escalations).Error; err != nil {
		return nil, fmt.Errorf("failed to find escalations: %w", err)
	}
	return escalations, nil
}

// FindLatestByTask retrieves the latest escalation for a task
func (r *EscalationRepository) FindLatestByTask(ctx context.Context, taskID uuid.UUID) (*models.Escalation, error) {
	var escalation models.Escalation
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		First(&escalation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No escalation found is not an error
		}
		return nil, fmt.Errorf("failed to find latest escalation: %w", err)
	}
	return &escalation, nil
}

// FindUnacknowledged retrieves unacknowledged escalations
func (r *EscalationRepository) FindUnacknowledged(ctx context.Context) ([]models.Escalation, error) {
	var escalations []models.Escalation
	if err := r.db.WithContext(ctx).
		Preload("Task").
		Where("acknowledged = ?", false).
		Order("level DESC, created_at ASC").
		Find(&escalations).Error; err != nil {
		return nil, fmt.Errorf("failed to find unacknowledged escalations: %w", err)
	}
	return escalations, nil
}

// Acknowledge marks an escalation as acknowledged
func (r *EscalationRepository) Acknowledge(ctx context.Context, id uuid.UUID, acknowledgedBy uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Escalation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"acknowledged":    true,
			"acknowledged_at": now,
			"acknowledged_by": acknowledgedBy,
			"status":          models.EscalationStatusAcknowledged,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to acknowledge escalation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Resolve marks an escalation as resolved
func (r *EscalationRepository) Resolve(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID, resolution string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.Escalation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.EscalationStatusResolved,
			"resolved_at": now,
			"resolved_by": resolvedBy,
			"resolution":  resolution,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to resolve escalation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.logger.WithFields(logrus.Fields{
		"escalation_id": id,
		"resolved_by":   resolvedBy,
	}).Debug("Escalation resolved")
	return nil
}

// FindWithFilters retrieves escalations matching the given filters
func (r *EscalationRepository) FindWithFilters(ctx context.Context, statuses []models.EscalationStatus, levels []models.EscalationLevel, page, pageSize int) ([]models.Escalation, int64, error) {
	var escalations []models.Escalation
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Escalation{}).Preload("Task")

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if len(levels) > 0 {
		query = query.Where("level IN ?", levels)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count escalations: %w", err)
	}

	// Apply pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize).Order("created_at DESC")

	if err := query.Find(&escalations).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find escalations: %w", err)
	}

	return escalations, total, nil
}

// GetEscalationSummary retrieves escalation statistics
func (r *EscalationRepository) GetEscalationSummary(ctx context.Context) (*models.EscalationSummary, error) {
	var summary models.EscalationSummary

	// Total escalations
	r.db.WithContext(ctx).Model(&models.Escalation{}).Count(&summary.TotalEscalations)

	// By level
	r.db.WithContext(ctx).Model(&models.Escalation{}).
		Where("level = ?", models.EscalationWarning).Count(&summary.WarningCount)
	r.db.WithContext(ctx).Model(&models.Escalation{}).
		Where("level = ?", models.EscalationUrgent).Count(&summary.UrgentCount)
	r.db.WithContext(ctx).Model(&models.Escalation{}).
		Where("level = ?", models.EscalationCritical).Count(&summary.CriticalCount)
	r.db.WithContext(ctx).Model(&models.Escalation{}).
		Where("level = ?", models.EscalationExecutive).Count(&summary.ExecutiveCount)

	// Unacknowledged
	r.db.WithContext(ctx).Model(&models.Escalation{}).
		Where("acknowledged = ?", false).Count(&summary.UnacknowledgedCount)

	// Average response time (minutes between created and acknowledged)
	var avgMinutes *float64
	r.db.WithContext(ctx).Model(&models.Escalation{}).
		Where("acknowledged = ?", true).
		Select("AVG(EXTRACT(EPOCH FROM (acknowledged_at - created_at)) / 60)").
		Scan(&avgMinutes)
	if avgMinutes != nil {
		summary.AverageResponseTime = int(*avgMinutes)
	}

	return &summary, nil
}
