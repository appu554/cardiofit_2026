package views

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// RIRTrajectory holds a pharmacist's own Recommendation Implementation Rate trajectory.
type RIRTrajectory struct {
	AuthorID uuid.UUID
	Points   []TrajectoryPoint
}

// TrajectoryPoint is a single period data point within a RIRTrajectory.
type TrajectoryPoint struct {
	PeriodStart string  // ISO date, e.g. "2026-04-01"
	RIR         float64 // Recommendation Implementation Rate for the period
}

// PharmacistSource is implemented by the upstream substrate query layer.
type PharmacistSource interface {
	RIRTrajectoryFor(ctx context.Context, pharmacistID uuid.UUID, days int) (RIRTrajectory, error)
}

// ErrCrossPharmacistAccess is returned when a query attempts to read another
// pharmacist's POA / PDP-class data through the self-view adapter.
var ErrCrossPharmacistAccess = errors.New("views: pharmacist self-view rejects cross-pharmacist queries")

// PharmacistView returns POA / PDP-class data for the pharmacist viewing their
// own work. Cross-pharmacist queries are refused at this layer regardless of
// what the underlying source returns.
type PharmacistView struct {
	src PharmacistSource
}

// NewPharmacistView constructs a PharmacistView backed by src.
func NewPharmacistView(src PharmacistSource) *PharmacistView {
	return &PharmacistView{src: src}
}

// OwnRIRTrajectory returns the RIR trajectory for pharmacistID.
// It rejects the result if the source returns a trajectory whose AuthorID
// does not match the requested pharmacistID.
func (v *PharmacistView) OwnRIRTrajectory(ctx context.Context, pharmacistID uuid.UUID, days int) (RIRTrajectory, error) {
	traj, err := v.src.RIRTrajectoryFor(ctx, pharmacistID, days)
	if err != nil {
		return RIRTrajectory{}, err
	}
	if traj.AuthorID != pharmacistID {
		return RIRTrajectory{}, ErrCrossPharmacistAccess
	}
	return traj, nil
}
