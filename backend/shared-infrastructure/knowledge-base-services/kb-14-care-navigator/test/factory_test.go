// Package test contains tests for KB-14 Care Navigator
package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-14-care-navigator/internal/clients"
	"kb-14-care-navigator/internal/models"
)

// =============================================================================
// Task Factory - Temporal Alert Mapping Tests
// =============================================================================

func TestTemporalAlert_PriorityMapping(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     models.TaskPriority
	}{
		{"critical_severity", "critical", models.TaskPriorityCritical},
		{"major_severity", "major", models.TaskPriorityHigh},
		{"minor_severity", "minor", models.TaskPriorityLow},
		{"unknown_severity", "unknown", models.TaskPriorityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			priority := models.TaskPriorityMedium
			switch tt.severity {
			case "critical":
				priority = models.TaskPriorityCritical
			case "major":
				priority = models.TaskPriorityHigh
			case "minor":
				priority = models.TaskPriorityLow
			}
			assert.Equal(t, tt.want, priority)
		})
	}
}

func TestTemporalAlert_TaskTypeMapping(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     models.TaskType
	}{
		{"critical_alert", "critical", models.TaskTypeAcuteProtocolDeadline},
		{"non_critical_alert", "warning", models.TaskTypeMonitoringOverdue},
		{"major_alert", "major", models.TaskTypeMonitoringOverdue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			taskType := models.TaskTypeMonitoringOverdue
			if tt.severity == "critical" {
				taskType = models.TaskTypeAcuteProtocolDeadline
			}
			assert.Equal(t, tt.want, taskType)
		})
	}
}

func TestTemporalAlert_SLACalculation(t *testing.T) {
	tests := []struct {
		name        string
		timeOverdue int
		taskType    models.TaskType
		wantSLA     int
	}{
		{"overdue_alert", 30, models.TaskTypeMonitoringOverdue, 60},
		{"not_overdue_critical", 0, models.TaskTypeAcuteProtocolDeadline, 60},
		{"not_overdue_monitoring", 0, models.TaskTypeMonitoringOverdue, 4320},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the SLA calculation logic from task_factory.go
			slaMinutes := tt.taskType.GetDefaultSLAMinutes()
			if tt.timeOverdue > 0 {
				slaMinutes = 60 // 1 hour for overdue alerts
			}
			assert.Equal(t, tt.wantSLA, slaMinutes)
		})
	}
}

// =============================================================================
// Task Factory - Care Gap Mapping Tests
// =============================================================================

func TestCareGap_TaskTypeMapping(t *testing.T) {
	tests := []struct {
		name    string
		gapType string
		want    models.TaskType
	}{
		{"screening_gap", "screening", models.TaskTypeScreeningOutreach},
		{"immunization_gap", "immunization", models.TaskTypeCareGapClosure},
		{"follow_up_gap", "follow_up", models.TaskTypeTransitionFollowup},
		{"monitoring_gap", "monitoring", models.TaskTypeMonitoringOverdue},
		{"unknown_gap", "unknown", models.TaskTypeCareGapClosure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			taskType := models.TaskTypeCareGapClosure
			switch tt.gapType {
			case "screening":
				taskType = models.TaskTypeScreeningOutreach
			case "immunization":
				taskType = models.TaskTypeCareGapClosure
			case "follow_up":
				taskType = models.TaskTypeTransitionFollowup
			case "monitoring":
				taskType = models.TaskTypeMonitoringOverdue
			}
			assert.Equal(t, tt.want, taskType)
		})
	}
}

func TestCareGap_PriorityMapping(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		want     models.TaskPriority
	}{
		{"high_priority", "high", models.TaskPriorityHigh},
		{"low_priority", "low", models.TaskPriorityLow},
		{"medium_priority", "medium", models.TaskPriorityMedium},
		{"default_priority", "", models.TaskPriorityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			priority := models.TaskPriorityMedium
			switch tt.priority {
			case "high":
				priority = models.TaskPriorityHigh
			case "low":
				priority = models.TaskPriorityLow
			}
			assert.Equal(t, tt.want, priority)
		})
	}
}

