// Package audit implements the s2-aggregator's EvidenceTrace integration,
// visibility-class enforcement, and EscalationEvent capture per S2 v1.0
// Part 13 (audit trail integration) and the S2 Adaptive Cognition
// Architectural Commitment Addendum Part 4.6 (audit trail as shared
// primitive) and Part 5.5 (cognitive escalation log-only commitment).
//
// Emission model:
//
//   - AuditEvent is the canonical s2-aggregator audit shape, covering the
//     five v1.0 Part 13.1 event categories: view rendering, pharmacist
//     actions, drill-throughs, system lifecycle events, and cognitive
//     escalation.
//
//   - Emitter is the persistence port. Three implementations:
//     PostgresEmitter writes the local s2_audit_events table (migration 004
//     — the local-canonical audit substrate); EthicsLogEmitter fans the
//     event out to the shared Phase 1c ethics_log; CompositeEmitter chains
//     them with fail-fast semantics on the first underlying error.
//
//   - MemoryEmitter is the in-package test fake with a retrieval API.
//
// Failure-mode contract (CRITICAL — differs from kb-32):
//
//   - kb-32's Stage 7 EvidenceTrace is fail-hard: any emitter error fails
//     the pipeline run because the trace IS the audit substrate.
//
//   - s2-aggregator's primary audit substrate is the local s2_audit_events
//     table (PostgresEmitter). The shared ethics_log fan-out is a
//     secondary downstream copy. Within this package the CompositeEmitter
//     is fail-fast (same as kb-32), but the boundary callers (action
//     handlers, view builder, drill-through handlers) treat audit
//     emission as best-effort: emission failure is logged but does NOT
//     fail the pharmacist action. This is the deliberate inversion
//     documented in Task 7 plan — the architectural rationale differs.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// AuditEventType classifies the kind of S2 audit event being recorded.
// The five values correspond directly to S2 v1.0 Part 13.1's five
// categories.
type AuditEventType string

const (
	// EventViewRender records every S2 workspace render (entry path,
	// layer, content rendered) per v1.0 Part 13.1.
	EventViewRender AuditEventType = "view_render"
	// EventPharmacistAction records each of the eleven Part 12.1
	// pharmacist actions per v1.0 Part 13.1.
	EventPharmacistAction AuditEventType = "pharmacist_action"
	// EventDrillThrough records substrate observation / trajectory /
	// negative-evidence drill-throughs per v1.0 Part 13.1.
	EventDrillThrough AuditEventType = "drill_through"
	// EventSystemLifecycle records recommendation lifecycle transitions
	// and other system events visible in S2 per v1.0 Part 13.1.
	EventSystemLifecycle AuditEventType = "system_lifecycle"
	// EventCognitiveEscalation records pharmacist layer-escalation per
	// Addendum Part 5. LOG-ONLY in Phase 1 per Addendum Part 5.5; the
	// emitter writes, no reader exists.
	EventCognitiveEscalation AuditEventType = "cognitive_escalation"
)

// AuditEvent is the canonical audit record shape for the s2-aggregator.
// It carries the structural fields S2 v1.0 Part 13.2 specifies for
// S2AuditEvent: pharmacist + resident + session identity, event type,
// substrate subject, free-form payload, and trace identifiers.
type AuditEvent struct {
	// TraceID is the unique identifier for this audit row.
	TraceID uuid.UUID
	// EventType is one of the five v1.0 Part 13.1 categories.
	EventType AuditEventType
	// Severity is the 1..5 scale shared with the Phase 1c ethics_log.
	// Severity=1 is reserved for primary algorithmic / cognitive events.
	Severity int
	// PharmacistID identifies the pharmacist whose action is being audited.
	PharmacistID uuid.UUID
	// ResidentID identifies the resident in S2 context (may be zero for
	// session-lifecycle events that span residents).
	ResidentID uuid.UUID
	// SessionID coheres events into a single pharmacist S2 session.
	SessionID uuid.UUID
	// Subject is a free-form short tag (e.g., recommendation_id,
	// "view_render", action enum value) for log-search ergonomics.
	Subject string
	// Payload carries the structured event-type-specific detail.
	Payload map[string]any
	// OccurredAt is the wall-clock time the event occurred.
	OccurredAt time.Time
}

