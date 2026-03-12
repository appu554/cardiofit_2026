// Package models contains domain models for KB-14 Care Navigator
package models

import (
	"time"

	"github.com/google/uuid"
)

// WorklistItem represents a task item in a worklist view
type WorklistItem struct {
	// Task info
	TaskID       uuid.UUID    `json:"task_id"`
	TaskNumber   string       `json:"task_number"`
	Type         TaskType     `json:"type"`
	Status       TaskStatus   `json:"status"`
	Priority     TaskPriority `json:"priority"`
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`

	// Patient info
	PatientID   string `json:"patient_id"`
	PatientName string `json:"patient_name,omitempty"`

	// Assignment info
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	AssigneeName string     `json:"assignee_name,omitempty"`
	AssignedRole string     `json:"assigned_role,omitempty"`
	TeamID       *uuid.UUID `json:"team_id,omitempty"`
	TeamName     string     `json:"team_name,omitempty"`

	// Timing
	CreatedAt   time.Time  `json:"created_at"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	SLAMinutes  int        `json:"sla_minutes"`

	// Calculated fields
	IsOverdue        bool   `json:"is_overdue"`
	TimeRemaining    int    `json:"time_remaining_minutes"` // Negative if overdue
	EscalationLevel  int    `json:"escalation_level"`
	SLAElapsedPercent float64 `json:"sla_elapsed_percent"`

	// Source info
	Source   TaskSource `json:"source"`
	SourceID string     `json:"source_id,omitempty"`

	// Action count
	TotalActions     int `json:"total_actions"`
	CompletedActions int `json:"completed_actions"`
}

// WorklistFilters defines filters for worklist queries
type WorklistFilters struct {
	// Assignment filters
	UserID    *uuid.UUID   `json:"user_id,omitempty"`
	TeamID    *uuid.UUID   `json:"team_id,omitempty"`
	Unassigned bool        `json:"unassigned,omitempty"`

	// Patient filter
	PatientID string `json:"patient_id,omitempty"`

	// Status filters
	Statuses []TaskStatus `json:"statuses,omitempty"`

	// Priority filters
	Priorities []TaskPriority `json:"priorities,omitempty"`

	// Type filters
	Types []TaskType `json:"types,omitempty"`

	// Source filter
	Sources []TaskSource `json:"sources,omitempty"`

	// Time filters
	Overdue  bool       `json:"overdue,omitempty"`
	DueBefore *time.Time `json:"due_before,omitempty"`
	DueAfter  *time.Time `json:"due_after,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`

	// Escalation filter
	MinEscalationLevel *int `json:"min_escalation_level,omitempty"`

	// Pagination
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // "due_date", "created_at", "priority", "status"
	SortOrder string `json:"sort_order,omitempty"` // "asc", "desc"
}

// DefaultFilters returns default worklist filters
func DefaultFilters() WorklistFilters {
	return WorklistFilters{
		Page:      1,
		PageSize:  20,
		SortBy:    "due_date",
		SortOrder: "asc",
	}
}

// WorklistResponse represents the response for worklist queries
type WorklistResponse struct {
	Success bool           `json:"success"`
	Data    []WorklistItem `json:"data,omitempty"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	PageSize int           `json:"page_size"`
	Error   string         `json:"error,omitempty"`
}

// WorklistSummary provides a summary of worklist items
type WorklistSummary struct {
	TotalTasks      int64 `json:"total_tasks"`
	OverdueTasks    int64 `json:"overdue_tasks"`
	UrgentTasks     int64 `json:"urgent_tasks"`
	DueTodayTasks   int64 `json:"due_today_tasks"`
	DueThisWeekTasks int64 `json:"due_this_week_tasks"`
	UnassignedTasks int64 `json:"unassigned_tasks"`

	// By status
	TasksByStatus map[TaskStatus]int64 `json:"tasks_by_status"`

	// By priority
	TasksByPriority map[TaskPriority]int64 `json:"tasks_by_priority"`

	// By type
	TasksByType map[TaskType]int64 `json:"tasks_by_type"`
}

// WorklistSummaryResponse wraps the worklist summary for API responses
type WorklistSummaryResponse struct {
	Success bool             `json:"success"`
	Data    *WorklistSummary `json:"data,omitempty"`
	Error   string           `json:"error,omitempty"`
}