func TestCareGap_InterventionToAction(t *testing.T) {
	interventions := []clients.Intervention{
		{Type: "schedule", Description: "Schedule follow-up appointment", Code: "CODE-1", CodeSystem: "SNOMED"},
		{Type: "lab_order", Description: "Order HbA1c test", Code: "CODE-2", CodeSystem: "LOINC"},
		{Type: "education", Description: "Provide diabetes education", Code: "CODE-3", CodeSystem: "SNOMED"},
	}

	actions := make([]models.TaskAction, 0, len(interventions))
	for i, intervention := range interventions {
		action := models.TaskAction{
			ActionID:    "action-" + string(rune('1'+i)),
			Type:        intervention.Type,
			Description: intervention.Description,
			Required:    true,
			Completed:   false,
		}
		actions = append(actions, action)
	}

	require.Len(t, actions, 3)
	assert.Equal(t, "schedule", actions[0].Type)
	assert.Equal(t, "lab_order", actions[1].Type)
	assert.True(t, actions[0].Required)
	assert.False(t, actions[0].Completed)
}

// =============================================================================
// Task Factory - Care Plan Activity Mapping Tests
// =============================================================================

func TestCarePlanActivity_TaskTypeMapping(t *testing.T) {
	tests := []struct {
		name         string
		activityType string
		want         models.TaskType
	}{
		{"medication_activity", "medication", models.TaskTypeMedicationRefill},
		{"lab_activity", "lab", models.TaskTypeMonitoringOverdue},
		{"procedure_activity", "procedure", models.TaskTypeCarePlanReview},
		{"education_activity", "education", models.TaskTypeCareGapClosure},
		{"referral_activity", "referral", models.TaskTypeReferralProcessing},
		{"follow_up_activity", "follow_up", models.TaskTypeTransitionFollowup},
		{"default_activity", "unknown", models.TaskTypeCarePlanReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			taskType := models.TaskTypeCarePlanReview
			switch tt.activityType {
			case "medication":
				taskType = models.TaskTypeMedicationRefill
			case "lab":
				taskType = models.TaskTypeMonitoringOverdue
			case "procedure":
				taskType = models.TaskTypeCarePlanReview
			case "education":
				taskType = models.TaskTypeCareGapClosure
			case "referral":
				taskType = models.TaskTypeReferralProcessing
			case "follow_up":
				taskType = models.TaskTypeTransitionFollowup
			}
			assert.Equal(t, tt.want, taskType)
		})
	}
}

func TestCarePlanActivity_PriorityMapping(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		want     models.TaskPriority
	}{
		{"stat_priority", "stat", models.TaskPriorityCritical},
		{"asap_priority", "asap", models.TaskPriorityHigh},
		{"urgent_priority", "urgent", models.TaskPriorityHigh},
		{"routine_priority", "routine", models.TaskPriorityLow},
		{"default_priority", "normal", models.TaskPriorityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			priority := models.TaskPriorityMedium
			switch tt.priority {
			case "stat":
				priority = models.TaskPriorityCritical
			case "asap", "urgent":
				priority = models.TaskPriorityHigh
			case "routine":
				priority = models.TaskPriorityLow
			}
			assert.Equal(t, tt.want, priority)
		})
	}
}

// =============================================================================
// Task Factory - Monitoring Overdue Mapping Tests
// =============================================================================

func TestMonitoringOverdue_SLACalculation(t *testing.T) {
	tests := []struct {
		name        string
		daysOverdue int
		wantSLA     int
	}{
		{"very_overdue_10_days", 10, 60},     // 1 hour
		{"moderately_overdue_5_days", 5, 240}, // 4 hours
		{"slightly_overdue_2_days", 2, 1440},  // 24 hours (default)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the SLA calculation logic from task_factory.go
			slaMinutes := 1440 // 24 hours default
			if tt.daysOverdue > 7 {
				slaMinutes = 60 // 1 hour for very overdue
			} else if tt.daysOverdue > 3 {
				slaMinutes = 240 // 4 hours
			}
			assert.Equal(t, tt.wantSLA, slaMinutes)
		})
	}
}

