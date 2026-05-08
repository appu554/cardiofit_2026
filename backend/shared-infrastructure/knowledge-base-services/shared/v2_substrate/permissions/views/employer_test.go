package views

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeEmployerSource struct {
	gateSatisfied    bool
	pharmacistCount  int
}

func (f *fakeEmployerSource) AggregateRIRForEmployer(_ context.Context, _ uuid.UUID, _ int) (AggregateRIR, error) {
	return AggregateRIR{
		Distribution:    []float64{0.85, 0.90, 0.88, 0.92, 0.87},
		PharmacistCount: f.pharmacistCount,
		GateSatisfied:   f.gateSatisfied,
	}, nil
}

func TestEmployerView_GateUnsatisfied(t *testing.T) {
	src := &fakeEmployerSource{gateSatisfied: false, pharmacistCount: 10}
	v := NewEmployerView(src)
	_, err := v.AggregateRIR(context.Background(), uuid.New(), 28)
	if err == nil {
		t.Fatalf("expected ErrAggregationGateUnsatisfied, got nil")
	}
	if err != ErrAggregationGateUnsatisfied {
		t.Errorf("expected ErrAggregationGateUnsatisfied, got: %v", err)
	}
}

func TestEmployerView_ReidentificationFloor(t *testing.T) {
	src := &fakeEmployerSource{gateSatisfied: true, pharmacistCount: 3}
	v := NewEmployerView(src)
	_, err := v.AggregateRIR(context.Background(), uuid.New(), 28)
	if err == nil {
		t.Fatalf("expected ErrReidentificationFloor, got nil")
	}
	if err != ErrReidentificationFloor {
		t.Errorf("expected ErrReidentificationFloor, got: %v", err)
	}
}

func TestEmployerView_HappyPath(t *testing.T) {
	src := &fakeEmployerSource{gateSatisfied: true, pharmacistCount: 8}
	v := NewEmployerView(src)
	agg, err := v.AggregateRIR(context.Background(), uuid.New(), 28)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !agg.GateSatisfied {
		t.Errorf("expected GateSatisfied to be true")
	}
	if agg.PharmacistCount != 8 {
		t.Errorf("expected PharmacistCount 8, got %d", agg.PharmacistCount)
	}
}
