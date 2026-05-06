package storage

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/delta"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// openTestBaselineStore opens a *sql.DB + BaselineStore against the test
// Postgres pointed to by KB20_TEST_DATABASE_URL. Skips cleanly when unset
// so the suite remains green in environments without a database.
func openTestBaselineStore(t *testing.T) (*BaselineStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated baseline store test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return NewBaselineStore(db), db
}

func TestBaselineStore_GetMissing_ReturnsErrNoBaseline(t *testing.T) {
	bs, db := openTestBaselineStore(t)
	defer db.Close()
	_, err := bs.Get(context.Background(), uuid.New(), "nonexistent-vital-key-"+uuid.NewString())
	if !errors.Is(err, delta.ErrNoBaseline) {
		t.Fatalf("expected ErrNoBaseline, got %v", err)
	}
}

func TestBaselineStore_UpsertGet_RoundTrip(t *testing.T) {
	bs, db := openTestBaselineStore(t)
	defer db.Close()
	rid := uuid.New()
	vt := "8480-6"
	in := delta.Baseline{
		BaselineValue: 130.0,
		StdDev:        5.0,
		SampleSize:    10,
		ComputedAt:    time.Now().UTC(),
	}
	if err := bs.Upsert(context.Background(), rid, vt, in); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := bs.Get(context.Background(), rid, vt)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if math.Abs(got.BaselineValue-in.BaselineValue) > 1e-6 {
		t.Errorf("baseline_value drift: got %v want %v", got.BaselineValue, in.BaselineValue)
	}
	if got.SampleSize != in.SampleSize {
		t.Errorf("sample_size drift: got %v want %v", got.SampleSize, in.SampleSize)
	}
	// IQR/StdDev round-trip is approximate (σ ≈ IQR/1.349); allow ~1%.
	if math.Abs(got.StdDev-in.StdDev) > 0.01*in.StdDev {
		t.Errorf("stddev drift > 1%%: got %v want %v", got.StdDev, in.StdDev)
	}
}

func TestBaselineStore_UpsertInsufficient_GetReturnsErrNoBaseline(t *testing.T) {
	bs, db := openTestBaselineStore(t)
	defer db.Close()
	rid := uuid.New()
	vt := "8480-6"
	if err := bs.Upsert(context.Background(), rid, vt, delta.Baseline{
		BaselineValue: 0, StdDev: 0, SampleSize: 1, ComputedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	_, err := bs.Get(context.Background(), rid, vt)
	if !errors.Is(err, delta.ErrNoBaseline) {
		t.Fatalf("expected ErrNoBaseline for n<3, got %v", err)
	}
}

// TestBaselineStore_RecomputeAndUpsert_FromObservations seeds the
// observations table with a known set of values and asserts the computed
// median + confidence tier matches the spec.
func TestBaselineStore_RecomputeAndUpsert_FromObservations(t *testing.T) {
	bs, db := openTestBaselineStore(t)
	defer db.Close()

	// Use V2SubstrateStore to seed observations (writes through the same
	// schema the recompute query reads from).
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	loinc := "8480-6"

	// Seven values with tight spread → expect HIGH confidence.
	values := []float64{120, 121, 122, 123, 124, 125, 126}
	for i, v := range values {
		val := v
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: loinc, Value: &val, Unit: "mmHg",
			ObservedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour).Truncate(time.Second),
		})
		if err != nil {
			t.Fatalf("seed observation %d: %v", i, err)
		}
	}

	got, err := bs.RecomputeAndUpsert(context.Background(), rid, loinc, 14)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	// Median of 120..126 = 123.
	if math.Abs(got.BaselineValue-123.0) > 1e-6 {
		t.Errorf("median: got %v want 123", got.BaselineValue)
	}
	if got.SampleSize != 7 {
		t.Errorf("sample_size: got %v want 7", got.SampleSize)
	}

	// Confirm the persisted row reports HIGH confidence.
	var conf string
	row := db.QueryRowContext(context.Background(),
		`SELECT confidence FROM baseline_state WHERE resident_id=$1 AND vital_type_key=$2`,
		rid, loinc)
	if err := row.Scan(&conf); err != nil {
		t.Fatalf("read confidence: %v", err)
	}
	if conf != string(delta.BaselineConfidenceHigh) {
		t.Errorf("expected HIGH confidence (n=7, tight IQR), got %s", conf)
	}
}

