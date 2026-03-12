// Package models contains domain models for KB-14 Care Navigator
package models

import "github.com/google/uuid"

// AssignmentSuggestion represents a suggested assignee for a task
type AssignmentSuggestion struct {
	MemberID      uuid.UUID `json:"member_id"`
	MemberName    string    `json:"member_name"`
	Role          string    `json:"role"`
	TeamID        uuid.UUID `json:"team_id"`
	TeamName      string    `json:"team_name"`
	Score         float64   `json:"score"`
	Reason        string    `json:"reason"`
	CurrentTasks  int       `json:"current_tasks"`
	MaxTasks      int       `json:"max_tasks"`
	AvailableCapacity int   `json:"available_capacity"`
}

// AssignmentScoreWeights defines the weights for different scoring factors
type AssignmentScoreWeights struct {
	WorkloadBalance   float64 // Weight for workload distribution (0-1)
	RoleMatch         float64 // Weight for role matching (0-1)
	PanelAttribution  float64 // Weight for patient's PCP panel (0-1)
	SkillMatch        float64 // Weight for skill matching (0-1)
	Availability      float64 // Weight for current availability (0-1)
}

// DefaultAssignmentWeights returns the default scoring weights
func DefaultAssignmentWeights() AssignmentScoreWeights {
	return AssignmentScoreWeights{
		WorkloadBalance:  0.30,
		RoleMatch:        0.25,
		PanelAttribution: 0.25,
		SkillMatch:       0.10,
		Availability:     0.10,
	}
}

// BulkAssignRequest represents a request to assign multiple tasks
type BulkAssignRequest struct {
	TaskIDs    []string  `json:"task_ids" binding:"required,min=1"`
	AssigneeID uuid.UUID `json:"assignee_id" binding:"required"`
	Role       string    `json:"role,omitempty"`
}

// BulkAssignResponse represents the response for bulk assignment
type BulkAssignResponse struct {
	Success       bool     `json:"success"`
	AssignedCount int      `json:"assigned_count"`
	FailedCount   int      `json:"failed_count"`
	FailedTaskIDs []string `json:"failed_task_ids,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// WorkloadInfo represents workload information for a team member
type WorkloadInfo struct {
	MemberID          uuid.UUID `json:"member_id"`
	MemberName        string    `json:"member_name"`
	Role              string    `json:"role"`
	CurrentTasks      int       `json:"current_tasks"`
	MaxTasks          int       `json:"max_tasks"`
	AvailableCapacity int       `json:"available_capacity"`
	UtilizationRate   float64   `json:"utilization_rate"` // Percentage 0-100

	// Task breakdown by status
	TasksByStatus map[TaskStatus]int `json:"tasks_by_status"`

	// Task breakdown by priority
	TasksByPriority map[TaskPriority]int `json:"tasks_by_priority"`

	// Overdue count
	OverdueTasks int `json:"overdue_tasks"`

	// Due soon (within 24 hours)
	DueSoonTasks int `json:"due_soon_tasks"`
}

// WorkloadResponse wraps workload info for API responses
type WorkloadResponse struct {
	Success bool          `json:"success"`
	Data    *WorkloadInfo `json:"data,omitempty"`
	Error   string        `json:"error,omitempty"`
}

// AssignmentSuggestResponse wraps assignment suggestions for API responses
type AssignmentSuggestResponse struct {
	Success bool                   `json:"success"`
	Data    []AssignmentSuggestion `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// AssignmentCriteria represents criteria for finding the best assignee
type AssignmentCriteria struct {
	TaskType       TaskType     `json:"task_type"`
	TaskPriority   TaskPriority `json:"task_priority"`
	PatientID      string       `json:"patient_id"`
	RequiredRole   string       `json:"required_role,omitempty"`
	RequiredSkills []string     `json:"required_skills,omitempty"`
	PreferredTeam  *uuid.UUID   `json:"preferred_team,omitempty"`
}
