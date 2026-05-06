package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/delta"
	"github.com/cardiofit/shared/v2_substrate/models"

	"kb-patient-profile/internal/storage"
)

func openIntegrationStore(t *testing.T) (*storage.V2SubstrateStore, *storage.InMemoryBaselineProvider) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping delta-on-write integration test")
	}
	store, err := storage.NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	bp := storage.NewInMemoryBaselineProvider()
	store.SetBaselineProvider(bp)
	return store, bp
}

func ptr(f float64) *float64 { return &f }

// TestDeltaOnWrite_EndToEnd seeds a baseline, inserts 4 observations with
// distinct value/kind/baseline-availability profiles, and asserts each lands
// the expected DirectionalFlag. Covers spec §9 acceptance item 10.
func TestDeltaOnWrite_EndToEnd(t *testing.T) {
	store, bp := openIntegrationStore(t)
	defer store.Close()

	rid := uuid.New()
	bp.Seed(rid, "8480-6", delta.Baseline{
		BaselineValue: 130.0,
		StdDev:        8.0,
		SampleSize:    50,
		ComputedAt:    time.Now().UTC(),
	})

	// Case 1: within baseline (val=132, dev≈0.25 stddev)
	o1, err := store.UpsertObservation(context.Background(), models.Observation{
		ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
		LOINCCode: "8480-6", Value: ptr(132.0), Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	if o1.Delta == nil || o1.Delta.DirectionalFlag != models.DeltaFlagWithinBaseline {
		t.Errorf("case 1: expected within_baseline, got %+v", o1.Delta)
	}

	// Case 2: severely elevated (val=160, dev≈3.75 stddev)
	o2, err := store.UpsertObservation(context.Background(), models.Observation{
		ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
		LOINCCode: "8480-6", Value: ptr(160.0), Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("upsert 2: %v", err)
	}
	if o2.Delta == nil || o2.Delta.DirectionalFlag != models.DeltaFlagSeverelyElevated {
		t.Errorf("case 2: expected severely_elevated, got %+v", o2.Delta)
	}

	// Case 3: behavioural — must yield no_baseline regardless of seeded data
	o3, err := store.UpsertObservation(context.Background(), models.Observation{
		ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindBehavioural,
		ValueText: "agitation episode 14:30", ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("upsert 3: %v", err)
	}
	if o3.Delta == nil || o3.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
		t.Errorf("case 3: expected no_baseline for behavioural, got %+v", o3.Delta)
	}

	// Case 4: vital with NO seeded baseline (different LOINC) — no_baseline
	o4, err := store.UpsertObservation(context.Background(), models.Observation{
		ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
		LOINCCode: "8462-4", // diastolic — not seeded
		Value: ptr(85.0), Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("upsert 4: %v", err)
	}
	if o4.Delta == nil || o4.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
		t.Errorf("case 4: expected no_baseline for unseeded LOINC, got %+v", o4.Delta)
	}
}
