package incident_response

import (
	"context"
	"errors"
	"testing"
)

// TestHoldOrchestrator_TriggersOnSeverity2 verifies that a severity-2 incident
// causes all registered handlers to be invoked.
func TestHoldOrchestrator_TriggersOnSeverity2(t *testing.T) {
	called := false
	h := HoldHandlerFunc(func(_ context.Context, inc Incident) error {
		called = true
		return nil
	})

	o := NewOrchestrator()
	o.Register(h)

	inc := Incident{
		Severity: 2,
		Kind:     "trust_violation",
	}
	if err := o.Trigger(context.Background(), inc); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	if !called {
		t.Errorf("handler was not called for severity 2 incident")
	}
}

// TestHoldOrchestrator_DoesNotTriggerOnSeverity4 verifies that a severity-4
// incident (procedural) does NOT activate any hold handlers.
func TestHoldOrchestrator_DoesNotTriggerOnSeverity4(t *testing.T) {
	called := false
	h := HoldHandlerFunc(func(_ context.Context, inc Incident) error {
		called = true
		return nil
	})

	o := NewOrchestrator()
	o.Register(h)

	inc := Incident{
		Severity: 4,
		Kind:     "procedural",
	}
	if err := o.Trigger(context.Background(), inc); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	if called {
		t.Errorf("handler must NOT be called for severity 4 incident")
	}
}

// TestHoldOrchestrator_PropagatesHandlerError verifies that if a registered
// handler returns an error, Trigger surfaces it immediately.
func TestHoldOrchestrator_PropagatesHandlerError(t *testing.T) {
	sentinel := errors.New("handler failed")
	h := HoldHandlerFunc(func(_ context.Context, _ Incident) error {
		return sentinel
	})

	o := NewOrchestrator()
	o.Register(h)

	inc := Incident{Severity: 1, Kind: "clinical_safety"}
	err := o.Trigger(context.Background(), inc)
	if !errors.Is(err, sentinel) {
		t.Errorf("Trigger returned %v, want sentinel error", err)
	}
}

// TestHoldOrchestrator_MultipleHandlersAllInvoked registers 3 handlers and
// verifies all three are called when a severity-1 incident is triggered.
func TestHoldOrchestrator_MultipleHandlersAllInvoked(t *testing.T) {
	counts := [3]int{}
	handlers := []HoldHandler{
		HoldHandlerFunc(func(_ context.Context, _ Incident) error { counts[0]++; return nil }),
		HoldHandlerFunc(func(_ context.Context, _ Incident) error { counts[1]++; return nil }),
		HoldHandlerFunc(func(_ context.Context, _ Incident) error { counts[2]++; return nil }),
	}

	o := NewOrchestrator()
	for _, h := range handlers {
		o.Register(h)
	}

	inc := Incident{Severity: 1, Kind: "clinical_safety"}
	if err := o.Trigger(context.Background(), inc); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	for i, c := range counts {
		if c != 1 {
			t.Errorf("handler[%d] called %d times, want 1", i, c)
		}
	}
}
