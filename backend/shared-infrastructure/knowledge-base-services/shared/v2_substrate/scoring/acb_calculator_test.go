package scoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestComputeACB_NoMedicationsReturnsZero(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	got, err := ComputeACB(context.Background(), nil, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeACB: %v", err)
	}
	if got.Score != 0 {
		t.Errorf("Score: got %d want 0", got.Score)
	}
}

func TestComputeACB_AllUnknownDrugsScoreZero(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	meds := []models.MedicineUse{
		med("UnlistedAgent", models.MedicineUseStatusActive),
	}
	got, err := ComputeACB(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeACB: %v", err)
	}
	if got.Score != 0 {
		t.Errorf("Score: got %d want 0", got.Score)
	}
	if len(got.UnknownDrugs) != 1 {
		t.Errorf("expected 1 unknown; got %d", len(got.UnknownDrugs))
	}
}

func TestComputeACB_KnownMedicationsSumExpected(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	// amitriptyline=3 + temazepam=1 + diphenhydramine=3 = 7
	meds := []models.MedicineUse{
		med("amitriptyline", models.MedicineUseStatusActive),
		med("temazepam", models.MedicineUseStatusActive),
		med("diphenhydramine", models.MedicineUseStatusActive),
	}
	got, err := ComputeACB(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeACB: %v", err)
	}
	if got.Score != 7 {
		t.Errorf("Score: got %d want 7", got.Score)
	}
	if len(got.ComputationInputs) != 3 {
		t.Errorf("expected 3 inputs; got %d", len(got.ComputationInputs))
	}
}

func TestComputeACB_OnlyActiveCounted(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	meds := []models.MedicineUse{
		med("amitriptyline", models.MedicineUseStatusActive), // 3
		med("temazepam", models.MedicineUseStatusCeased),     // ignored
		med("oxybutynin", models.MedicineUseStatusPaused),    // ignored
	}
	got, err := ComputeACB(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeACB: %v", err)
	}
	if got.Score != 3 {
		t.Errorf("Score: got %d want 3 (only active counted)", got.Score)
	}
}
