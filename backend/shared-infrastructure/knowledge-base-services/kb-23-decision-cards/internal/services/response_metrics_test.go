package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func ptr64(v int64) *int64   { return &v }
func ptrTime(t time.Time) *time.Time { return &t }

func makeLifecycle(clinicianID, patientID string) models.DetectionLifecycle {
	return models.DetectionLifecycle{
		ID:                  uuid.New(),
		DetectionType:       "TEST",
		PatientID:           patientID,
		AssignedClinicianID: clinicianID,
		CurrentState:        string(models.LifecycleNotified),
		TierAtDetection:     "STANDARD",
		DetectedAt:          time.Now().Add(-24 * time.Hour),
	}
}

// TestMetrics_ClinicianMedians — 10 lifecycles with AcknowledgmentLatencyMs
// [100000..1000000]. Median of 10 even values = avg(5th, 6th) = (500000+600000)/2 = 550000.
func TestMetrics_ClinicianMedians(t *testing.T) {
	svc := NewResponseMetricsService()

	var lifecycles []models.DetectionLifecycle
	for i := 1; i <= 10; i++ {
		lc := makeLifecycle("doc-1", "patient-1")
		val := int64(i * 100000)
		lc.AcknowledgmentLatencyMs = &val
		lifecycles = append(lifecycles, lc)
	}

	result := svc.ComputeClinicianMetrics(lifecycles, "doc-1", 30)

	if result.TotalDetections != 10 {
		t.Fatalf("expected 10 detections, got %d", result.TotalDetections)
	}
	if result.MedianAcknowledgmentMs == nil {
		t.Fatal("expected non-nil MedianAcknowledgmentMs")
	}
	if *result.MedianAcknowledgmentMs != 550000 {
		t.Fatalf("expected median 550000, got %d", *result.MedianAcknowledgmentMs)
	}
}

// TestMetrics_ActionCompletionRate — 10 lifecycles, 8 have ActionedAt → rate 0.80
func TestMetrics_ActionCompletionRate(t *testing.T) {
	svc := NewResponseMetricsService()

	var lifecycles []models.DetectionLifecycle
	now := time.Now()
	for i := 0; i < 10; i++ {
		lc := makeLifecycle("doc-2", "patient-2")
		if i < 8 {
			lc.ActionedAt = ptrTime(now)
		}
		lifecycles = append(lifecycles, lc)
	}

	result := svc.ComputeClinicianMetrics(lifecycles, "doc-2", 30)

	if result.ActionCompletionRate != 0.8 {
		t.Fatalf("expected action completion rate 0.80, got %.4f", result.ActionCompletionRate)
	}
}

// TestMetrics_OutcomeRate — 8 actioned, 5 have ResolvedAt → rate 5/8 = 0.625
func TestMetrics_OutcomeRate(t *testing.T) {
	svc := NewResponseMetricsService()

	var lifecycles []models.DetectionLifecycle
	now := time.Now()
	for i := 0; i < 10; i++ {
		lc := makeLifecycle("doc-3", "patient-3")
		if i < 8 {
			lc.ActionedAt = ptrTime(now)
			if i < 5 {
				lc.ResolvedAt = ptrTime(now)
			}
		}
		lifecycles = append(lifecycles, lc)
	}

	result := svc.ComputeClinicianMetrics(lifecycles, "doc-3", 30)

	if result.OutcomeRate != 0.625 {
		t.Fatalf("expected outcome rate 0.625, got %.4f", result.OutcomeRate)
	}
}

// TestMetrics_SystemLevel — 30 lifecycles across 3 clinicians, 5 timed out → TimeoutRate 5/30
func TestMetrics_SystemLevel(t *testing.T) {
	svc := NewResponseMetricsService()

	clinicians := []string{"doc-a", "doc-b", "doc-c"}
	var lifecycles []models.DetectionLifecycle
	for i := 0; i < 30; i++ {
		cID := clinicians[i%3]
		lc := makeLifecycle(cID, "patient-sys")
		if i < 5 {
			lc.CurrentState = string(models.LifecycleTimedOut)
		}
		lifecycles = append(lifecycles, lc)
	}

	result := svc.ComputeSystemMetrics(lifecycles, 30)

	if result.TotalDetections != 30 {
		t.Fatalf("expected 30 detections, got %d", result.TotalDetections)
	}
	// 5/30 = 0.16666... rounded to 3 decimals = 0.167
	expectedTimeout := 0.167
	if result.TimeoutRate != expectedTimeout {
		t.Fatalf("expected timeout rate %.3f, got %.3f", expectedTimeout, result.TimeoutRate)
	}
}

// TestMetrics_PilotKPIs — 20 lifecycles with various action types
func TestMetrics_PilotKPIs(t *testing.T) {
	svc := NewResponseMetricsService()

	actionTypes := []string{
		// 8 CALL_PATIENT → OutreachCalls
		"CALL_PATIENT", "CALL_PATIENT", "CALL_PATIENT", "CALL_PATIENT",
		"CALL_PATIENT", "CALL_PATIENT", "CALL_PATIENT", "CALL_PATIENT",
		// 5 MEDICATION_REVIEW → MedicationChanges
		"MEDICATION_REVIEW", "MEDICATION_REVIEW", "MEDICATION_REVIEW",
		"MEDICATION_REVIEW", "MEDICATION_REVIEW",
		// 3 SCHEDULE_APPOINTMENT → AppointmentsScheduled
		"SCHEDULE_APPOINTMENT", "SCHEDULE_APPOINTMENT", "SCHEDULE_APPOINTMENT",
		// 4 with no matching action type
		"OTHER", "OTHER", "OTHER", "OTHER",
	}

	now := time.Now()
	var lifecycles []models.DetectionLifecycle
	for i := 0; i < 20; i++ {
		lc := makeLifecycle("doc-pilot", "patient-pilot")
		lc.ActionType = actionTypes[i]
		lc.ActionedAt = ptrTime(now)
		lifecycles = append(lifecycles, lc)
	}

	result := svc.ComputePilotMetrics(lifecycles)

	if result.TotalDetections != 20 {
		t.Fatalf("expected 20 total detections, got %d", result.TotalDetections)
	}
	if result.OutreachCalls != 8 {
		t.Fatalf("expected 8 outreach calls, got %d", result.OutreachCalls)
	}
	if result.MedicationChanges != 5 {
		t.Fatalf("expected 5 medication changes, got %d", result.MedicationChanges)
	}
	if result.AppointmentsScheduled != 3 {
		t.Fatalf("expected 3 appointments scheduled, got %d", result.AppointmentsScheduled)
	}
}
