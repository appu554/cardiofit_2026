package delta

import (
	"context"
	"errors"
	"time"
)

// BaselineConfig captures per-observation-type parameters for the running
// baseline recompute. Layer 2 doc §2.2 specifies that different vitals
// require different lookback windows and filter rules: weight is slow-
// changing (90d), systolic BP is morning-only (avoid post-meal/post-
// activity confounding), eGFR flags velocity (≥20% decline in 14 days),
// behavioural agitation excludes acute-infection windows, etc.
//
// One row per observation_type lives in the baseline_configs table
// (kb-20 migration 014); BaselineStore.RecomputeAndUpsertTx consults the
// row to parameterise its SELECT and confidence classifier. Unknown
// observation types fall through to DefaultConfig.
//
// observation_type matching mirrors V2SubstrateStore.vitalTypeKey():
// LOINC code preferred, SNOMED code fallback, Observation.Kind enum
// value as last resort. The seed data uses LOINC codes where available
// (e.g. "8480-6" for systolic BP) and human-readable keys otherwise.
type BaselineConfig struct {
	// ObservationType is the vital_type_key the config applies to.
	ObservationType string
	// WindowDays is the rolling lookback window in days for the
	// observations the recompute should consider.
	WindowDays int
	// MinObsForHighConfidence overrides the default n>=7 threshold for
	// the HIGH confidence tier. Layer 2 §2.2 calls for higher thresholds
	// on noisier types (e.g. systolic BP n=21, agitation n=7).
	MinObsForHighConfidence int
	// ExcludeDuringActiveConcerns lists active concern type names whose
	// presence should exclude observations from the lookup window.
	// Wired in Wave 2.3 once the active_concerns table lands.
	ExcludeDuringActiveConcerns []string
	// MorningOnly restricts the recompute to observations recorded
	// between 06:00 and 11:00 local time. Used for systolic BP to avoid
	// post-meal / post-activity confounding.
	MorningOnly bool
	// FlagVelocity instructs the recompute to additionally compute a
	// 14-day decline percentage and surface a velocity alert when the
	// decline crosses the threshold (≥20% per Layer 2 §2.2 for eGFR).
	FlagVelocity bool
	// Notes is operator-facing context describing the rationale for
	// the parameter choices. Persisted but not consumed in the algorithm.
	Notes string
	// UpdatedAt is when the config row was last modified.
	UpdatedAt time.Time
}

// ErrBaselineConfigNotFound is returned by BaselineConfigStore.Get when
// no row matches the supplied observation type. Callers SHOULD fall
// back to DefaultConfig rather than failing — unknown types are common
// (the seed table covers only the 5 canonical types from Layer 2 §2.2).
var ErrBaselineConfigNotFound = errors.New("baseline_config: not found for observation type")

// BaselineConfigStore is the persistence contract for BaselineConfig
// rows. Implementations live in kb-20 (Postgres-backed against the
// baseline_configs table). The delta package owns the algorithm and
// the interface; it never speaks SQL itself.
type BaselineConfigStore interface {
	// Get returns the config for observationType. Returns
	// ErrBaselineConfigNotFound when no row exists; other errors
	// (decode/network) are propagated as-is.
	Get(ctx context.Context, observationType string) (*BaselineConfig, error)
	// List returns every config row, ordered by observation_type.
	List(ctx context.Context) ([]BaselineConfig, error)
	// Upsert writes the supplied config (insert or replace by PK).
	Upsert(ctx context.Context, c BaselineConfig) error
}

// DefaultConfig returns the fallback parameters used when no config row
// matches the supplied observationType. Mirrors the Wave 2.1 hardcoded
// behaviour: 14-day window, n>=7 for HIGH confidence, no filters.
func DefaultConfig(observationType string) BaselineConfig {
	return BaselineConfig{
		ObservationType:         observationType,
		WindowDays:              DefaultBaselineLookbackDays,
		MinObsForHighConfidence: 7,
	}
}

// VelocityDeclineThreshold is the 14-day decline percentage that triggers
// a velocity alert when BaselineConfig.FlagVelocity is set. Layer 2 §2.2
// specifies ≥20% for eGFR; expressed as 0.20 here for direct comparison
// against the computed decline ratio.
const VelocityDeclineThreshold = 0.20
