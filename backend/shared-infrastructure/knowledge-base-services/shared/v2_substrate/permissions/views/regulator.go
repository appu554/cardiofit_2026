package views

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// AuditPack is an audit-grade evidence pack with cryptographic provenance,
// suitable for regulatory consumption (AD-class data).
type AuditPack struct {
	PackID      uuid.UUID
	ContentHash string // must be non-empty; e.g. "sha256:<hex>"
	Items       []AuditItem
}

// AuditItem is a single auditable event within an AuditPack.
type AuditItem struct {
	ID        uuid.UUID
	Class     string    // e.g. "recommendation_decision", "consent_state_change"
	OriginRef uuid.UUID // reference to the originating entity
}

// RegulatorSource is implemented by the upstream substrate query layer.
type RegulatorSource interface {
	AuditPackFor(ctx context.Context, scope string, since time.Time, until time.Time) (AuditPack, error)
}

// ErrMissingProvenance is returned when an AuditPack lacks a content hash,
// indicating the pack cannot be treated as cryptographically provenance-bearing.
var ErrMissingProvenance = errors.New("views: regulator audit pack missing content hash provenance")

// RegulatorView returns audit-grade evidence packs for regulatory consumption.
// It validates that every returned pack carries a non-empty content hash
// before surfacing the data.
//
// Phase 1a stub: enforces provenance presence only.
//
// Phase 1c will own: AD-class data-sharing-agreement enforcement, ERM
// review of regulator queries (per Ethical Architecture Guidelines §4 +
// §10 ethics-based auditing), and data-minimisation for the cryptographic
// audit pack. See
// docs/superpowers/plans/2026-05-07-phase-1c-ethical-architecture-substrate.md.
type RegulatorView struct {
	src RegulatorSource
}

// NewRegulatorView constructs a RegulatorView backed by src.
func NewRegulatorView(src RegulatorSource) *RegulatorView {
	return &RegulatorView{src: src}
}

// AuditPack returns the audit evidence pack for the given scope and time window.
// Returns ErrMissingProvenance if the pack has no content hash.
func (v *RegulatorView) AuditPack(ctx context.Context, scope string, since time.Time, until time.Time) (AuditPack, error) {
	pack, err := v.src.AuditPackFor(ctx, scope, since, until)
	if err != nil {
		return AuditPack{}, err
	}
	if pack.ContentHash == "" {
		return AuditPack{}, ErrMissingProvenance
	}
	return pack, nil
}
