package aggregation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

// EscalationCapturer is the boundary interface defaultViewBuilder uses
// to delegate Phase 1 LOG-ONLY escalation capture to the audit package.
// The audit package's EscalationEventEmitter satisfies this interface
// at the boundary (set via WithEscalationCapturer); the default sink is
// the stdout logger from Task 1.
type EscalationCapturer interface {
	Capture(ctx context.Context, ev EscalationEvent) error
}

// ViewRenderEmitter is the boundary interface defaultViewBuilder uses
// to emit EventViewRender audit rows from BuildLayer1Baseline. The
// audit package's Emitter satisfies a compatible adapter; this local
// interface keeps the aggregation package free of an audit-package
// import.
type ViewRenderEmitter interface {
	EmitViewRender(ctx context.Context, req WorkspaceRequest, layer int) error
}

// S2ViewBuilder is the layer-aware view-construction interface per the
// S2 Adaptive Cognition Architectural Commitment Addendum, Part 8.1
// (lines 386–397).
//
// The interface commits the team to a layer-aware design pattern that
// supports adding Layer 2–5 rendering capabilities without rebuilding the
// aggregation pipeline. In Phase 1 only BuildLayer1Baseline is meaningful;
// Layers 2–5 return sentinel errors citing Addendum Part 6's content-
// deferral discipline.
type S2ViewBuilder interface {
	BuildLayer1Baseline(ctx context.Context, req WorkspaceRequest) (Layer1View, error)
	BuildLayer2Escalated(ctx context.Context, req WorkspaceRequest) (Layer2View, error)
	BuildLayer3Complex(ctx context.Context, req WorkspaceRequest) (Layer3View, error)
	BuildLayer4SituationBoard(ctx context.Context, req WorkspaceRequest) (Layer4View, error)
	BuildLayer5Investigation(ctx context.Context, req WorkspaceRequest) (Layer5View, error)
	EscalateToLayer(ctx context.Context, currentLayer int, targetLayer int, req WorkspaceRequest) (View, error)
	LogEscalation(ctx context.Context, escalation EscalationEvent) error
}

// defaultViewBuilder is the Phase 1 concrete implementation. Layer 1
// returns a zero-value Layer1View (Tasks 3–7 populate it); Layers 2–5
// return sentinel errors that cite the Addendum so any caller that hits
// them surfaces the architectural reason rather than a generic "not
// implemented".
//
// LogEscalation writes the event to a structured log sink — stdout in
// Phase 1 per Addendum §5.5 (log-only commitment). Task 7 swaps the sink
// for a real audit hook (EvidenceTrace + visibility-class enforcement).
type defaultViewBuilder struct {
	// logger writes escalation audit lines. Defaults to log.Default()
	// (stdout) so production boots are observable without extra wiring.
	// The field is exported via NewDefaultViewBuilderWithLogger so tests
	// can capture output.
	logger *log.Logger

	// escalationCapturer, when set, receives every LogEscalation call
	// in addition to the stdout log line. Task 7 wires this to the
	// audit package's EscalationEventEmitter so escalation captures
	// reach the s2_audit_events table.
	escalationCapturer EscalationCapturer

	// viewRenderEmitter, when set, receives an EventViewRender audit
	// row on every BuildLayer1Baseline call. Task 7 wires this to the
	// audit package's Emitter via an adapter.
	viewRenderEmitter ViewRenderEmitter
}

// WithEscalationCapturer returns a view builder that delegates
// LogEscalation to capturer in addition to writing the stdout log
// line. nil capturer disables the delegate (preserves Task 1 behaviour).
func WithEscalationCapturer(b S2ViewBuilder, capturer EscalationCapturer) S2ViewBuilder {
	if dv, ok := b.(*defaultViewBuilder); ok {
		dv.escalationCapturer = capturer
		return dv
	}
	return b
}

// WithViewRenderEmitter returns a view builder that emits an audit row
// per BuildLayer1Baseline call. nil emitter disables the audit fan-out.
func WithViewRenderEmitter(b S2ViewBuilder, emitter ViewRenderEmitter) S2ViewBuilder {
	if dv, ok := b.(*defaultViewBuilder); ok {
		dv.viewRenderEmitter = emitter
		return dv
	}
	return b
}

// NewDefaultViewBuilder returns a Phase 1 view builder that logs escalation
// events to stdout via the standard library logger.
func NewDefaultViewBuilder() S2ViewBuilder {
	return &defaultViewBuilder{logger: log.New(os.Stdout, "s2-escalation: ", log.LstdFlags|log.LUTC)}
}

