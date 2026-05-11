// Package lifecycle — evidence_trace.go implements Stage 7 of the craft engine:
// emission of an immutable audit-defensibility record on every successful
// detected → drafted recommendation transition.
//
// Phase 2-completion Task 4 ("EvidenceTrace emission") closes pre-pilot
// production blocker #3 — the audit-defensibility ledger that lets a regulator
// reconstruct exactly why and when each recommendation drafted.
//
// # Dual emission
//
// Every successful pipeline run emits two records:
//
//  1. An entry on the Phase 1c EthicsLog (EntryType=decision, Severity=1) so
//     the trace participates in the cross-service ethics audit feed.
//  2. A row in the kb-32 evidence_trace_entries table (migration 045) so the
//     trace is durably queryable from the craft service itself.
//
// CompositeEmitter fans out to both. Fail-hard semantics: ANY emitter error
// propagates and fails the pipeline run.
//
// VisibilityClass: AD — fire-time evidence trace per Guidelines §4 Stage 7
package lifecycle

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/citations"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// DraftedTransitionEntry is the immutable Stage 7 ledger record produced when
// a recommendation transitions from detected to drafted. It captures every
// element a regulator needs to reconstruct the recommendation event:
//
//   - which rule fired (RuleID)
//   - the deterministic content hash of the framed recommendation
//   - the appropriateness assessment that cleared the Stage 4 gate
//   - the fire-time citation pins (Stage 5b)
//   - the urgency tier
//   - the wall-clock fire time
//
// All fields are populated by the pipeline at the success-only return; held
// recommendations (capacity hold, appropriateness hold) do NOT produce a
// DraftedTransitionEntry.
type DraftedTransitionEntry struct {
	// RecommendationID is the unique identifier for the drafted recommendation.
	RecommendationID uuid.UUID `json:"recommendation_id"`

	// AuthorID is the UUID of the human or system author the packet is attributed to.
	AuthorID uuid.UUID `json:"author_id"`

	// RuleID identifies the CQL rule that produced the recommendation.
	RuleID string `json:"rule_id"`

	// ContentHash is the SHA-256 hex digest from Stage 5 (framing.ContentHash).
	ContentHash string `json:"content_hash"`

	// Assessment is the appropriateness Assessment that cleared the Stage 4 gate.
	Assessment appropriateness.Assessment `json:"assessment"`

	// Citations is the slice of fire-time citation pins produced in Stage 5b.
	// May be empty when the rule pack ships no EvidenceAnchors.
	Citations []citations.RecommendationCitation `json:"citations"`

	// Urgency is the urgency tier tag derived from the ClinicalSnapshot.
	Urgency string `json:"urgency"`

	// FiredAt is the UTC timestamp of the detected → drafted transition.
	FiredAt time.Time `json:"fired_at"`
}

// EvidenceTraceEmitter is the port through which the pipeline emits a
// DraftedTransitionEntry. Production wiring uses CompositeEmitter to fan out
// to both the EthicsLog and Postgres.
type EvidenceTraceEmitter interface {
	// EmitDraftedTransition persists entry. A non-nil error fails the pipeline
	// run — the caller MUST NOT treat emission errors as best-effort.
	EmitDraftedTransition(ctx context.Context, entry DraftedTransitionEntry) error
}

// ---------------------------------------------------------------------------
// EthicsLogEmitter
// ---------------------------------------------------------------------------

// EthicsLogEmitter writes a DraftedTransitionEntry to the Phase 1c EthicsLog
// substrate as an EntryTypeDecision entry with Severity=1.
//
// The entry's Description field carries the JSON-serialized
// DraftedTransitionEntry so a Layer-4 audit reader can recover the full
// record. DecisionID is set to entry.RecommendationID so the entry is keyed
// by the recommendation it documents.
type EthicsLogEmitter struct {
	logger *ethics_log.Logger
}

// NewEthicsLogEmitter constructs an EthicsLogEmitter that writes through
// logger. logger must be non-nil.
func NewEthicsLogEmitter(logger *ethics_log.Logger) *EthicsLogEmitter {
	return &EthicsLogEmitter{logger: logger}
}

