package dashboards

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Fake source
// ---------------------------------------------------------------------------

type fakeReasoningSrc struct {
	trajectory []TrajectoryPoint
	classRates map[string]float64
	trajErr    error
	ratesErr   error
}

func (f *fakeReasoningSrc) RIRTrajectory(_ context.Context, _ uuid.UUID) ([]TrajectoryPoint, error) {
	return f.trajectory, f.trajErr
}

func (f *fakeReasoningSrc) ClassSpecificRates(_ context.Context, _ uuid.UUID) (map[string]float64, error) {
	return f.classRates, f.ratesErr
}

// ---------------------------------------------------------------------------
// Plan-verbatim Test 1
// ---------------------------------------------------------------------------

// TestReasoning_TrajectoryFirstNoPeerRank verifies that the returned
// ReasoningView carries the full trajectory supplied by the source and that
// PeerPercentile is always nil (Self-Visibility Guidelines §3.4).
func TestReasoning_TrajectoryFirstNoPeerRank(t *testing.T) {
	src := &fakeReasoningSrc{
		trajectory: []TrajectoryPoint{{PeriodStart: 0, RIRPct: 0.40}, {PeriodStart: 1, RIRPct: 0.55}},
	}
	d := NewReasoning(src)
	view, _ := d.For(context.Background(), uuid.New())
	if len(view.Trajectory) != 2 {
		t.Errorf("trajectory pts = %d", len(view.Trajectory))
	}
	if view.PeerPercentile != nil {
		t.Errorf("peer percentile must NOT be present in self-view")
	}
}

// ---------------------------------------------------------------------------
// Plan-verbatim Test 2
// ---------------------------------------------------------------------------

// TestReasoning_RamseyBaselineSurfacedAsCeiling verifies that when the source
// supplies a class rate, the RamseyComparison entry for that class carries the
// correct Ramsey 2025 baseline and FramedAsCeiling == true.
func TestReasoning_RamseyBaselineSurfacedAsCeiling(t *testing.T) {
	src := &fakeReasoningSrc{
		classRates: map[string]float64{"colecalciferol": 0.42},
	}
	d := NewReasoning(src)
	view, _ := d.For(context.Background(), uuid.New())
	if got, want := view.RamseyComparison["colecalciferol"].Baseline, 0.37; got != want {
		t.Errorf("colecalciferol baseline = %v, want %v", got, want)
	}
	if !view.RamseyComparison["colecalciferol"].FramedAsCeiling {
		t.Errorf("Ramsey comparison must be framed as ceiling, not peer rank")
	}
}

// ---------------------------------------------------------------------------
// Augmentation 1 — PeerPercentileAlwaysNil
// ---------------------------------------------------------------------------

