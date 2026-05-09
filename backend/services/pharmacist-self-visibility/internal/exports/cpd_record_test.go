package exports_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cardiofit/pharmacist-self-visibility/internal/exports"
)

// stubCPDSource is a test double for CPDExportSource.
type stubCPDSource struct {
	activities []exports.ActivityRow
	actErr     error
	reflErr    error
}

func (s *stubCPDSource) ActivitiesInCycle(ctx context.Context, pharmacistID string, cycleStart, cycleEnd int) ([]exports.ActivityRow, error) {
	if s.actErr != nil {
		return nil, s.actErr
	}
	return s.activities, nil
}

func (s *stubCPDSource) ReflectionsForActivity(ctx context.Context, activityID string) ([]string, error) {
	if s.reflErr != nil {
		return nil, s.reflErr
	}
	return nil, nil
}

// TestCPDRecord_HoursByCategory — verbatim test from plan.
// Verifies that confirmed activities are summed correctly by category.
func TestCPDRecord_HoursByCategory(t *testing.T) {
	activities := []exports.ActivityRow{
		{ID: "a1", Category: "clinical", Hours: 2.0, Confirmed: true},
		{ID: "a2", Category: "clinical", Hours: 1.5, Confirmed: true},
		{ID: "a3", Category: "communication", Hours: 3.0, Confirmed: true},
	}

	src := &stubCPDSource{activities: activities}
	gen := exports.NewCPDRecordGenerator(src)

	rec, err := gen.Generate(context.Background(), "pharm-10", 2025, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.HoursByCategory["clinical"] != 3.5 {
		t.Errorf("expected clinical=3.5, got %v", rec.HoursByCategory["clinical"])
	}
	if rec.HoursByCategory["communication"] != 3.0 {
		t.Errorf("expected communication=3.0, got %v", rec.HoursByCategory["communication"])
	}
	if rec.PharmacistID != "pharm-10" {
		t.Errorf("unexpected pharmacist ID %q", rec.PharmacistID)
	}
	if rec.CycleStart != 2025 || rec.CycleEnd != 2026 {
		t.Errorf("unexpected cycle %d–%d", rec.CycleStart, rec.CycleEnd)
	}
}

// TestCPDRecord_UnconfirmedExcluded — explicit test that Confirmed: false
// activities are not counted in HoursByCategory.
func TestCPDRecord_UnconfirmedExcluded(t *testing.T) {
	activities := []exports.ActivityRow{
		{ID: "b1", Category: "clinical", Hours: 5.0, Confirmed: true},
		{ID: "b2", Category: "clinical", Hours: 2.0, Confirmed: false}, // must be excluded
		{ID: "b3", Category: "management", Hours: 1.0, Confirmed: false}, // must be excluded
	}

	src := &stubCPDSource{activities: activities}
	gen := exports.NewCPDRecordGenerator(src)

	rec, err := gen.Generate(context.Background(), "pharm-11", 2025, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.HoursByCategory["clinical"] != 5.0 {
		t.Errorf("expected clinical=5.0 (unconfirmed excluded), got %v", rec.HoursByCategory["clinical"])
	}
	if _, exists := rec.HoursByCategory["management"]; exists {
		t.Error("management category should not appear: all activities were unconfirmed")
	}
}

// TestCPDRecord_PropagatesActivitiesError — source error from ActivitiesInCycle
// propagates to caller.
func TestCPDRecord_PropagatesActivitiesError(t *testing.T) {
	wantErr := errors.New("query timeout")
	src := &stubCPDSource{actErr: wantErr}
	gen := exports.NewCPDRecordGenerator(src)

	_, err := gen.Generate(context.Background(), "pharm-12", 2025, 2026)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// TestCPDRecord_EmptyActivitiesReturnsEmptyMap — when there are no activities
// HoursByCategory is a non-nil empty map, not nil.
func TestCPDRecord_EmptyActivitiesReturnsEmptyMap(t *testing.T) {
	src := &stubCPDSource{activities: []exports.ActivityRow{}}
	gen := exports.NewCPDRecordGenerator(src)

	rec, err := gen.Generate(context.Background(), "pharm-13", 2025, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.HoursByCategory == nil {
		t.Error("HoursByCategory must be non-nil even for empty activities")
	}
	if len(rec.HoursByCategory) != 0 {
		t.Errorf("expected empty map, got %v", rec.HoursByCategory)
	}
}

// TestCPDRecord_GeneratedAtIsUTC — GeneratedAt must be in UTC.
func TestCPDRecord_GeneratedAtIsUTC(t *testing.T) {
	src := &stubCPDSource{activities: []exports.ActivityRow{}}
	gen := exports.NewCPDRecordGenerator(src)

	rec, err := gen.Generate(context.Background(), "pharm-14", 2025, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.GeneratedAt.Location() != time.UTC {
		t.Errorf("expected GeneratedAt in UTC, got %v", rec.GeneratedAt.Location())
	}
}
