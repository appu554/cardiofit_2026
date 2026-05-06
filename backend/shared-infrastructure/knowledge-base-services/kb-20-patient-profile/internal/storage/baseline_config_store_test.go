package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

// openTestBaselineConfigStore opens a *sql.DB + BaselineConfigStore against
// the test Postgres pointed to by KB20_TEST_DATABASE_URL. Skips cleanly when
// the env var is unset so the suite remains green in environments without
// a database.
func openTestBaselineConfigStore(t *testing.T) (*BaselineConfigStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated baseline_configs test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return NewBaselineConfigStore(db), db
}

// TestBaselineConfigStore_SeedRowsLanded verifies migration 014 inserted
// the 5 canonical rows from Layer 2 doc §2.2 (potassium, systolic BP,
// weight, behavioural agitation, eGFR). Acceptance criterion of Wave 2.2.
func TestBaselineConfigStore_SeedRowsLanded(t *testing.T) {
	cs, db := openTestBaselineConfigStore(t)
	defer db.Close()

	expected := map[string]struct {
		windowDays int
		morning    bool
		velocity   bool
	}{
		"potassium":                          {windowDays: 14, morning: false, velocity: false},
		"8480-6":                             {windowDays: 30, morning: true, velocity: false},
		"weight":                             {windowDays: 90, morning: false, velocity: false},
		"behavioural_agitation_episode_count": {windowDays: 14, morning: false, velocity: false},
		"egfr":                               {windowDays: 90, morning: false, velocity: true},
	}
	for ot, want := range expected {
		got, err := cs.Get(context.Background(), ot)
		if err != nil {
			t.Errorf("seed %q missing: %v", ot, err)
			continue
		}
		if got.WindowDays != want.windowDays {
			t.Errorf("%q window_days: got %d want %d", ot, got.WindowDays, want.windowDays)
		}
		if got.MorningOnly != want.morning {
			t.Errorf("%q morning_only: got %v want %v", ot, got.MorningOnly, want.morning)
		}
		if got.FlagVelocity != want.velocity {
			t.Errorf("%q flag_velocity: got %v want %v", ot, got.FlagVelocity, want.velocity)
		}
	}
}

func TestBaselineConfigStore_Get_NotFound(t *testing.T) {
	cs, db := openTestBaselineConfigStore(t)
	defer db.Close()
	_, err := cs.Get(context.Background(), "nonexistent-vital-"+t.Name())
	if !errors.Is(err, delta.ErrBaselineConfigNotFound) {
		t.Fatalf("expected ErrBaselineConfigNotFound, got %v", err)
	}
}

func TestBaselineConfigStore_UpsertRoundTrip(t *testing.T) {
	cs, db := openTestBaselineConfigStore(t)
	defer db.Close()

	in := delta.BaselineConfig{
		ObservationType:             "test-vital-" + t.Name(),
		WindowDays:                  21,
		MinObsForHighConfidence:     5,
		ExcludeDuringActiveConcerns: []string{"test_concern_a", "test_concern_b"},
		MorningOnly:                 true,
		FlagVelocity:                true,
		Notes:                       "test note",
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(),
			`DELETE FROM baseline_configs WHERE observation_type = $1`, in.ObservationType)
	}()

	if err := cs.Upsert(context.Background(), in); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := cs.Get(context.Background(), in.ObservationType)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.WindowDays != in.WindowDays || got.MorningOnly != in.MorningOnly ||
		got.FlagVelocity != in.FlagVelocity || got.Notes != in.Notes ||
		got.MinObsForHighConfidence != in.MinObsForHighConfidence {
		t.Errorf("round-trip drift: got %+v want %+v", *got, in)
	}
	if len(got.ExcludeDuringActiveConcerns) != len(in.ExcludeDuringActiveConcerns) {
		t.Fatalf("excludes len: got %v want %v",
			got.ExcludeDuringActiveConcerns, in.ExcludeDuringActiveConcerns)
	}
	for i, v := range in.ExcludeDuringActiveConcerns {
		if got.ExcludeDuringActiveConcerns[i] != v {
			t.Errorf("excludes[%d]: got %q want %q",
				i, got.ExcludeDuringActiveConcerns[i], v)
		}
	}
}

func TestBaselineConfigStore_List_IncludesSeeds(t *testing.T) {
	cs, db := openTestBaselineConfigStore(t)
	defer db.Close()

	rows, err := cs.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) < 5 {
		t.Fatalf("expected >=5 rows after migration 014 seed, got %d", len(rows))
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.ObservationType] = true
	}
	for _, ot := range []string{"potassium", "8480-6", "weight",
		"behavioural_agitation_episode_count", "egfr"} {
		if !seen[ot] {
			t.Errorf("seed row missing from List: %s", ot)
		}
	}
}
