// VisibilityClass: WO (workflow-operational)
package dashboards

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Shared fake risk source
// ---------------------------------------------------------------------------

type fakeRiskSource struct {
	scores    map[uuid.UUID]int
	restraint map[uuid.UUID][]string
	err       error // if non-nil, ResidentsWithCompositeRisk returns this error
}

func (f *fakeRiskSource) ResidentsWithCompositeRisk(_ context.Context, _ uuid.UUID) (map[uuid.UUID]int, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.scores, nil
}
func (f *fakeRiskSource) RestraintSignalsFor(_ context.Context, residentID uuid.UUID) ([]string, error) {
	return f.restraint[residentID], nil
}
func (f *fakeRiskSource) TopReasons(_ context.Context, _ uuid.UUID) ([]string, error) {
	return []string{"recent_fall", "overdue_monitoring"}, nil
}

// ---------------------------------------------------------------------------
// Plan tests (verbatim from plan)
// ---------------------------------------------------------------------------

// TestWorklist_RankByCompositeRiskScore verifies that Today() returns items
// sorted by CompositeRisk descending so the highest-risk resident appears first.
func TestWorklist_RankByCompositeRiskScore(t *testing.T) {
	pharm := uuid.New()
	resA := uuid.New()
	resB := uuid.New()
	src := &fakeRiskSource{scores: map[uuid.UUID]int{resA: 9, resB: 4}}
	wl := NewWorklist(src)

	items, err := wl.Today(context.Background(), pharm)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].ResidentID != resA {
		t.Errorf("expected highest-risk first; got %v", items[0].ResidentID)
	}
}

// TestWorklist_RestraintSignalsSurfacedInline verifies that restraint signals
// for a resident are included inline on the WorklistItem.
func TestWorklist_RestraintSignalsSurfacedInline(t *testing.T) {
	pharm := uuid.New()
	res := uuid.New()
	src := &fakeRiskSource{
		scores:    map[uuid.UUID]int{res: 7},
		restraint: map[uuid.UUID][]string{res: {"recent_fall_within_72h"}},
	}
	wl := NewWorklist(src)

	items, _ := wl.Today(context.Background(), pharm)
	if len(items[0].RestraintSignals) != 1 {
		t.Errorf("expected restraint signal surfaced inline; got %v", items[0].RestraintSignals)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 1: TestWorklist_EmptyResidentList
// ---------------------------------------------------------------------------

// TestWorklist_EmptyResidentList guards against a nil slice being returned when
// the source has no residents. The API layer must distinguish "no work today"
// (empty, non-nil slice) from an uninitialized result.
func TestWorklist_EmptyResidentList(t *testing.T) {
	pharm := uuid.New()
	src := &fakeRiskSource{scores: map[uuid.UUID]int{}}
	wl := NewWorklist(src)

	items, err := wl.Today(context.Background(), pharm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Error("Today() returned nil slice; want empty non-nil slice")
	}
	if len(items) != 0 {
		t.Errorf("Today() returned %d items, want 0", len(items))
	}
}

// ---------------------------------------------------------------------------
// Augmentation 2: TestWorklist_TieBreakStable
// ---------------------------------------------------------------------------

// TestWorklist_TieBreakStable verifies that equal-score residents are ordered
// deterministically by ResidentID (UUID byte order ascending).
//
// Because Go map iteration is unordered, the implementation must apply a
// deterministic tie-breaker in the sort comparator. This test:
// 1. Generates 3 random UUIDs
// 2. Computes their expected order by UUID byte comparison
// 3. Sets all to the same CompositeRisk
// 4. Verifies the returned order matches the UUID-sorted order
//
// The test is run 10 times to catch any residual non-determinism.
func TestWorklist_TieBreakStable(t *testing.T) {
	// Run the test multiple times to ensure tie-break is truly deterministic.
	for run := 0; run < 10; run++ {
		pharm := uuid.New()
		resA := uuid.New()
		resB := uuid.New()
		resC := uuid.New()

		// Compute the expected order by sorting UUIDs by byte order.
		residents := []uuid.UUID{resA, resB, resC}
		sort.Slice(residents, func(i, j int) bool {
			return bytes.Compare(residents[i][:], residents[j][:]) < 0
		})
		expected := residents // This is now sorted by UUID byte order

		// All three residents share the same composite risk score.
		src := &fakeRiskSource{
			scores: map[uuid.UUID]int{resA: 5, resB: 5, resC: 5},
		}
		wl := NewWorklist(src)

		items, err := wl.Today(context.Background(), pharm)
		if err != nil {
			t.Fatalf("run %d: today: %v", run, err)
		}
		if len(items) != 3 {
			t.Fatalf("run %d: got %d items, want 3", run, len(items))
		}

		// Verify that returned items match the UUID-sorted order.
		for i := 0; i < 3; i++ {
			if items[i].ResidentID != expected[i] {
				t.Errorf("run %d: position %d: got %v, want %v",
					run, i, items[i].ResidentID, expected[i])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Augmentation 3: TestWorklist_PropagatesSourceError
// ---------------------------------------------------------------------------

// TestWorklist_PropagatesSourceError verifies that if ResidentsWithCompositeRisk
// returns an error, Today() propagates it rather than returning a silent empty result.
func TestWorklist_PropagatesSourceError(t *testing.T) {
	pharm := uuid.New()
	sentinel := errors.New("risk source unavailable")
	src := &fakeRiskSource{err: sentinel}
	wl := NewWorklist(src)

	_, err := wl.Today(context.Background(), pharm)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 4: TestWorklist_ContextCancellation
// ---------------------------------------------------------------------------

// TestWorklist_ContextCancellation verifies the defensive context check: if the
// context is already cancelled when Today() inspects it after fetching scores,
// it returns ctx.Err() rather than silently building an empty/partial result.
func TestWorklist_ContextCancellation(t *testing.T) {
	pharm := uuid.New()
	res := uuid.New()
	src := &fakeRiskSource{scores: map[uuid.UUID]int{res: 5}}
	wl := NewWorklist(src)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Today()

	_, err := wl.Today(ctx, pharm)
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
