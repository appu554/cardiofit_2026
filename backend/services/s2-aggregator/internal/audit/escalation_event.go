// escalation_event.go captures pharmacist cognitive escalation events as
// audit rows. Phase 1 commitment per S2 Adaptive Cognition Architectural
// Commitment Addendum Part 5.5 is LOG-ONLY: every escalation is written
// to the audit substrate; NO read path exists for "give me pharmacist X's
// escalation patterns" because such a read would be surveillance per
// Addendum Part 5.2.
//
// ============================================================================
// MAY-NOT-USE — Phase 1 commitment per Addendum Part 5.2 lines 214–220
// ============================================================================
//
// Per ethical architecture §8 algorithmic management protections, the
// platform commits that pharmacist cognitive escalation patterns will
// NOT be used for:
//
//   1. Performance evaluation: a pharmacist who escalates less is not
//      "more efficient"; a pharmacist who escalates more is not
//      "less skilled".
//
//   2. Productivity surveillance: escalation patterns are not aggregated
//      for employer view (PEV visibility class is restricted).
//
//   3. Comparative pharmacist ranking: patterns are not used to rank
//      pharmacists relative to each other.
//
//   4. Decisions affecting pharmacist employment: patterns are not
//      shared with employer in any form that affects employment
//      decisions.
//
//   5. Differential treatment of pharmacists: patterns do not generate
//      differential platform behaviour that affects pharmacist work
//      conditions.
//
// CONSEQUENCE for engineering: this file MUST NOT export any function
// whose name suggests reading escalation patterns by pharmacist. The
// audit package's structural test (tests/structural/no_surveillance_reader_test.go)
// asserts this by grep against the source code: any addition of a
// GetEscalationPattern / QueryEscalationsForPharmacist /
// EscalationsByPharmacist function (or equivalent SQL in this package)
// fails the build. The safeguard cannot be bypassed without explicit
// Ethics Steering Committee approval (Addendum Part 5.5).
//
// Phase 4 (≥12 months evidence + ESC approval + external review +
// pharmacist self-visibility operational) gates any future read path.
// Until then this file is write-only.
// ============================================================================
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
)

// EscalationEventEmitter wraps an Emitter and translates an
// aggregation.EscalationEvent into an AuditEvent of type
// EventCognitiveEscalation. The wrapper exposes only Capture (a write
// path) — there is no read API by design.
type EscalationEventEmitter struct {
	emitter Emitter
}

// NewEscalationEventEmitter constructs an EscalationEventEmitter
// wrapping the supplied Emitter. emitter must be non-nil.
func NewEscalationEventEmitter(emitter Emitter) *EscalationEventEmitter {
	return &EscalationEventEmitter{emitter: emitter}
}

// Capture writes the escalation event as an AuditEvent. Severity=1 (the
// primary-decision severity) per kb-32 Stage 7 convention: cognitive
// escalation IS an algorithmic-significance event even though Phase 1
// does not act on it.
//
// LOG-ONLY commitment: this method is the ONLY public way escalation
// data enters the audit substrate. There is no corresponding read
// method on this type or anywhere else in the audit package.
func (e *EscalationEventEmitter) Capture(ctx context.Context, ev aggregation.EscalationEvent) error {
	if e == nil || e.emitter == nil {
		return errors.New("escalation_event_emitter: nil emitter")
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	trace := ev.AuditTraceID
	if trace == uuid.Nil {
		trace = uuid.New()
	}
	// Payload carries the structured layer-transition + trigger detail
	// so audit replay can reconstruct the event without a SubstrateRef
	// lookup. JSON-round-tripped through map[string]any for parity with
	// the AuditEvent.Payload shape.
	payload := map[string]any{
		"from_layer":   ev.FromLayer,
		"to_layer":     ev.ToLayer,
		"triggered_by": ev.TriggeredBy.String(),
	}
	// Defensive: ensure the payload is JSON-clean so downstream
	// PostgresEmitter's JSONB column does not reject the row.
	if _, err := json.Marshal(payload); err != nil {
		return err
	}
	audit := AuditEvent{
		TraceID:      trace,
		EventType:    EventCognitiveEscalation,
		Severity:     1,
		PharmacistID: ev.PharmacistID,
		ResidentID:   ev.ResidentID,
		SessionID:    ev.SessionID,
		Subject:      "cognitive_escalation",
		Payload:      payload,
		OccurredAt:   ev.Timestamp,
	}
	return e.emitter.Emit(ctx, audit)
}