// TestBaselineStore_RecomputeAndUpsert_ConfidenceTransitions exercises the
// HIGH → MEDIUM → LOW transitions by varying IQR while keeping n constant.
func TestBaselineStore_RecomputeAndUpsert_ConfidenceTransitions(t *testing.T) {
	bs, db := openTestBaselineStore(t)
	defer db.Close()

	store := NewV2SubstrateStoreWithDB(db)
	loinc := "8480-6"

	cases := []struct {
		name   string
		values []float64
		want   delta.BaselineConfidence
	}{
		{
			// n=7, median=120, IQR≈3.0 → 3.0 < 0.25*120 (=30) → HIGH
			name:   "high",
			values: []float64{118, 119, 120, 120, 120, 121, 122},
			want:   delta.BaselineConfidenceHigh,
		},
		{
			// n=4 (so HIGH path closed), median≈110, IQR=20.5 → 20.5 < 0.5*110 (=55) → MEDIUM
			name:   "medium",
			values: []float64{100, 105, 115, 125},
			want:   delta.BaselineConfidenceMedium,
		},
		{
			// n=3 (HIGH+MEDIUM both closed) → LOW
			name:   "low",
			values: []float64{80, 100, 130},
			want:   delta.BaselineConfidenceLow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rid := uuid.New() // fresh resident → fresh observation set
			for i, v := range tc.values {
				val := v
				_, err := store.UpsertObservation(context.Background(), models.Observation{
					ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
					LOINCCode: loinc, Value: &val, Unit: "mmHg",
					ObservedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour).Truncate(time.Second),
				})
				if err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			if _, err := bs.RecomputeAndUpsert(context.Background(), rid, loinc, 14); err != nil {
				t.Fatalf("recompute: %v", err)
			}
			var conf string
			err := db.QueryRowContext(context.Background(),
				`SELECT confidence FROM baseline_state WHERE resident_id=$1 AND vital_type_key=$2`,
				rid, loinc).Scan(&conf)
			if err != nil {
				t.Fatalf("read confidence: %v", err)
			}
			if conf != string(tc.want) {
				t.Errorf("confidence: got %s want %s", conf, tc.want)
			}
		})
	}
}

