// Package citations — versioning.go defines the core data structures and
// value-object methods for citation source versioning.
//
// VisibilityClass: AD — citation versioning per Guidelines §6 audit defensibility
package citations

import (
	"time"
)

// ---------------------------------------------------------------------------
// VersionStatus
// ---------------------------------------------------------------------------

// VersionStatus represents the lifecycle state of a SourceVersion.
type VersionStatus string

const (
	// StatusActive is the current, authoritative version of a source.
	StatusActive VersionStatus = "active"

	// StatusAmended means this version was superseded by a newer version of the
	// same source. Existing citations remain valid — the amendment does NOT
	// retroactively invalidate pinned recommendations.
	StatusAmended VersionStatus = "amended"

	// StatusRetracted means the source has been withdrawn. Pinned citations retain
	// the retracted version so dashboards can surface a "retracted source" flag.
	StatusRetracted VersionStatus = "retracted"

	// StatusSuperseded means this entire source has been replaced by a different
	// source identifier. Ongoing recommendations should be re-cited against the
	// new source.
	StatusSuperseded VersionStatus = "superseded"
)

// IsValidStatus returns true when s is one of the four canonical VersionStatus
// values accepted by the citation registry.
func IsValidStatus(s string) bool {
	switch VersionStatus(s) {
	case StatusActive, StatusAmended, StatusRetracted, StatusSuperseded:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// SourceVersion
// ---------------------------------------------------------------------------

// SourceVersion is a point-in-time snapshot of an evidence source. The
// closed-open interval [EffectiveFrom, EffectiveTo) defines when this version
// is authoritative. A nil EffectiveTo means the version has no planned expiry
// (i.e. it is currently active).
//
// VisibilityClass: AD — audit-defensible source record
type SourceVersion struct {
	// SourceID is the stable identifier for the evidence source (e.g. a
	// guideline DOI, a drug database key, or an internal clinical KB ID).
	SourceID string

	// Version is the version label within the source (e.g. "1", "2", "2024-01").
	Version string

	// EffectiveFrom is the inclusive start of this version's authority window.
	EffectiveFrom time.Time

	// EffectiveTo is the exclusive end of the authority window. Nil means open.
	EffectiveTo *time.Time

	// ContentHash is a SHA-256 hex digest of the source content at this version.
	// It enables integrity verification without re-fetching the source.
	ContentHash string

	// Status reflects the current lifecycle state of this version.
	Status VersionStatus
}

// ActiveAt returns true when this SourceVersion is the authoritative version
// at the given point in time. The interval semantics are closed-open:
//
//	[EffectiveFrom, EffectiveTo)
//
// This means:
//   - asOf == EffectiveFrom → active (inclusive lower bound)
//   - asOf == EffectiveTo   → NOT active (exclusive upper bound)
//   - EffectiveTo == nil    → active for any asOf >= EffectiveFrom (open interval)
func (v SourceVersion) ActiveAt(asOf time.Time) bool {
	if asOf.Before(v.EffectiveFrom) {
		return false
	}
	if v.EffectiveTo != nil && !asOf.Before(*v.EffectiveTo) {
		// asOf >= EffectiveTo → outside the closed-open window
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// RecommendationCitation
// ---------------------------------------------------------------------------

// RecommendationCitation is the immutable fire-time pin linking a
// recommendation to the exact source version that was active when the
// recommendation was generated.
//
// The audit-defensibility guarantee: once a RecommendationCitation is created,
// subsequent amendments or retractions of the source do NOT change this record.
// The Version field points permanently to the SourceVersion that was current
// at PinnedAt.
//
// VisibilityClass: AD — fire-time citation per Guidelines §6
type RecommendationCitation struct {
	// RecommendationID is the UUID of the clinical recommendation (references
	// the recommendations table per Plan 0.1).
	RecommendationID string

	// SourceID is the stable identifier of the evidence source.
	SourceID string

	// Version is the exact version of SourceID that was active at PinnedAt.
	// This field is immutable after creation.
	Version string

	// PinnedAt is the UTC timestamp at which the recommendation was generated
	// and this citation was pinned. It equals the "fire time" of the recommendation.
	PinnedAt time.Time
}
