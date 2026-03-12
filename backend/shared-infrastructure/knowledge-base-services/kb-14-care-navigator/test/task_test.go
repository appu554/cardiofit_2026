// Package test contains tests for KB-14 Care Navigator
package test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-14-care-navigator/internal/models"
)

// =============================================================================
// Task Model Tests
// =============================================================================

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status models.TaskStatus
		want   bool
	}{
		{"created", models.TaskStatusCreated, true},
		{"assigned", models.TaskStatusAssigned, true},
		{"in_progress", models.TaskStatusInProgress, true},
		{"completed", models.TaskStatusCompleted, true},
		{"verified", models.TaskStatusVerified, true},
		{"declined", models.TaskStatusDeclined, true},
		{"blocked", models.TaskStatusBlocked, true},
		{"escalated", models.TaskStatusEscalated, true},
		{"cancelled", models.TaskStatusCancelled, true},
		{"invalid", models.TaskStatus("INVALID"), false},
		{"empty", models.TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTaskType_GetDefaultRole(t *testing.T) {
	tests := []struct {
		name     string
		taskType models.TaskType
		want     string
	}{
		{"critical_lab_review", models.TaskTypeCriticalLabReview, "Physician"},
		{"medication_review", models.TaskTypeMedicationReview, "Pharmacist"},
		{"abnormal_result", models.TaskTypeAbnormalResult, "Ordering MD"},
		{"care_plan_review", models.TaskTypeCarePlanReview, "PCP"},
		{"acute_protocol_deadline", models.TaskTypeAcuteProtocolDeadline, "Attending"},
		{"care_gap_closure", models.TaskTypeCareGapClosure, "Care Coordinator"},
		{"monitoring_overdue", models.TaskTypeMonitoringOverdue, "Care Coordinator"},
		{"transition_followup", models.TaskTypeTransitionFollowup, "Transition Coordinator"},
		{"annual_wellness", models.TaskTypeAnnualWellness, "Nurse"},
		{"chronic_care_mgmt", models.TaskTypeChronicCareMgmt, "Care Manager"},
		{"appointment_remind", models.TaskTypeAppointmentRemind, "Scheduler"},
		{"missed_appointment", models.TaskTypeMissedAppointment, "Outreach Specialist"},
		{"screening_outreach", models.TaskTypeScreeningOutreach, "Outreach Specialist"},
		{"medication_refill", models.TaskTypeMedicationRefill, "Outreach Specialist"},
		{"prior_auth_needed", models.TaskTypePriorAuthNeeded, "Auth Specialist"},
		{"referral_processing", models.TaskTypeReferralProcessing, "Referral Coordinator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.taskType.GetDefaultRole()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTaskType_GetDefaultSLAMinutes(t *testing.T) {
	tests := []struct {
		name     string
		taskType models.TaskType
		want     int
	}{
		{"critical_lab_review", models.TaskTypeCriticalLabReview, 60},       // 1 hour
		{"acute_protocol_deadline", models.TaskTypeAcuteProtocolDeadline, 60}, // 1 hour
		{"medication_review", models.TaskTypeMedicationReview, 240},           // 4 hours
		{"abnormal_result", models.TaskTypeAbnormalResult, 1440},              // 24 hours
		{"therapeutic_change", models.TaskTypeTherapeuticChange, 2880},        // 48 hours
		{"referral_processing", models.TaskTypeReferralProcessing, 7200},      // 5 days
		{"care_gap_closure", models.TaskTypeCareGapClosure, 43200},            // 30 days
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.taskType.GetDefaultSLAMinutes()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTaskType_GetDefaultPriority(t *testing.T) {
	tests := []struct {
		name     string
		taskType models.TaskType
		want     models.TaskPriority
	}{
		{"critical_lab_review", models.TaskTypeCriticalLabReview, models.TaskPriorityCritical},
		{"acute_protocol_deadline", models.TaskTypeAcuteProtocolDeadline, models.TaskPriorityCritical},
		{"medication_review", models.TaskTypeMedicationReview, models.TaskPriorityHigh},
		{"abnormal_result", models.TaskTypeAbnormalResult, models.TaskPriorityHigh},
		{"therapeutic_change", models.TaskTypeTherapeuticChange, models.TaskPriorityMedium},
		{"care_gap_closure", models.TaskTypeCareGapClosure, models.TaskPriorityMedium},
		{"annual_wellness", models.TaskTypeAnnualWellness, models.TaskPriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.taskType.GetDefaultPriority()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTask_IsOverdue(t *testing.T) {
	now := time.Now().UTC()
	pastDue := now.Add(-1 * time.Hour)
	futureDue := now.Add(1 * time.Hour)

	tests := []struct {
		name    string
		dueDate *time.Time
		want    bool
	}{
		{"no_due_date", nil, false},
		{"past_due", &pastDue, true},
		{"future_due", &futureDue, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{DueDate: tt.dueDate}
			got := task.IsOverdue()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTask_IsDueSoon(t *testing.T) {
	now := time.Now().UTC()
	dueIn30Min := now.Add(30 * time.Minute)
	dueIn2Hours := now.Add(2 * time.Hour)
	pastDue := now.Add(-1 * time.Hour)

	tests := []struct {
		name       string
		dueDate    *time.Time
		hoursAhead int
		want       bool
	}{
		{"no_due_date", nil, 1, false},
		{"due_in_30min_check_1hr", &dueIn30Min, 1, true},
		{"due_in_2hrs_check_1hr", &dueIn2Hours, 1, false},
		{"past_due", &pastDue, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{DueDate: tt.dueDate}
			got := task.IsDueSoon(tt.hoursAhead)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTask_GetSLAElapsedPercent(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name       string
		createdAt  time.Time
		slaMinutes int
		wantMin    float64
		wantMax    float64
	}{
		{"no_sla", now, 0, 0, 0},
		{"fresh_task", now, 60, 0, 5},
		{"50_percent_elapsed", now.Add(-30 * time.Minute), 60, 45, 55},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &models.Task{
				CreatedAt:  tt.createdAt,
				SLAMinutes: tt.slaMinutes,
			}
			got := task.GetSLAElapsedPercent()
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}

// =============================================================================
// Task Action Tests
// =============================================================================

func TestActionSlice_ValueAndScan(t *testing.T) {
	actions := models.ActionSlice{
		{
			ActionID:    "action-1",
			Type:        "review",
			Description: "Review lab results",
			Required:    true,
			Completed:   false,
		},
		{
			ActionID:    "action-2",
			Type:        "order",
			Description: "Order follow-up test",
			Required:    false,
			Completed:   true,
		},
	}

	// Test Value()
	value, err := actions.Value()
	require.NoError(t, err)
	require.NotNil(t, value)

	// Test Scan()
	var scanned models.ActionSlice
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Len(t, scanned, 2)
	assert.Equal(t, "action-1", scanned[0].ActionID)
	assert.Equal(t, "review", scanned[0].Type)
	assert.True(t, scanned[0].Required)
	assert.False(t, scanned[0].Completed)
}

func TestActionSlice_ScanNil(t *testing.T) {
	var actions models.ActionSlice
	err := actions.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, actions)
}

func TestNoteSlice_ValueAndScan(t *testing.T) {
	now := time.Now().UTC()
	notes := models.NoteSlice{
		{
			NoteID:    "note-1",
			Author:    "Dr. Smith",
			AuthorID:  uuid.NewString(),
			Content:   "Patient condition improved",
			CreatedAt: now,
		},
	}

	// Test Value()
	value, err := notes.Value()
	require.NoError(t, err)
	require.NotNil(t, value)

	// Test Scan()
	var scanned models.NoteSlice
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Len(t, scanned, 1)
	assert.Equal(t, "note-1", scanned[0].NoteID)
	assert.Equal(t, "Dr. Smith", scanned[0].Author)
}

func TestJSONMap_ValueAndScan(t *testing.T) {
	metadata := models.JSONMap{
		"protocol_id":   "PROTO-001",
		"protocol_name": "Sepsis Management",
		"severity":      "critical",
		"score":         42.5,
	}

	// Test Value()
	value, err := metadata.Value()
	require.NoError(t, err)
	require.NotNil(t, value)

	// Test Scan()
	var scanned models.JSONMap
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Equal(t, "PROTO-001", scanned["protocol_id"])
	assert.Equal(t, "Sepsis Management", scanned["protocol_name"])
}

// =============================================================================
// CreateTaskRequest Validation Tests
// =============================================================================

func TestCreateTaskRequest_Defaults(t *testing.T) {
	req := &models.CreateTaskRequest{
		Type:      models.TaskTypeCriticalLabReview,
		Source:    models.TaskSourceKB3,
		PatientID: "patient-123",
		Title:     "Review critical lab result",
	}

	// Test that task type provides sensible defaults
	assert.Equal(t, models.TaskPriorityCritical, req.Type.GetDefaultPriority())
	assert.Equal(t, 60, req.Type.GetDefaultSLAMinutes())
	assert.Equal(t, "Physician", req.Type.GetDefaultRole())
}

// =============================================================================
// Escalation Model Tests
// =============================================================================

func TestEscalationLevel_Thresholds(t *testing.T) {
	tests := []struct {
		name       string
		slaElapsed float64
		priority   models.TaskPriority
		want       models.EscalationLevel
	}{
		{"standard_25_percent", 0.25, models.TaskPriorityMedium, models.EscalationNone},
		{"standard_55_percent", 0.55, models.TaskPriorityMedium, models.EscalationWarning},
		{"standard_80_percent", 0.80, models.TaskPriorityMedium, models.EscalationUrgent},
		{"standard_110_percent", 1.10, models.TaskPriorityMedium, models.EscalationCritical},
		{"critical_30_percent", 0.30, models.TaskPriorityCritical, models.EscalationWarning},
		{"critical_60_percent", 0.60, models.TaskPriorityCritical, models.EscalationUrgent},
		{"critical_85_percent", 0.85, models.TaskPriorityCritical, models.EscalationCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.CalculateEscalationLevel(tt.slaElapsed, tt.priority)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// Table Name Tests
// =============================================================================

func TestTask_TableName(t *testing.T) {
	task := &models.Task{}
	assert.Equal(t, "tasks", task.TableName())
}
