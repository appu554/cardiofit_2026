// Package kb3 provides integration with KB-3 Temporal/Guidelines service.
// KB-3 is the "Temporal Brain" that manages when care obligations are due,
// while KB-9 is the "Accountability Engine" that identifies what obligations exist.
//
// Together they form Tier 7: Longitudinal Intelligence Platform.
package kb3

import "time"

// ConstraintStatus represents the temporal status of a care obligation.
// These statuses map directly to KB-3's PathwayEngine constraint evaluation.
type ConstraintStatus string

const (
	// StatusPending indicates the obligation is not yet due.
	StatusPending ConstraintStatus = "PENDING"

	// StatusApproaching indicates the deadline is within the alert threshold.
	StatusApproaching ConstraintStatus = "APPROACHING"

	// StatusOverdue indicates the deadline has passed but within grace period.
	StatusOverdue ConstraintStatus = "OVERDUE"

	// StatusMissed indicates the grace period has expired.
	StatusMissed ConstraintStatus = "MISSED"

	// StatusMet indicates the obligation was fulfilled.
	StatusMet ConstraintStatus = "MET"
)

// ScheduleItemType represents the type of scheduled care item.
type ScheduleItemType string

const (
	ItemTypeLab         ScheduleItemType = "lab"
	ItemTypeAppointment ScheduleItemType = "appointment"
	ItemTypeMedication  ScheduleItemType = "medication"
	ItemTypeProcedure   ScheduleItemType = "procedure"
	ItemTypeScreening   ScheduleItemType = "screening"
	ItemTypeAssessment  ScheduleItemType = "assessment"
)

// ScheduleStatus represents the status of a scheduled item.
type ScheduleStatus string

const (
	SchedulePending   ScheduleStatus = "pending"
	ScheduleCompleted ScheduleStatus = "completed"
	ScheduleOverdue   ScheduleStatus = "overdue"
	ScheduleSkipped   ScheduleStatus = "skipped"
	ScheduleCancelled ScheduleStatus = "cancelled"
)

// Frequency represents recurrence frequency.
type Frequency string

const (
	FrequencyDaily   Frequency = "daily"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
	FrequencyYearly  Frequency = "yearly"
)

// RecurrencePattern defines how a scheduled item repeats.
// Matches KB-3's scheduling.go RecurrencePattern struct.
type RecurrencePattern struct {
	// Frequency determines the base time unit (daily, weekly, monthly, yearly)
	Frequency Frequency `json:"frequency"`

	// Interval is the number of frequency units between occurrences
	// e.g., Interval=3 with FrequencyMonthly = every 3 months
	Interval int `json:"interval"`

	// DaysOfWeek for weekly recurrence (0=Sunday, 6=Saturday)
	DaysOfWeek []int `json:"days_of_week,omitempty"`

	// DayOfMonth for monthly recurrence
	DayOfMonth int `json:"day_of_month,omitempty"`

	// MonthOfYear for annual recurrence (1-12)
	MonthOfYear int `json:"month_of_year,omitempty"`

	// EndDate when the recurrence stops (nil = never)
	EndDate *time.Time `json:"end_date,omitempty"`

	// MaxOccurrences limits total number of occurrences
	MaxOccurrences int `json:"max_occurrences,omitempty"`
}

