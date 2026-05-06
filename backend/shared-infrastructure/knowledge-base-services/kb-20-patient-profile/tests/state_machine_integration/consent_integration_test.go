// Wave 6.2 — Consent state-machine integration test.
//
// Layer 2 doc §4.5: "a CapacityAssessment outcome change triggers a
// Consent re-evaluation event. The substrate emits the trigger event;
// Layer 3's Consent state machine consumes it."
package state_machine_integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// capacityOutcome is the kb-20 substrate's CapacityAssessment outcome
// projection. The real shape lives in
// shared/v2_substrate/models/capacity_assessment.go; for this mock
// integration test we use a slim subset.
type capacityOutcome struct {
	ResidentRef        uuid.UUID
	HasCapacityForCare bool
	AssessedAt         time.Time
}

// consentReevalEvent is the substrate-emitted trigger Layer 3 consumes.
type consentReevalEvent struct {
	ResidentRef uuid.UUID
	TriggeredAt time.Time
	Reason      string
}

// emitOnCapacityChange compares two capacity outcomes and emits a Consent
// re-eval event when the HasCapacityForCare flag flipped.
func emitOnCapacityChange(prev, next capacityOutcome) (consentReevalEvent, bool) {
	if prev.HasCapacityForCare == next.HasCapacityForCare {
		return consentReevalEvent{}, false
	}
	return consentReevalEvent{
		ResidentRef: next.ResidentRef,
		TriggeredAt: next.AssessedAt,
		Reason:      "capacity_outcome_change",
	}, true
}

func TestConsent_CapacityFlipTriggersReeval(t *testing.T) {
	residentRef := uuid.New()
	prev := capacityOutcome{ResidentRef: residentRef, HasCapacityForCare: true, AssessedAt: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)}
	next := capacityOutcome{ResidentRef: residentRef, HasCapacityForCare: false, AssessedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)}
	ev, emitted := emitOnCapacityChange(prev, next)
	if !emitted {
		t.Fatal("capacity flip MUST emit Consent re-eval event")
	}
	if ev.ResidentRef != residentRef {
		t.Fatal("event must carry resident_ref")
	}
	if ev.TriggeredAt != next.AssessedAt {
		t.Fatal("event triggered_at must match the assessment time of the new outcome")
	}
	if ev.Reason != "capacity_outcome_change" {
		t.Fatalf("reason drift: %s", ev.Reason)
	}
}

func TestConsent_NoFlipNoEvent(t *testing.T) {
	residentRef := uuid.New()
	prev := capacityOutcome{ResidentRef: residentRef, HasCapacityForCare: true, AssessedAt: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)}
	next := capacityOutcome{ResidentRef: residentRef, HasCapacityForCare: true, AssessedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)}
	if _, emitted := emitOnCapacityChange(prev, next); emitted {
		t.Fatal("unchanged capacity outcome MUST NOT emit a re-eval event")
	}
}
