package contracts

import "context"

// VetoContract defines the authority boundary between ICU Intelligence and other KBs.
//
// ARCHITECTURE CRITICAL (CTO/CMO Directive):
//
//	"CQL explains. KB-19 recommends. ICU decides."
//
// This interface enforces the authority hierarchy:
//   - ICU Intelligence can veto KB-19, KB-18, KB-14, and CQL outputs
//   - KB-19 can only RECOMMEND, never DECIDE
//   - KB-18 governance can be overridden in crisis states
//   - KB-14 workflows can be interrupted by ICU dominance
//
// The VetoContract is implemented by the DominanceEngine and consumed by
// all workflow KBs (KB-14, KB-18, KB-19) before executing any action.
type VetoContract interface {
	// ═══════════════════════════════════════════════════════════════════════════
	// AUTHORITY QUERIES - Determine what ICU can/cannot override
	// ═══════════════════════════════════════════════════════════════════════════

	// CanICUVeto returns true if ICU dominance can override the given action type.
	// Per directive: ICU can veto everything except reality.
	CanICUVeto(actionType ActionType) bool

	// CanKB19Recommend returns true if KB-19 is allowed to make recommendations
	// in the given dominance state. KB-19 can always recommend, but its
	// recommendations may be ignored.
	CanKB19Recommend(state DominanceState) bool

	// MustDeferToICU returns true if the proposed action must wait for
	// ICU dominance evaluation before proceeding.
	MustDeferToICU(action ProposedAction, state DominanceState) bool

	// ═══════════════════════════════════════════════════════════════════════════
	// VETO EVALUATION - Actual veto decision
	// ═══════════════════════════════════════════════════════════════════════════

	// EvaluateVeto checks if ICU dominance vetoes the proposed action.
	// Returns a VetoResult containing the decision and reasoning.
	EvaluateVeto(ctx context.Context, action ProposedAction, state DominanceState) (*VetoResult, error)

	// ═══════════════════════════════════════════════════════════════════════════
	// OVERRIDE TRACKING - Audit and accountability
	// ═══════════════════════════════════════════════════════════════════════════

	// RecordOverride logs when ICU dominance overrides a KB recommendation.
	// Required for clinical audit trail and governance compliance.
	RecordOverride(ctx context.Context, override OverrideRecord) error
}

// DominanceState represents ICU dominance states (imported concept).
// This is a string type matching icu.DominanceState for contract purposes.
type DominanceState string

// Dominance state constants (mirror icu package)
const (
	StateNone               DominanceState = "NONE"
	StateShock              DominanceState = "SHOCK"
	StateHypoxia            DominanceState = "HYPOXIA"
	StateActiveBleed        DominanceState = "ACTIVE_BLEED"
	StateLowOutputFailure   DominanceState = "LOW_OUTPUT_FAILURE"
	StateNeurologicCollapse DominanceState = "NEUROLOGIC_COLLAPSE"
)

// ActionType categorizes proposed clinical actions for veto evaluation.
type ActionType string

const (
	// Medication actions
	ActionMedicationOrder    ActionType = "MEDICATION_ORDER"
	ActionMedicationHold     ActionType = "MEDICATION_HOLD"
	ActionMedicationModify   ActionType = "MEDICATION_MODIFY"
	ActionMedicationDispense ActionType = "MEDICATION_DISPENSE"

	// Procedure actions
	ActionProcedureOrder   ActionType = "PROCEDURE_ORDER"
	ActionProcedureStart   ActionType = "PROCEDURE_START"
	ActionProcedureAbort   ActionType = "PROCEDURE_ABORT"

	// Care plan actions
	ActionCarePlanActivate ActionType = "CAREPLAN_ACTIVATE"
	ActionCarePlanModify   ActionType = "CAREPLAN_MODIFY"
	ActionCarePlanComplete ActionType = "CAREPLAN_COMPLETE"

	// Workflow actions
	ActionWorkflowStart    ActionType = "WORKFLOW_START"
	ActionWorkflowAdvance  ActionType = "WORKFLOW_ADVANCE"
	ActionWorkflowComplete ActionType = "WORKFLOW_COMPLETE"

	// Transition actions
	ActionDischarge        ActionType = "DISCHARGE"
	ActionTransfer         ActionType = "TRANSFER"
	ActionEscalation       ActionType = "ESCALATION"
)

