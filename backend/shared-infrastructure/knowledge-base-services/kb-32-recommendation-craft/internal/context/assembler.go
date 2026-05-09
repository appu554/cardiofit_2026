// Package context provides Stage 1 of the six-stage rendering pipeline:
// clinical context assembly.
//
// VisibilityClass: PDP — pharmacist's clinical context for own recommendation work
//
// The Assembler pulls a ClinicalSnapshot from a SubstrateClient and surfaces it
// for consumption by the reasoning-chain and generator stages. The SubstrateClient
// interface decouples the assembler from any specific transport or storage; a
// Postgres-backed implementation is deferred to Task 13 wiring.
package context

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// ClinicalSnapshot
// ---------------------------------------------------------------------------

// ClinicalSnapshot is the craft-engine's own view of a resident's clinical state,
// optimised for recommendation-generation consumption. It is intentionally
// separate from the shared substrate's resident snapshot types so that kb-32 can
// evolve its fields without coupling to platform-wide schema changes.
type ClinicalSnapshot struct {
	// ResidentID identifies the resident this snapshot belongs to.
	ResidentID uuid.UUID

	// EGFR is the estimated glomerular filtration rate (mL/min/1.73 m²).
	// Used by the reasoning chain to gate renal-dose adjustments.
	EGFR float64

	// DBI is the Drug Burden Index score.
	DBI float64

	// ACB is the Anticholinergic Cognitive Burden score.
	ACB int

	// CFS is the Clinical Frailty Scale score (1–9).
	CFS int

	// CareIntensity describes the resident's goals-of-care trajectory.
	// Valid values: "active", "comfort", "palliative", "end_of_life".
	// Use IsValidCareIntensity to validate.
	CareIntensity string

	// RecentFall72h is true when the resident has had a documented fall
	// within the past 72 hours.
	RecentFall72h bool

	// RecentAdmission72h is true when the resident has been admitted
	// to hospital within the past 72 hours.
	RecentAdmission72h bool

	// AssessedAt is the wall-clock time at which this snapshot was captured.
	// Used by Task 9's appropriateness checker to detect stale state.
	AssessedAt time.Time
}

// Stale reports whether the snapshot was assessed more than ttl ago.
// A stale snapshot may affect goals-of-care alignment scoring in the
// appropriateness gate (Task 9).
func (s ClinicalSnapshot) Stale(ttl time.Duration) bool {
	return time.Since(s.AssessedAt) > ttl
}

// ---------------------------------------------------------------------------
// Care intensity validation
// ---------------------------------------------------------------------------

// validCareIntensities is the canonical set of accepted care-intensity values.
var validCareIntensities = map[string]struct{}{
	"active":      {},
	"comfort":     {},
	"palliative":  {},
	"end_of_life": {},
}

// IsValidCareIntensity reports whether s is one of the four recognised
// care-intensity descriptors. The check is case-sensitive.
func IsValidCareIntensity(s string) bool {
	_, ok := validCareIntensities[s]
	return ok
}

// ---------------------------------------------------------------------------
// SubstrateClient interface
// ---------------------------------------------------------------------------

// SubstrateClient is the port through which the Assembler retrieves resident
// clinical state. Implementations are expected to honour context cancellation.
//
// The Postgres-backed implementation is wired in Task 13; tests use
// InMemorySubstrateClient defined in assembler_test.go.
type SubstrateClient interface {
	SnapshotFor(ctx context.Context, residentID uuid.UUID) (ClinicalSnapshot, error)
}

// ---------------------------------------------------------------------------
// Assembler
// ---------------------------------------------------------------------------

// Assembler is Stage 1 of the rendering pipeline. It retrieves a
// ClinicalSnapshot for the given resident and makes it available to
// downstream stages.
type Assembler struct {
	src SubstrateClient
}

// NewAssembler constructs an Assembler backed by src.
func NewAssembler(src SubstrateClient) *Assembler {
	return &Assembler{src: src}
}

// Assemble retrieves the ClinicalSnapshot for residentID from the configured
// SubstrateClient. It checks ctx for cancellation before delegating to the
// source, ensuring that a pre-cancelled context is never forwarded silently.
func (a *Assembler) Assemble(ctx context.Context, residentID uuid.UUID) (ClinicalSnapshot, error) {
	// Honour pre-cancelled contexts before incurring any I/O.
	if err := ctx.Err(); err != nil {
		return ClinicalSnapshot{}, fmt.Errorf("context cancelled before substrate call: %w", err)
	}

	snap, err := a.src.SnapshotFor(ctx, residentID)
	if err != nil {
		return ClinicalSnapshot{}, fmt.Errorf("assembler: substrate error for resident %v: %w", residentID, err)
	}
	return snap, nil
}
