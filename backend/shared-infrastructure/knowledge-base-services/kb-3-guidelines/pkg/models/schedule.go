package models

import "time"

// ScheduleItemType categorizes scheduled items
type ScheduleItemType string

const (
	ScheduleLab         ScheduleItemType = "lab"
	ScheduleAppointment ScheduleItemType = "appointment"
	ScheduleMedication  ScheduleItemType = "medication"
	ScheduleProcedure   ScheduleItemType = "procedure"
	ScheduleScreening   ScheduleItemType = "screening"
	ScheduleAssessment  ScheduleItemType = "assessment"
)

// ScheduleStatus for scheduled items
type ScheduleStatus string

const (
	SchedulePending   ScheduleStatus = "pending"
	ScheduleCompleted ScheduleStatus = "completed"
	ScheduleOverdue   ScheduleStatus = "overdue"
	ScheduleCancelled ScheduleStatus = "cancelled"
	ScheduleSkipped   ScheduleStatus = "skipped"
)

// Frequency for recurrence patterns
type Frequency string

const (
	FreqDaily   Frequency = "daily"
	FreqWeekly  Frequency = "weekly"
	FreqMonthly Frequency = "monthly"
	FreqYearly  Frequency = "yearly"
)

// ScheduledItem represents a scheduled care item
type ScheduledItem struct {
	ItemID         string             `json:"item_id"`
	PatientID      string             `json:"patient_id"`
	Type           ScheduleItemType   `json:"type"`
	Name           string             `json:"name"`
	Description    string             `json:"description,omitempty"`
	DueDate        time.Time          `json:"due_date"`
	Priority       int                `json:"priority"` // 1=highest, 5=lowest
	IsRecurring    bool               `json:"is_recurring"`
	Recurrence     *RecurrencePattern `json:"recurrence,omitempty"`
	Status         ScheduleStatus     `json:"status"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
	SourceProtocol string             `json:"source_protocol,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// RecurrencePattern per README specification
type RecurrencePattern struct {
	Frequency      Frequency  `json:"frequency"`
	Interval       int        `json:"interval"`                 // Every N frequency units
	DaysOfWeek     []int      `json:"days_of_week,omitempty"`   // 0=Sunday, 6=Saturday
	DayOfMonth     int        `json:"day_of_month,omitempty"`
	MonthOfYear    int        `json:"month_of_year,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	MaxOccurrences int        `json:"max_occurrences,omitempty"`
}

// ChronicSchedule for chronic disease management
type ChronicSchedule struct {
	ScheduleID      string           `json:"schedule_id"`
	Name            string           `json:"name"`
	GuidelineSource string           `json:"guideline_source"`
	Description     string           `json:"description,omitempty"`
	MonitoringItems []MonitoringItem `json:"monitoring_items"`
	FollowUpRules   []FollowUpRule   `json:"follow_up_rules"`
}

// MonitoringItem for chronic disease monitoring
type MonitoringItem struct {
	ItemID     string            `json:"item_id"`
	Name       string            `json:"name"`
	Type       ScheduleItemType  `json:"type"`
	Recurrence RecurrencePattern `json:"recurrence"`
	Conditions []Condition       `json:"conditions,omitempty"` // When this applies
}

// FollowUpRule for dynamic follow-up scheduling
type FollowUpRule struct {
	RuleID  string            `json:"rule_id"`
	Trigger Condition         `json:"trigger"`
	Action  string            `json:"action"`
	Timing  RecurrencePattern `json:"timing"`
}

// PreventiveSchedule for preventive care
type PreventiveSchedule struct {
	ScheduleID       string             `json:"schedule_id"`
	Name             string             `json:"name"`
	Description      string             `json:"description,omitempty"`
	TargetPopulation PopulationCriteria `json:"target_population"`
	ScreeningItems   []ScreeningItem    `json:"screening_items"`
}

// PopulationCriteria for targeting preventive care
type PopulationCriteria struct {
	AgeMin      *int     `json:"age_min,omitempty"`
	AgeMax      *int     `json:"age_max,omitempty"`
	Sex         string   `json:"sex,omitempty"` // M, F, any
	Conditions  []string `json:"conditions,omitempty"`
	RiskFactors []string `json:"risk_factors,omitempty"`
}

// ScreeningItem for preventive screening
type ScreeningItem struct {
	ItemID         string            `json:"item_id"`
	Name           string            `json:"name"`
	Recommendation string            `json:"recommendation"`
	StartAge       int               `json:"start_age"`
	EndAge         int               `json:"end_age"`
	Interval       RecurrencePattern `json:"interval"`
	Sex            string            `json:"sex"` // M, F, any
	EvidenceGrade  string            `json:"evidence_grade"`
	Source         string            `json:"source"` // USPSTF, ACIP, etc.
}

// AddScheduleRequest for API
type AddScheduleRequest struct {
	Type        ScheduleItemType   `json:"type" binding:"required"`
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description,omitempty"`
	DueDate     time.Time          `json:"due_date" binding:"required"`
	Priority    int                `json:"priority"`
	IsRecurring bool               `json:"is_recurring"`
	Recurrence  *RecurrencePattern `json:"recurrence,omitempty"`
}

// ScheduleSummary for patient schedule overview
type ScheduleSummary struct {
	PatientID       string `json:"patient_id"`
	TotalItems      int    `json:"total_items"`
	PendingItems    int    `json:"pending_items"`
	OverdueItems    int    `json:"overdue_items"`
	CompletedItems  int    `json:"completed_items"`
	UpcomingInWeek  int    `json:"upcoming_in_week"`
	UpcomingInMonth int    `json:"upcoming_in_month"`
}

// CalculateNextOccurrence calculates the next occurrence based on recurrence pattern
func (r *RecurrencePattern) CalculateNextOccurrence(from time.Time) time.Time {
	switch r.Frequency {
	case FreqDaily:
		return from.AddDate(0, 0, r.Interval)
	case FreqWeekly:
		return from.AddDate(0, 0, 7*r.Interval)
	case FreqMonthly:
		return from.AddDate(0, r.Interval, 0)
	case FreqYearly:
		return from.AddDate(r.Interval, 0, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}

// Helper functions
func IntPtr(i int) *int { return &i }
