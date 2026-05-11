package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
)

func testEscalation() aggregation.EscalationEvent {
	return aggregation.EscalationEvent{
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		FromLayer:    1,
		ToLayer:      3,
		TriggeredBy:  aggregation.TriggerPharmacistInitiated,
		Timestamp:    time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
		AuditTraceID: uuid.New(),
	}
}

func TestEscalationEventEmitter_Capture_WritesCognitiveEscalationEvent(t *testing.T) {
	mem := NewMemoryEmitter()
	em := NewEscalationEventEmitter(mem)
	ev := testEscalation()
	if err := em.Capture(context.Background(), ev); err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	if mem.Count() != 1 {
		t.Fatalf("expected 1 audit event captured, got %d", mem.Count())
	}
	last, _ := mem.Last()
	if last.EventType != EventCognitiveEscalation {
		t.Errorf("EventType = %v, want %v", last.EventType, EventCognitiveEscalation)
	}
	if last.Severity != 1 {
		t.Errorf("Severity = %d, want 1 (primary-decision severity per kb-32 convention)", last.Severity)
	}
	if last.PharmacistID != ev.PharmacistID {
		t.Errorf("PharmacistID drift")
	}
	if last.ResidentID != ev.ResidentID {
		t.Errorf("ResidentID drift")
	}
	if last.SessionID != ev.SessionID {
		t.Errorf("SessionID drift")
	}
	if last.TraceID != ev.AuditTraceID {
		t.Errorf("TraceID should mirror AuditTraceID, got %v want %v", last.TraceID, ev.AuditTraceID)
	}
	// Payload structure
	if fl, _ := last.Payload["from_layer"].(int); fl != 1 {
		t.Errorf("payload.from_layer = %v, want 1", last.Payload["from_layer"])
	}
	if tl, _ := last.Payload["to_layer"].(int); tl != 3 {
		t.Errorf("payload.to_layer = %v, want 3", last.Payload["to_layer"])
	}
	if tb, _ := last.Payload["triggered_by"].(string); tb != "pharmacist_initiated" {
		t.Errorf("payload.triggered_by = %v, want pharmacist_initiated", last.Payload["triggered_by"])
	}
}

func TestEscalationEventEmitter_Capture_FillsDefaults(t *testing.T) {
	mem := NewMemoryEmitter()
	em := NewEscalationEventEmitter(mem)
	ev := testEscalation()
	ev.Timestamp = time.Time{}
	ev.AuditTraceID = uuid.Nil
	if err := em.Capture(context.Background(), ev); err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	last, _ := mem.Last()
	if last.OccurredAt.IsZero() {
		t.Errorf("OccurredAt should be defaulted to time.Now when input is zero")
	}
	if last.TraceID == uuid.Nil {
		t.Errorf("TraceID should be defaulted to uuid.New when input is nil")
	}
}

func TestEscalationEventEmitter_NilEmitter(t *testing.T) {
	em := NewEscalationEventEmitter(nil)
	if err := em.Capture(context.Background(), testEscalation()); err == nil {
		t.Fatalf("expected error when wrapped emitter is nil")
	}
}
