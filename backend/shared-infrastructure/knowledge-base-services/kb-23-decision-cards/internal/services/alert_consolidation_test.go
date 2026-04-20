package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func TestConsolidate_InterventionTaken_ActionTypePresent(t *testing.T) {
	lc := models.DetectionLifecycle{
		ID: uuid.New(), PatientID: "P100", CohortID: "hcf_catalyst_chf",
		DetectedAt: time.Now().Add(-40 * 24 * time.Hour),
		ActionType: "nurse_phone_followup",
	}
	t1 := lc.DetectedAt.Add(5 * time.Minute); lc.DeliveredAt = &t1
	t2 := lc.DetectedAt.Add(30 * time.Minute); lc.AcknowledgedAt = &t2
	t3 := lc.DetectedAt.Add(2 * time.Hour); lc.ActionedAt = &t3

	outcomeID := uuid.New()
	occurred := false
	outcome := models.OutcomeRecord{
		ID: outcomeID, PatientID: "P100", OutcomeType: "READMISSION_30D",
		OutcomeOccurred: occurred,
		Reconciliation: string(models.ReconciliationResolved),
	}
	riskScore := 62.0
	record, err := BuildConsolidatedRecord(lc, &outcome, riskScore, "HIGH", "gap20-heuristic-v1", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.TreatmentStrategy != string(models.TreatmentInterventionTaken) {
		t.Fatalf("expected INTERVENTION_TAKEN, got %s", record.TreatmentStrategy)
	}
	if record.ActionType != "nurse_phone_followup" {
		t.Fatalf("expected action_type preserved")
	}
	if record.TimeZero != lc.DetectedAt {
		t.Fatalf("expected time_zero = T0 (DetectedAt)")
	}
	if record.HorizonDays != 30 {
		t.Fatalf("expected horizon_days=30, got %d", record.HorizonDays)
	}
}

func TestConsolidate_Override_OverrideReasonPreserved(t *testing.T) {
	lc := models.DetectionLifecycle{
		ID: uuid.New(), PatientID: "P101",
		DetectedAt: time.Now().Add(-40 * 24 * time.Hour),
		ActionType: "OVERRIDE",
		ActionDetail: "reason=already_addressed",
	}
	record, err := BuildConsolidatedRecord(lc, nil, 55.0, "HIGH", "gap20-heuristic-v1", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.TreatmentStrategy != string(models.TreatmentAlreadyAddressed) {
		t.Fatalf("expected ALREADY_ADDRESSED, got %s", record.TreatmentStrategy)
	}
	if record.OverrideReason != "already_addressed" {
		t.Fatalf("expected override_reason=already_addressed, got %s", record.OverrideReason)
	}
}

func TestConsolidate_NoResponse_HorizonClosure(t *testing.T) {
	lc := models.DetectionLifecycle{
		ID: uuid.New(), PatientID: "P102",
		DetectedAt: time.Now().Add(-45 * 24 * time.Hour),
		// no DeliveredAt / AcknowledgedAt / ActionedAt
	}
	record, err := BuildConsolidatedRecord(lc, nil, 48.0, "MODERATE", "gap20-heuristic-v1", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.TreatmentStrategy != string(models.TreatmentNoResponse) {
		t.Fatalf("expected NO_RESPONSE, got %s", record.TreatmentStrategy)
	}
}
