// Package services provides business logic for KB-14 Care Navigator
package services

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
)

// AnalyticsService provides dashboard and analytics functionality
type AnalyticsService struct {
	taskRepo       *database.TaskRepository
	teamRepo       *database.TeamRepository
	escalationRepo *database.EscalationRepository
	log            *logrus.Entry
}

// NewAnalyticsService creates a new AnalyticsService
func NewAnalyticsService(
	taskRepo *database.TaskRepository,
	teamRepo *database.TeamRepository,
	escalationRepo *database.EscalationRepository,
	log *logrus.Entry,
) *AnalyticsService {
	return &AnalyticsService{
		taskRepo:       taskRepo,
		teamRepo:       teamRepo,
		escalationRepo: escalationRepo,
		log:            log.WithField("service", "analytics"),
	}
}

// DashboardMetrics represents the main dashboard metrics
type DashboardMetrics struct {
	// Task counts
	TotalActiveTasks  int64 `json:"total_active_tasks"`
	OverdueTasks      int64 `json:"overdue_tasks"`
	UrgentTasks       int64 `json:"urgent_tasks"`
	DueTodayTasks     int64 `json:"due_today_tasks"`
	UnassignedTasks   int64 `json:"unassigned_tasks"`

	// By status
	TasksByStatus map[string]int64 `json:"tasks_by_status"`

	// By priority
	TasksByPriority map[string]int64 `json:"tasks_by_priority"`

	// By source
	TasksBySource map[string]int64 `json:"tasks_by_source"`

	// Escalation metrics
	TotalEscalations     int64 `json:"total_escalations"`
	UnacknowledgedEsc    int64 `json:"unacknowledged_escalations"`
	CriticalEscalations  int64 `json:"critical_escalations"`

	// SLA metrics
	SLAComplianceRate float64 `json:"sla_compliance_rate"`
	AvgCompletionTime int     `json:"avg_completion_time_minutes"`

	// Timestamp
	GeneratedAt time.Time `json:"generated_at"`
}

// GetDashboardMetrics retrieves dashboard metrics
func (s *AnalyticsService) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	metrics := &DashboardMetrics{
		TasksByStatus:   make(map[string]int64),
		TasksByPriority: make(map[string]int64),
		TasksBySource:   make(map[string]int64),
		GeneratedAt:     time.Now().UTC(),
	}

	// Get worklist summary
	summary, err := s.taskRepo.GetTaskSummary(ctx, database.TaskFilters{})
	if err != nil {
		return nil, err
	}

	metrics.TotalActiveTasks = summary.TotalTasks
	metrics.OverdueTasks = summary.OverdueTasks
	metrics.UrgentTasks = summary.UrgentTasks
	metrics.DueTodayTasks = summary.DueTodayTasks
	metrics.UnassignedTasks = summary.UnassignedTasks

	// Convert status map
	for status, count := range summary.TasksByStatus {
		metrics.TasksByStatus[string(status)] = count
	}

	// Convert priority map
	for priority, count := range summary.TasksByPriority {
		metrics.TasksByPriority[string(priority)] = count
	}

	// Get escalation summary
	escSummary, err := s.escalationRepo.GetEscalationSummary(ctx)
	if err == nil {
		metrics.TotalEscalations = escSummary.TotalEscalations
		metrics.UnacknowledgedEsc = escSummary.UnacknowledgedCount
		metrics.CriticalEscalations = escSummary.CriticalCount + escSummary.ExecutiveCount
	}

	// Calculate SLA compliance (tasks completed on time / total completed)
	// This is a simplified calculation
	if completedCount, ok := metrics.TasksByStatus[string(models.TaskStatusCompleted)]; ok && completedCount > 0 {
		// In a real implementation, we'd query tasks completed within SLA
		metrics.SLAComplianceRate = 85.0 // Placeholder
	}

	return metrics, nil
}

