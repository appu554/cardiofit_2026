package views

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// AggregateRIR holds an anonymised aggregate of RIR data across pharmacists
// for employer-level consumption. No per-pharmacist identifiable data is
// included unless DataAggregationConsent has been collected upstream.
type AggregateRIR struct {
	Distribution    []float64 // anonymised distribution across pharmacists
	PharmacistCount int       // must be >= reidentification floor
	GateSatisfied   bool      // AggregationGate must be satisfied before data is returned
}

// EmployerSource is implemented by the upstream substrate query layer.
type EmployerSource interface {
	AggregateRIRForEmployer(ctx context.Context, employerID uuid.UUID, days int) (AggregateRIR, error)
}

// ErrAggregationGateUnsatisfied is returned when the upstream aggregate does
// not satisfy the AggregationGate condition (e.g. insufficient consent coverage).
var ErrAggregationGateUnsatisfied = errors.New("views: aggregation gate not satisfied")

// ErrReidentificationFloor is returned when the pharmacist count in the
// aggregate falls below the minimum re-identification protection floor.
var ErrReidentificationFloor = errors.New("views: pharmacist count below reidentification floor")

// EmployerView returns aggregate RIR data for employer-level dashboards.
// It enforces the AggregationGate and re-identification floor before
// returning any data.
type EmployerView struct {
	src                  EmployerSource
	reidentificationFloor int // default 5
}

// NewEmployerView constructs an EmployerView with the default re-identification
// floor of 5 pharmacists.
func NewEmployerView(src EmployerSource) *EmployerView {
	return &EmployerView{src: src, reidentificationFloor: 5}
}

// AggregateRIR returns the anonymised aggregate for employerID over the
// requested day window. Returns ErrAggregationGateUnsatisfied if the gate
// condition is not met, or ErrReidentificationFloor if the pharmacist count
// is too small to protect individual identity.
func (v *EmployerView) AggregateRIR(ctx context.Context, employerID uuid.UUID, days int) (AggregateRIR, error) {
	agg, err := v.src.AggregateRIRForEmployer(ctx, employerID, days)
	if err != nil {
		return AggregateRIR{}, err
	}
	if !agg.GateSatisfied {
		return AggregateRIR{}, ErrAggregationGateUnsatisfied
	}
	if agg.PharmacistCount < v.reidentificationFloor {
		return AggregateRIR{}, ErrReidentificationFloor
	}
	return agg, nil
}
