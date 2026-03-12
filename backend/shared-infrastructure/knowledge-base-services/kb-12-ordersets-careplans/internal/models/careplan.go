// Package models defines data structures for KB-12 Order Sets & Care Plans
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CarePlanStatus represents the status of a care plan
type CarePlanStatus string

const (
	CarePlanStatusDraft      CarePlanStatus = "draft"
	CarePlanStatusActive     CarePlanStatus = "active"
	CarePlanStatusOnHold     CarePlanStatus = "on-hold"
	CarePlanStatusCompleted  CarePlanStatus = "completed"
	CarePlanStatusCancelled  CarePlanStatus = "cancelled"
	CarePlanStatusRevoked    CarePlanStatus = "revoked"
)

// CarePlanIntent represents the intent of a care plan
type CarePlanIntent string

const (
	IntentPlan       CarePlanIntent = "plan"
	IntentOrder      CarePlanIntent = "order"
	IntentOption     CarePlanIntent = "option"
	IntentProposal   CarePlanIntent = "proposal"
	IntentDirective  CarePlanIntent = "directive"
)

// GoalStatus represents the status of a goal
type GoalStatus string

const (
	GoalStatusProposed      GoalStatus = "proposed"
	GoalStatusAccepted      GoalStatus = "accepted"
	GoalStatusInProgress    GoalStatus = "in-progress"
	GoalStatusAchieved      GoalStatus = "achieved"
	GoalStatusNotAchieved   GoalStatus = "not-achieved"
	GoalStatusCancelled     GoalStatus = "cancelled"
)

// ActivityStatus represents the status of an activity
type ActivityStatus string

const (
	ActivityStatusScheduled   ActivityStatus = "scheduled"
	ActivityStatusInProgress  ActivityStatus = "in-progress"
	ActivityStatusCompleted   ActivityStatus = "completed"
	ActivityStatusNotDone     ActivityStatus = "not-done"
	ActivityStatusCancelled   ActivityStatus = "cancelled"
)

// ClinicalCondition represents a clinical condition code reference
type ClinicalCondition struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
}

// GuidelineReference represents a clinical guideline reference
type GuidelineReference struct {
	GuidelineID string `json:"guideline_id,omitempty"`
	Name        string `json:"name"`
	Source      string `json:"source,omitempty"`
	URL         string `json:"url,omitempty"`
	Year        int    `json:"year,omitempty"`
}

