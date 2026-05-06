package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

func openTestCareIntensityStore(t *testing.T) (*CareIntensityStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated care_intensity store test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	v2 := NewV2SubstrateStoreWithDB(db)
	return NewCareIntensityStore(db, v2), db
}

func TestCareIntensityStore_FirstTransitionThenCurrentLookup(t *testing.T) {
	s, db := openTestCareIntensityStore(t)
	defer db.Close()

	rid := uuid.New()
	roleRef := uuid.New()
	in := models.CareIntensity{
		ResidentRef:         rid,
		Tag:                 models.CareIntensityTagActiveTreatment,
		EffectiveDate:       time.Now().UTC().Truncate(time.Second),
		DocumentedByRoleRef: roleRef,
	}
	out, err := s.CreateCareIntensityTransition(context.Background(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.CareIntensity == nil || out.CareIntensity.Tag != models.CareIntensityTagActiveTreatment {
		t.Fatalf("CareIntensity drift: %+v", out.CareIntensity)
	}
	if out.Event == nil || out.Event.EventType != models.EventTypeCareIntensityTransition {
		t.Errorf("Event drift: %+v", out.Event)
	}
	// First-ever transition: no automatic cascades for active_treatment.
	if len(out.Cascades) != 0 {
		t.Errorf("expected 0 cascades for first→active_treatment; got %+v", out.Cascades)
	}

	cur, err := s.GetCurrentCareIntensity(context.Background(), rid)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if cur.ID != out.CareIntensity.ID {
		t.Errorf("current ID drift: got %s want %s", cur.ID, out.CareIntensity.ID)
	}
}

func TestCareIntensityStore_ActiveToPalliativeProducesThreeCascades(t *testing.T) {
	s, db := openTestCareIntensityStore(t)
	defer db.Close()

	rid := uuid.New()
	roleRef := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := s.CreateCareIntensityTransition(context.Background(), models.CareIntensity{
		ResidentRef:         rid,
		Tag:                 models.CareIntensityTagActiveTreatment,
		EffectiveDate:       now.Add(-24 * time.Hour),
		DocumentedByRoleRef: roleRef,
	}); err != nil {
		t.Fatalf("seed active: %v", err)
	}
	out, err := s.CreateCareIntensityTransition(context.Background(), models.CareIntensity{
		ResidentRef:         rid,
		Tag:                 models.CareIntensityTagPalliative,
		EffectiveDate:       now,
		DocumentedByRoleRef: roleRef,
	})
	if err != nil {
		t.Fatalf("transition active→palliative: %v", err)
	}
	if len(out.Cascades) != 3 {
		t.Errorf("expected 3 cascades for active→palliative; got %d: %+v", len(out.Cascades), out.Cascades)
	}
	if out.Event.Severity != models.EventSeverityModerate {
		t.Errorf("expected severity=moderate; got %s", out.Event.Severity)
	}
	// supersedes_ref should auto-link to the previous active row.
	if out.CareIntensity.SupersedesRef == nil {
		t.Errorf("expected supersedes_ref to be auto-linked")
	}
}

func TestCareIntensityStore_HistoryOrderedDesc(t *testing.T) {
	s, db := openTestCareIntensityStore(t)
	defer db.Close()

	rid := uuid.New()
	roleRef := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	tags := []struct {
		t  string
		at time.Time
	}{
		{models.CareIntensityTagActiveTreatment, now.Add(-72 * time.Hour)},
		{models.CareIntensityTagComfortFocused, now.Add(-24 * time.Hour)},
		{models.CareIntensityTagPalliative, now},
	}
	for _, x := range tags {
		if _, err := s.CreateCareIntensityTransition(context.Background(), models.CareIntensity{
			ResidentRef:         rid,
			Tag:                 x.t,
			EffectiveDate:       x.at,
			DocumentedByRoleRef: roleRef,
		}); err != nil {
			t.Fatalf("seed %s: %v", x.t, err)
		}
	}

	hist, err := s.ListCareIntensityHistory(context.Background(), rid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hist) != 3 {
		t.Fatalf("expected 3 rows; got %d", len(hist))
	}
	if hist[0].Tag != models.CareIntensityTagPalliative {
		t.Errorf("expected newest=palliative; got %s", hist[0].Tag)
	}
	if hist[2].Tag != models.CareIntensityTagActiveTreatment {
		t.Errorf("expected oldest=active_treatment; got %s", hist[2].Tag)
	}

	// Current = newest.
	cur, err := s.GetCurrentCareIntensity(context.Background(), rid)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if cur.Tag != models.CareIntensityTagPalliative {
		t.Errorf("current drift: got %s", cur.Tag)
	}
}

func TestCareIntensityStore_GetCurrentNoHistoryReturnsNotFound(t *testing.T) {
	s, db := openTestCareIntensityStore(t)
	defer db.Close()

	_, err := s.GetCurrentCareIntensity(context.Background(), uuid.New())
	if !errors.Is(err, interfaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestCareIntensityStore_RejectsInvalidTag(t *testing.T) {
	s, db := openTestCareIntensityStore(t)
	defer db.Close()
	bad := models.CareIntensity{
		ResidentRef:         uuid.New(),
		Tag:                 "active", // legacy short form, not a valid v2.4 tag
		EffectiveDate:       time.Now().UTC(),
		DocumentedByRoleRef: uuid.New(),
	}
	if _, err := s.CreateCareIntensityTransition(context.Background(), bad); err == nil {
		t.Errorf("expected error for legacy short tag")
	}
}

func TestCareIntensityStore_PalliativeToComfortFocusedOneCascade(t *testing.T) {
	s, db := openTestCareIntensityStore(t)
	defer db.Close()

	rid := uuid.New()
	roleRef := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := s.CreateCareIntensityTransition(context.Background(), models.CareIntensity{
		ResidentRef:         rid,
		Tag:                 models.CareIntensityTagPalliative,
		EffectiveDate:       now.Add(-24 * time.Hour),
		DocumentedByRoleRef: roleRef,
	}); err != nil {
		t.Fatalf("seed palliative: %v", err)
	}
	out, err := s.CreateCareIntensityTransition(context.Background(), models.CareIntensity{
		ResidentRef:         rid,
		Tag:                 models.CareIntensityTagComfortFocused,
		EffectiveDate:       now,
		DocumentedByRoleRef: roleRef,
	})
	if err != nil {
		t.Fatalf("transition palliative→comfort: %v", err)
	}
	if len(out.Cascades) != 1 {
		t.Errorf("expected 1 cascade for palliative→comfort_focused; got %+v", out.Cascades)
	}
	if len(out.Cascades) == 1 && out.Cascades[0].Kind != "revisit_monitoring_plan" {
		t.Errorf("expected revisit_monitoring_plan; got %s", out.Cascades[0].Kind)
	}
}
