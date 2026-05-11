package drill_through

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

func TestGetTrajectoryHistory_HappyPath(t *testing.T) {
	rid := uuid.New()
	client := aggregation.NewInMemorySubstrateClient().WithObservations(
		substrate_types.Observation{ID: uuid.New(), ResidentID: rid, Parameter: "egfr", Value: 50, ObservedAt: time.Now().AddDate(-1, 0, 0), Source: "kb-20"},
		substrate_types.Observation{ID: uuid.New(), ResidentID: rid, Parameter: "egfr", Value: 42, ObservedAt: time.Now(), Source: "kb-20"},
	)
	h, err := GetTrajectoryHistory(context.Background(), client, rid, "egfr")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(h.Observations) != 2 {
		t.Errorf("want 2 obs; got %d", len(h.Observations))
	}
	if !h.Observations[0].ObservedAt.Before(h.Observations[1].ObservedAt) {
		t.Error("observations should be sorted oldest first")
	}
}

func TestGetTrajectoryHistory_EmptySeries(t *testing.T) {
	client := aggregation.NewInMemorySubstrateClient()
	h, err := GetTrajectoryHistory(context.Background(), client, uuid.New(), "egfr")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if h.Observations == nil {
		t.Error("Observations must be non-nil empty slice")
	}
	if len(h.Observations) != 0 {
		t.Errorf("expected zero observations; got %d", len(h.Observations))
	}
}

func TestGetTrajectoryHistory_NilClient(t *testing.T) {
	_, err := GetTrajectoryHistory(context.Background(), nil, uuid.New(), "egfr")
	if err == nil {
		t.Fatal("expected error on nil client")
	}
}

func TestGetTrajectoryHistory_EmptyParam(t *testing.T) {
	_, err := GetTrajectoryHistory(context.Background(), aggregation.NewInMemorySubstrateClient(), uuid.New(), "")
	if err == nil {
		t.Fatal("expected error on empty parameter")
	}
}