// TestBaselineStore_PersistsAcrossReopen simulates a process restart by
// closing the store and reopening it. Proves we have real persistence,
// not in-memory state.
func TestBaselineStore_PersistsAcrossReopen(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	rid := uuid.New()
	vt := "test-persistence-" + uuid.NewString()

	// Round 1: open, write, close.
	db1, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	bs1 := NewBaselineStore(db1)
	if err := bs1.Upsert(context.Background(), rid, vt, delta.Baseline{
		BaselineValue: 137.0, StdDev: 4.0, SampleSize: 8, ComputedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	_ = db1.Close()

	// Round 2: open fresh connection, read.
	db2, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()
	bs2 := NewBaselineStore(db2)
	got, err := bs2.Get(context.Background(), rid, vt)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if math.Abs(got.BaselineValue-137.0) > 1e-6 {
		t.Errorf("persisted baseline drifted across reopen: got %v want 137", got.BaselineValue)
	}
}

// ============================================================================
// Wave 2.2: per-observation-type config tests
// ============================================================================

// TestBaselineStore_WindowDayOverride_PullsBeyondDefault14d verifies that
// when a config row specifies window_days=30 (e.g. systolic BP), the
// recompute pulls observations from the full 30-day window — not the
// Wave 2.1 hardcoded 14-day window.
func TestBaselineStore_WindowDayOverride_PullsBeyondDefault14d(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	cs := NewBaselineConfigStore(db)
	bs := NewBaselineStore(db).WithConfigStore(cs)
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	loinc := "8480-6" // systolic BP — seeded with window_days=30, morning_only=true

	// Seed observations spread across 30 days, all in the morning so the
	// morning_only filter doesn't drop them. With window_days=14 the
	// 20-29-day-old rows would be excluded; with =30 they're all included.
	now := time.Now().UTC().Truncate(time.Hour)
	syd, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("load tz: %v", err)
	}
	morning := func(daysAgo int) time.Time {
		// 8 AM Sydney to land squarely inside the 6-10 AM window.
		base := now.Add(-time.Duration(daysAgo) * 24 * time.Hour).In(syd)
		return time.Date(base.Year(), base.Month(), base.Day(), 8, 0, 0, 0, syd).UTC()
	}
	for i, daysAgo := range []int{1, 4, 7, 10, 13, 16, 22, 28} {
		val := 120.0 + float64(i)
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: loinc, Value: &val, Unit: "mmHg",
			ObservedAt: morning(daysAgo),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	got, err := bs.RecomputeAndUpsert(context.Background(), rid, loinc, 0)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if got.SampleSize != 8 {
		t.Errorf("with window_days=30, expected n=8 across 28d span, got %d", got.SampleSize)
	}
	// Verify the persisted window_days field matches the config (30).
	var persistedWindow int
	row := db.QueryRowContext(context.Background(),
		`SELECT baseline_window_days FROM baseline_state WHERE resident_id=$1 AND vital_type_key=$2`,
		rid, loinc)
	if err := row.Scan(&persistedWindow); err != nil {
		t.Fatalf("read window: %v", err)
	}
	if persistedWindow != 30 {
		t.Errorf("persisted window_days: got %d want 30 (from config)", persistedWindow)
	}
}

// TestBaselineStore_MorningOnlyFilter_ExcludesAfternoon verifies that the
// morning_only=true config (systolic BP) restricts the recompute to AM
// observations and drops afternoon/evening readings.
func TestBaselineStore_MorningOnlyFilter_ExcludesAfternoon(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	cs := NewBaselineConfigStore(db)
	bs := NewBaselineStore(db).WithConfigStore(cs)
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	loinc := "8480-6"

	syd, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("tz: %v", err)
	}
	at := func(daysAgo, hour int) time.Time {
		base := time.Now().UTC().In(syd).Add(-time.Duration(daysAgo) * 24 * time.Hour)
		return time.Date(base.Year(), base.Month(), base.Day(), hour, 0, 0, 0, syd).UTC()
	}
	// 4 morning + 4 afternoon readings within the 30-day window. Morning
	// readings have value 120; afternoon readings have value 200 (so a
	// failure to filter would dramatically shift the median).
	for i, daysAgo := range []int{1, 3, 5, 7} {
		val := 120.0 + float64(i)
		if _, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: loinc, Value: &val, Unit: "mmHg", ObservedAt: at(daysAgo, 8),
		}); err != nil {
			t.Fatalf("seed AM %d: %v", i, err)
		}
	}
	for i, daysAgo := range []int{2, 4, 6, 8} {
		val := 200.0 + float64(i)
		if _, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: loinc, Value: &val, Unit: "mmHg", ObservedAt: at(daysAgo, 15),
		}); err != nil {
			t.Fatalf("seed PM %d: %v", i, err)
		}
	}

	got, err := bs.RecomputeAndUpsert(context.Background(), rid, loinc, 0)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if got.SampleSize != 4 {
		t.Errorf("morning_only filter should leave n=4, got %d", got.SampleSize)
	}
	// Median of 120,121,122,123 = 121.5 — anything close to 200 means
	// afternoon readings leaked through.
	if math.Abs(got.BaselineValue-121.5) > 1.0 {
		t.Errorf("morning-only median: got %v want ~121.5", got.BaselineValue)
	}
}

