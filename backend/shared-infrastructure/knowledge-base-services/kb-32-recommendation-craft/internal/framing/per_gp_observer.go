// Package framing implements Stage 5 of the six-stage rendering pipeline:
// frame-vs-content separation for audit-defensible multi-audience delivery.
//
// VisibilityClass: AD — per-GP framing learning with toxicity guards per
// Guidelines §8.
//
// ARCHITECTURAL PROHIBITION: The per-GP observation model is aggregate-only
// across all pharmacists. No pharmacist_id is stored, transmitted, or inferable
// from the observation record. This is a non-negotiable constraint that prevents
// pharmacist surveillance via GP-acceptance patterns. The schema (migration 040)
// enforces this structurally; this package enforces it at the application layer.
//
// Toxicity guard rules (Guidelines §8):
//  1. Aggregate-only — no pharmacist_id anywhere in the observation pipeline.
//  2. 30-observation floor (MinObservationsThreshold) — fewer than 30 accepted
//     observations returns "default" framing, not a learned tone.
//  3. Prescriber opt-out — if the GP has opted out (migration 041), "default"
//     is returned regardless of observation count; opt-out check runs first.
package framing

import (
	"context"
	"errors"
)

// MinObservationsThreshold is the minimum number of GP-decision observations
// required before a learned framing tone is returned. Below this floor the
// PerGPObserver returns "default" framing so that under-sampled GPs are never
// profiled on insufficient data.
const MinObservationsThreshold = 30

// ErrFramingOptedOut is returned by Suggest when the GP has registered an
// opt-out in the prescriber_framing_optout table. Callers should treat this
// as a signal to use default framing and MUST NOT retry with a different gpID.
var ErrFramingOptedOut = errors.New("framing: prescriber has opted out of per-GP framing adaptation")

// ErrInvalidOutcome is returned by Observe when the decision_outcome value is
// not one of the three accepted literals: "accepted", "declined", "deferred".
var ErrInvalidOutcome = errors.New("framing: invalid decision outcome; must be one of accepted/declined/deferred")

// validFramingTones is the canonical set of accepted framing tone values.
// Mirrors the CHECK constraint in migration 040; enforced here at the application layer.
var validFramingTones = map[string]struct{}{
	"concise":       {},
	"detailed":      {},
	"collaborative": {},
	"default":       {},
}

// validDecisionOutcomes is the canonical set of accepted decision_outcome values.
// Mirrors the CHECK constraint in migration 040.
var validDecisionOutcomes = map[string]struct{}{
	"accepted": {},
	"declined": {},
	"deferred": {},
}

// IsValidFramingTone reports whether s is one of the four recognised framing
// tone codes: "concise", "detailed", "collaborative", "default".
// The check is case-sensitive. This helper enforces the client-side equivalent
// of the CHECK constraint in migration 040 (framing_tone column).
func IsValidFramingTone(s string) bool {
	_, ok := validFramingTones[s]
	return ok
}

// FramingPattern holds the learned framing preference for a single GP,
// derived from aggregate observations. It deliberately carries no pharmacist
// attribution (see Guidelines §8 / architectural prohibition above).
type FramingPattern struct {
	// GPID is the unique identifier of the prescribing GP.
	GPID string

	// BestFramingTone is the tone that has most frequently accompanied accepted
	// recommendations for this GP. One of: "concise", "detailed", "collaborative".
	// Never "default" — "default" is the sentinel returned when data is insufficient.
	BestFramingTone string

	// ObservationCount is the total number of observations contributing to
	// BestFramingTone. Must reach MinObservationsThreshold before the pattern
	// is considered actionable.
	ObservationCount int
}

// ObservationSource abstracts the data layer so that PerGPObserver can be
// tested without a real database. Implementations must be safe for concurrent
// use from multiple goroutines.
type ObservationSource interface {
	// PatternFor returns the aggregate framing pattern for the given GP.
	// Returns (nil, nil) when no observations exist for the GP.
	PatternFor(ctx context.Context, gpID string) (*FramingPattern, error)

	// HasOptedOut reports whether the GP has registered an opt-out in the
	// prescriber_framing_optout table (migration 047).
	//
	// Postgres-backed implementations SHOULD delegate this call to
	// OptOutStore.IsOptedOut (see optout_store.go) so that exactly one
	// source of truth governs opt-out state: the same row a /v1/framing
	// HTTP write created/revoked is the row a Suggest-time read observes.
	HasOptedOut(ctx context.Context, gpID string) (bool, error)

	// RecordObservation persists a single framing observation to the
	// per_gp_framing_observations table. The pharmacist caller must NOT pass
	// any pharmacist identifying information — the implementation stores only
	// (gp_id, framing_tone, decision_outcome).
	//
	// outcome must be one of "accepted", "declined", "deferred"; tone must
	// pass IsValidFramingTone. Validation is also performed by Observe before
	// this method is called, so implementations may trust the inputs are valid.
	RecordObservation(ctx context.Context, gpID, tone, outcome string) error
}

// PerGPObserver implements the read-write surface for the per-GP framing
// observation system. It applies both toxicity guards (opt-out check and
// 30-observation floor) on every Suggest call.
//
// Construct with NewPerGPObserver; do not create directly.
type PerGPObserver struct {
	src ObservationSource
}

// NewPerGPObserver constructs a PerGPObserver backed by the given source.
// src must not be nil.
func NewPerGPObserver(src ObservationSource) *PerGPObserver {
	if src == nil {
		panic("framing: NewPerGPObserver: src must not be nil")
	}
	return &PerGPObserver{src: src}
}

// Suggest returns the recommended framing tone for the given GP.
//
// Guard evaluation order (both are non-negotiable per Guidelines §8):
//  1. Opt-out check — if the GP has opted out, returns ("", ErrFramingOptedOut).
//  2. Observation floor — if ObservationCount < MinObservationsThreshold,
//     returns ("default", nil).
//
// If both guards pass, returns the GP's learned BestFramingTone.
// If no pattern exists yet (nil from source), returns ("default", nil).
func (o *PerGPObserver) Suggest(ctx context.Context, gpID string) (string, error) {
	// Guard 1: opt-out check (always first, regardless of observation count).
	optedOut, err := o.src.HasOptedOut(ctx, gpID)
	if err != nil {
		return "", err
	}
	if optedOut {
		return "", ErrFramingOptedOut
	}

	// Guard 2: observation floor.
	pattern, err := o.src.PatternFor(ctx, gpID)
	if err != nil {
		return "", err
	}
	if pattern == nil || pattern.ObservationCount < MinObservationsThreshold {
		return "default", nil
	}

	return pattern.BestFramingTone, nil
}

// Observe records a single framing observation for the given GP.
// This is the write path; it persists (gpID, tone, outcome) to the
// per_gp_framing_observations table via the ObservationSource.
//
// Parameters:
//   - gpID: the GP's unique identifier
//   - tone: must pass IsValidFramingTone ("concise", "detailed", "collaborative", "default")
//   - outcome: must be one of "accepted", "declined", "deferred"
//
// Returns ErrInvalidOutcome if outcome is not one of the three accepted values.
// No pharmacist identifier is accepted or stored; callers must not attempt to
// pass pharmacist attribution — this method intentionally has no such parameter.
func (o *PerGPObserver) Observe(ctx context.Context, gpID, tone, outcome string) error {
	if _, ok := validDecisionOutcomes[outcome]; !ok {
		return ErrInvalidOutcome
	}
	return o.src.RecordObservation(ctx, gpID, tone, outcome)
}
