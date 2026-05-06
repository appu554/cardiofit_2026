package delta

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeBaselineStateStore is the unit-test double for BaselineStateStore.
// In-memory map keyed by (residentID, vitalTypeKey); no SQL.
type fakeBaselineStateStore struct {
	mu   sync.Mutex
	rows map[string]Baseline
}

func newFakeBaselineStateStore() *fakeBaselineStateStore {
	return &fakeBaselineStateStore{rows: map[string]Baseline{}}
}

func (f *fakeBaselineStateStore) key(rid uuid.UUID, vt string) string {
	return rid.String() + "::" + vt
}

func (f *fakeBaselineStateStore) Get(_ context.Context, rid uuid.UUID, vt string) (*Baseline, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.rows[f.key(rid, vt)]
	if !ok {
		return nil, ErrNoBaseline
	}
	bb := b
	return &bb, nil
}

func (f *fakeBaselineStateStore) Upsert(_ context.Context, rid uuid.UUID, vt string, b Baseline) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[f.key(rid, vt)] = b
	return nil
}

func (f *fakeBaselineStateStore) RecomputeAndUpsert(_ context.Context, _ uuid.UUID, _ string, _ int) (*Baseline, error) {
	return nil, errors.New("not used in this unit test")
}

func TestPersistentBaselineProvider_NoRow_ReturnsErrNoBaseline(t *testing.T) {
	p := NewPersistentBaselineProvider(newFakeBaselineStateStore())
	_, err := p.FetchBaseline(context.Background(), uuid.New(), "8480-6")
	if !errors.Is(err, ErrNoBaseline) {
		t.Fatalf("expected ErrNoBaseline, got %v", err)
	}
}

func TestPersistentBaselineProvider_InsufficientSamples_ReturnsErrNoBaseline(t *testing.T) {
	store := newFakeBaselineStateStore()
	rid := uuid.New()
	if err := store.Upsert(context.Background(), rid, "8480-6", Baseline{
		BaselineValue: 130, StdDev: 5, SampleSize: 2, ComputedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	p := NewPersistentBaselineProvider(store)
	_, err := p.FetchBaseline(context.Background(), rid, "8480-6")
	if !errors.Is(err, ErrNoBaseline) {
		t.Fatalf("expected ErrNoBaseline for n<3, got %v", err)
	}
}

func TestPersistentBaselineProvider_ZeroStdDev_ReturnsErrNoBaseline(t *testing.T) {
	store := newFakeBaselineStateStore()
	rid := uuid.New()
	_ = store.Upsert(context.Background(), rid, "8480-6", Baseline{
		BaselineValue: 130, StdDev: 0, SampleSize: 7, ComputedAt: time.Now().UTC(),
	})
	p := NewPersistentBaselineProvider(store)
	_, err := p.FetchBaseline(context.Background(), rid, "8480-6")
	if !errors.Is(err, ErrNoBaseline) {
		t.Fatalf("zero StdDev should map to ErrNoBaseline, got %v", err)
	}
}

func TestPersistentBaselineProvider_HealthyRow_ReturnsBaseline(t *testing.T) {
	store := newFakeBaselineStateStore()
	rid := uuid.New()
	want := Baseline{BaselineValue: 130, StdDev: 5, SampleSize: 7, ComputedAt: time.Now().UTC()}
	_ = store.Upsert(context.Background(), rid, "8480-6", want)
	p := NewPersistentBaselineProvider(store)
	got, err := p.FetchBaseline(context.Background(), rid, "8480-6")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.BaselineValue != want.BaselineValue || got.StdDev != want.StdDev || got.SampleSize != want.SampleSize {
		t.Errorf("baseline drift: got %+v want %+v", *got, want)
	}
}

func TestClassifyBaselineConfidence(t *testing.T) {
	cases := []struct {
		name   string
		n      int
		iqr    float64
		median float64
		want   BaselineConfidence
	}{
		{"insufficient n=2", 2, 1.0, 100.0, BaselineConfidenceInsufficientData},
		{"high n=7 iqr<25%", 7, 20.0, 100.0, BaselineConfidenceHigh},
		{"medium n=4 iqr<50%", 4, 40.0, 100.0, BaselineConfidenceMedium},
		{"low n=3 wide iqr", 3, 80.0, 100.0, BaselineConfidenceLow},
		{"low n=7 wide iqr falls through", 7, 60.0, 100.0, BaselineConfidenceLow},
		{"medium boundary: iqr at 50% is NOT < 50%", 4, 50.0, 100.0, BaselineConfidenceLow},
		{"high boundary: iqr at 25% is NOT < 25%", 7, 25.0, 100.0, BaselineConfidenceMedium},
		{"zero median falls back to low", 7, 0.0, 0.0, BaselineConfidenceLow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyBaselineConfidence(tc.n, tc.iqr, tc.median)
			if got != tc.want {
				t.Errorf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestPercentiles_KnownInputs(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := Percentiles(values, 0.25, 0.5, 0.75)
	want := []float64{3.25, 5.5, 7.75}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("p%v: got %v want %v", []float64{0.25, 0.5, 0.75}[i], got[i], want[i])
		}
	}
}

func TestPercentiles_Edges(t *testing.T) {
	if got := Percentiles([]float64{}, 0.5); !math.IsNaN(got[0]) {
		t.Errorf("empty input should return NaN, got %v", got[0])
	}
	if got := Percentiles([]float64{42}, 0.0, 0.5, 1.0); got[0] != 42 || got[1] != 42 || got[2] != 42 {
		t.Errorf("single-element percentiles should all equal 42, got %v", got)
	}
	// Already sorted vs reversed must yield identical results.
	asc := []float64{10, 20, 30, 40, 50}
	desc := []float64{50, 40, 30, 20, 10}
	a := Percentiles(asc, 0.5)
	d := Percentiles(desc, 0.5)
	if a[0] != d[0] {
		t.Errorf("Percentiles must be order-independent: asc=%v desc=%v", a[0], d[0])
	}
}

// Compile-time confirmation that the fake satisfies BaselineStateStore.
var _ BaselineStateStore = (*fakeBaselineStateStore)(nil)
