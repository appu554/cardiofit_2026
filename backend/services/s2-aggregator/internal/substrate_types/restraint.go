// Restraint substrate shapes — Phase 1 advisory-only pairing per S2 v1.0
// Part 7.
//
// SHAPE-DRIFT NOTE: kb-32 currently ships restraint.Signal (see kb-32/
// internal/restraint/signaler.go) but that type is a snapshot-time
// detector output — it has no SignalID, no PairedRecommendationID, no
// TriggeredAt. The S2 workspace pairing-and-acknowledgment workflow per
// v1.0 Part 7.1–7.4 needs all of those.
//
// We stub the persistent shape here and mark the drift:
//
// TODO(kb-32 alignment when restraint subsystem matures):
//
//	When kb-32's restraint subsystem grows a persistence layer with
//	signal_id + paired_recommendation_id + triggered_at columns
//	(expected per the v1.1 restraint template work), retire this stub
//	and import the canonical shape via the SubstrateClient interface.
//	Canonical path will be kb-32/internal/restraint/persisted.go
//	(speculative — name not yet committed in kb-32).
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// RestraintSignal is the persistent S2-side projection of a kb-32
// restraint detector output. Severity is an integer rather than the
// kb-32 "red"|"amber" string so the s2 renderer can sort signals
// numerically; mapping is amber=2, red=3 (placeholder until kb-32
// formalises).
//
// SubstrateRefs are attached at the aggregation layer (not here) — the
// substrate-level shape is intentionally free of rendering concerns.
type RestraintSignal struct {
	SignalID               uuid.UUID
	Type                   string // e.g., "care_intensity_transition_recent"
	Severity               int
	PairedRecommendationID uuid.UUID // uuid.Nil when unpaired (panel-level signal)
	TriggeredAt            time.Time
	SubstrateID            uuid.UUID // the kb-32 substrate row that triggered the signal
	SubstrateSource        string    // e.g., "kb-32-restraint"
}

// RestraintAcknowledgment captures the pharmacist's response to a
// restraint signal per v1.0 Part 7.2. Phase 1 commitment: ADVISORY ONLY
// — acknowledgment is logged in EvidenceTrace; the platform does NOT
// auto-suppress paired recommendations.
//
// Decision is one of:
//   - "acknowledge_advisory"            — proceed / defer / case-by-case
//   - "invoke_safety_critical_bypass"   — bypass with mandatory reasoning
type RestraintAcknowledgment struct {
	SignalID       uuid.UUID
	PharmacistID   uuid.UUID
	AcknowledgedAt time.Time
	Decision       string
}

// Decision constants. Kept as a closed set so renderer + audit pipeline
// can switch exhaustively.
const (
	RestraintDecisionAcknowledgeAdvisory  = "acknowledge_advisory"
	RestraintDecisionSafetyCriticalBypass = "invoke_safety_critical_bypass"
)
