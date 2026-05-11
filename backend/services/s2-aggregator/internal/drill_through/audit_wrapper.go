// audit_wrapper.go adds best-effort EventDrillThrough audit emission
// around the substrate-drill-through, trajectory-history, and
// negative-evidence handlers per S2 v1.0 Part 13.1 (every drill-through
// is audited).
//
// Best-effort semantics: emission failure is logged but never fails the
// drill-through itself (Task 7 contract — see audit/evidence_trace.go
// failure-mode header).
package drill_through

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/audit"
)

// AuditedDrillThrough wraps a drill-through with audit-emission. The
// caller invokes the inner handler then calls Record with the trace
// metadata; the wrapper handles the audit row.
type AuditedDrillThrough struct {
	emitter audit.Emitter
}

// NewAuditedDrillThrough constructs an AuditedDrillThrough wrapping
// emitter. nil emitter disables audit (used by tests that don't
// exercise the audit path).
func NewAuditedDrillThrough(emitter audit.Emitter) *AuditedDrillThrough {
	return &AuditedDrillThrough{emitter: emitter}
}

// RecordSubstrateObservation emits an EventDrillThrough audit row for
// the substrate-observation drill-through that just executed. Returns
// nil on emitter-failure (best-effort) but logs the error.
func (a *AuditedDrillThrough) RecordSubstrateObservation(
	ctx context.Context,
	pharmacistID, residentID, sessionID uuid.UUID,
	ref aggregation.SubstrateRef,
) {
	a.emit(ctx, pharmacistID, residentID, sessionID, "substrate_observation", map[string]any{
		"source":      ref.Source,
		"ref_id":      ref.ID,
		"description": ref.Description,
	})
}

// RecordTrajectoryHistory emits an EventDrillThrough audit row for the
// trajectory-history drill-through.
func (a *AuditedDrillThrough) RecordTrajectoryHistory(
	ctx context.Context,
	pharmacistID, residentID, sessionID uuid.UUID,
	parameter string,
) {
	a.emit(ctx, pharmacistID, residentID, sessionID, "trajectory_history", map[string]any{
		"parameter": parameter,
	})
}

// RecordNegativeEvidence emits an EventDrillThrough audit row for the
// negative-evidence drill-through.
func (a *AuditedDrillThrough) RecordNegativeEvidence(
	ctx context.Context,
	pharmacistID, residentID, sessionID uuid.UUID,
	claim string,
) {
	a.emit(ctx, pharmacistID, residentID, sessionID, "negative_evidence", map[string]any{
		"claim": claim,
	})
}

// emit is the shared best-effort dispatch.
func (a *AuditedDrillThrough) emit(
	ctx context.Context,
	pharmacistID, residentID, sessionID uuid.UUID,
	subject string,
	payload map[string]any,
) {
	if a == nil || a.emitter == nil {
		return
	}
	evt := audit.AuditEvent{
		TraceID:      uuid.New(),
		EventType:    audit.EventDrillThrough,
		Severity:     3,
		PharmacistID: pharmacistID,
		ResidentID:   residentID,
		SessionID:    sessionID,
		Subject:      subject,
		Payload:      payload,
		OccurredAt:   time.Now().UTC(),
	}
	if err := a.emitter.Emit(ctx, evt); err != nil {
		log.Printf("s2-aggregator: drill-through audit emission failed for subject=%s: %v", subject, err)
	}
}