// SLAMetrics represents SLA compliance metrics
type SLAMetrics struct {
	Period            string  `json:"period"`
	TotalCompleted    int64   `json:"total_completed"`
	CompletedOnTime   int64   `json:"completed_on_time"`
	CompletedLate     int64   `json:"completed_late"`
	ComplianceRate    float64 `json:"compliance_rate"`
	AvgTimeToComplete int     `json:"avg_time_to_complete_minutes"`

	// By priority
	ByPriority map[string]SLAPriorityMetrics `json:"by_priority"`

	// By type
	ByType map[string]SLATypeMetrics `json:"by_type"`
}

// SLAPriorityMetrics represents SLA metrics by priority
type SLAPriorityMetrics struct {
	Priority       string  `json:"priority"`
	Total          int64   `json:"total"`
	OnTime         int64   `json:"on_time"`
	Late           int64   `json:"late"`
	ComplianceRate float64 `json:"compliance_rate"`
}

// SLATypeMetrics represents SLA metrics by task type
type SLATypeMetrics struct {
	Type           string  `json:"type"`
	Total          int64   `json:"total"`
	OnTime         int64   `json:"on_time"`
	Late           int64   `json:"late"`
	ComplianceRate float64 `json:"compliance_rate"`
	DefaultSLA     int     `json:"default_sla_minutes"`
}

// GetSLAMetrics retrieves SLA compliance metrics for a time period
func (s *AnalyticsService) GetSLAMetrics(ctx context.Context, days int) (*SLAMetrics, error) {
	metrics := &SLAMetrics{
		Period:     days_to_period(days),
		ByPriority: make(map[string]SLAPriorityMetrics),
		ByType:     make(map[string]SLATypeMetrics),
	}

	// In a real implementation, this would query completed tasks and analyze SLA compliance
	// For now, we return placeholder data structure

	// Calculate date range
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -days)

	// Get completed tasks in date range
	filters := database.TaskFilters{
		Statuses:      []models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusVerified},
		CreatedAfter:  &startDate,
		CreatedBefore: &endDate,
		Page:          1,
		PageSize:      1000, // Get all for analysis
	}

	tasks, total, err := s.taskRepo.FindWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	metrics.TotalCompleted = total

	// Analyze each task
	for _, task := range tasks {
		wasOnTime := false
		if task.CompletedAt != nil && task.DueDate != nil {
			wasOnTime = !task.CompletedAt.After(*task.DueDate)
		}

		if wasOnTime {
			metrics.CompletedOnTime++
		} else {
			metrics.CompletedLate++
		}

		// Track by priority
		priority := string(task.Priority)
		pm, exists := metrics.ByPriority[priority]
		if !exists {
			pm = SLAPriorityMetrics{Priority: priority}
		}
		pm.Total++
		if wasOnTime {
			pm.OnTime++
		} else {
			pm.Late++
		}
		metrics.ByPriority[priority] = pm

		// Track by type
		taskType := string(task.Type)
		tm, exists := metrics.ByType[taskType]
		if !exists {
			tm = SLATypeMetrics{
				Type:       taskType,
				DefaultSLA: task.Type.GetDefaultSLAMinutes(),
			}
		}
		tm.Total++
		if wasOnTime {
			tm.OnTime++
		} else {
			tm.Late++
		}
		metrics.ByType[taskType] = tm
	}

	// Calculate compliance rates
	if metrics.TotalCompleted > 0 {
		metrics.ComplianceRate = float64(metrics.CompletedOnTime) / float64(metrics.TotalCompleted) * 100
	}

	for k, v := range metrics.ByPriority {
		if v.Total > 0 {
			v.ComplianceRate = float64(v.OnTime) / float64(v.Total) * 100
			metrics.ByPriority[k] = v
		}
	}

	for k, v := range metrics.ByType {
		if v.Total > 0 {
			v.ComplianceRate = float64(v.OnTime) / float64(v.Total) * 100
			metrics.ByType[k] = v
		}
	}

	return metrics, nil
}

