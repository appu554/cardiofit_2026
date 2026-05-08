// Package dashboards provides the pharmacist self-visibility dashboard surfaces.
//
// This file implements Surface 4: My Clinical Reasoning Patterns.
// Trajectory-first visualisation of recommendation type distribution over time.
// Class-specific implementation rates are compared against the Ramsey 2025
// national baselines using ceiling framing only — peer ranking is never surfaced.
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// RamseyBaselines are the Ramsey 2025 national implementation-rate baselines
// per Self-Visibility Guidelines §4.2.
//
// These values define the anonymised best-in-class ceiling used for aspiration
// framing. They MUST NOT be used to rank pharmacists against each other.
var RamseyBaselines = map[string]float64{
	"colecalciferol":  0.37,
	"calcium":         0.36,
	"ppi":             0.43,
	"cessation_total": 0.51,
	"dose_reduction":  0.49,
}

// TrajectoryPoint holds a single time-period observation of a pharmacist's
// recommendation-implementation rate (RIR).
//
// VisibilityClass: PFA for trajectory metrics; POA for any reflective annotation.
type TrajectoryPoint struct {
	// PeriodStart is an opaque ordinal (e.g. fiscal-month index) identifying the
	// start of the measurement period. It is intentionally not a time.Time so that
	// callers control the granularity and formatting.
	PeriodStart int
	// RIRPct is the recommendation-implementation rate for this period, expressed
	// as a fraction in [0,1]. E.g. 0.55 means 55 % of recommendations were
	// implemented by the patient's GP.
	RIRPct float64
}

// RamseyCompare holds a single class-specific comparison between a pharmacist's
// own implementation rate and the corresponding Ramsey 2025 national baseline.
//
// FramedAsCeiling is always true: this comparison communicates "what excellent
// ACOPs achieve" — it is never a peer-rank or percentile signal.
type RamseyCompare struct {
	// OwnRate is the pharmacist's own implementation rate for this drug class,
	// expressed as a fraction in [0,1].
	OwnRate float64
	// Baseline is the Ramsey 2025 national baseline for this class (from
	// RamseyBaselines), expressed as a fraction in [0,1].
	Baseline float64
	// FramedAsCeiling is always true for self-view records to guarantee that the
	// UI layer presents this as an aspiration target, not a competitive ranking.
	FramedAsCeiling bool
}

// ReasoningView is the pharmacist's read-only view of their own clinical
// reasoning patterns.
//
// VisibilityClass: PFA for trajectory metrics; POA for any reflective annotation.
//
// PeerPercentile is always nil in the self-view. The field is reserved solely
// for a hypothetical employer-facing view that requires separate consent. Any
// code path that sets PeerPercentile for a self-view is a privacy violation.
type ReasoningView struct {
	// Trajectory is the pharmacist's RIR time series, ordered by PeriodStart.
	// An empty (non-nil) slice means the pharmacist has no trajectory data yet.
	Trajectory []TrajectoryPoint
	// RamseyComparison maps drug-class keys (matching RamseyBaselines) to the
	// pharmacist's ceiling comparison for that class. Only the 5 known Ramsey
	// 2025 classes are ever present; unknown classes from the source are silently
	// excluded to prevent non-validated baselines from reaching the UI.
	RamseyComparison map[string]RamseyCompare
	// PeerPercentile MUST always be nil in self-view (Self-Visibility Guidelines
	// §3.4). It is retained as a typed field so the employer-facing view can share
	// the struct without re-defining it, but the self-view constructor (For) never
	// populates it.
	PeerPercentile *float64
}

// ReasoningSource is the data-access interface backing Reasoning.
//
// Implementations must:
//   - Respect context cancellation.
//   - Return data for the specified pharmacist only.
//   - Never include peer comparison data; that responsibility belongs to a
//     separate employer-view layer.
type ReasoningSource interface {
	// RIRTrajectory returns the time-series of recommendation-implementation rates
	// for the given pharmacist. An empty slice (with nil error) is valid and means
	// no trajectory data is available yet.
	RIRTrajectory(ctx context.Context, pharmacistID uuid.UUID) ([]TrajectoryPoint, error)
	// ClassSpecificRates returns a map of drug-class key → implementation rate for
	// the given pharmacist. Keys should match the RamseyBaselines map, but unknown
	// keys are tolerated (they are silently excluded by For).
	ClassSpecificRates(ctx context.Context, pharmacistID uuid.UUID) (map[string]float64, error)
}

// Reasoning implements Surface 4 — My Clinical Reasoning Patterns.
//
// Construct with NewReasoning; call For to obtain the ReasoningView for a
// specific pharmacist.
type Reasoning struct{ src ReasoningSource }

// NewReasoning returns a Reasoning backed by the given ReasoningSource.
func NewReasoning(s ReasoningSource) *Reasoning { return &Reasoning{src: s} }

// For returns the ReasoningView for the given pharmacist.
//
// Trajectory ordering: returned as-is from the source; the caller is
// responsible for any presentation-layer sort.
//
// Ramsey filter: only the 5 classes present in RamseyBaselines are included in
// RamseyComparison. Classes returned by ClassSpecificRates that are not in
// RamseyBaselines are silently excluded — this prevents non-validated baselines
// from reaching the UI layer.
//
// PeerPercentile guarantee: PeerPercentile is always nil in the returned view
// (Self-Visibility Guidelines §3.4). No caller should read it as meaningful.
func (r *Reasoning) For(ctx context.Context, pharmacistID uuid.UUID) (ReasoningView, error) {
	traj, err := r.src.RIRTrajectory(ctx, pharmacistID)
	if err != nil {
		return ReasoningView{}, err
	}
	rates, err := r.src.ClassSpecificRates(ctx, pharmacistID)
	if err != nil {
		return ReasoningView{}, err
	}

	// Ensure Trajectory is non-nil so callers can distinguish "no data" from
	// an uninitialised result without a nil check.
	if traj == nil {
		traj = []TrajectoryPoint{}
	}

	view := ReasoningView{
		Trajectory:       traj,
		RamseyComparison: make(map[string]RamseyCompare),
		// PeerPercentile intentionally nil — self-view only.
	}

	for class, ownRate := range rates {
		// Guard: only include classes with a known Ramsey 2025 baseline.
		baseline, ok := RamseyBaselines[class]
		if !ok {
			continue
		}
		view.RamseyComparison[class] = RamseyCompare{
			OwnRate:         ownRate,
			Baseline:        baseline,
			FramedAsCeiling: true, // ceiling framing, never peer rank
		}
	}

	return view, nil // PeerPercentile intentionally nil
}
