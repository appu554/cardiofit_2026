// EthicsLog substrate shapes — minimal copy of the shared Phase 1c
// ethics_log.Entry shape and the canonical EntryType / Severity constants
// used by kb-32's Stage 7 EvidenceTrace emitter and by the s2-aggregator
// audit package.
//
// Rationale (same as recommendation.go, prn_velocity.go, override.go): the
// canonical Logger / Store / Entry types live in
//   backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/ethics/ethics_log
// which is in a separate Go module (the knowledge-base-services repo).
// The s2-aggregator follows the shape-copy + structural-pin-test pattern
// to keep cross-module discipline: a thin local mirror plus a field-name
// pin test catches drift at CI time. Task 8 wires the production shared
// ethics_log.Logger via an adapter at the boundary.
//
// SOURCE OF TRUTH: shared/v2_substrate/ethics/ethics_log/logger.go.
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// EthicsLogEntryType classifies the kind of ethical event being logged.
// Mirrors ethics_log.EntryType — values must match the canonical string
// constants character-for-character.
type EthicsLogEntryType string

// Canonical EntryType constants from Phase 1c ethics_log. The pin test
// asserts these values match the canonical strings exactly.
const (
	// EthicsEntryTypeDecision records a primary algorithmic decision event.
	EthicsEntryTypeDecision EthicsLogEntryType = "decision"
	// EthicsEntryTypeConcernFlagged records a flagged ethical concern.
	EthicsEntryTypeConcernFlagged EthicsLogEntryType = "concern_flagged"
	// EthicsEntryTypeReviewRequested records a request for human/ERM review.
	EthicsEntryTypeReviewRequested EthicsLogEntryType = "review_requested"
	// EthicsEntryTypePatternDetected records detection of a systematic ethical pattern.
	EthicsEntryTypePatternDetected EthicsLogEntryType = "pattern_detected"
	// EthicsEntryTypeIncident records a confirmed ethical incident.
	EthicsEntryTypeIncident EthicsLogEntryType = "incident"
)

// Severity constants reflect the 1..5 scale used by Phase 1c ethics_log,
// where 1 is reserved for primary algorithmic decisions (kb-32 Stage 7
// uses Severity=1 per Phase 2-completion Task 4) and 5 is most severe.
const (
	// EthicsSeverityPrimaryDecision = 1: primary algorithmic decision event.
	EthicsSeverityPrimaryDecision = 1
	// EthicsSeverityLow = 2.
	EthicsSeverityLow = 2
	// EthicsSeverityModerate = 3.
	EthicsSeverityModerate = 3
	// EthicsSeverityHigh = 4.
	EthicsSeverityHigh = 4
	// EthicsSeverityCritical = 5.
	EthicsSeverityCritical = 5
)

// EthicsLogEntry mirrors the canonical ethics_log.Entry shape.
//
// Notable: there is NO generic Payload field on the canonical Entry —
// kb-32's Stage 7 emitter (Phase 2-completion Task 4) serializes the
// structured payload as JSON into Description. The s2-aggregator audit
// package follows the same convention.
//
// SOURCE OF TRUTH: shared/v2_substrate/ethics/ethics_log/logger.go (Entry).
type EthicsLogEntry struct {
	ID                 uuid.UUID
	DecisionID         uuid.UUID
	EntryType          EthicsLogEntryType
	Severity           int
	Description        string
	Reviewer           *string
	ReviewOutcome      *string
	RemediationActions []string
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
