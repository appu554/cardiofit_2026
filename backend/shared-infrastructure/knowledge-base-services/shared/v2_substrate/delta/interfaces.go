// Package delta provides the delta-on-write service for v2 substrate
// Observations. ComputeDelta is a pure function: given an Observation and an
// optional Baseline, it returns a Delta describing the directional deviation
// from baseline. Baselines are sourced via the BaselineProvider interface,
// which kb-26 (or any KB owning baseline data) implements as a thin adapter.
//
// Why service-layer (not DB trigger): triggers cannot cleanly call kb-26's
// AcuteRepository (separate service, separate DB); triggers create hidden
// coupling; service-layer is testable with mock BaselineProviders. See
// spec §2.1 for the architectural rationale.
package delta

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNoBaseline is returned by BaselineProvider.FetchBaseline when no
// historical data exists for (residentID, vitalType). ComputeDelta callers
// MUST translate this sentinel into a Delta with DirectionalFlag =
// models.DeltaFlagNoBaseline rather than failing the write.
var ErrNoBaseline = errors.New("delta: no baseline available")

// Baseline is the historical reference point for a single vital type at a
// single resident. Sourced from kb-26 (or a replica). SampleSize is the
// number of historical observations the BaselineValue + StdDev were derived
// from; ComputedAt is when kb-26 last refreshed the baseline.
//
// StdDev is the population standard deviation in the same unit as the
// associated Observation.Value. ComputeDelta divides (value - BaselineValue)
// by StdDev to derive the deviation in standard-deviation units; thresholds
// for the directional flag are defined in compute.go.
type Baseline struct {
	BaselineValue float64   `json:"baseline_value"`
	StdDev        float64   `json:"stddev"`
	SampleSize    int       `json:"sample_size"`
	ComputedAt    time.Time `json:"computed_at"`
	// VelocityFlag is set by the recompute path when the associated
	// BaselineConfig.FlagVelocity is true and the 14-day decline crosses
	// VelocityDeclineThreshold (Layer 2 §2.2 — eGFR ≥20% decline). Zero
	// value (false) preserves backwards compatibility for callers that
	// do not consult per-observation-type configs.
	VelocityFlag bool `json:"velocity_flag,omitempty"`
}

// BaselineProvider exposes baseline data for delta computation. kb-26's
// AcuteRepository is the production implementation; tests use in-memory
// mocks. vitalType is the LOINC code (vitals/labs) or model-internal kind
// identifier (e.g. "weight"); the provider resolves it to its own internal
// vital-type key.
//
// FetchBaseline returns ErrNoBaseline when no data exists. Other errors
// (network, decode) propagate to the caller; UpsertObservation translates
// them into Delta with DirectionalFlag = no_baseline + logs the error
// (decision: do not fail the write because the observation must persist
// regardless of baseline availability).
type BaselineProvider interface {
	FetchBaseline(ctx context.Context, residentID uuid.UUID, vitalType string) (*Baseline, error)
}
