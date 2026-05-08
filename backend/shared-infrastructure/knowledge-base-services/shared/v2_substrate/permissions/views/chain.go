package views

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ChainRollup holds network-level roll-up metrics for a pharmacy chain.
type ChainRollup struct {
	ChainID        uuid.UUID
	Metric         string
	AggregateValue float64
}

// ChainSource is implemented by the upstream substrate query layer.
type ChainSource interface {
	RollupForChain(ctx context.Context, chainID uuid.UUID, metric string) (ChainRollup, error)
}

// ErrWrongChainNetwork is returned when the source returns a rollup whose
// chain ID does not match the requested chain ID.
var ErrWrongChainNetwork = errors.New("views: chain source returned mismatched chain network ID")

// ChainView returns network-level roll-up data for a pharmacy chain.
// Phase 1a stub: wraps the source call and enforces chain ID consistency.
type ChainView struct {
	src ChainSource
}

// NewChainView constructs a ChainView backed by src.
func NewChainView(src ChainSource) *ChainView {
	return &ChainView{src: src}
}

// Rollup returns the network-level roll-up for chainID and the given metric.
// It rejects the result if the source returns a rollup whose chain ID does not
// match the requested chainID.
func (v *ChainView) Rollup(ctx context.Context, chainID uuid.UUID, metric string) (ChainRollup, error) {
	r, err := v.src.RollupForChain(ctx, chainID, metric)
	if err != nil {
		return ChainRollup{}, err
	}
	if r.ChainID != chainID {
		return ChainRollup{}, ErrWrongChainNetwork
	}
	return r, nil
}