// EmitDraftedTransition serializes entry to JSON and appends it to the
// EthicsLog as an EntryTypeDecision entry with Severity=1.
func (e *EthicsLogEmitter) EmitDraftedTransition(ctx context.Context, entry DraftedTransitionEntry) error {
	if e.logger == nil {
		return fmt.Errorf("ethics_log_emitter: logger is nil")
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("ethics_log_emitter: marshal entry: %w", err)
	}
	logEntry := ethics_log.Entry{
		DecisionID:  entry.RecommendationID,
		EntryType:   ethics_log.EntryTypeDecision,
		Severity:    1,
		Description: string(payload),
		CreatedAt:   entry.FiredAt,
		UpdatedAt:   entry.FiredAt,
	}
	if err := e.logger.Append(ctx, logEntry); err != nil {
		return fmt.Errorf("ethics_log_emitter: append: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PostgresEmitter
// ---------------------------------------------------------------------------

// PostgresEmitter writes a DraftedTransitionEntry to the kb-32
// evidence_trace_entries table (migration 045). The schema serializes the
// Assessment dimensions as a JSONB column and the citation pin set as a
// JSONB array so the row is self-contained for audit replay.
type PostgresEmitter struct {
	db *sql.DB
}

// NewPostgresEmitter constructs a PostgresEmitter over db. db must be
// non-nil and connected to a schema with migration 045 applied.
func NewPostgresEmitter(db *sql.DB) *PostgresEmitter {
	return &PostgresEmitter{db: db}
}

// EmitDraftedTransition inserts entry into evidence_trace_entries. The
// recommendation_id column is the primary key — duplicate emissions for the
// same recommendation are rejected by the DB so an audit-trail double-write
// surfaces as an error rather than a silent overwrite.
func (e *PostgresEmitter) EmitDraftedTransition(ctx context.Context, entry DraftedTransitionEntry) error {
	if e.db == nil {
		return fmt.Errorf("postgres_emitter: db is nil")
	}
	assessmentJSON, err := json.Marshal(entry.Assessment)
	if err != nil {
		return fmt.Errorf("postgres_emitter: marshal assessment: %w", err)
	}
	citationsJSON, err := json.Marshal(entry.Citations)
	if err != nil {
		return fmt.Errorf("postgres_emitter: marshal citations: %w", err)
	}
	const q = `
INSERT INTO evidence_trace_entries (
    recommendation_id,
    author_id,
    rule_id,
    content_hash,
    assessment,
    citations,
    urgency,
    fired_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	if _, err := e.db.ExecContext(ctx, q,
		entry.RecommendationID,
		entry.AuthorID,
		entry.RuleID,
		entry.ContentHash,
		assessmentJSON,
		citationsJSON,
		entry.Urgency,
		entry.FiredAt,
	); err != nil {
		return fmt.Errorf("postgres_emitter: insert: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// CompositeEmitter
// ---------------------------------------------------------------------------

// CompositeEmitter fans EmitDraftedTransition out to a sequence of underlying
// emitters. It is the production wiring shape: both EthicsLog and Postgres
// must succeed for the pipeline run to succeed.
//
// Failure semantics are fail-fast: on the first emitter error, CompositeEmitter
// returns immediately without invoking subsequent emitters. This preserves
// the fail-hard contract — a partial emission is still treated as an
// audit-trail miss and surfaces as a pipeline error.
type CompositeEmitter struct {
	emitters []EvidenceTraceEmitter
}

// NewCompositeEmitter constructs a CompositeEmitter wrapping the supplied
// emitters in the order given. A nil or empty slice produces an emitter that
// succeeds without doing anything — callers should construct at least one
// underlying emitter for production use.
func NewCompositeEmitter(emitters ...EvidenceTraceEmitter) *CompositeEmitter {
	return &CompositeEmitter{emitters: emitters}
}

// EmitDraftedTransition invokes each underlying emitter in order. It returns
// the first non-nil error and stops; subsequent emitters are NOT invoked.
func (c *CompositeEmitter) EmitDraftedTransition(ctx context.Context, entry DraftedTransitionEntry) error {
	for i, em := range c.emitters {
		if em == nil {
			return fmt.Errorf("composite_emitter: emitter[%d] is nil", i)
		}
		if err := em.EmitDraftedTransition(ctx, entry); err != nil {
			return fmt.Errorf("composite_emitter: emitter[%d]: %w", i, err)
		}
	}
	return nil
}