// NewDefaultViewBuilderWithLogger is the test-facing constructor. It lets
// unit tests pass an io.Writer-backed *log.Logger to capture LogEscalation
// output without redirecting os.Stdout.
func NewDefaultViewBuilderWithLogger(w io.Writer) S2ViewBuilder {
	return &defaultViewBuilder{logger: log.New(w, "s2-escalation: ", log.LstdFlags|log.LUTC)}
}

// BuildLayer1Baseline returns a zero-value Layer1View in Task 1.
// Tasks 3–7 populate the view fields per S2 v1.0 Parts 4–13.
//
// Task 7: when a ViewRenderEmitter is wired, emit an EventViewRender
// audit row before returning. Audit-emission failure is best-effort —
// logged to stdout but not returned — because v1.0 Part 13.1 view-
// render audit is downstream observability, not a load-bearing
// substrate write.
func (b *defaultViewBuilder) BuildLayer1Baseline(
	ctx context.Context, req WorkspaceRequest,
) (Layer1View, error) {
	if b.viewRenderEmitter != nil {
		if err := b.viewRenderEmitter.EmitViewRender(ctx, req, 1); err != nil && b.logger != nil {
			b.logger.Printf("view_render audit emission failed (best-effort): %v", err)
		}
	}
	return Layer1View{}, nil
}

// notImplementedSentinel returns the canonical Phase 1 sentinel error for
// Layers 2–5. The message names the layer and cites the Addendum so
// downstream callers (frontend rendering, error-classifying middleware)
// can distinguish "deferred by architectural discipline" from runtime
// failures.
func notImplementedSentinel(layer int) error {
	return fmt.Errorf(
		"layer %d not yet implemented per S2 Adaptive Cognition Addendum Part 6 (content deferred to senior consultant pharmacist authoring)",
		layer,
	)
}

func (b *defaultViewBuilder) BuildLayer2Escalated(
	_ context.Context, _ WorkspaceRequest,
) (Layer2View, error) {
	return Layer2View{}, notImplementedSentinel(2)
}

func (b *defaultViewBuilder) BuildLayer3Complex(
	_ context.Context, _ WorkspaceRequest,
) (Layer3View, error) {
	return Layer3View{}, notImplementedSentinel(3)
}

func (b *defaultViewBuilder) BuildLayer4SituationBoard(
	_ context.Context, _ WorkspaceRequest,
) (Layer4View, error) {
	return Layer4View{}, notImplementedSentinel(4)
}

func (b *defaultViewBuilder) BuildLayer5Investigation(
	_ context.Context, _ WorkspaceRequest,
) (Layer5View, error) {
	return Layer5View{}, notImplementedSentinel(5)
}

// EscalateToLayer is not yet operational. Layer-to-layer transitions
// require Layers 2–5 to exist; in Phase 1 the only layer present is
// Layer 1 so any escalation request is by definition unfulfillable.
//
// Task 7 wires this method to LogEscalation so even unfulfillable
// requests leave an audit record (Addendum §5.5).
func (b *defaultViewBuilder) EscalateToLayer(
	_ context.Context, _ int, _ int, _ WorkspaceRequest,
) (View, error) {
	return nil, errors.New("escalation not implemented at Layer 1 — Addendum Part 6 defers Layer 2–5 content")
}

// LogEscalation writes the audit event to the configured logger. Phase 1
// is log-only per Addendum §5.5; Task 7 replaces this sink with a real
// audit emitter (EvidenceTrace + visibility-class enforcement).
//
// Format: a single line with the structured fields named, so log scrapers
// can pivot on field names without parsing prose.
func (b *defaultViewBuilder) LogEscalation(
	ctx context.Context, e EscalationEvent,
) error {
	if b.logger == nil {
		return errors.New("s2-aggregator: LogEscalation called with nil logger — construct via NewDefaultViewBuilder")
	}
	b.logger.Printf(
		"escalation_event pharmacist_id=%s resident_id=%s session_id=%s from_layer=%d to_layer=%d triggered_by=%s timestamp=%s audit_trace_id=%s",
		e.PharmacistID, e.ResidentID, e.SessionID,
		e.FromLayer, e.ToLayer, e.TriggeredBy,
		e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		e.AuditTraceID,
	)
	// Task 7: delegate to the audit-package EscalationEventEmitter when
	// wired. Phase 1 commitment per Addendum Part 5.5 is LOG-ONLY; the
	// emitter writes only, and the audit package has no read path for
	// per-pharmacist escalation patterns (enforced by the structural
	// no-surveillance-reader test).
	if b.escalationCapturer != nil {
		if err := b.escalationCapturer.Capture(ctx, e); err != nil {
			return fmt.Errorf("escalation capture: %w", err)
		}
	}
	return nil
}