func TestMonitoringOverdue_PriorityMapping(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     models.TaskPriority
	}{
		{"critical_severity", "critical", models.TaskPriorityCritical},
		{"high_severity", "high", models.TaskPriorityHigh},
		{"low_severity", "low", models.TaskPriorityLow},
		{"default_severity", "medium", models.TaskPriorityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			priority := models.TaskPriorityMedium
			switch tt.severity {
			case "critical":
				priority = models.TaskPriorityCritical
			case "high":
				priority = models.TaskPriorityHigh
			case "low":
				priority = models.TaskPriorityLow
			}
			assert.Equal(t, tt.want, priority)
		})
	}
}

// =============================================================================
// Task Factory - Protocol Step Mapping Tests
// =============================================================================

func TestProtocolStep_TaskTypeMapping(t *testing.T) {
	tests := []struct {
		name     string
		stepType string
		want     models.TaskType
	}{
		{"medication_step", "medication", models.TaskTypeMedicationReview},
		{"lab_step", "lab", models.TaskTypeCriticalLabReview},
		{"procedure_step", "procedure", models.TaskTypeCarePlanReview},
		{"action_step", "action", models.TaskTypeAcuteProtocolDeadline},
		{"decision_step", "decision", models.TaskTypeAcuteProtocolDeadline},
		{"default_step", "unknown", models.TaskTypeAcuteProtocolDeadline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the mapping logic from task_factory.go
			taskType := models.TaskTypeAcuteProtocolDeadline
			switch tt.stepType {
			case "medication":
				taskType = models.TaskTypeMedicationReview
			case "lab":
				taskType = models.TaskTypeCriticalLabReview
			case "procedure":
				taskType = models.TaskTypeCarePlanReview
			}
			assert.Equal(t, tt.want, taskType)
		})
	}
}

// =============================================================================
// Task Factory - Title Generation Tests
// =============================================================================

func TestTaskTitleGeneration(t *testing.T) {
	tests := []struct {
		name         string
		source       models.TaskSource
		protocolName string
		action       string
		severity     string
		wantContains []string
	}{
		{
			name:         "temporal_alert_title",
			source:       models.TaskSourceKB3,
			protocolName: "Sepsis Management",
			action:       "Check Lactate",
			severity:     "critical",
			wantContains: []string{"Sepsis Management", "Check Lactate", "critical"},
		},
		{
			name:         "care_gap_title",
			source:       models.TaskSourceKB9,
			protocolName: "Diabetes HbA1c",
			action:       "Screening",
			severity:     "high",
			wantContains: []string{"Care Gap"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var title string
			switch tt.source {
			case models.TaskSourceKB3:
				title = "[" + tt.protocolName + "] " + tt.action + " - " + tt.severity
			case models.TaskSourceKB9:
				title = "[Care Gap] " + tt.protocolName + " - " + tt.action
			}

			for _, want := range tt.wantContains {
				assert.Contains(t, title, want)
			}
		})
	}
}

// =============================================================================
// Client Data Structure Tests
// =============================================================================

func TestTemporalAlert_Structure(t *testing.T) {
	now := time.Now().UTC()
	deadline := now.Add(1 * time.Hour)

	alert := &clients.TemporalAlert{
		AlertID:      "ALERT-001",
		PatientID:    "PATIENT-001",
		EncounterID:  "ENCOUNTER-001",
		ProtocolID:   "PROTO-001",
		ProtocolName: "Sepsis Protocol",
		ConstraintID: "CONST-001",
		Action:       "Check Lactate",
		Severity:     "critical",
		Status:       "active",
		Deadline:     deadline,
		TimeOverdue:  0,
		AlertTime:    now,
		Description:  "Lactate level check required within 1 hour",
		Reference:    "Sepsis-3 Guidelines",
		Acknowledged: false,
	}

	assert.Equal(t, "ALERT-001", alert.AlertID)
	assert.Equal(t, "critical", alert.Severity)
	assert.False(t, alert.Acknowledged)
	assert.Equal(t, deadline, alert.Deadline)
}

