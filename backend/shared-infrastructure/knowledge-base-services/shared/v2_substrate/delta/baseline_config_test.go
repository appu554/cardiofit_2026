package delta

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDefaultConfig_FallbackValues(t *testing.T) {
	c := DefaultConfig("unknown-type")
	if c.ObservationType != "unknown-type" {
		t.Errorf("ObservationType: got %q want %q", c.ObservationType, "unknown-type")
	}
	if c.WindowDays != DefaultBaselineLookbackDays {
		t.Errorf("WindowDays: got %d want %d", c.WindowDays, DefaultBaselineLookbackDays)
	}
	if c.MinObsForHighConfidence != 7 {
		t.Errorf("MinObsForHighConfidence: got %d want 7", c.MinObsForHighConfidence)
	}
	if c.MorningOnly || c.FlagVelocity {
		t.Errorf("default filters should be off: morning=%v velocity=%v",
			c.MorningOnly, c.FlagVelocity)
	}
	if len(c.ExcludeDuringActiveConcerns) != 0 {
		t.Errorf("default excludes should be empty, got %v", c.ExcludeDuringActiveConcerns)
	}
}

func TestBaselineConfig_StructRoundTrip(t *testing.T) {
	in := BaselineConfig{
		ObservationType:             "8480-6",
		WindowDays:                  30,
		MinObsForHighConfidence:     21,
		ExcludeDuringActiveConcerns: []string{"acute_pain", "infection"},
		MorningOnly:                 true,
		FlagVelocity:                false,
		Notes:                       "systolic BP",
		UpdatedAt:                   time.Now().UTC(),
	}
	// Round-trip via List/Get of the in-memory fake.
	store := newFakeBaselineConfigStore()
	if err := store.Upsert(context.Background(), in); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := store.Get(context.Background(), in.ObservationType)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.WindowDays != in.WindowDays || got.MorningOnly != in.MorningOnly ||
		got.MinObsForHighConfidence != in.MinObsForHighConfidence ||
		len(got.ExcludeDuringActiveConcerns) != len(in.ExcludeDuringActiveConcerns) {
		t.Errorf("round-trip drift: got %+v want %+v", *got, in)
	}
}

func TestBaselineConfigStore_ErrNotFound(t *testing.T) {
	store := newFakeBaselineConfigStore()
	_, err := store.Get(context.Background(), "nonexistent")
	if !errors.Is(err, ErrBaselineConfigNotFound) {
		t.Fatalf("expected ErrBaselineConfigNotFound, got %v", err)
	}
}

// fakeBaselineConfigStore is the in-memory test double used both here
// and by tests in this package that exercise the recompute path with a
// known config (without the Postgres dependency).
type fakeBaselineConfigStore struct {
	mu   sync.Mutex
	rows map[string]BaselineConfig
}

func newFakeBaselineConfigStore() *fakeBaselineConfigStore {
	return &fakeBaselineConfigStore{rows: map[string]BaselineConfig{}}
}

func (f *fakeBaselineConfigStore) Get(_ context.Context, ot string) (*BaselineConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.rows[ot]
	if !ok {
		return nil, ErrBaselineConfigNotFound
	}
	cc := c
	return &cc, nil
}

func (f *fakeBaselineConfigStore) List(_ context.Context) ([]BaselineConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]BaselineConfig, 0, len(f.rows))
	for _, c := range f.rows {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeBaselineConfigStore) Upsert(_ context.Context, c BaselineConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c.UpdatedAt = time.Now().UTC()
	f.rows[c.ObservationType] = c
	return nil
}

// TestPersistentBaselineProvider_WithConfigStore_ResolvesConfig verifies
// that when a BaselineConfigStore is wired, FetchBaseline behaviour is
// unchanged on the read path (the config governs writes/recomputes, not
// reads). A separate test for the recompute path lives in kb-20.
func TestPersistentBaselineProvider_WithConfigStore_PreservesReadSemantics(t *testing.T) {
	state := newFakeBaselineStateStore()
	cfg := newFakeBaselineConfigStore()
	rid := uuid.New()
	_ = state.Upsert(context.Background(), rid, "8480-6", Baseline{
		BaselineValue: 130, StdDev: 5, SampleSize: 7, ComputedAt: time.Now().UTC(),
	})

	p := NewPersistentBaselineProvider(state).WithConfigStore(cfg)
	got, err := p.FetchBaseline(context.Background(), rid, "8480-6")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.BaselineValue != 130 || got.SampleSize != 7 {
		t.Errorf("baseline drift via config-aware provider: got %+v", *got)
	}
}

// Compile-time confirmation that the fake satisfies BaselineConfigStore.
var _ BaselineConfigStore = (*fakeBaselineConfigStore)(nil)
