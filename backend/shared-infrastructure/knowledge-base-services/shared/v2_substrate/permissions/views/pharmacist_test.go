package views

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// fakeSource satisfies PharmacistSource.
// By default OwnRIR returns a trajectory whose AuthorID matches the requested pharmacistID.
// When mismatchAuthor is set, it returns a trajectory with a *different* AuthorID.
type fakeSource struct {
	mismatchAuthor bool
}

func (f *fakeSource) RIRTrajectoryFor(_ context.Context, pharmacistID uuid.UUID, _ int) (RIRTrajectory, error) {
	authorID := pharmacistID
	if f.mismatchAuthor {
		authorID = uuid.New() // deliberately different
	}
	return RIRTrajectory{
		AuthorID: authorID,
		Points: []TrajectoryPoint{
			{PeriodStart: "2026-04-01", RIR: 0.92},
		},
	}, nil
}

func TestPharmacistView_OwnRIRTrajectory(t *testing.T) {
	pharmacist := uuid.New()
	source := &fakeSource{}
	v := NewPharmacistView(source)

	traj, err := v.OwnRIRTrajectory(context.Background(), pharmacist, 28)
	if err != nil {
		t.Fatalf("traj: %v", err)
	}
	if traj.AuthorID != pharmacist {
		t.Errorf("returned wrong author trajectory: %v", traj.AuthorID)
	}
}

func TestPharmacistView_RejectsCrossPharmacistAccess(t *testing.T) {
	v := NewPharmacistView(&fakeSource{mismatchAuthor: true})
	_, err := v.OwnRIRTrajectory(context.Background(), uuid.New(),
		28 /* fakeSource returns mismatched ID; verify rejection */)
	if err == nil {
		t.Errorf("expected cross-pharmacist access to error")
	}
}
