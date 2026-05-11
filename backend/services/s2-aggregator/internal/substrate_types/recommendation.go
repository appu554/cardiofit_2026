// Recommendation substrate shapes — minimal copies of kb-32 types used by
// the s2-aggregator's pending recommendations pipeline.
//
// Rationale (same as prn_velocity.go): s2-aggregator is a separate Go
// module from kb-32. kb-32's authoritative types live behind internal/
// and are unimportable. The shape-copy + structural-pin-test pattern
// established in Task 3 is reused here.
//
// Each type below carries a SOURCE OF TRUTH comment naming the canonical
// kb-32 file. Pin tests in recommendation_pin_test.go enforce field-name
// parity at CI time so drift is caught early.
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// RecommendationPacket is the s2-aggregator's view of kb-32 generator.Packet.
//
// SOURCE OF TRUTH: backend/shared-infrastructure/knowledge-base-services/
// kb-32-recommendation-craft/internal/generator/recommendation.go (Packet).
//
// Field-shape note: kb-32 Packet carries an embedded reasoning.ApplicableRule
// in its AppliedRule field. The aggregator only needs the RuleID + Urgency
// for rendering, so a slim AppliedRule shape is captured here rather than
// the full reasoning.ApplicableRule tree.
type RecommendationPacket struct {
	RecommendationID uuid.UUID
	AuthorID         uuid.UUID
	Type             string // "STOP" | "MONITOR" | "DOSE_CHANGE" | "ADD"
	Sections         map[string]string
	AppliedRule      AppliedRule
	SnapshotRef      uuid.UUID
}

// AppliedRule is the slim s2-side projection of kb-32
// reasoning.ApplicableRule. Only the fields the pending-recommendation
// renderer needs are pinned.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/reasoning
// (ApplicableRule).
type AppliedRule struct {
	RuleID  string
	Type    string
	Urgency string
}

// AssessmentScores mirrors kb-32 appropriateness.Assessment — the five-
// dimension clinical-safety rubric per Phase 2-completion Task 2.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/appropriateness/
// checker.go (Assessment).
type AssessmentScores struct {
	ClinicalWarrant        int
	EvidenceSolidity       int
	AlternativesConsidered int
	RestraintConsidered    int
	GoalsOfCareAlignment   int
}

// Citation mirrors kb-32 citations.RecommendationCitation — the immutable
// fire-time source pin per Guidelines §6.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/citations/versioning.go
// (RecommendationCitation).
type Citation struct {
	RecommendationID string
	SourceID         string
	Version          string
	PinnedAt         time.Time
}