// TestBaselineStore_CustomMinObsForHighConfidence_Downgrades verifies that
// a config with min_obs_for_high_confidence above the default 7 prevents
// the HIGH tier when n is below it, even if the default classifier would
// have promoted (n>=7 + tight IQR).
func TestBaselineStore_CustomMinObsForHighConfidence_Downgrades(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	cs := NewBaselineConfigStore(db)
	bs := NewBaselineStore(db).WithConfigStore(cs)
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	loinc := "8480-6" // seeded with min_obs_for_high_confidence=21

	syd, _ := time.LoadLocation("Australia/Sydney")
	for i := 0; i < 10; i++ {
		val := 120.0 + float64(i%2)
		ts := time.Now().UTC().In(syd).Add(-time.Duration(i+1) * 24 * time.Hour)
		ts = time.Date(ts.Year(), ts.Month(), ts.Day(), 8, 0, 0, 0, syd).UTC()
		if _, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: loinc, Value: &val, Unit: "mmHg", ObservedAt: ts,
		}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	if _, err := bs.RecomputeAndUpsert(context.Background(), rid, loinc, 0); err != nil {
		t.Fatalf("recompute: %v", err)
	}
	var conf string
	err = db.QueryRowContext(context.Background(),
		`SELECT confidence FROM baseline_state WHERE resident_id=$1 AND vital_type_key=$2`,
		rid, loinc).Scan(&conf)
	if err != nil {
		t.Fatalf("read conf: %v", err)
	}
	// n=10, tight IQR — default would be HIGH; with min=21, expect MEDIUM.
	if conf != string(delta.BaselineConfidenceMedium) {
		t.Errorf("expected MEDIUM (min_obs_for_high=21, n=10), got %s", conf)
	}
}

