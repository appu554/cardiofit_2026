package services

import (
	"testing"

	"kb-23-decision-cards/internal/models"
)

func newTestItem() *models.WorklistItem {
	return &models.WorklistItem{
		PatientID:       "patient-1",
		PAIScore:        72,
		ResolutionState: models.ResolutionPending,
	}
}

func TestResolution_Acknowledge_Resolved(t *testing.T) {
	item := newTestItem()
	req := models.WorklistActionRequest{
		PatientID:   "patient-1",
		ClinicianID: "doc-1",
		ActionCode:  "ACKNOWLEDGE",
	}

	result := HandleWorklistAction(item, req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.UpdatedItem.ResolutionState != models.ResolutionResolved {
		t.Errorf("expected RESOLVED, got %s", result.UpdatedItem.ResolutionState)
	}
}

func TestResolution_Defer_24h(t *testing.T) {
	item := newTestItem()
	req := models.WorklistActionRequest{
		PatientID:   "patient-1",
		ClinicianID: "doc-1",
		ActionCode:  "DEFER",
		DeferHours:  24,
	}

	result := HandleWorklistAction(item, req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.UpdatedItem.ResolutionState != models.ResolutionDeferred {
		t.Errorf("expected DEFERRED, got %s", result.UpdatedItem.ResolutionState)
	}
}

func TestResolution_Dismiss_CreatesFeedback(t *testing.T) {
	item := newTestItem()
	req := models.WorklistActionRequest{
		PatientID:   "patient-1",
		ClinicianID: "doc-1",
		ActionCode:  "DISMISS",
		Notes:       "not clinically relevant",
	}

	result := HandleWorklistAction(item, req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.UpdatedItem.ResolutionState != models.ResolutionResolved {
		t.Errorf("expected RESOLVED, got %s", result.UpdatedItem.ResolutionState)
	}
	if result.Feedback == nil {
		t.Fatal("expected feedback to be created for DISMISS")
	}
	if result.Feedback.FeedbackType != "NOT_USEFUL" {
		t.Errorf("expected feedback type NOT_USEFUL, got %s", result.Feedback.FeedbackType)
	}
	if result.Feedback.PatientID != "patient-1" {
		t.Errorf("expected patient-1, got %s", result.Feedback.PatientID)
	}
	if result.Feedback.ClinicianID != "doc-1" {
		t.Errorf("expected doc-1, got %s", result.Feedback.ClinicianID)
	}
	if result.Feedback.SubmittedAt.IsZero() {
		t.Error("expected SubmittedAt to be set")
	}
}

func TestResolution_CallPatient_InProgress(t *testing.T) {
	item := newTestItem()
	req := models.WorklistActionRequest{
		PatientID:   "patient-1",
		ClinicianID: "doc-1",
		ActionCode:  "CALL_PATIENT",
	}

	result := HandleWorklistAction(item, req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.UpdatedItem.ResolutionState != models.ResolutionInProgress {
		t.Errorf("expected IN_PROGRESS, got %s", result.UpdatedItem.ResolutionState)
	}
}

func TestResolution_EscalateToGP_Escalated(t *testing.T) {
	item := newTestItem()
	req := models.WorklistActionRequest{
		PatientID:   "patient-1",
		ClinicianID: "nurse-1",
		ActionCode:  "ESCALATE_TO_GP",
	}

	result := HandleWorklistAction(item, req)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.UpdatedItem.ResolutionState != models.ResolutionEscalated {
		t.Errorf("expected ESCALATED, got %s", result.UpdatedItem.ResolutionState)
	}
}

func TestResolution_UnknownAction_Error(t *testing.T) {
	item := newTestItem()
	req := models.WorklistActionRequest{
		PatientID:   "patient-1",
		ClinicianID: "doc-1",
		ActionCode:  "DANCE_PARTY",
	}

	result := HandleWorklistAction(item, req)

	if result.Error == nil {
		t.Fatal("expected error for unknown action")
	}
}
