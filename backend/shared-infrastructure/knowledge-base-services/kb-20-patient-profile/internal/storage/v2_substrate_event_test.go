package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

func openEventTestStore(t *testing.T) *V2SubstrateStore {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated event storage test")
	}
	store, err := NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func TestUpsertGetEvent_RoundTrip(t *testing.T) {
	store := openEventTestStore(t)
	defer store.Close()

	fac := uuid.New()
	in := models.Event{
		ID:                 uuid.New(),
		EventType:          models.EventTypeFall,
		OccurredAt:         time.Now().UTC().Truncate(time.Second),
		OccurredAtFacility: &fac,
		ResidentID:         uuid.New(),
		ReportedByRef:      uuid.New(),
		WitnessedByRefs:    []uuid.UUID{uuid.New()},
		Severity:           models.EventSeverityModerate,
		DescriptionStructured: json.RawMessage(`{"location":"bathroom"}`),
		ReportableUnder:    []string{"QI Program"},
	}
	out, err := store.UpsertEvent(context.Background(), in)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if out.EventType != in.EventType || out.Severity != in.Severity {
		t.Errorf("upsert drift: %+v", out)
	}

	got, err := store.GetEvent(context.Background(), in.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.OccurredAtFacility == nil || *got.OccurredAtFacility != fac {
		t.Errorf("OccurredAtFacility lost: got %v", got.OccurredAtFacility)
	}
	if len(got.ReportableUnder) != 1 || got.ReportableUnder[0] != "QI Program" {
		t.Errorf("ReportableUnder lost: got %v", got.ReportableUnder)
	}
	if len(got.WitnessedByRefs) != 1 {
		t.Errorf("WitnessedByRefs lost: got %v", got.WitnessedByRefs)
	}
}

func TestGetEvent_NotFoundSentinel(t *testing.T) {
	store := openEventTestStore(t)
	defer store.Close()
	_, err := store.GetEvent(context.Background(), uuid.New())
	if !errors.Is(err, interfaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListEventsByResident(t *testing.T) {
	store := openEventTestStore(t)
	defer store.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		_, err := store.UpsertEvent(context.Background(), models.Event{
			ID:            uuid.New(),
			EventType:     models.EventTypeGPVisit,
			OccurredAt:    now.Add(time.Duration(i) * time.Hour),
			ResidentID:    rid,
			ReportedByRef: uuid.New(),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	out, err := store.ListEventsByResident(context.Background(), rid, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("expected 3 events, got %d", len(out))
	}
	// Newest first
	if !out[0].OccurredAt.After(out[2].OccurredAt) {
		t.Errorf("expected ORDER BY occurred_at DESC")
	}
}

func TestListEventsByType_DateRangeFilter(t *testing.T) {
	store := openEventTestStore(t)
	defer store.Close()

	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	rid := uuid.New()
	for i := 0; i < 3; i++ {
		_, err := store.UpsertEvent(context.Background(), models.Event{
			ID:            uuid.New(),
			EventType:     models.EventTypeRuleFire,
			OccurredAt:    base.AddDate(0, i, 0),
			ResidentID:    rid,
			ReportedByRef: uuid.New(),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	from := base.AddDate(0, 1, -1) // start of month 2 minus 1 day
	to := base.AddDate(0, 2, 1)    // month 3 plus 1 day
	out, err := store.ListEventsByType(context.Background(), models.EventTypeRuleFire, from, to, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Only the month-2 (i=1) and month-3 (i=2) seeds should match.
	if len(out) < 2 {
		t.Errorf("expected at least 2 events in [from, to); got %d", len(out))
	}
}
