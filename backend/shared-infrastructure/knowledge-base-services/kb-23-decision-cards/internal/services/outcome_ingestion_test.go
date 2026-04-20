package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func TestOutcomeIngest_SingleSource_AutoResolves(t *testing.T) {
	lifecycleID := uuid.New()
	records := []models.OutcomeRecord{
		{
			PatientID:       "P001",
			LifecycleID:     &lifecycleID,
			CohortID:        "hcf_catalyst_chf",
			OutcomeType:     "READMISSION_30D",
			OutcomeOccurred: true,
			Source:          string(models.OutcomeSourceHospitalDischarge),
			IngestedAt:      time.Now(),
		},
	}
	result, err := ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("expected RESOLVED, got %s", result.Reconciliation)
	}
	if !result.OutcomeOccurred {
		t.Fatalf("expected outcome_occurred=true")
	}
}

func TestOutcomeIngest_MultipleSourcesAgree_Resolves(t *testing.T) {
	lifecycleID := uuid.New()
	occurredAt := time.Now().Add(-10 * 24 * time.Hour)
	records := []models.OutcomeRecord{
		{
			PatientID: "P002", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: true, OccurredAt: &occurredAt,
			Source: string(models.OutcomeSourceHospitalDischarge),
		},
		{
			PatientID: "P002", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: true, OccurredAt: &occurredAt,
			Source: string(models.OutcomeSourceClaimsFeed),
		},
	}
	result, err := ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("expected RESOLVED, got %s", result.Reconciliation)
	}
}

func TestOutcomeIngest_MultipleSourcesDisagree_Conflicts(t *testing.T) {
	lifecycleID := uuid.New()
	records := []models.OutcomeRecord{
		{
			PatientID: "P003", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: true,
			Source: string(models.OutcomeSourceHospitalDischarge),
		},
		{
			PatientID: "P003", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: false,
			Source: string(models.OutcomeSourceClaimsFeed),
		},
	}
	result, err := ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reconciliation != string(models.ReconciliationConflicted) {
		t.Fatalf("expected CONFLICTED, got %s", result.Reconciliation)
	}
}