// TestReasoning_PeerPercentileAlwaysNil explicitly asserts PeerPercentile is
// nil for any returned ReasoningView. This is a regression guard against a
// future refactor accidentally populating it in the self-view path.
func TestReasoning_PeerPercentileAlwaysNil(t *testing.T) {
	cases := []struct {
		name string
		src  *fakeReasoningSrc
	}{
		{
			name: "non-empty trajectory and rates",
			src: &fakeReasoningSrc{
				trajectory: []TrajectoryPoint{{PeriodStart: 0, RIRPct: 0.60}},
				classRates: map[string]float64{"ppi": 0.50, "calcium": 0.38},
			},
		},
		{
			name: "empty trajectory and rates",
			src:  &fakeReasoningSrc{},
		},
		{
			name: "all five Ramsey classes",
			src: &fakeReasoningSrc{
				classRates: map[string]float64{
					"colecalciferol": 0.45,
					"calcium":        0.40,
					"ppi":            0.48,
					"cessation_total": 0.55,
					"dose_reduction": 0.52,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewReasoning(tc.src)
			view, err := d.For(context.Background(), uuid.New())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if view.PeerPercentile != nil {
				t.Errorf("PeerPercentile must always be nil in self-view, got %v", *view.PeerPercentile)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Augmentation 2 — UnknownClassNotInRamseyComparison
// ---------------------------------------------------------------------------

// TestReasoning_UnknownClassNotInRamseyComparison verifies that if the source
// returns a classRates entry for a class not present in RamseyBaselines (e.g.
// "experimental_class"), that class does NOT appear in view.RamseyComparison.
// Only the 5 known Ramsey 2025 baseline classes may appear.
func TestReasoning_UnknownClassNotInRamseyComparison(t *testing.T) {
	src := &fakeReasoningSrc{
		classRates: map[string]float64{
			"colecalciferol":   0.40,
			"experimental_class": 0.60,
		},
	}
	d := NewReasoning(src)
	view, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, found := view.RamseyComparison["experimental_class"]; found {
		t.Error("unknown class 'experimental_class' must NOT appear in RamseyComparison")
	}
	// Verify the known class is still present.
	if _, found := view.RamseyComparison["colecalciferol"]; !found {
		t.Error("known class 'colecalciferol' should be present in RamseyComparison")
	}
}

// ---------------------------------------------------------------------------
// Augmentation 3 — PropagatesSourceError
// ---------------------------------------------------------------------------

// TestReasoning_PropagatesSourceError verifies that errors from both
// RIRTrajectory and ClassSpecificRates propagate correctly and return a zero
// ReasoningView with the original error.
func TestReasoning_PropagatesSourceError(t *testing.T) {
	trajSentinel := errors.New("trajectory db unavailable")
	ratesSentinel := errors.New("rates db unavailable")

	t.Run("trajectory error propagates", func(t *testing.T) {
		src := &fakeReasoningSrc{trajErr: trajSentinel}
		d := NewReasoning(src)
		view, err := d.For(context.Background(), uuid.New())
		if !errors.Is(err, trajSentinel) {
			t.Errorf("expected trajectory sentinel error, got %v", err)
		}
		if len(view.Trajectory) != 0 || view.RamseyComparison != nil {
			t.Errorf("expected zero ReasoningView on trajectory error, got %+v", view)
		}
	})

	t.Run("rates error propagates", func(t *testing.T) {
		src := &fakeReasoningSrc{
			trajectory: []TrajectoryPoint{{PeriodStart: 0, RIRPct: 0.50}},
			ratesErr:   ratesSentinel,
		}
		d := NewReasoning(src)
		view, err := d.For(context.Background(), uuid.New())
		if !errors.Is(err, ratesSentinel) {
			t.Errorf("expected rates sentinel error, got %v", err)
		}
		if len(view.Trajectory) != 0 || view.RamseyComparison != nil {
			t.Errorf("expected zero ReasoningView on rates error, got %+v", view)
		}
	})
}

// ---------------------------------------------------------------------------
// Augmentation 4 — EmptyTrajectoryStillReturnsView
// ---------------------------------------------------------------------------

// TestReasoning_EmptyTrajectoryStillReturnsView verifies that when the source
// returns an empty trajectory slice, For() returns a valid ReasoningView (not
// an error) with an empty Trajectory field. RamseyComparison may be populated
// if classRates are supplied, or empty otherwise.
func TestReasoning_EmptyTrajectoryStillReturnsView(t *testing.T) {
	t.Run("empty trajectory with classRates", func(t *testing.T) {
		src := &fakeReasoningSrc{
			trajectory: []TrajectoryPoint{},
			classRates: map[string]float64{"ppi": 0.48},
		}
		d := NewReasoning(src)
		view, err := d.For(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error for empty trajectory: %v", err)
		}
		if view.Trajectory == nil {
			t.Error("Trajectory must be non-nil (even if empty) for empty-trajectory case")
		}
		if len(view.Trajectory) != 0 {
			t.Errorf("expected empty Trajectory, got %d points", len(view.Trajectory))
		}
		// RamseyComparison should still be populated from classRates.
		if _, ok := view.RamseyComparison["ppi"]; !ok {
			t.Error("RamseyComparison should include 'ppi' even with empty trajectory")
		}
	})

	t.Run("empty trajectory and empty classRates", func(t *testing.T) {
		src := &fakeReasoningSrc{
			trajectory: []TrajectoryPoint{},
			classRates: map[string]float64{},
		}
		d := NewReasoning(src)
		view, err := d.For(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error for empty trajectory+classRates: %v", err)
		}
		if view.Trajectory == nil {
			t.Error("Trajectory must be non-nil even when empty")
		}
		if view.RamseyComparison == nil {
			t.Error("RamseyComparison must be non-nil map even when empty")
		}
	})
}
