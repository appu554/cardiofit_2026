// aggregation_adapter.go bridges the audit package's Emitter shape to
// the aggregation package's ViewRenderEmitter boundary interface so the
// view builder can fan view-render rows to the audit substrate without
// taking a direct dependency on this package's types.
package audit

import (
	"context"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
)

// ViewRenderAdapter wraps an Emitter to satisfy
// aggregation.ViewRenderEmitter. Each call produces an
// EventViewRender AuditEvent with severity 3 (moderate — view renders
// are observability events, not primary algorithmic decisions).
type ViewRenderAdapter struct {
	emitter Emitter
}

// NewViewRenderAdapter constructs a ViewRenderAdapter wrapping emitter.
func NewViewRenderAdapter(emitter Emitter) *ViewRenderAdapter {
	return &ViewRenderAdapter{emitter: emitter}
}

// EmitViewRender writes an EventViewRender row.
func (a *ViewRenderAdapter) EmitViewRender(ctx context.Context, req aggregation.WorkspaceRequest, layer int) error {
	if a == nil || a.emitter == nil {
		return nil
	}
	evt := AuditEvent{
		TraceID:      uuid.New(),
		EventType:    EventViewRender,
		Severity:     3,
		PharmacistID: req.PharmacistID,
		ResidentID:   req.ResidentID,
		SessionID:    req.SessionID,
		Subject:      "view_render",
		Payload: map[string]any{
			"layer":      layer,
			"entry_path": string(req.EntryPath),
		},
		OccurredAt: req.AsOf,
	}
	return a.emitter.Emit(ctx, evt)
}

// Compile-time assertion that ViewRenderAdapter satisfies the
// aggregation boundary interface.
var _ aggregation.ViewRenderEmitter = (*ViewRenderAdapter)(nil)

// Compile-time assertion that EscalationEventEmitter satisfies the
// aggregation boundary interface (it already has a Capture(ctx, ev)
// method with the right signature).
var _ aggregation.EscalationCapturer = (*EscalationEventEmitter)(nil)