// ProposedAction represents an action that requires ICU veto evaluation.
type ProposedAction struct {
	// ID is a unique identifier for this action proposal
	ID string `json:"id"`

	// Type categorizes the action for veto rules
	Type ActionType `json:"type"`

	// ActionType is a string version of the action type (for runtime clients)
	ActionType string `json:"actionType,omitempty"`

	// Source identifies the originating KB (e.g., "KB-19", "KB-14")
	Source string `json:"source"`

	// Description is human-readable action description
	Description string `json:"description"`

	// Urgency indicates clinical urgency (0=routine, 10=immediate)
	Urgency int `json:"urgency"`

	// RiskLevel indicates action risk (0=minimal, 10=critical)
	RiskLevel int `json:"risk_level"`

	// PatientID is the FHIR Patient resource ID
	PatientID string `json:"patient_id"`

	// EncounterID is the current FHIR Encounter resource ID
	EncounterID string `json:"encounter_id"`

	// ActionID is the identifier for the specific action (for approval workflows)
	ActionID string `json:"action_id,omitempty"`

	// RuleSetID is the rule set being evaluated (for rule evaluation actions)
	RuleSetID string `json:"rule_set_id,omitempty"`

	// Metadata contains action-specific data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VetoResult contains the outcome of a veto evaluation.
type VetoResult struct {
	// Vetoed is true if ICU dominance blocks this action
	Vetoed bool `json:"vetoed"`

	// Reason explains why the action was vetoed (if vetoed)
	Reason string `json:"reason,omitempty"`

	// DominanceState is the active state that triggered the veto
	DominanceState DominanceState `json:"dominance_state"`

	// TriggeringRule identifies which safety rule triggered the veto
	TriggeringRule string `json:"triggering_rule,omitempty"`

	// SafetyFlags active safety flags that influenced the decision
	SafetyFlags SafetyFlags `json:"safety_flags,omitempty"`

	// AllowedAlternatives lists permitted actions in current state
	AllowedAlternatives []ActionType `json:"allowed_alternatives,omitempty"`

	// MustNotify lists roles that must be notified of this veto
	MustNotify []string `json:"must_notify,omitempty"`

	// Confidence is the classifier's confidence (0.0-1.0)
	Confidence float64 `json:"confidence"`
}

// OverrideRecord captures when ICU dominance overrides another KB's recommendation.
type OverrideRecord struct {
	// Timestamp when the override occurred
	Timestamp int64 `json:"timestamp"`

	// PatientID affected by the override
	PatientID string `json:"patient_id"`

	// EncounterID during which the override occurred
	EncounterID string `json:"encounter_id"`

	// DominanceState that triggered the override
	DominanceState DominanceState `json:"dominance_state"`

	// OverriddenKB identifies which KB was overridden (e.g., "KB-19")
	OverriddenKB string `json:"overridden_kb"`

	// OriginalRecommendation describes what the KB recommended
	OriginalRecommendation string `json:"original_recommendation"`

	// OverrideReason explains why ICU dominance took precedence
	OverrideReason string `json:"override_reason"`

	// ClinicalJustification provides clinical rationale
	ClinicalJustification string `json:"clinical_justification"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// KB-19 SPECIFIC CONTRACTS
// ═══════════════════════════════════════════════════════════════════════════════

// KB19Recommendation represents a recommendation from KB-19 Protocol Orchestrator.
//
// ARCHITECTURE CONSTRAINT (CTO/CMO):
// KB-19 provides RECOMMENDATIONS and EXPLANATIONS only.
// It cannot veto, override, or make safety decisions.
// ICU Intelligence has authority over all KB-19 recommendations.
type KB19Recommendation struct {
	// RecommendedProtocol is the suggested protocol/guideline
	RecommendedProtocol string `json:"recommended_protocol"`

	// Rationale explains why this protocol is recommended
	Rationale string `json:"rationale"`

	// EvidenceGrade is the evidence quality (from KB-15)
	EvidenceGrade string `json:"evidence_grade"`

	// DeferToICUIfDominant must always be true for high-risk actions
	// This flag acknowledges ICU authority
	DeferToICUIfDominant bool `json:"defer_to_icu_if_dominant"`

	// CanBeOverriddenByICU must always be true
	// KB-19 cannot make binding decisions
	CanBeOverriddenByICU bool `json:"can_be_overridden_by_icu"`
}

// NewKB19Recommendation creates a properly configured KB-19 recommendation.
// Per architecture: DeferToICUIfDominant and CanBeOverriddenByICU are always true.
func NewKB19Recommendation(protocol, rationale, evidenceGrade string) *KB19Recommendation {
	return &KB19Recommendation{
		RecommendedProtocol:  protocol,
		Rationale:            rationale,
		EvidenceGrade:        evidenceGrade,
		DeferToICUIfDominant: true, // MANDATORY: Always defer to ICU
		CanBeOverriddenByICU: true, // MANDATORY: ICU can always override
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════════

// IsHighRiskAction returns true if the action type is considered high-risk
// and must always check with ICU dominance first.
func IsHighRiskAction(actionType ActionType) bool {
	switch actionType {
	case ActionMedicationOrder, ActionProcedureStart, ActionDischarge:
		return true
	default:
		return false
	}
}

// RequiresICUClearance returns true if the action requires explicit
// ICU clearance before proceeding.
func RequiresICUClearance(action ProposedAction) bool {
	// High urgency or high risk always needs clearance
	if action.Urgency >= 7 || action.RiskLevel >= 7 {
		return true
	}
	// Specific action types need clearance
	return IsHighRiskAction(action.Type)
}
