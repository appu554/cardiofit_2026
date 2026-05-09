package views

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// RACHRollup holds pharmacy-partner (RACH) level roll-up metrics.
type RACHRollup struct {
	RACHID         uuid.UUID
	Metric         string
	AggregateValue float64
}

// RACHSource is implemented by the upstream substrate query layer.
type RACHSource interface {
	RollupForRACH(ctx context.Context, rachID uuid.UUID, metric string) (RACHRollup, error)
}

// ErrMismatchedRACHID is returned when the source returns a rollup whose
// RACH ID does not match the requested ID.
var ErrMismatchedRACHID = errors.New("views: RACH source returned mismatched RACH ID")

// RACHView returns pharmacy-partner level roll-up data.
//
// Phase 1a stub: wraps the source call and enforces RACH ID consistency.
// No aggregation logic is performed here.
//
// Phase 1c will own: aggregation invariants for the RACH view-type
// (contestation-aware pharmacy-partner roll-ups, restrictive-practice
// filtering per Ethical Architecture Guidelines §6, and resident
// vulnerability respecting per §5.3). See
// docs/superpowers/plans/2026-05-07-phase-1c-ethical-architecture-substrate.md.
type RACHView struct {
	src RACHSource
}

// NewRACHView constructs a RACHView backed by src.
func NewRACHView(src RACHSource) *RACHView {
	return &RACHView{src: src}
}

// Rollup returns the pharmacy-partner roll-up for rachID and the given metric.
// It rejects the result if the source returns a rollup whose RACH ID does not
// match the requested rachID.
func (v *RACHView) Rollup(ctx context.Context, rachID uuid.UUID, metric string) (RACHRollup, error) {
	r, err := v.src.RollupForRACH(ctx, rachID, metric)
	if err != nil {
		return RACHRollup{}, err
	}
	if r.RACHID != rachID {
		return RACHRollup{}, ErrMismatchedRACHID
	}
	return r, nil
}