// CarePlanTemplate represents a template for a chronic care plan
// Uses Go native types with GORM JSONB serialization for clean API usage
type CarePlanTemplate struct {
	ID              string               `gorm:"type:varchar(50);primaryKey" json:"id"`
	PlanID          string               `gorm:"uniqueIndex;size:50" json:"plan_id,omitempty"`
	TemplateID      string               `gorm:"size:50" json:"template_id,omitempty"` // Alias for PlanID
	Condition       string               `gorm:"size:100;not null;index" json:"condition"`
	ConditionRef    *ClinicalCondition   `gorm:"type:jsonb" json:"condition_ref,omitempty"` // Structured condition
	Category        string               `gorm:"size:50" json:"category,omitempty"`
	Subcategory     string               `gorm:"size:50" json:"subcategory,omitempty"`
	Name            string               `gorm:"size:200;not null" json:"name"`
	Description     string               `gorm:"type:text" json:"description,omitempty"`
	GuidelineSource string               `gorm:"size:200" json:"guideline_source,omitempty"`
	Guidelines      []GuidelineReference `gorm:"type:jsonb" json:"guidelines,omitempty"`
	Version         string               `gorm:"size:20" json:"version,omitempty"`
	Status          string               `gorm:"size:20;default:'active'" json:"status,omitempty"`
	Duration        string               `gorm:"size:50" json:"duration,omitempty"`      // e.g., "ongoing", "6 months"
	ReviewPeriod    string               `gorm:"size:100" json:"review_period,omitempty"` // e.g., "3-6 months"
	Goals           GoalSlice            `gorm:"type:jsonb;not null" json:"goals"`
	Activities      ActivitySlice        `gorm:"type:jsonb;not null" json:"activities"`
	Monitoring      MonitoringSlice      `gorm:"type:jsonb" json:"monitoring,omitempty"`
	MonitoringItems MonitoringSlice      `gorm:"type:jsonb" json:"monitoring_items,omitempty"`
	Active          bool                 `gorm:"default:true" json:"active"`
	CreatedAt       time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for CarePlanTemplate
func (CarePlanTemplate) TableName() string {
	return "care_plan_templates"
}

// BeforeCreate generates ID if not set
func (c *CarePlanTemplate) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// CarePlanInstance represents an activated care plan for a patient
type CarePlanInstance struct {
	ID                  uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID          string              `gorm:"uniqueIndex;size:50;not null" json:"instance_id"`
	TemplateID          string              `gorm:"size:50;not null;index" json:"template_id"`
	PatientID           string              `gorm:"size:50;not null;index" json:"patient_id"`
	Status              CarePlanStatus      `gorm:"size:20;not null;index" json:"status"`
	StartDate           time.Time           `gorm:"not null" json:"start_date"`
	EndDate             *time.Time          `json:"end_date,omitempty"`
	GoalsProgress       ProgressSlice       `gorm:"type:jsonb" json:"goals_progress,omitempty"`
	ActivitiesCompleted CompletionSlice     `gorm:"type:jsonb" json:"activities_completed,omitempty"`
	CreatedAt           time.Time           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time           `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for CarePlanInstance
func (CarePlanInstance) TableName() string {
	return "care_plan_instances"
}

// BeforeCreate generates UUID and instance ID before creating
func (c *CarePlanInstance) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.InstanceID == "" {
		c.InstanceID = "CPI-" + uuid.New().String()[:8]
	}
	return nil
}

// Goal represents a clinical goal in a care plan
type Goal struct {
	GoalID        string          `json:"goal_id,omitempty"`
	ID            string          `json:"id,omitempty"`
	Description   string          `json:"description"`
	Category      string          `json:"category,omitempty"`
	Priority      string          `json:"priority,omitempty"`
	TargetDate    string          `json:"target_date,omitempty"`
	Targets       []GoalTarget    `json:"targets,omitempty"`
	Addresses     []CodeReference `json:"addresses,omitempty"`
	Status        GoalStatus      `json:"status,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	Condition     string          `json:"condition,omitempty"` // Conditional goal activation
}

// CarePlanGoal is an alias for Goal for backward compatibility
type CarePlanGoal = Goal

// GoalTarget represents a measurable target for a goal
type GoalTarget struct {
	TargetID    string `json:"target_id,omitempty"`
	Measure     string `json:"measure,omitempty"`
	Metric      string `json:"metric,omitempty"`
	Code        string `json:"code,omitempty"`
	Value       string `json:"value,omitempty"`
	TargetValue string `json:"target_value,omitempty"`
	Timeframe   string `json:"timeframe,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Achieved    bool   `json:"achieved,omitempty"`
}

// GoalProgress represents progress toward a goal
type GoalProgress struct {
	GoalID       string     `json:"goal_id"`
	Status       GoalStatus `json:"status"`
	ProgressPct  float64    `json:"progress_pct"`
	CurrentValue string     `json:"current_value,omitempty"`
	TargetValue  string     `json:"target_value,omitempty"`
	LastMeasured *time.Time `json:"last_measured,omitempty"`
	Notes        string     `json:"notes,omitempty"`
}

// Activity represents an activity in a care plan
type Activity struct {
	ActivityID     string                 `json:"activity_id,omitempty"`
	ID             string                 `json:"id,omitempty"`
	ActivityType   string                 `json:"activity_type,omitempty"`
	Type           string                 `json:"type,omitempty"`
	Description    string                 `json:"description"`
	Detail         ActivityDetail         `json:"detail,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"` // Flexible details map
	Recurrence     *RecurrencePattern     `json:"recurrence,omitempty"`
	Status         ActivityStatus         `json:"status,omitempty"`
	GoalReferences []string               `json:"goal_references,omitempty"`
	Frequency      string                 `json:"frequency,omitempty"`
	Condition      string                 `json:"condition,omitempty"`
}

// ActivityDetail represents the detailed specification of an activity
type ActivityDetail struct {
	// For medications
	DrugCode    string `json:"drug_code,omitempty"`
	DrugName    string `json:"drug_name,omitempty"`
	RxNormCode  string `json:"rxnorm_code,omitempty"`
	Dose        string `json:"dose,omitempty"`
	Route       string `json:"route,omitempty"`
	Frequency   string `json:"frequency,omitempty"`
	PRN         bool   `json:"prn,omitempty"`
	PRNReason   string `json:"prn_reason,omitempty"`

	// For appointments/visits
	ServiceType string `json:"service_type,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Location    string `json:"location,omitempty"`
	Duration    string `json:"duration,omitempty"`

	// For monitoring
	LabCode  string `json:"lab_code,omitempty"`
	LabName  string `json:"lab_name,omitempty"`
	Interval string `json:"interval,omitempty"`

	// For education
	Topic     string `json:"topic,omitempty"`
	Materials string `json:"materials,omitempty"`
	Format    string `json:"format,omitempty"`

	// For lifestyle
	Intervention string `json:"intervention,omitempty"`
	Target       string `json:"target,omitempty"`

	// Common
	Instructions string `json:"instructions,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

// RecurrencePattern represents a schedule recurrence pattern
type RecurrencePattern struct {
	PatternID   string    `json:"pattern_id,omitempty"`
	Type        string    `json:"type,omitempty"`
	Frequency   int       `json:"frequency,omitempty"`
	Interval    string    `json:"interval,omitempty"`
	DaysOfWeek  []string  `json:"days_of_week,omitempty"`
	DayOfMonth  int       `json:"day_of_month,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty"`
	EndDate     time.Time `json:"end_date,omitempty"`
	Occurrences int       `json:"occurrences,omitempty"`
}

// ActivityCompletion represents completion status of an activity
type ActivityCompletion struct {
	ActivityID      string         `json:"activity_id"`
	Status          ActivityStatus `json:"status"`
	CompletedDate   *time.Time     `json:"completed_date,omitempty"`
	NextDueDate     *time.Time     `json:"next_due_date,omitempty"`
	CompletionCount int            `json:"completion_count"`
	Notes           string         `json:"notes,omitempty"`
}

// MonitoringItem represents a monitoring requirement
type MonitoringItem struct {
	ItemID         string             `json:"item_id,omitempty"`
	ID             string             `json:"id,omitempty"`
	Name           string             `json:"name"`
	LabCode        string             `json:"lab_code,omitempty"`
	LOINCCode      string             `json:"loinc_code,omitempty"`
	VitalType      string             `json:"vital_type,omitempty"`
	Parameter      string             `json:"parameter,omitempty"`
	Frequency      string             `json:"frequency,omitempty"`
	Recurrence     *RecurrencePattern `json:"recurrence,omitempty"`
	NormalRange    string             `json:"normal_range,omitempty"`
	AlertRange     string             `json:"alert_range,omitempty"`
	Target         string             `json:"target,omitempty"`          // Target value for monitoring
	AlertThreshold string             `json:"alert_threshold,omitempty"` // Threshold for alerts
	Instructions   string             `json:"instructions,omitempty"`
}

// ==================== Custom GORM Types for JSONB ====================

// GoalSlice is a []Goal that serializes to JSONB
type GoalSlice []Goal

func (s GoalSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *GoalSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// ActivitySlice is a []Activity that serializes to JSONB
type ActivitySlice []Activity

func (s ActivitySlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *ActivitySlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// MonitoringSlice is a []MonitoringItem that serializes to JSONB
type MonitoringSlice []MonitoringItem

func (s MonitoringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *MonitoringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// ProgressSlice is a []GoalProgress that serializes to JSONB
type ProgressSlice []GoalProgress

func (s ProgressSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *ProgressSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// CompletionSlice is a []ActivityCompletion that serializes to JSONB
type CompletionSlice []ActivityCompletion

func (s CompletionSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *CompletionSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// ==================== Compatibility Methods ====================

// GetGoals returns the goals directly (compatibility method)
func (c *CarePlanTemplate) GetGoals() ([]Goal, error) {
	return c.Goals, nil
}

// SetGoals sets the goals directly (compatibility method)
func (c *CarePlanTemplate) SetGoals(goals []Goal) error {
	c.Goals = goals
	return nil
}

// GetActivities returns the activities directly (compatibility method)
func (c *CarePlanTemplate) GetActivities() ([]Activity, error) {
	return c.Activities, nil
}

// SetActivities sets the activities directly (compatibility method)
func (c *CarePlanTemplate) SetActivities(activities []Activity) error {
	c.Activities = activities
	return nil
}

// GetMonitoringItems returns the monitoring items directly (compatibility method)
func (c *CarePlanTemplate) GetMonitoringItems() ([]MonitoringItem, error) {
	if c.MonitoringItems != nil {
		return c.MonitoringItems, nil
	}
	return c.Monitoring, nil
}

// SetMonitoringItems sets the monitoring items directly (compatibility method)
func (c *CarePlanTemplate) SetMonitoringItems(items []MonitoringItem) error {
	c.MonitoringItems = items
	c.Monitoring = items
	return nil
}

// GetGoalsProgress returns the goals progress directly (compatibility method)
func (c *CarePlanInstance) GetGoalsProgress() ([]GoalProgress, error) {
	return c.GoalsProgress, nil
}

// SetGoalsProgress sets the goals progress directly (compatibility method)
func (c *CarePlanInstance) SetGoalsProgress(progress []GoalProgress) error {
	c.GoalsProgress = progress
	return nil
}

// GetActivitiesCompleted returns the activities completed directly (compatibility method)
func (c *CarePlanInstance) GetActivitiesCompleted() ([]ActivityCompletion, error) {
	return c.ActivitiesCompleted, nil
}

// SetActivitiesCompleted sets the activities completed directly (compatibility method)
func (c *CarePlanInstance) SetActivitiesCompleted(completions []ActivityCompletion) error {
	c.ActivitiesCompleted = completions
	return nil
}

// IsActive returns true if the care plan is currently active
func (c *CarePlanInstance) IsActive() bool {
	return c.Status == CarePlanStatusActive
}

// CalculateOverallProgress calculates overall care plan progress
func (c *CarePlanInstance) CalculateOverallProgress() (float64, error) {
	if len(c.GoalsProgress) == 0 {
		return 0, nil
	}

	var total float64
	for _, p := range c.GoalsProgress {
		total += p.ProgressPct
	}
	return total / float64(len(c.GoalsProgress)), nil
}
