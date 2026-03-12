// Package services provides business logic for KB-14 Care Navigator
package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
)

// AssignmentEngine handles intelligent task assignment
type AssignmentEngine struct {
	taskRepo *database.TaskRepository
	teamRepo *database.TeamRepository
	log      *logrus.Entry
}

// NewAssignmentEngine creates a new AssignmentEngine
func NewAssignmentEngine(
	taskRepo *database.TaskRepository,
	teamRepo *database.TeamRepository,
	log *logrus.Entry,
) *AssignmentEngine {
	return &AssignmentEngine{
		taskRepo: taskRepo,
		teamRepo: teamRepo,
		log:      log.WithField("service", "assignment-engine"),
	}
}

// SuggestAssignees suggests the best assignees for a task based on multiple factors
func (e *AssignmentEngine) SuggestAssignees(ctx context.Context, criteria *models.AssignmentCriteria) ([]models.AssignmentSuggestion, error) {
	// Get the required role from task type if not specified
	role := criteria.RequiredRole
	if role == "" {
		role = criteria.TaskType.GetDefaultRole()
	}

	// Get available members with capacity - first try preferred team
	members, err := e.teamRepo.GetAvailableMembers(ctx, criteria.PreferredTeam, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get available members: %w", err)
	}

	// Fallback: If no members found in preferred team, search across all teams for the role
	if len(members) == 0 && criteria.PreferredTeam != nil {
		e.log.WithFields(logrus.Fields{
			"role":           role,
			"preferred_team": criteria.PreferredTeam,
			"task_type":      criteria.TaskType,
		}).Debug("No members in preferred team, searching all teams")

		// Search across all teams (pass nil for teamID)
		members, err = e.teamRepo.GetAvailableMembers(ctx, nil, role)
		if err != nil {
			return nil, fmt.Errorf("failed to get available members across teams: %w", err)
		}
	}

	if len(members) == 0 {
		e.log.WithFields(logrus.Fields{
			"role":      role,
			"task_type": criteria.TaskType,
		}).Warn("No available members found")
		return []models.AssignmentSuggestion{}, nil
	}

	// Get default scoring weights
	weights := models.DefaultAssignmentWeights()

	// Score each member
	var suggestions []models.AssignmentSuggestion
	for _, member := range members {
		score := e.calculateScore(ctx, &member, criteria, weights)
		reason := e.generateReason(&member, criteria, score)

		suggestion := models.AssignmentSuggestion{
			MemberID:          member.ID,
			MemberName:        member.Name,
			Role:              member.Role,
			TeamID:            member.TeamID,
			Score:             score,
			Reason:            reason,
			CurrentTasks:      member.CurrentTasks,
			MaxTasks:          member.MaxTasks,
			AvailableCapacity: member.GetAvailableCapacity(),
		}

		// Get team name
		if team, err := e.teamRepo.GetTeamByID(ctx, member.TeamID); err == nil && team != nil {
			suggestion.TeamName = team.Name
		}

		suggestions = append(suggestions, suggestion)
	}

	// Sort by score (highest first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Return top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	e.log.WithFields(logrus.Fields{
		"task_type":   criteria.TaskType,
		"role":        role,
		"suggestions": len(suggestions),
	}).Debug("Generated assignment suggestions")

	return suggestions, nil
}

// calculateScore calculates the assignment score for a member
func (e *AssignmentEngine) calculateScore(ctx context.Context, member *models.TeamMember, criteria *models.AssignmentCriteria, weights models.AssignmentScoreWeights) float64 {
	var score float64

	// 1. Workload Balance Score (higher score for less loaded members)
	workloadScore := 1.0 - (float64(member.CurrentTasks) / float64(member.MaxTasks))
	if workloadScore < 0 {
		workloadScore = 0
	}
	score += workloadScore * weights.WorkloadBalance

	// 2. Role Match Score
	roleScore := 0.0
	if member.Role == criteria.RequiredRole || criteria.RequiredRole == "" {
		roleScore = 1.0
	} else if criteria.TaskType.GetDefaultRole() == member.Role {
		roleScore = 0.8
	}
	score += roleScore * weights.RoleMatch

	// 3. Panel Attribution Score (check if patient's PCP is in team's panel)
	// This would require patient context - simplified for now
	panelScore := 0.5 // Default neutral score
	score += panelScore * weights.PanelAttribution

	// 4. Skill Match Score
	skillScore := 0.0
	if len(criteria.RequiredSkills) > 0 {
		matchedSkills := 0
		for _, requiredSkill := range criteria.RequiredSkills {
			if member.HasSkill(requiredSkill) {
				matchedSkills++
			}
		}
		skillScore = float64(matchedSkills) / float64(len(criteria.RequiredSkills))
	} else {
		skillScore = 0.5 // Neutral if no skills required
	}
	score += skillScore * weights.SkillMatch

	// 5. Availability Score
	availabilityScore := 0.0
	if member.IsAvailable() {
		availabilityScore = 1.0
	}
	score += availabilityScore * weights.Availability

	return score
}

