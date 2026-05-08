package consent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// recommendationTypeToConsentClass is a test mapper. Production wires a
// real mapping table; this matches the test's specific recType values.
func recommendationTypeToConsentClass(recType string) (string, bool) {
	switch recType {
	case "psychotropic_stop", "psychotropic_dose", "psychotropic_add":
		return models.ConsentClassPsychotropic, true
	case "chemo_stop", "chemo_dose", "chemo_add":
		return models.ConsentClassChemotherapy, true
	}
	return "", false
}

func TestPostgresConsentChecker_ActiveConsentSatisfies(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	checker := NewPostgresConsentChecker(store, recommendationTypeToConsentClass)
	ctx := context.Background()

	resident := uuid.New()
	active := models.Consent{
		ID:            uuid.New(),
		ResidentID:    resident,
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "substitute_decision_maker",
		ValidFrom:     time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &active); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE resident_id = $1", resident)
	})

	ok, err := checker.ConsentActive(ctx, resident, "psychotropic_stop")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !ok {
		t.Errorf("expected active consent to satisfy check; got false")
	}
}

func TestPostgresConsentChecker_NoConsentReturnsFalse(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	checker := NewPostgresConsentChecker(store, recommendationTypeToConsentClass)
	ctx := context.Background()

	// No consent seeded.
	ok, err := checker.ConsentActive(ctx, uuid.New(), "psychotropic_stop")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if ok {
		t.Errorf("expected no-consent case to return false; got true")
	}
}

func TestPostgresConsentChecker_NonGatedRecTypePasses(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	checker := NewPostgresConsentChecker(store, recommendationTypeToConsentClass)
	ctx := context.Background()

	// "general_optimisation" is NOT mapped → mapper returns ("", false) →
	// no consent required → checker returns (true, nil) without DB call.
	ok, err := checker.ConsentActive(ctx, uuid.New(), "general_optimisation")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !ok {
		t.Errorf("expected non-gated recType to pass without consent; got false")
	}
}

func TestPostgresConsentChecker_WrongClassFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	checker := NewPostgresConsentChecker(store, recommendationTypeToConsentClass)
	ctx := context.Background()

	resident := uuid.New()
	psychoActive := models.Consent{
		ID:            uuid.New(),
		ResidentID:    resident,
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "sdm",
		ValidFrom:     time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &psychoActive); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE resident_id = $1", resident)
	})

	// chemo_stop maps to ConsentClassChemotherapy → no chemo consent exists
	// → checker returns (false, nil) even though psychotropic consent does.
	ok, _ := checker.ConsentActive(ctx, resident, "chemo_stop")
	if ok {
		t.Errorf("expected chemo recType to fail despite psychotropic consent; got true")
	}
}