// TestBaselineStore_VelocityFlag_Triggers_OnEgfrDecline verifies that for
// the eGFR config (flag_velocity=true), a 14-day decline of ≥20% sets
// VelocityFlag on the returned Baseline.
func TestBaselineStore_VelocityFlag_Triggers_OnEgfrDecline(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	cs := NewBaselineConfigStore(db)
	bs := NewBaselineStore(db).WithConfigStore(cs)
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	vt := "egfr"

	// Decline series: oldest=60, newest=45 → 25% decline (>20% threshold).
	values := []float64{60, 58, 55, 52, 50, 48, 45}
	for i, v := range values {
		val := v
		// Insert in chronological order: oldest first → most recent last.
		ts := time.Now().UTC().Add(-time.Duration(len(values)-i) * 24 * time.Hour)
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindLab,
			LOINCCode: vt, Value: &val, Unit: "mL/min/1.73m2", ObservedAt: ts,
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	got, err := bs.RecomputeAndUpsert(context.Background(), rid, vt, 0)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if !got.VelocityFlag {
		t.Errorf("expected VelocityFlag=true for 25%% eGFR decline, got false (baseline=%+v)", *got)
	}
}

// TestBaselineStore_VelocityFlag_StableEgfr verifies that when eGFR is
// stable (no significant decline), VelocityFlag stays false.
func TestBaselineStore_VelocityFlag_StableEgfr(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	cs := NewBaselineConfigStore(db)
	bs := NewBaselineStore(db).WithConfigStore(cs)
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	vt := "egfr"

	// Flat-ish series: 60→58 = ~3% drift, well below threshold.
	values := []float64{60, 59, 60, 58, 59, 60, 58}
	for i, v := range values {
		val := v
		ts := time.Now().UTC().Add(-time.Duration(len(values)-i) * 24 * time.Hour)
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindLab,
			LOINCCode: vt, Value: &val, Unit: "mL/min/1.73m2", ObservedAt: ts,
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	got, err := bs.RecomputeAndUpsert(context.Background(), rid, vt, 0)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if got.VelocityFlag {
		t.Errorf("stable eGFR should not trigger VelocityFlag, got true (baseline=%+v)", *got)
	}
}

// TestBaselineStore_UnknownObservationType_FallsBackToDefault verifies
// that a vital type with no matching baseline_configs row uses the
// default (14d window, no filters), preserving Wave 2.1 behaviour.
func TestBaselineStore_UnknownObservationType_FallsBackToDefault(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	cs := NewBaselineConfigStore(db)
	bs := NewBaselineStore(db).WithConfigStore(cs)
	store := NewV2SubstrateStoreWithDB(db)
	rid := uuid.New()
	vt := "potassium-extra-" + uuid.NewString() // not in seed table

	// 5 readings within the default 14-day window.
	for i := 0; i < 5; i++ {
		val := 4.0 + float64(i)*0.1
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindLab,
			LOINCCode: vt, Value: &val, Unit: "mmol/L",
			ObservedAt: time.Now().UTC().Add(-time.Duration(i+1) * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	got, err := bs.RecomputeAndUpsert(context.Background(), rid, vt, 0)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	// All 5 readings should be included (default 14d window covers them).
	if got.SampleSize != 5 {
		t.Errorf("unknown type should use default 14d window with all 5 readings, got n=%d", got.SampleSize)
	}
	// Persisted window_days should be the default 14.
	var persistedWindow int
	if err := db.QueryRowContext(context.Background(),
		`SELECT baseline_window_days FROM baseline_state WHERE resident_id=$1 AND vital_type_key=$2`,
		rid, vt).Scan(&persistedWindow); err != nil {
		t.Fatalf("read window: %v", err)
	}
	if persistedWindow != delta.DefaultBaselineLookbackDays {
		t.Errorf("unknown-type fallback window: got %d want %d",
			persistedWindow, delta.DefaultBaselineLookbackDays)
	}
}

// ============================================================================
// Pre-existing test
// ============================================================================

// TestUpsertObservation_RecomputesBaselineTransactionally proves the
// observation INSERT and the baseline_state recompute commit together.
// Inserts ten observations; asserts the running baseline matches the
// median of the persisted set.
func TestUpsertObservation_RecomputesBaselineTransactionally(t *testing.T) {
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store := NewV2SubstrateStoreWithDB(db)
	bs := NewBaselineStore(db)
	store.SetBaselineStore(bs)
	store.SetBaselineProvider(delta.NewPersistentBaselineProvider(bs))

	rid := uuid.New()
	loinc := "8480-6"

	// Insert ten values: median = 5.5 (per Percentiles[0.5] of 1..10).
	for i := 1; i <= 10; i++ {
		val := float64(i)
		_, err := store.UpsertObservation(context.Background(), models.Observation{
			ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
			LOINCCode: loinc, Value: &val, Unit: "x",
			ObservedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour).Truncate(time.Second),
		})
		if err != nil {
			t.Fatalf("upsert obs %d: %v", i, err)
		}
	}

	// The baseline_state row should exist and its baseline_value should
	// be the median of the inserted set.
	var (
		baselineValue sql.NullFloat64
		nObs          int
	)
	err = db.QueryRowContext(context.Background(),
		`SELECT baseline_value, n_observations FROM baseline_state
		  WHERE resident_id=$1 AND vital_type_key=$2`, rid, loinc).Scan(&baselineValue, &nObs)
	if err != nil {
		t.Fatalf("read baseline_state row: %v", err)
	}
	if !baselineValue.Valid {
		t.Fatalf("baseline_value should be non-NULL after 10 inserts")
	}
	if math.Abs(baselineValue.Float64-5.5) > 1e-6 {
		t.Errorf("median: got %v want 5.5", baselineValue.Float64)
	}
	if nObs != 10 {
		t.Errorf("n_observations: got %v want 10", nObs)
	}
}
