// Package models contains domain models for KB-14 Care Navigator
package models

import "time"

// CareGap represents a care gap from KB-9
type CareGap struct {
	GapID         string                 `json:"gap_id"`
	PatientID     string                 `json:"patient_id"`
	GapType       string                 `json:"gap_type"`     // screenings, vaccinations, chronic_care, etc.
	GapCategory   string                 `json:"gap_category"` // preventive, chronic, quality
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	DueDate       *time.Time             `json:"due_date,omitempty"`
	Priority      string                 `json:"priority"` // high, medium, low
	Interventions []CareGapIntervention  `json:"interventions,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CareGapIntervention represents an intervention for closing a care gap
type CareGapIntervention struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Code        string `json:"code,omitempty"`
	CodeSystem  string `json:"code_system,omitempty"`
}

// TemporalAlert represents a temporal alert from KB-3
type TemporalAlert struct {
	AlertID      string                 `json:"alert_id"`
	PatientID    string                 `json:"patient_id"`
	EncounterID  string                 `json:"encounter_id,omitempty"`
	ProtocolID   string                 `json:"protocol_id"`
	ProtocolName string                 `json:"protocol_name"`
	ConstraintID string                 `json:"constraint_id,omitempty"`
	Action       string                 `json:"action"`
	Severity     string                 `json:"severity"` // critical, major, minor
	Status       string                 `json:"status"`   // pending, acknowledged, resolved
	Deadline     *time.Time             `json:"deadline,omitempty"`
	TimeOverdue  int                    `json:"time_overdue_minutes"`
	AlertTime    *time.Time             `json:"alert_time,omitempty"`
	Description  string                 `json:"description"`
	Reference    string                 `json:"reference,omitempty"`
	Acknowledged bool                   `json:"acknowledged"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProtocolDeadline represents a protocol deadline from KB-3
type ProtocolDeadline struct {
	DeadlineID    string                 `json:"deadline_id"`
	PatientID     string                 `json:"patient_id"`
	EncounterID   string                 `json:"encounter_id,omitempty"`
	ProtocolID    string                 `json:"protocol_id"`
	ProtocolName  string                 `json:"protocol_name"`
	StageID       string                 `json:"stage_id,omitempty"`
	StageName     string                 `json:"stage_name,omitempty"`
	ActionID      string                 `json:"action_id,omitempty"`
	ActionName    string                 `json:"action_name,omitempty"`
	Deadline      *time.Time             `json:"deadline,omitempty"`
	SLAMinutes    int                    `json:"sla_minutes"`
	Priority      string                 `json:"priority"` // critical, high, medium, low
	CurrentStatus string                 `json:"current_status"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CarePlanActivity represents a care plan activity from KB-12
type CarePlanActivity struct {
	ActivityID    string                 `json:"activity_id"`
	CarePlanID    string                 `json:"care_plan_id"`
	PatientID     string                 `json:"patient_id"`
	EncounterID   string                 `json:"encounter_id,omitempty"`
	Type          string                 `json:"type"` // medication, procedure, observation, appointment, etc.
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	Status        string                 `json:"status"`   // scheduled, in-progress, completed, cancelled
	Priority      string                 `json:"priority"` // routine, urgent, asap, stat
	DueDate       *time.Time             `json:"due_date,omitempty"`
	ScheduledTime *time.Time             `json:"scheduled_time,omitempty"`
	Instructions  []string               `json:"instructions,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ProtocolStep represents a protocol step from KB-12
type ProtocolStep struct {
	StepID       string                 `json:"step_id"`
	ProtocolID   string                 `json:"protocol_id"`
	ProtocolName string                 `json:"protocol_name"`
	PatientID    string                 `json:"patient_id"`
	EncounterID  string                 `json:"encounter_id,omitempty"`
	StepType     string                 `json:"step_type"` // action, decision, medication, lab, procedure
	StepNumber   int                    `json:"step_number,omitempty"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description,omitempty"`
	Status       string                 `json:"status"` // pending, in-progress, completed, skipped
	Priority     string                 `json:"priority"`
	DueDate      *time.Time             `json:"due_date,omitempty"`
	SLAMinutes   int                    `json:"sla_minutes,omitempty"`
	Actions      []ProtocolStepAction   `json:"actions,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProtocolStepAction represents an action within a protocol step
type ProtocolStepAction struct {
	ActionID    string `json:"action_id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}