// Emitter is the persistence port for AuditEvent. Production wiring
// composes a PostgresEmitter (local-canonical) and an EthicsLogEmitter
// (downstream copy) via CompositeEmitter.
type Emitter interface {
	// Emit persists the event. A non-nil error indicates persistence
	// failure; CompositeEmitter is fail-fast on the first error.
	Emit(ctx context.Context, entry AuditEvent) error
}

// ---------------------------------------------------------------------------
// EthicsLogEmitter
// ---------------------------------------------------------------------------

// Logger is the s2-aggregator-local boundary interface for the shared
// Phase 1c ethics_log.Logger. Task 8 wires the production logger via an
// adapter that satisfies this interface; the in-package fake satisfies
// it directly for tests.
//
// Append must accept the local-shape EthicsLogEntry; the adapter
// translates field-for-field to the canonical ethics_log.Entry at the
// boundary.
type Logger interface {
	Append(ctx context.Context, entry substrate_types.EthicsLogEntry) error
}

// EthicsLogEmitter writes an AuditEvent to the shared Phase 1c
// ethics_log substrate by serializing it as JSON into the Description
// field of an EthicsLogEntry — the same pattern Phase 2-completion
// Task 4 (kb-32 Stage 7) established. DecisionID is set to the
// AuditEvent's TraceID so each entry is keyed by the audit row it
// documents.
type EthicsLogEmitter struct {
	logger Logger
}

// NewEthicsLogEmitter constructs an EthicsLogEmitter wrapping logger.
// logger must be non-nil at first call (verified inside Emit so the
// constructor remains side-effect-free).
func NewEthicsLogEmitter(logger Logger) *EthicsLogEmitter {
	return &EthicsLogEmitter{logger: logger}
}

