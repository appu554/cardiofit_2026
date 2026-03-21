package enrollment

import "fmt"

// State represents a step in the enrollment lifecycle.
type State string

const (
	StateCreated          State = "CREATED"
	StateIdentityVerified State = "IDENTITY_VERIFIED"
	StateIntakeReady      State = "INTAKE_READY"
	StateIntakeInProgress State = "INTAKE_IN_PROGRESS"
	StateHardStopped      State = "HARD_STOPPED"
	StateIntakePaused     State = "INTAKE_PAUSED"
	StateIntakeCompleted  State = "INTAKE_COMPLETED"
	StateEnrolled         State = "ENROLLED"
)

// AllStates returns every state in the enrollment lifecycle.
func AllStates() []State {
	return []State{
		StateCreated, StateIdentityVerified, StateIntakeReady,
		StateIntakeInProgress, StateHardStopped, StateIntakePaused,
		StateIntakeCompleted, StateEnrolled,
	}
}

// validTransitions defines the allowed state graph.
// Terminal states (HARD_STOPPED, ENROLLED) have no outgoing edges.
var validTransitions = map[State][]State{
	StateCreated:          {StateIdentityVerified},
	StateIdentityVerified: {StateIntakeReady},
	StateIntakeReady:      {StateIntakeInProgress},
	StateIntakeInProgress: {StateHardStopped, StateIntakePaused, StateIntakeCompleted},
	StateHardStopped:      {},
	StateIntakePaused:     {StateIntakeInProgress},
	StateIntakeCompleted:  {StateEnrolled},
	StateEnrolled:         {},
}

// CanTransition returns true when the from->to edge exists in the state graph.
func CanTransition(from, to State) bool {
	targets, exists := validTransitions[from]
	if !exists {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// ErrInvalidTransition is returned when a caller attempts a forbidden state change.
type ErrInvalidTransition struct {
	From State
	To   State
}

func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid enrollment transition: %s -> %s", e.From, e.To)
}