// ScheduledItem represents a scheduled care item in KB-3.
// Matches KB-3's models/schedule.go ScheduledItem struct.
type ScheduledItem struct {
	// ItemID is the unique identifier
	ItemID string `json:"item_id"`

	// PatientID links to the patient
	PatientID string `json:"patient_id"`

	// Type indicates what kind of care item this is
	Type ScheduleItemType `json:"type"`

	// Name is a human-readable description
	Name string `json:"name"`

	// Description provides additional context
	Description string `json:"description,omitempty"`

	// DueDate is when the item should be completed
	DueDate time.Time `json:"due_date"`

	// Priority (1=highest, 5=lowest)
	Priority int `json:"priority"`

	// IsRecurring indicates if this item repeats
	IsRecurring bool `json:"is_recurring"`

	// Recurrence defines the repeat pattern if IsRecurring
	Recurrence *RecurrencePattern `json:"recurrence,omitempty"`

	// Status tracks completion state
	Status ScheduleStatus `json:"status"`

	// SourceProtocol links back to the originating guideline/measure
	SourceProtocol string `json:"source_protocol,omitempty"`

	// SourceMeasureID links to the CMS measure that identified this gap
	SourceMeasureID string `json:"source_measure_id,omitempty"`

	// CreatedAt is when the item was scheduled
	CreatedAt time.Time `json:"created_at"`

	// CompletedAt is when the item was marked complete
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TemporalContext contains temporal enrichment data from KB-3.
// This is added to care gaps to provide deadline information.
type TemporalContext struct {
	// DueDate is when the care gap should be addressed
	DueDate *time.Time `json:"due_date,omitempty"`

	// OverdueDate is when the gap becomes overdue (after grace period)
	OverdueDate *time.Time `json:"overdue_date,omitempty"`

	// GracePeriod before the gap is marked overdue
	GracePeriod time.Duration `json:"grace_period,omitempty"`

	// Status is the current temporal status
	Status ConstraintStatus `json:"status"`

	// LastCompletedDate is when this care item was last fulfilled
	LastCompletedDate *time.Time `json:"last_completed_date,omitempty"`

	// NextDueDate is when the next occurrence is due (for recurring items)
	NextDueDate *time.Time `json:"next_due_date,omitempty"`

	// Recurrence pattern if this is a recurring care item
	Recurrence *RecurrencePattern `json:"recurrence,omitempty"`

	// DaysUntilDue is the number of days until due date (negative if overdue)
	DaysUntilDue int `json:"days_until_due"`

	// DaysOverdue is the number of days past due (0 if not overdue)
	DaysOverdue int `json:"days_overdue"`
}

// ScheduleRequest is the request to create a new scheduled item.
type ScheduleRequest struct {
	PatientID       string            `json:"patient_id"`
	Type            ScheduleItemType  `json:"type"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	DueDate         time.Time         `json:"due_date"`
	Priority        int               `json:"priority"`
	IsRecurring     bool              `json:"is_recurring"`
	Recurrence      *RecurrencePattern `json:"recurrence,omitempty"`
	SourceProtocol  string            `json:"source_protocol,omitempty"`
	SourceMeasureID string            `json:"source_measure_id,omitempty"`
}

// ScheduleResponse is the response from KB-3 schedule operations.
type ScheduleResponse struct {
	Items []ScheduledItem `json:"items"`
	Total int             `json:"total"`
}

// PathwayOverdueItem represents an overdue pathway action from KB-3.
// This matches the structure returned by /v1/alerts/overdue for pathway items.
type PathwayOverdueItem struct {
	InstanceID   string    `json:"instance_id"`
	PatientID    string    `json:"patient_id"`
	PathwayID    string    `json:"pathway_id"`
	ActionID     string    `json:"action_id"`
	ActionName   string    `json:"action_name"`
	Deadline     time.Time `json:"deadline"`
	OverdueBy    int64     `json:"overdue_by"` // Nanoseconds overdue
	Severity     string    `json:"severity"`   // critical, major, minor
	CurrentStage string    `json:"current_stage"`
}

// OverdueAlert represents an overdue item alert from KB-3 (unified format).
type OverdueAlert struct {
	ItemID        string           `json:"item_id"`
	PatientID     string           `json:"patient_id"`
	Type          ScheduleItemType `json:"type"`
	Name          string           `json:"name"`
	DueDate       time.Time        `json:"due_date"`
	DaysOverdue   int              `json:"days_overdue"`
	Priority      int              `json:"priority"`
	Severity      string           `json:"severity"` // critical, major, minor
	SourceMeasure string           `json:"source_measure,omitempty"`
}

// OverdueAlertsResponse is the response from KB-3 overdue alerts query.
type OverdueAlertsResponse struct {
	Alerts []OverdueAlert `json:"alerts"`
	Total  int            `json:"total"`
}

// MeasureTemporalMapping maps CMS measures to their temporal requirements.
// This defines how KB-9 measures map to KB-3 scheduling.
type MeasureTemporalMapping struct {
	MeasureID  string           `json:"measure_id"`
	ItemType   ScheduleItemType `json:"item_type"`
	Recurrence *RecurrencePattern `json:"recurrence,omitempty"`
	GracePeriod time.Duration   `json:"grace_period"`
	Priority   int              `json:"priority"`
}

// DefaultMeasureMappings returns the standard temporal mappings for CMS measures.
func DefaultMeasureMappings() map[string]MeasureTemporalMapping {
	return map[string]MeasureTemporalMapping{
		"CMS122": {
			MeasureID: "CMS122",
			ItemType:  ItemTypeLab,
			Recurrence: &RecurrencePattern{
				Frequency: FrequencyMonthly,
				Interval:  3, // Every 3 months (quarterly)
			},
			GracePeriod: 30 * 24 * time.Hour, // 30 days grace
			Priority:    2,                    // High priority
		},
		"CMS165": {
			MeasureID: "CMS165",
			ItemType:  ItemTypeAppointment,
			Recurrence: &RecurrencePattern{
				Frequency: FrequencyYearly,
				Interval:  1, // Annual
			},
			GracePeriod: 60 * 24 * time.Hour, // 60 days grace
			Priority:    2,                    // High priority
		},
		"CMS130": {
			MeasureID: "CMS130",
			ItemType:  ItemTypeScreening,
			Recurrence: &RecurrencePattern{
				Frequency: FrequencyYearly,
				Interval:  10, // Every 10 years
			},
			GracePeriod: 365 * 24 * time.Hour, // 1 year grace
			Priority:    3,                     // Medium priority
		},
		"CMS2": {
			MeasureID: "CMS2",
			ItemType:  ItemTypeAssessment,
			Recurrence: &RecurrencePattern{
				Frequency: FrequencyYearly,
				Interval:  1, // Annual
			},
			GracePeriod: 90 * 24 * time.Hour, // 90 days grace
			Priority:    3,                    // Medium priority
		},
	}
}
