package storage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/delta"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

func openTestStore(t *testing.T) (*V2SubstrateStore, *InMemoryBaselineProvider) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated observation storage test")
	}
	store, err := NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	bp := NewInMemoryBaselineProvider()
	store.SetBaselineProvider(bp)
	return store, bp
}

func ptr(f float64) *float64 { return &f }

func TestUpsertGetObservation_RoundTrip(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()

	in := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		LOINCCode:  "8480-6",
		Kind:       models.ObservationKindVital,
		Value:      ptr(132.0),
		Unit:       "mmHg",
		ObservedAt: time.Now().UTC().Truncate(time.Second),
	}
	out, err := store.UpsertObservation(context.Background(), in)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if out.Delta == nil || out.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
		t.Errorf("expected Delta.Flag=no_baseline (no provider seed), got %+v", out.Delta)
	}

	got, err := store.GetObservation(context.Background(), in.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.LOINCCode != in.LOINCCode || got.Kind != in.Kind {
		t.Errorf("round-trip drift: got %+v want %+v", got, in)
	}
	if got.Value == nil || *got.Value != *in.Value {
		t.Errorf("Value drift: got %v want %v", got.Value, in.Value)
	}
}

func TestGetObservation_NotFoundSentinel(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()
	_, err := store.GetObservation(context.Background(), uuid.New())
	if !errors.Is(err, interfaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpsertObservation_BehaviouralValueText(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()

	in := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindBehavioural,
		ValueText:  "agitation episode 14:30, paced corridor",
		ObservedAt: time.Now().UTC().Truncate(time.Second),
	}
	out, err := store.UpsertObservation(context.Background(), in)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if out.Value != nil {
		t.Errorf("Value should be nil for behavioural; got %v", *out.Value)
	}
	if out.ValueText != in.ValueText {
		t.Errorf("ValueText drift: got %q want %q", out.ValueText, in.ValueText)
	}
	if out.Delta == nil || out.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
		t.Errorf("behavioural must yield no_baseline Delta, got %+v", out.Delta)
	}
}

func TestListObservationsByResident(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()
	rid := uuid.New()
	for i := 0; i < 3; i++ {
		v := 120.0 + float64(i)
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: "8480-6", Value: &v, Unit: "mmHg",
			ObservedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour).Truncate(time.Second),
		})
		if err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}
	got, err := store.ListObservationsByResident(context.Background(), rid, 100, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 observations, got %d", len(got))
	}
}

func TestListObservationsByResidentAndKind(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()
	rid := uuid.New()
	v := 132.0
	_, _ = store.UpsertObservation(context.Background(), models.Observation{
		ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
		LOINCCode: "8480-6", Value: &v, Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	w := 78.0
	_, _ = store.UpsertObservation(context.Background(), models.Observation{
		ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindWeight,
		Value: &w, Unit: "kg", ObservedAt: time.Now().UTC().Truncate(time.Second),
	})
	got, err := store.ListObservationsByResidentAndKind(context.Background(), rid, models.ObservationKindWeight, 100, 0)
	if err != nil {
		t.Fatalf("list-by-kind: %v", err)
	}
	if len(got) != 1 || got[0].Kind != models.ObservationKindWeight {
		t.Errorf("expected exactly 1 weight observation, got %d (%+v)", len(got), got)
	}
}

// Ensure the unused import is referenced at compile time (delta.ErrNoBaseline
// is the sentinel returned by InMemoryBaselineProvider when unseeded).
var _ = delta.ErrNoBaseline
