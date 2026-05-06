package validation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateEvidenceTraceNode reports any structural problem with n.
//
// Universal rules (apply to every node):
//   - StateMachine must be one of the EvidenceTraceStateMachine* enum values
//   - StateChangeType must be non-empty (free-form structured tag)
//   - RecordedAt must be non-zero
//   - OccurredAt must be non-zero
//   - Each Inputs[i].InputType must be non-empty
//   - Each Inputs[i].InputRef must be non-zero
//   - Each Inputs[i].RoleInDecision must be one of TraceRoleInDecision*
//   - Each Outputs[i].OutputType must be non-empty
//   - Each Outputs[i].OutputRef must be non-zero
//
// Note: ResidentRef is intentionally NOT required — system-only nodes
// (e.g. global rule_fire, credential checks not yet bound to a resident)
// have no resident. Actor sub-fields are also nullable (system actors).
func ValidateEvidenceTraceNode(n models.EvidenceTraceNode) error {
	if !models.IsValidEvidenceTraceStateMachine(n.StateMachine) {
		return fmt.Errorf("invalid state_machine %q", n.StateMachine)
	}
	if n.StateChangeType == "" {
		return errors.New("state_change_type is required")
	}
	if n.RecordedAt.IsZero() {
		return errors.New("recorded_at is required")
	}
	if n.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	for i, in := range n.Inputs {
		if in.InputType == "" {
			return fmt.Errorf("inputs[%d].input_type is required", i)
		}
		if in.InputRef == uuid.Nil {
			return fmt.Errorf("inputs[%d].input_ref is required", i)
		}
		if !models.IsValidTraceRoleInDecision(in.RoleInDecision) {
			return fmt.Errorf("inputs[%d].role_in_decision %q is invalid", i, in.RoleInDecision)
		}
	}
	for i, out := range n.Outputs {
		if out.OutputType == "" {
			return fmt.Errorf("outputs[%d].output_type is required", i)
		}
		if out.OutputRef == uuid.Nil {
			return fmt.Errorf("outputs[%d].output_ref is required", i)
		}
	}
	return nil
}
