package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

func testEvent() AuditEvent {
	return AuditEvent{
		TraceID:      uuid.New(),
		EventType:    EventPharmacistAction,
		Severity:     substrate_types.EthicsSeverityPrimaryDecision,
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		Subject:      "override",
		Payload:      map[string]any{"reason_code": "patient_preference"},
		OccurredAt:   time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC),
	}
}

func TestMemoryEmitter_RecordsEvents(t *testing.T) {
	em := NewMemoryEmitter()
	evt := testEvent()
	if err := em.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if got := em.Count(); got != 1 {
		t.Fatalf("Count = %d, want 1", got)
	}
	last, ok := em.Last()
	if !ok || last.TraceID != evt.TraceID {
		t.Fatalf("Last() = %v, %v; want %v, true", last, ok, evt)
	}
}

func TestMemoryEmitter_EventsOfType(t *testing.T) {
	em := NewMemoryEmitter()
	a := testEvent()
	a.EventType = EventViewRender
	b := testEvent()
	b.EventType = EventDrillThrough
	c := testEvent()
	c.EventType = EventViewRender
	for _, e := range []AuditEvent{a, b, c} {
		_ = em.Emit(context.Background(), e)
	}
	views := em.EventsOfType(EventViewRender)
	if len(views) != 2 {
		t.Fatalf("EventsOfType(view_render) returned %d events, want 2", len(views))
	}
}

func TestEthicsLogEmitter_WritesDecisionEntry(t *testing.T) {
	log := NewMemoryLogger()
	em := NewEthicsLogEmitter(log)
	evt := testEvent()
	if err := em.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	entries := log.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 ethics_log entry, got %d", len(entries))
	}
	got := entries[0]
	if got.DecisionID != evt.TraceID {
		t.Errorf("DecisionID = %v, want %v (trace_id)", got.DecisionID, evt.TraceID)
	}
	if got.EntryType != substrate_types.EthicsEntryTypeDecision {
		t.Errorf("EntryType = %v, want %v", got.EntryType, substrate_types.EthicsEntryTypeDecision)
	}
	if got.Severity != evt.Severity {
		t.Errorf("Severity = %d, want %d", got.Severity, evt.Severity)
	}
	if got.Description == "" {
		t.Errorf("Description (JSON payload) is empty")
	}
}

func TestEthicsLogEmitter_NilLogger(t *testing.T) {
	em := NewEthicsLogEmitter(nil)
	if err := em.Emit(context.Background(), testEvent()); err == nil {
		t.Fatalf("expected error from nil logger, got nil")
	}
}

// failEmitter is a test Emitter that always returns the configured error.
type failEmitter struct{ err error }

func (f *failEmitter) Emit(_ context.Context, _ AuditEvent) error { return f.err }

func TestCompositeEmitter_FailFast(t *testing.T) {
	a := NewMemoryEmitter()
	boom := &failEmitter{err: errors.New("simulated")}
	c := NewMemoryEmitter()
	comp := NewCompositeEmitter(a, boom, c)
	err := comp.Emit(context.Background(), testEvent())
	if err == nil {
		t.Fatalf("expected fail-fast error, got nil")
	}
	if a.Count() != 1 {
		t.Errorf("first emitter should have been invoked: a.Count = %d, want 1", a.Count())
	}
	if c.Count() != 0 {
		t.Errorf("third emitter should NOT have been invoked (fail-fast): c.Count = %d, want 0", c.Count())
	}
}

func TestCompositeEmitter_AllSucceed(t *testing.T) {
	a := NewMemoryEmitter()
	b := NewMemoryEmitter()
	comp := NewCompositeEmitter(a, b)
	if err := comp.Emit(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Count() != 1 || b.Count() != 1 {
		t.Errorf("both emitters should receive the event: a=%d b=%d", a.Count(), b.Count())
	}
}

func TestCompositeEmitter_NilUnderlying(t *testing.T) {
	comp := NewCompositeEmitter(nil)
	if err := comp.Emit(context.Background(), testEvent()); err == nil {
		t.Fatalf("expected error when underlying emitter is nil")
	}
}

// fakeExecer captures the args passed to ExecContext for PostgresEmitter tests.
type fakeExecer struct {
	gotQuery string
	gotArgs  []any
	failWith error
}

func (f *fakeExecer) ExecContext(_ context.Context, q string, args ...any) (any, error) {
	f.gotQuery = q
	f.gotArgs = args
	return nil, f.failWith
}

func TestPostgresEmitter_InsertsEvent(t *testing.T) {
	db := &fakeExecer{}
	em := NewPostgresEmitter(db)
	evt := testEvent()
	if err := em.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Emit error: %v", err)
	}
	if len(db.gotArgs) != 9 {
		t.Fatalf("expected 9 args to ExecContext, got %d", len(db.gotArgs))
	}
	if db.gotArgs[0] != evt.TraceID {
		t.Errorf("arg[0] (trace_id) = %v, want %v", db.gotArgs[0], evt.TraceID)
	}
	if db.gotArgs[1] != string(evt.EventType) {
		t.Errorf("arg[1] (event_type) = %v, want %v", db.gotArgs[1], evt.EventType)
	}
}

func TestPostgresEmitter_NilResidentAndSession_AsSQLNull(t *testing.T) {
	db := &fakeExecer{}
	em := NewPostgresEmitter(db)
	evt := testEvent()
	evt.ResidentID = uuid.Nil
	evt.SessionID = uuid.Nil
	if err := em.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Emit error: %v", err)
	}
	if db.gotArgs[4] != nil {
		t.Errorf("arg[4] (resident_id) should be nil for uuid.Nil, got %v", db.gotArgs[4])
	}
	if db.gotArgs[5] != nil {
		t.Errorf("arg[5] (session_id) should be nil for uuid.Nil, got %v", db.gotArgs[5])
	}
}
