package checkin

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CheckinState represents a state in the M0-CI biweekly check-in state machine.
type CheckinState string

const (
	CS1_SCHEDULED  CheckinState = "CS1_SCHEDULED"
	CS2_REMINDED   CheckinState = "CS2_REMINDED"
	CS3_COLLECTING CheckinState = "CS3_COLLECTING"
	CS4_PAUSED     CheckinState = "CS4_PAUSED"
	CS5_SCORING    CheckinState = "CS5_SCORING"
	CS6_DISPATCHED CheckinState = "CS6_DISPATCHED"
	CS7_CLOSED     CheckinState = "CS7_CLOSED"
)

// AllCheckinStates returns all 7 check-in states in order.
func AllCheckinStates() []CheckinState {
	return []CheckinState{
		CS1_SCHEDULED,
		CS2_REMINDED,
		CS3_COLLECTING,
		CS4_PAUSED,
		CS5_SCORING,
		CS6_DISPATCHED,
		CS7_CLOSED,
	}
}

// validCheckinTransitions defines allowed state transitions.
var validCheckinTransitions = map[CheckinState][]CheckinState{
	CS1_SCHEDULED:  {CS2_REMINDED},
	CS2_REMINDED:   {CS3_COLLECTING},
	CS3_COLLECTING: {CS4_PAUSED, CS5_SCORING},
	CS4_PAUSED:     {CS3_COLLECTING, CS7_CLOSED},
	CS5_SCORING:    {CS6_DISPATCHED},
	CS6_DISPATCHED: {CS7_CLOSED},
	CS7_CLOSED:     {},
}

// CanCheckinTransition returns true if transitioning from → to is allowed.
func CanCheckinTransition(from, to CheckinState) bool {
	targets, ok := validCheckinTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// CheckinSlotDef defines a single slot in the check-in form.
type CheckinSlotDef struct {
	Name     string
	Domain   string
	LOINCCode string
	Unit     string
	Required bool
}

// CheckinSlots returns the 12 check-in slots (8 required).
func CheckinSlots() []CheckinSlotDef {
	return []CheckinSlotDef{
		{Name: "fbg", Domain: "glycemic", LOINCCode: "1558-6", Unit: "mg/dL", Required: true},
		{Name: "ppbg", Domain: "glycemic", LOINCCode: "1521-4", Unit: "mg/dL", Required: true},
		{Name: "hba1c", Domain: "glycemic", LOINCCode: "4548-4", Unit: "%", Required: false},
		{Name: "systolic_bp", Domain: "cardiovascular", LOINCCode: "8480-6", Unit: "mmHg", Required: true},
		{Name: "diastolic_bp", Domain: "cardiovascular", LOINCCode: "8462-4", Unit: "mmHg", Required: true},
		{Name: "egfr", Domain: "renal", LOINCCode: "48642-3", Unit: "mL/min/1.73m2", Required: false},
		{Name: "weight", Domain: "anthropometric", LOINCCode: "29463-7", Unit: "kg", Required: true},
		{Name: "medication_adherence", Domain: "behavioral", LOINCCode: "71950-0", Unit: "score", Required: true},
		{Name: "physical_activity_minutes", Domain: "lifestyle", LOINCCode: "68516-4", Unit: "min/week", Required: false},
		{Name: "sleep_hours", Domain: "lifestyle", LOINCCode: "93832-4", Unit: "hours", Required: false},
		{Name: "symptom_severity", Domain: "clinical", LOINCCode: "72514-3", Unit: "score", Required: true},
		{Name: "side_effects", Domain: "clinical", LOINCCode: "85354-9", Unit: "score", Required: true},
	}
}

// CheckinSession represents a single biweekly check-in session.
type CheckinSession struct {
	ID          uuid.UUID
	PatientID   uuid.UUID
	EncounterID uuid.UUID
	CycleNumber int
	State       CheckinState
	Trajectory  string
	SlotsFilled int
	SlotsTotal  int
	ScheduledAt time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ErrInvalidCheckinTransition is returned when a state transition is not allowed.
type ErrInvalidCheckinTransition struct {
	From CheckinState
	To   CheckinState
}

func (e *ErrInvalidCheckinTransition) Error() string {
	return fmt.Sprintf("invalid check-in transition: %s → %s", e.From, e.To)
}

// Transition validates and applies a state transition.
func (s *CheckinSession) Transition(to CheckinState) error {
	if !CanCheckinTransition(s.State, to) {
		return &ErrInvalidCheckinTransition{From: s.State, To: to}
	}
	s.State = to
	now := time.Now()
	s.UpdatedAt = now

	if to == CS3_COLLECTING && s.StartedAt == nil {
		s.StartedAt = &now
	}
	if to == CS5_SCORING {
		s.CompletedAt = &now
	}
	return nil
}

// IsTerminal returns true if the session is in a terminal state.
func (s *CheckinSession) IsTerminal() bool {
	return s.State == CS7_CLOSED
}

// RequiredSlotsFilled returns true if all required slots have been filled.
func (s *CheckinSession) RequiredSlotsFilled(filledSlots map[string]bool) bool {
	for _, slot := range CheckinSlots() {
		if slot.Required && !filledSlots[slot.Name] {
			return false
		}
	}
	return true
}

// BiweeklyInterval is the standard interval between check-in sessions.
var BiweeklyInterval = 14 * 24 * time.Hour

// NextScheduledAt computes the next check-in time from the previous one.
func NextScheduledAt(previous time.Time) time.Time {
	return previous.Add(BiweeklyInterval)
}
