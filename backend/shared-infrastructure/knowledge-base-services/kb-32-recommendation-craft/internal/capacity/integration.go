// Package capacity implements the Stage 3.5 capacity + restrictive-practice
// consent gate for the kb-32 recommendation craft pipeline.
//
// The gate enforces Ethical Architecture Guidelines §6.4–6.6:
//
//   - §6.4 (Capacity is contextual): residents whose CognitiveCapacity is
//     uncertain, moderate_impairment, or severe_impairment may not have a
//     recommendation auto-drafted unless a substitute decision-maker (SDM)
//     workflow is in place. Mild impairment is explicitly excluded — mild
//     residents retain decisional capacity for routine matters under §6.6 and
//     do not require SDM.
//   - §6.5 (SDM workflow): when SDM is required, Assessment.SDMRequired MUST
//     be true to proceed. The conservative default when SDM is required but
//     not yet in place is to hold the recommendation.
//   - §6.6 (Restrictive practice): when the recommendation is for a
//     restrictive practice (chemical/physical/environmental restraint or
//     seclusion), an active RestrictivePracticeConsent MUST exist and
//     Allows(assessment.AssessedAt) MUST return true. No active consent ⇒ hold.
//
// The gate is purely advisory at the pipeline level: failure modes are
// returned as sentinel errors and converted by the pipeline into a
// PipelineResult.HoldReason rather than a hard error, mirroring the existing
// Stage 4 appropriateness-hold pattern.
package capacity

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/ethics/consent_extension"
	"github.com/cardiofit/shared/v2_substrate/ethics/vulnerability"
)

// CapacitySource is the port through which the Gate fetches the resident's
// vulnerability assessment and (when relevant) their restrictive-practice
// consent record. Production wiring is Postgres-backed; tests use a fake.
type CapacitySource interface {
	// AssessmentFor returns the current vulnerability.Assessment for residentID.
	// An error from the source is propagated by Gate.Evaluate as-is.
	AssessmentFor(ctx context.Context, residentID uuid.UUID) (vulnerability.Assessment, error)

	// RestrictivePracticeConsentFor returns the active
	// RestrictivePracticeConsent for residentID and practice, or nil when no
	// consent record exists. An error from the source is propagated as-is.
	RestrictivePracticeConsentFor(ctx context.Context, residentID uuid.UUID,
		practice consent_extension.PracticeType) (*consent_extension.RestrictivePracticeConsent, error)
}

// Gate is the Stage 3.5 capacity + consent gate.
type Gate struct {
	src CapacitySource
}

// NewGate constructs a Gate backed by src. src MUST NOT be nil; callers
// who do not have a real source wired (e.g. early Phase 2 deployments)
// should leave Pipeline.capacityGate nil instead of constructing a Gate
// with a nil source.
func NewGate(src CapacitySource) *Gate {
	return &Gate{src: src}
}

// Sentinel errors returned by Gate.Evaluate. Callers — currently the kb-32
// pipeline — convert these into a PipelineResult.HoldReason so the held
// recommendation is surfaced as a non-error outcome (State="detected"),
// matching the existing Stage 4 appropriateness-gate hold pattern.
var (
	// ErrSDMRequired is returned when CognitiveCapacity is uncertain,
	// moderate_impairment, or severe_impairment AND Assessment.SDMRequired
	// is false. Per Guidelines §6.4–6.5, the conservative default is to
	// hold until the SDM workflow is in place.
	ErrSDMRequired = errors.New("capacity: SDM workflow required for this resident")

	// ErrRestrictivePracticeNoConsent is returned when the recommendation is
	// for a restrictive practice AND either no consent record exists OR the
	// existing consent.Allows returns false (expired, withdrawn, alternatives
	// not documented, etc.). Per Guidelines §6.6 the practice MUST NOT
	// proceed without active, audit-defensible consent.
	ErrRestrictivePracticeNoConsent = errors.New("capacity: restrictive practice recommended but no active consent")
)

// Evaluate runs the two-step gate: capacity then (if relevant) restrictive
// practice consent.
//
// restrictivePractice is the consent_extension.PracticeType the recommendation
// maps to (one of PracticeChemicalRestraint, PracticePhysicalRestraint,
// PracticeEnvironmentalRestraint, PracticeSeclusion) or the empty
// PracticeType("") when the recommendation is not a restrictive practice.
// When empty, only the capacity check runs.
//
// Return values:
//   - nil                              → proceed (Stage 4 follows)
//   - ErrSDMRequired                   → capacity check failed; hold
//   - ErrRestrictivePracticeNoConsent  → consent check failed; hold
//   - any other error                  → infrastructure failure from src
func (g *Gate) Evaluate(ctx context.Context, residentID uuid.UUID, restrictivePractice consent_extension.PracticeType) error {
	assessment, err := g.src.AssessmentFor(ctx, residentID)
	if err != nil {
		return err
	}

	// Capacity check (Guidelines §6.4–6.5).
	// Mild impairment intentionally excluded per §6.6: mild capacity retains
	// decisional capacity for routine matters and does not trigger SDM.
	if requiresSDM(assessment.CognitiveCapacity) && !assessment.SDMRequired {
		return ErrSDMRequired
	}

	// Restrictive-practice consent check (Guidelines §6.6).
	// Only runs when the recommendation maps to a restrictive practice type.
	if restrictivePractice == "" {
		return nil
	}
	consent, err := g.src.RestrictivePracticeConsentFor(ctx, residentID, restrictivePractice)
	if err != nil {
		return err
	}
	if consent == nil || !consent.Allows(assessment.AssessedAt) {
		return ErrRestrictivePracticeNoConsent
	}
	return nil
}

// requiresSDM reports whether the given CognitiveCapacity triggers the
// SDM-required gate per Guidelines §6.4–6.6.
//
// Per §6.6, MildImpairment is explicitly NOT included — mild residents
// retain decisional capacity for routine clinical matters.
func requiresSDM(c vulnerability.CognitiveCapacity) bool {
	switch c {
	case vulnerability.CapacityUncertain,
		vulnerability.CapacityModerateImpairment,
		vulnerability.CapacitySevereImpairment:
		return true
	default:
		return false
	}
}
