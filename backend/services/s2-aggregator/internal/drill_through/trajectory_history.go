package drill_through

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TrajectoryHistory bundles the full observation series for one parameter
// per v1.0 Part 5.5 (trajectory drill-through). Observations are ordered
// chronologically (oldest first); each observation carries its own
// substrate confidence so the renderer can mark variable-confidence
// series per v1.0 Part 10.3.
type TrajectoryHistory struct {
	ResidentID   uuid.UUID
	Parameter    string
	Observations []substrate_types.Observation
}

// GetTrajectoryHistory returns the full observation series for one
// parameter. Returns an empty Observations slice (not nil) when no
// observations exist.
func GetTrajectoryHistory(
	ctx context.Context,
	client aggregation.SubstrateClient,
	residentID uuid.UUID,
	parameter string,
) (TrajectoryHistory, error) {
	if client == nil {
		return TrajectoryHistory{}, fmt.Errorf("GetTrajectoryHistory: nil client")
	}
	if parameter == "" {
		return TrajectoryHistory{}, fmt.Errorf("GetTrajectoryHistory: empty parameter")
	}

	obs, err := client.TrajectoryHistory(ctx, residentID, parameter)
	if err != nil {
		return TrajectoryHistory{}, fmt.Errorf("GetTrajectoryHistory(%s): %w", parameter, err)
	}
	if obs == nil {
		obs = []substrate_types.Observation{}
	}
	return TrajectoryHistory{
		ResidentID:   residentID,
		Parameter:    parameter,
		Observations: obs,
	}, nil
}