// Emit serializes evt and appends it to the wrapped Logger.
func (e *EthicsLogEmitter) Emit(ctx context.Context, evt AuditEvent) error {
	if e == nil || e.logger == nil {
		return errors.New("ethics_log_emitter: nil logger")
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("ethics_log_emitter: marshal event: %w", err)
	}
	entry := substrate_types.EthicsLogEntry{
		DecisionID:  evt.TraceID,
		EntryType:   substrate_types.EthicsEntryTypeDecision,
		Severity:    evt.Severity,
		Description: string(payload),
		CreatedAt:   evt.OccurredAt,
		UpdatedAt:   evt.OccurredAt,
	}
	if err := e.logger.Append(ctx, entry); err != nil {
		return fmt.Errorf("ethics_log_emitter: append: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PostgresEmitter
// ---------------------------------------------------------------------------

// PostgresExecer is the minimal DB port PostgresEmitter requires. The
// adapter in cmd/server (Task 8 wiring) supplies a *sql.DB; this package
// only needs ExecContext-style access so tests can stub it.
type PostgresExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (any, error)
}

// PostgresEmitter writes an AuditEvent to the local s2_audit_events
// table (migration 004). This table is the s2-aggregator's
// LOCAL-CANONICAL audit substrate — every action / view / drill-through
// row written here is the source of truth for s2 audit replay. The
// EthicsLog fan-out is a secondary downstream copy.
type PostgresEmitter struct {
	db PostgresExecer
}

// NewPostgresEmitter constructs a PostgresEmitter over db. db must be
// non-nil and connected to a schema with migration 004 applied.
func NewPostgresEmitter(db PostgresExecer) *PostgresEmitter {
	return &PostgresEmitter{db: db}
}

// Emit inserts evt into s2_audit_events. The payload column is JSONB.
func (e *PostgresEmitter) Emit(ctx context.Context, evt AuditEvent) error {
	if e == nil || e.db == nil {
		return errors.New("postgres_emitter: nil db")
	}
	payloadJSON, err := json.Marshal(evt.Payload)
	if err != nil {
		return fmt.Errorf("postgres_emitter: marshal payload: %w", err)
	}
	const q = `
INSERT INTO s2_audit_events (
    trace_id, event_type, severity, pharmacist_id,
    resident_id, session_id, subject, payload, occurred_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`
	if _, err := e.db.ExecContext(ctx, q,
		evt.TraceID,
		string(evt.EventType),
		evt.Severity,
		evt.PharmacistID,
		nullableUUID(evt.ResidentID),
		nullableUUID(evt.SessionID),
		evt.Subject,
		payloadJSON,
		evt.OccurredAt,
	); err != nil {
		return fmt.Errorf("postgres_emitter: insert: %w", err)
	}
	return nil
}

// nullableUUID returns nil for uuid.Nil so the column receives SQL NULL
// rather than the zero-UUID sentinel.
func nullableUUID(u uuid.UUID) any {
	if u == uuid.Nil {
		return nil
	}
	return u
}

// ---------------------------------------------------------------------------
// CompositeEmitter
// ---------------------------------------------------------------------------

// CompositeEmitter fans Emit out to a sequence of underlying Emitters
// with fail-fast semantics on the first error — same shape as the kb-32
// Stage 7 CompositeEmitter (Phase 2-completion Task 4).
type CompositeEmitter struct {
	emitters []Emitter
}

// NewCompositeEmitter constructs a CompositeEmitter wrapping emitters in
// the order given.
func NewCompositeEmitter(emitters ...Emitter) *CompositeEmitter {
	return &CompositeEmitter{emitters: emitters}
}

// Emit invokes each underlying emitter in order. It returns the first
// non-nil error and stops; subsequent emitters are NOT invoked.
func (c *CompositeEmitter) Emit(ctx context.Context, evt AuditEvent) error {
	for i, em := range c.emitters {
		if em == nil {
			return fmt.Errorf("composite_emitter: emitter[%d] is nil", i)
		}
		if err := em.Emit(ctx, evt); err != nil {
			return fmt.Errorf("composite_emitter: emitter[%d]: %w", i, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// MemoryEmitter
// ---------------------------------------------------------------------------

// MemoryEmitter is an in-memory Emitter for tests. It retains every
// emitted event under a mutex and exposes retrieval helpers so callers
// can assert on the audit trail.
type MemoryEmitter struct {
	mu     sync.Mutex
	events []AuditEvent
}

// NewMemoryEmitter returns an empty MemoryEmitter.
func NewMemoryEmitter() *MemoryEmitter { return &MemoryEmitter{} }

// Emit appends the event to the in-memory log.
func (m *MemoryEmitter) Emit(_ context.Context, evt AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, evt)
	return nil
}

// Events returns a snapshot copy of all emitted events.
func (m *MemoryEmitter) Events() []AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AuditEvent, len(m.events))
	copy(out, m.events)
	return out
}

// Count returns the number of events emitted.
func (m *MemoryEmitter) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

// Last returns the most-recently-emitted event and true, or a zero
// event and false when the store is empty.
func (m *MemoryEmitter) Last() (AuditEvent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return AuditEvent{}, false
	}
	return m.events[len(m.events)-1], true
}

// EventsOfType returns all emitted events with the given EventType.
func (m *MemoryEmitter) EventsOfType(t AuditEventType) []AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AuditEvent, 0)
	for _, e := range m.events {
		if e.EventType == t {
			out = append(out, e)
		}
	}
	return out
}

// MemoryLogger is an in-memory test fake for the Logger interface.
type MemoryLogger struct {
	mu      sync.Mutex
	entries []substrate_types.EthicsLogEntry
}

// NewMemoryLogger returns an empty MemoryLogger.
func NewMemoryLogger() *MemoryLogger { return &MemoryLogger{} }

// Append appends entry to the in-memory log.
func (l *MemoryLogger) Append(_ context.Context, entry substrate_types.EthicsLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	return nil
}

// Entries returns a snapshot copy.
func (l *MemoryLogger) Entries() []substrate_types.EthicsLogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]substrate_types.EthicsLogEntry, len(l.entries))
	copy(out, l.entries)
	return out
}