// generateReason generates a human-readable reason for the assignment suggestion
func (e *AssignmentEngine) generateReason(member *models.TeamMember, criteria *models.AssignmentCriteria, score float64) string {
	reasons := []string{}

	// Workload reason
	capacityPercent := int((1.0 - float64(member.CurrentTasks)/float64(member.MaxTasks)) * 100)
	if capacityPercent > 50 {
		reasons = append(reasons, fmt.Sprintf("%d%% capacity available", capacityPercent))
	}

	// Role match reason
	if member.Role == criteria.RequiredRole {
		reasons = append(reasons, fmt.Sprintf("Exact role match: %s", member.Role))
	} else if criteria.TaskType.GetDefaultRole() == member.Role {
		reasons = append(reasons, fmt.Sprintf("Default role for task type: %s", member.Role))
	}

	// Skills reason
	matchedSkills := 0
	for _, skill := range criteria.RequiredSkills {
		if member.HasSkill(skill) {
			matchedSkills++
		}
	}
	if matchedSkills > 0 {
		reasons = append(reasons, fmt.Sprintf("%d/%d required skills", matchedSkills, len(criteria.RequiredSkills)))
	}

	if len(reasons) == 0 {
		return fmt.Sprintf("Score: %.2f - Available for assignment", score)
	}

	return fmt.Sprintf("Score: %.2f - %s", score, reasons[0])
}

// AutoAssign automatically assigns a task to the best available member
func (e *AssignmentEngine) AutoAssign(ctx context.Context, task *models.Task) (*models.TeamMember, error) {
	criteria := &models.AssignmentCriteria{
		TaskType:     task.Type,
		TaskPriority: task.Priority,
		PatientID:    task.PatientID,
		RequiredRole: task.AssignedRole,
		PreferredTeam: task.TeamID,
	}

	suggestions, err := e.SuggestAssignees(ctx, criteria)
	if err != nil {
		return nil, err
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no available assignees found for task %s", task.ID)
	}

	// Get the top suggestion
	topSuggestion := suggestions[0]

	// Get the member
	member, err := e.teamRepo.GetMemberByID(ctx, topSuggestion.MemberID)
	if err != nil {
		return nil, err
	}

	e.log.WithFields(logrus.Fields{
		"task_id":      task.ID,
		"assigned_to":  member.ID,
		"member_name":  member.Name,
		"score":        topSuggestion.Score,
	}).Info("Auto-assigned task")

	return member, nil
}

// BulkAssign assigns multiple tasks to a single assignee
func (e *AssignmentEngine) BulkAssign(ctx context.Context, req *models.BulkAssignRequest) (*models.BulkAssignResponse, error) {
	response := &models.BulkAssignResponse{
		Success:       true,
		AssignedCount: 0,
		FailedCount:   0,
		FailedTaskIDs: []string{},
	}

	// Validate assignee exists and has capacity
	member, err := e.teamRepo.GetMemberByID(ctx, req.AssigneeID)
	if err != nil {
		return nil, fmt.Errorf("invalid assignee: %w", err)
	}

	if member.CurrentTasks+len(req.TaskIDs) > member.MaxTasks {
		return nil, fmt.Errorf("assignee %s does not have capacity for %d tasks", member.Name, len(req.TaskIDs))
	}

	for _, taskIDStr := range req.TaskIDs {
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			response.FailedCount++
			response.FailedTaskIDs = append(response.FailedTaskIDs, taskIDStr)
			continue
		}

		task, err := e.taskRepo.GetByID(ctx, taskID)
		if err != nil {
			response.FailedCount++
			response.FailedTaskIDs = append(response.FailedTaskIDs, taskIDStr)
			continue
		}

		// Update assignment
		task.AssignedTo = &req.AssigneeID
		if req.Role != "" {
			task.AssignedRole = req.Role
		}
		task.Status = models.TaskStatusAssigned

		if err := e.taskRepo.Update(ctx, task); err != nil {
			response.FailedCount++
			response.FailedTaskIDs = append(response.FailedTaskIDs, taskIDStr)
			continue
		}

		response.AssignedCount++
	}

	// Update member's task count
	for i := 0; i < response.AssignedCount; i++ {
		_ = e.teamRepo.IncrementMemberTaskCount(ctx, req.AssigneeID)
	}

	response.Success = response.FailedCount == 0

	e.log.WithFields(logrus.Fields{
		"assigned_count": response.AssignedCount,
		"failed_count":   response.FailedCount,
		"assignee":       req.AssigneeID,
	}).Info("Bulk assignment completed")

	return response, nil
}

// GetWorkload retrieves workload information for a member
func (e *AssignmentEngine) GetWorkload(ctx context.Context, memberID uuid.UUID) (*models.WorkloadInfo, error) {
	member, err := e.teamRepo.GetMemberByID(ctx, memberID)
	if err != nil {
		return nil, err
	}

	// Get task breakdown
	tasks, err := e.taskRepo.FindByAssignee(ctx, memberID)
	if err != nil {
		return nil, err
	}

	workload := &models.WorkloadInfo{
		MemberID:          member.ID,
		MemberName:        member.Name,
		Role:              member.Role,
		CurrentTasks:      member.CurrentTasks,
		MaxTasks:          member.MaxTasks,
		AvailableCapacity: member.GetAvailableCapacity(),
		UtilizationRate:   float64(member.CurrentTasks) / float64(member.MaxTasks) * 100,
		TasksByStatus:     make(map[models.TaskStatus]int),
		TasksByPriority:   make(map[models.TaskPriority]int),
		OverdueTasks:      0,
		DueSoonTasks:      0,
	}

	// Categorize tasks
	for _, task := range tasks {
		// Skip completed/cancelled
		if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusCancelled {
			continue
		}

		workload.TasksByStatus[task.Status]++
		workload.TasksByPriority[task.Priority]++

		// Check overdue/due soon
		if task.DueDate != nil {
			if task.IsOverdue() {
				workload.OverdueTasks++
			} else if task.IsDueSoon(24) { // Due within 24 hours
				workload.DueSoonTasks++
			}
		}
	}

	return workload, nil
}
