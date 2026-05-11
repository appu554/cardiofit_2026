// Package actions implements the eleven pharmacist actions, reasoning
// capture rules, and session context per S2 Resident Workspace
// Implementation Guidelines v1.0 Part 12 and the S2 Adaptive Cognition
// Architectural Commitment Addendum Part 4.7 (pharmacist actions as a
// shared primitive across all five cognitive depth states; the eleven
// actions are layer-invariant — deeper layers may surface additional
// actions but never remove these).
package actions

import (
	"time"

	"github.com/google/uuid"
)

// Action is the canonical enumeration of the eleven pharmacist actions
// per v1.0 Part 12.1. The string values are the lower_snake_case names
// used at API ingress, in audit rows, and across cross-service contracts.
type Action string

// The eleven canonical actions per v1.0 Part 12.1 in the order they
// appear in the spec (open through invoke safety-critical bypass).
const (
	ActionOpen                       Action = "open"
	ActionModify                     Action = "modify"
	ActionDefer                      Action = "defer"
	ActionOverride                   Action = "override"
	ActionMarkReviewed               Action = "mark_reviewed"
	ActionFlagForFollowUp            Action = "flag_for_follow_up"
	ActionAddNote                    Action = "add_note"
	ActionOpenComplexWorkspace       Action = "open_complex_workspace"
	ActionDrillIntoSubstrate         Action = "drill_into_substrate"
	ActionAcknowledgeRestraintSignal Action = "acknowledge_restraint_signal"
	ActionInvokeSafetyCriticalBypass Action = "invoke_safety_critical_bypass"
)

// allActions is the canonical ordered set of valid actions, used by
// IsValidAction and the reasoning-requirement table.
var allActions = []Action{
	ActionOpen,
	ActionModify,
	ActionDefer,
	ActionOverride,
	ActionMarkReviewed,
	ActionFlagForFollowUp,
	ActionAddNote,
	ActionOpenComplexWorkspace,
	ActionDrillIntoSubstrate,
	ActionAcknowledgeRestraintSignal,
	ActionInvokeSafetyCriticalBypass,
}

// IsValidAction reports whether s names one of the eleven canonical
// pharmacist actions.
func IsValidAction(s string) bool {
	for _, a := range allActions {
		if string(a) == s {
			return true
		}
	}
	return false
}

// ReasoningRequirement classifies how the reasoning field is treated for
// a given action per v1.0 Part 12.3.
type ReasoningRequirement int

// Reasoning requirement values per v1.0 Part 12.3.
const (
	// ReasoningNotApplicable means the action carries no reasoning field;
	// any non-empty reasoning is a contract violation.
	ReasoningNotApplicable ReasoningRequirement = iota
	// ReasoningOptional means reasoning may be supplied but is not
	// required for the action to be accepted.
	ReasoningOptional
	// ReasoningMandatory means the action is rejected without a non-
	// trivial reasoning string (and, for override, a taxonomy code).
	ReasoningMandatory
	// ReasoningIsNote means the action carries free-text in NoteBody and
	// the note itself IS the reasoning (per v1.0 Part 12.3 add-note row).
	ReasoningIsNote
)

// ReasoningRequirementFor returns the reasoning-capture requirement for a
// given action per the v1.0 Part 12.3 table:
//
//	open                         → not applicable
//	modify                       → MANDATORY
//	defer                        → optional
//	override                     → MANDATORY + taxonomy code
//	mark reviewed                → not applicable
//	flag for follow-up           → not applicable
//	add note                     → the note IS the reasoning
//	open complex workspace       → not applicable (escalation is the audit event)
//	drill into substrate         → not applicable
//	acknowledge restraint signal → optional
//	invoke safety-critical bypass→ MANDATORY (audit-prioritised)
func ReasoningRequirementFor(a Action) ReasoningRequirement {
	switch a {
	case ActionModify, ActionOverride, ActionInvokeSafetyCriticalBypass:
		return ReasoningMandatory
	case ActionDefer, ActionAcknowledgeRestraintSignal:
		return ReasoningOptional
	case ActionAddNote:
		return ReasoningIsNote
	case ActionOpen, ActionMarkReviewed, ActionFlagForFollowUp,
		ActionOpenComplexWorkspace, ActionDrillIntoSubstrate:
		return ReasoningNotApplicable
	default:
		return ReasoningNotApplicable
	}
}

// ActionRequest is the inbound shape captured by every pharmacist
// action — the wire payload from Layer 1 client through to the
// pharmacist_actions audit row.
//
// SubjectID is the recommendation_id, restraint_signal_id, or uuid.Nil
// when the action is resident-level rather than artefact-level (e.g.
// mark-reviewed at session scope).
type ActionRequest struct {
	Action                  Action
	PharmacistID            uuid.UUID
	ResidentID              uuid.UUID
	SessionID               uuid.UUID
	SubjectID               uuid.UUID
	Reasoning               string
	OverrideReasonCode      string // snake_case (override action only)
	OverrideReasonCodeShort string // 3-letter Guidelines Part 5 code
	AppropriatenessFlag     string // override action only
	NoteBody                string // add note action only
	Timestamp               time.Time
}

// ActionAcknowledgment is the response returned after an action is
// recorded — the audit trace handle the caller stores for reconciliation.
//
// Task 7 wires AuditTraceID to the real EvidenceTrace emitter; in Task 6
// it is a uuid.New() so the field is stable on the wire.
type ActionAcknowledgment struct {
	ActionID     uuid.UUID
	AcceptedAt   time.Time
	AuditTraceID uuid.UUID
}
