package scoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// fixtureLookup returns a small static lookup covering the canonical
// Hilmer 2007 weights for the test fixtures used in this file.
func fixtureLookup() *StaticDrugWeightLookup {
	return NewStaticDrugWeightLookup(map[string]DrugWeight{
		"amitriptyline":   {DrugName: "amitriptyline", AnticholinergicWeight: 0.5, SedativeWeight: 0.5, ACBWeight: 3},
		"oxybutynin":      {DrugName: "oxybutynin", AnticholinergicWeight: 0.5, SedativeWeight: 0.0, ACBWeight: 3},
		"temazepam":       {DrugName: "temazepam", AnticholinergicWeight: 0.0, SedativeWeight: 0.5, ACBWeight: 1},
		"diphenhydramine": {DrugName: "diphenhydramine", AnticholinergicWeight: 0.5, SedativeWeight: 0.5, ACBWeight: 3},
	})
}

func med(displayName, status string) models.MedicineUse {
	return models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  uuid.New(),
		DisplayName: displayName,
		Status:      status,
		StartedAt:   time.Now().UTC(),
	}
}

func TestComputeDBI_NoMedicationsReturnsZero(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	got, err := ComputeDBI(context.Background(), nil, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeDBI: %v", err)
	}
	if got.Score != 0 {
		t.Errorf("Score: got %v want 0", got.Score)
	}
	if got.AnticholinergicComponent != 0 || got.SedativeComponent != 0 {
		t.Errorf("components nonzero: %+v", got)
	}
	if len(got.ComputationInputs) != 0 {
		t.Errorf("expected no inputs; got %d", len(got.ComputationInputs))
	}
	if len(got.UnknownDrugs) != 0 {
		t.Errorf("expected no unknowns; got %d", len(got.UnknownDrugs))
	}
	if got.ResidentRef != rid {
		t.Errorf("ResidentRef drift")
	}
	if !got.ComputedAt.Equal(now) {
		t.Errorf("ComputedAt drift")
	}
}

func TestComputeDBI_AllUnknownDrugsScoreZero(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	meds := []models.MedicineUse{
		med("ExperimentalAgentX", models.MedicineUseStatusActive),
		med("UnlistedDrugY", models.MedicineUseStatusActive),
	}
	got, err := ComputeDBI(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeDBI: %v", err)
	}
	if got.Score != 0 {
		t.Errorf("Score: got %v want 0", got.Score)
	}
	if len(got.UnknownDrugs) != 2 {
		t.Errorf("expected 2 unknowns; got %d (%v)", len(got.UnknownDrugs), got.UnknownDrugs)
	}
	if len(got.ComputationInputs) != 0 {
		t.Errorf("expected no inputs; got %d", len(got.ComputationInputs))
	}
}

func TestComputeDBI_KnownMedicationsSumExpected(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	// amitriptyline (0.5 ach + 0.5 sed) + oxybutynin (0.5 ach + 0 sed)
	// + temazepam (0 ach + 0.5 sed) = ach 1.0 + sed 1.0 = 2.0
	meds := []models.MedicineUse{
		med("Amitriptyline 25mg", models.MedicineUseStatusActive),
		med("Oxybutynin 5mg", models.MedicineUseStatusActive),
		med("Temazepam 10mg", models.MedicineUseStatusActive),
	}
	got, err := ComputeDBI(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeDBI: %v", err)
	}
	if got.AnticholinergicComponent != 1.0 {
		t.Errorf("Ach: got %v want 1.0", got.AnticholinergicComponent)
	}
	if got.SedativeComponent != 1.0 {
		t.Errorf("Sed: got %v want 1.0", got.SedativeComponent)
	}
	if got.Score != 2.0 {
		t.Errorf("Score: got %v want 2.0", got.Score)
	}
	if len(got.ComputationInputs) != 3 {
		t.Errorf("inputs: got %d want 3", len(got.ComputationInputs))
	}
	if len(got.UnknownDrugs) != 0 {
		t.Errorf("unknowns: got %d want 0", len(got.UnknownDrugs))
	}
}

func TestComputeDBI_OnlyActiveCounted(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	meds := []models.MedicineUse{
		med("amitriptyline", models.MedicineUseStatusActive),  // 0.5+0.5 = 1.0
		med("oxybutynin", models.MedicineUseStatusCeased),     // ignored
		med("temazepam", models.MedicineUseStatusCompleted),   // ignored
		med("diphenhydramine", models.MedicineUseStatusPaused), // ignored
	}
	got, err := ComputeDBI(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeDBI: %v", err)
	}
	if got.Score != 1.0 {
		t.Errorf("Score: got %v want 1.0 (only active counted)", got.Score)
	}
	if len(got.ComputationInputs) != 1 {
		t.Errorf("expected 1 active contribution; got %d", len(got.ComputationInputs))
	}
}

func TestComputeDBI_CaseInsensitivePrefixMatch(t *testing.T) {
	rid := uuid.New()
	now := time.Now().UTC()
	meds := []models.MedicineUse{
		med("AMITRIPTYLINE 25 MG ORAL", models.MedicineUseStatusActive),
	}
	got, err := ComputeDBI(context.Background(), meds, fixtureLookup(), rid, now)
	if err != nil {
		t.Fatalf("ComputeDBI: %v", err)
	}
	if got.Score != 1.0 {
		t.Errorf("Score: got %v want 1.0; case-insensitive prefix match should fire", got.Score)
	}
}
