package bias_stratification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cardiofit/shared/v2_substrate/ethics/pattern_detection"
)

// fakeSource is a deterministic in-memory MetricSource for tests.
type fakeSource struct {
	samples []Sample
	err     error
	// blockBetween, if > 0, sleeps between sample emits to enable cancellation tests.
	blockBetween time.Duration
}

func (f *fakeSource) StreamMetrics(ctx context.Context, metric string) (<-chan Sample, error) {
	if f.err != nil {
		return nil, f.err
	}
	ch := make(chan Sample)
	go func() {
		defer close(ch)
		for _, s := range f.samples {
			if f.blockBetween > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(f.blockBetween):
				}
			}
			select {
			case <-ctx.Done():
				return
			case ch <- s:
			}
		}
	}()
	return ch, nil
}

func TestStratifyByDimension_BasicMeans(t *testing.T) {
	src := &fakeSource{
		samples: []Sample{
			{ResidentID: "r1", Value: 10, Demographics: map[Dimension]string{DimAgeBand: "65-74"}},
			{ResidentID: "r2", Value: 20, Demographics: map[Dimension]string{DimAgeBand: "65-74"}},
			{ResidentID: "r3", Value: 30, Demographics: map[Dimension]string{DimAgeBand: "75-84"}},
			{ResidentID: "r4", Value: 50, Demographics: map[Dimension]string{DimAgeBand: "75-84"}},
			{ResidentID: "r5", Value: 100, Demographics: map[Dimension]string{DimAgeBand: "85+"}},
			{ResidentID: "r6", Value: 100, Demographics: map[Dimension]string{DimAgeBand: "85+"}},
		},
	}
	s := NewStratifier(src)
	got, err := s.StratifyByDimension(context.Background(), "appropriateness", DimAgeBand)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := map[string]float64{"65-74": 15, "75-84": 40, "85+": 100}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("stratum %q: got %v, want %v", k, got[k], v)
		}
	}
}

func TestStratifyByDimension_DropsUnclassified(t *testing.T) {
	src := &fakeSource{
		samples: []Sample{
			{ResidentID: "r1", Value: 10, Demographics: map[Dimension]string{DimSex: "F"}},
			{ResidentID: "r2", Value: 99, Demographics: map[Dimension]string{DimSex: ""}},
			{ResidentID: "r3", Value: 99, Demographics: map[Dimension]string{}},
			{ResidentID: "r4", Value: 30, Demographics: map[Dimension]string{DimSex: "M"}},
		},
	}
	s := NewStratifier(src)
	got, err := s.StratifyByDimension(context.Background(), "m", DimSex)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, has := got[""]; has {
		t.Errorf("empty stratum key must be dropped: %v", got)
	}
	if got["F"] != 10 || got["M"] != 30 {
		t.Errorf("unexpected stratum means: %v", got)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 strata, got %d: %v", len(got), got)
	}
}

func TestStratifyByDimension_EmptyStream(t *testing.T) {
	src := &fakeSource{samples: nil}
	s := NewStratifier(src)
	got, err := s.StratifyByDimension(context.Background(), "m", DimAgeBand)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestStratifyByDimension_SourceError(t *testing.T) {
	want := errors.New("boom")
	src := &fakeSource{err: want}
	s := NewStratifier(src)
	_, err := s.StratifyByDimension(context.Background(), "m", DimAgeBand)
	if !errors.Is(err, want) {
		t.Fatalf("got err %v, want %v", err, want)
	}
}

func TestStratifyByDimension_ContextCancel(t *testing.T) {
	src := &fakeSource{
		samples: []Sample{
			{ResidentID: "r1", Value: 1, Demographics: map[Dimension]string{DimAgeBand: "65-74"}},
			{ResidentID: "r2", Value: 2, Demographics: map[Dimension]string{DimAgeBand: "65-74"}},
			{ResidentID: "r3", Value: 3, Demographics: map[Dimension]string{DimAgeBand: "65-74"}},
		},
		blockBetween: 50 * time.Millisecond,
	}
	s := NewStratifier(src)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.StratifyByDimension(ctx, "m", DimAgeBand)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected ctx.Canceled, got %v", err)
	}
}

func TestStratifyAll_FansOutSingleSweep(t *testing.T) {
	full := func(age, sex, frail, cald, ses, fac string) map[Dimension]string {
		return map[Dimension]string{
			DimAgeBand:     age,
			DimSex:         sex,
			DimFrailtyTier: frail,
			DimCALD:        cald,
			DimSocioecon:   ses,
			DimFacility:    fac,
		}
	}
	samples := []Sample{
		{Value: 1, Demographics: full("65-74", "F", "low", "anglo", "Q1", "siteA")},
		{Value: 3, Demographics: full("65-74", "F", "low", "anglo", "Q1", "siteA")},
		{Value: 10, Demographics: full("75-84", "M", "med", "cald", "Q3", "siteB")},
		{Value: 20, Demographics: full("75-84", "M", "med", "cald", "Q3", "siteB")},
		{Value: 100, Demographics: full("85+", "F", "high", "cald", "Q5", "siteC")},
		{Value: 100, Demographics: full("85+", "F", "high", "cald", "Q5", "siteC")},
	}
	src := &fakeSource{samples: samples}
	s := NewStratifier(src)
	got, err := s.StratifyAll(context.Background(), "m")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 6 {
		t.Fatalf("expected 6 dimensions, got %d: %v", len(got), got)
	}
	if got[DimAgeBand]["65-74"] != 2 || got[DimAgeBand]["75-84"] != 15 || got[DimAgeBand]["85+"] != 100 {
		t.Errorf("DimAgeBand wrong: %v", got[DimAgeBand])
	}
	if got[DimSex]["F"] != (1+3+100+100)/4.0 {
		t.Errorf("DimSex F wrong: %v", got[DimSex])
	}
	if got[DimFacility]["siteB"] != 15 {
		t.Errorf("DimFacility siteB wrong: %v", got[DimFacility])
	}
}

func TestStratifyAll_ConsumesDetectBiasDisparity(t *testing.T) {
	// One stratum at 100, others at 1 — obvious disparity, ratio = 100.
	samples := []Sample{
		{Value: 100, Demographics: map[Dimension]string{DimAgeBand: "85+"}},
		{Value: 100, Demographics: map[Dimension]string{DimAgeBand: "85+"}},
		{Value: 1, Demographics: map[Dimension]string{DimAgeBand: "65-74"}},
		{Value: 1, Demographics: map[Dimension]string{DimAgeBand: "75-84"}},
	}
	src := &fakeSource{samples: samples}
	s := NewStratifier(src)
	strat, err := s.StratifyByDimension(context.Background(), "appropriateness", DimAgeBand)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !pattern_detection.DetectBiasDisparity(strat, 1.5) {
		t.Errorf("DetectBiasDisparity should flag obvious disparity, got false. strat=%v", strat)
	}
}
