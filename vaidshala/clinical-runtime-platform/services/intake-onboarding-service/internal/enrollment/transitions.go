package enrollment

import (
	"time"

	"github.com/google/uuid"
)

// ChannelType identifies the enrollment channel (payer pathway).
type ChannelType string

const (
	ChannelCorporate  ChannelType = "CORPORATE"
	ChannelInsurance  ChannelType = "INSURANCE"
	ChannelGovernment ChannelType = "GOVERNMENT"
)

// Enrollment is the aggregate root for a patient's onboarding lifecycle.
type Enrollment struct {
	PatientID          uuid.UUID   `json:"patient_id"`
	TenantID           uuid.UUID   `json:"tenant_id"`
	ChannelType        ChannelType `json:"channel_type"`
	State              State       `json:"state"`
	EncounterID        uuid.UUID   `json:"encounter_id"`
	AssignedPharmacist *uuid.UUID  `json:"assigned_pharmacist,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// Transition moves the enrollment to the target state if the transition is valid.
// Returns ErrInvalidTransition when the edge does not exist in the state graph.
func (e *Enrollment) Transition(to State) error {
	if !CanTransition(e.State, to) {
		return &ErrInvalidTransition{From: e.State, To: to}
	}
	e.State = to
	e.UpdatedAt = time.Now().UTC()
	return nil
}

// IsTerminal returns true when the enrollment is in a final state
// (HARD_STOPPED or ENROLLED) with no further transitions possible.
func (e *Enrollment) IsTerminal() bool {
	return e.State == StateHardStopped || e.State == StateEnrolled
}
