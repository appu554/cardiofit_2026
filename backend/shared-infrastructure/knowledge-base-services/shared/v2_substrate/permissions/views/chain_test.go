package views

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeChainSource struct {
	wrongNetwork bool
}

func (f *fakeChainSource) RollupForChain(_ context.Context, chainID uuid.UUID, metric string) (ChainRollup, error) {
	id := chainID
	if f.wrongNetwork {
		id = uuid.New() // deliberately different
	}
	return ChainRollup{
		ChainID:        id,
		Metric:         metric,
		AggregateValue: 0.89,
	}, nil
}

func TestChainView_HappyPath(t *testing.T) {
	src := &fakeChainSource{}
	v := NewChainView(src)
	chainID := uuid.New()

	rollup, err := v.Rollup(context.Background(), chainID, "rir_p90")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rollup.ChainID != chainID {
		t.Errorf("expected ChainID %v, got %v", chainID, rollup.ChainID)
	}
	if rollup.Metric != "rir_p90" {
		t.Errorf("expected metric rir_p90, got %s", rollup.Metric)
	}
}

func TestChainView_WrongNetwork(t *testing.T) {
	src := &fakeChainSource{wrongNetwork: true}
	v := NewChainView(src)

	_, err := v.Rollup(context.Background(), uuid.New(), "rir_p90")
	if err == nil {
		t.Errorf("expected error for wrong network ID, got nil")
	}
}