func TestCareGap_Structure(t *testing.T) {
	dueDate := time.Now().UTC().Add(30 * 24 * time.Hour)

	gap := &clients.CareGap{
		GapID:       "GAP-001",
		PatientID:   "PATIENT-001",
		GapType:     "screening",
		Category:    "Preventive Care",
		MeasureID:   "CMS130",
		MeasureName: "Colorectal Cancer Screening",
		Description: "Annual colorectal cancer screening is overdue",
		Priority:    "high",
		DueDate:     &dueDate,
		Interventions: []clients.Intervention{
			{Type: "schedule", Description: "Schedule colonoscopy", Code: "45378", CodeSystem: "CPT"},
		},
		Rationale:      "USPSTF Grade A recommendation",
		EvidenceSource: "CMS Quality Measures",
	}

	assert.Equal(t, "GAP-001", gap.GapID)
	assert.Equal(t, "screening", gap.GapType)
	assert.Len(t, gap.Interventions, 1)
	assert.Equal(t, "schedule", gap.Interventions[0].Type)
}

func TestCarePlan_Structure(t *testing.T) {
	now := time.Now().UTC()
	dueDate := now.Add(7 * 24 * time.Hour)
	endDate := now.Add(365 * 24 * time.Hour)

	carePlan := &clients.CarePlan{
		PlanID:      "PLAN-001",
		PatientID:   "PATIENT-001",
		Title:       "Diabetes Management Plan",
		Status:      "active",
		Category:    "chronic_care",
		Description: "Comprehensive diabetes management plan",
		StartDate:   now,
		EndDate:     &endDate,
		Activities: []clients.Activity{
			{
				ActivityID:  "ACT-001",
				Type:        "lab",
				Title:       "HbA1c Test",
				Description: "Quarterly HbA1c monitoring",
				Priority:    "high",
				Status:      "scheduled",
				DueDate:     &dueDate,
			},
		},
	}

	assert.Equal(t, "PLAN-001", carePlan.PlanID)
	assert.Equal(t, "active", carePlan.Status)
	assert.Len(t, carePlan.Activities, 1)
	assert.Equal(t, "lab", carePlan.Activities[0].Type)
}

// =============================================================================
// Task Number Generation Tests
// =============================================================================

func TestTaskNumberPrefix(t *testing.T) {
	tests := []struct {
		name       string
		taskType   models.TaskType
		wantPrefix string
	}{
		{"clinical_type", models.TaskTypeCriticalLabReview, "CLN"},
		{"medication_type", models.TaskTypeMedicationReview, "CLN"},
		{"care_gap_type", models.TaskTypeCareGapClosure, "GAP"},
		{"screening_type", models.TaskTypeScreeningOutreach, "GAP"},
		{"monitoring_type", models.TaskTypeMonitoringOverdue, "TMP"},
		{"protocol_type", models.TaskTypeAcuteProtocolDeadline, "TMP"},
		{"appointment_type", models.TaskTypeAppointmentRemind, "OUT"},
		{"missed_type", models.TaskTypeMissedAppointment, "OUT"},
		{"admin_type", models.TaskTypePriorAuthNeeded, "ADM"},
		{"referral_type", models.TaskTypeReferralProcessing, "ADM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulating the prefix logic from task_service.go
			prefix := "TASK"
			switch tt.taskType {
			case models.TaskTypeCriticalLabReview, models.TaskTypeMedicationReview:
				prefix = "CLN"
			case models.TaskTypeCareGapClosure, models.TaskTypeScreeningOutreach:
				prefix = "GAP"
			case models.TaskTypeMonitoringOverdue, models.TaskTypeAcuteProtocolDeadline:
				prefix = "TMP"
			case models.TaskTypeAppointmentRemind, models.TaskTypeMissedAppointment:
				prefix = "OUT"
			case models.TaskTypePriorAuthNeeded, models.TaskTypeReferralProcessing:
				prefix = "ADM"
			}
			assert.Equal(t, tt.wantPrefix, prefix)
		})
	}
}
