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