// TaskTrend represents task volume trends
type TaskTrend struct {
	Date        string `json:"date"`
	Created     int64  `json:"created"`
	Completed   int64  `json:"completed"`
	Overdue     int64  `json:"overdue"`
	Escalations int64  `json:"escalations"`
}

// TrendMetrics represents task trend metrics
type TrendMetrics struct {
	Period string      `json:"period"`
	Days   int         `json:"days"`
	Trends []TaskTrend `json:"trends"`
}

// GetTrendMetrics retrieves task volume trends
func (s *AnalyticsService) GetTrendMetrics(ctx context.Context, days int) (*TrendMetrics, error) {
	metrics := &TrendMetrics{
		Period: days_to_period(days),
		Days:   days,
		Trends: make([]TaskTrend, 0, days),
	}

	// Generate trend data for each day
	// In a real implementation, this would query aggregated daily metrics
	for i := days - 1; i >= 0; i-- {
		date := time.Now().UTC().AddDate(0, 0, -i)
		trend := TaskTrend{
			Date: date.Format("2006-01-02"),
			// These would be actual counts from the database
			Created:     0,
			Completed:   0,
			Overdue:     0,
			Escalations: 0,
		}
		metrics.Trends = append(metrics.Trends, trend)
	}

	return metrics, nil
}

// CareGapAnalytics represents care gap closure analytics
type CareGapAnalytics struct {
	Period            string                    `json:"period"`
	TotalGapTasks     int64                     `json:"total_gap_tasks"`
	ClosedGaps        int64                     `json:"closed_gaps"`
	ClosureRate       float64                   `json:"closure_rate"`
	AvgTimeToClose    int                       `json:"avg_time_to_close_days"`
	ByCategory        map[string]CategoryMetric `json:"by_category"`
}

// CategoryMetric represents metrics for a care gap category
type CategoryMetric struct {
	Category    string  `json:"category"`
	Total       int64   `json:"total"`
	Closed      int64   `json:"closed"`
	ClosureRate float64 `json:"closure_rate"`
}

// GetCareGapAnalytics retrieves care gap closure analytics
func (s *AnalyticsService) GetCareGapAnalytics(ctx context.Context, days int) (*CareGapAnalytics, error) {
	analytics := &CareGapAnalytics{
		Period:     days_to_period(days),
		ByCategory: make(map[string]CategoryMetric),
	}

	// Get tasks from KB-9 source
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -days)

	filters := database.TaskFilters{
		Sources:       []models.TaskSource{models.TaskSourceKB9},
		CreatedAfter:  &startDate,
		CreatedBefore: &endDate,
		Page:          1,
		PageSize:      1000,
	}

	tasks, total, err := s.taskRepo.FindWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	analytics.TotalGapTasks = total

	// Analyze tasks
	for _, task := range tasks {
		if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusVerified {
			analytics.ClosedGaps++
		}

		// Extract category from metadata
		category := "other"
		if cat, ok := task.Metadata["gap_category"].(string); ok {
			category = cat
		}

		cm, exists := analytics.ByCategory[category]
		if !exists {
			cm = CategoryMetric{Category: category}
		}
		cm.Total++
		if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusVerified {
			cm.Closed++
		}
		analytics.ByCategory[category] = cm
	}

	// Calculate closure rates
	if analytics.TotalGapTasks > 0 {
		analytics.ClosureRate = float64(analytics.ClosedGaps) / float64(analytics.TotalGapTasks) * 100
	}

	for k, v := range analytics.ByCategory {
		if v.Total > 0 {
			v.ClosureRate = float64(v.Closed) / float64(v.Total) * 100
			analytics.ByCategory[k] = v
		}
	}

	return analytics, nil
}

// Helper function to convert days to period string
func days_to_period(days int) string {
	switch {
	case days <= 7:
		return "week"
	case days <= 30:
		return "month"
	case days <= 90:
		return "quarter"
	default:
		return "year"
	}
}
