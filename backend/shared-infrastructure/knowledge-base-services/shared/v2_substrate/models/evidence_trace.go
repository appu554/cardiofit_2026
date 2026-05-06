package models

import (
	"time"

	"github.com/google/uuid"
)

// EvidenceTraceNode is the v2 substrate's clinical-reasoning audit node — the
// architectural moat. Per Layer 2 doc §1.6 and Recommendation 3 of Part 7.
//
// EvidenceTrace vs Provenance vs AuditEvent (the dual+1 pattern):
//   - FHIR Provenance: who modified which clinical resource, when, on what
//     authority, from what inputs (resource-history layer).
//   - FHIR AuditEvent: system/security events (login, query, access)
//     (operational-logging layer).
//   - EvidenceTraceNode (Vaidshala-specific, sits on top): clinical-reasoning
//     chain — bidirectional graph linking observations → interpretations →
//     recommendations → decisions → outcomes.
//
// Edges are NOT embedded in the node; they live in their own evidence_trace
// edges table (see shared/v2_substrate/evidence_trace) so traversal queries
// can use a recursive CTE without parsing JSON arrays. The graph is queryable
// in BOTH directions from day 1.
//
// Canonical storage: kb-20-patient-profile (evidence_trace_nodes +
// evidence_trace_edges, migration 009). Wave 5 will harden retention and
// indexing; this is the foundational schema and write path.
type EvidenceTraceNode struct {
	ID              uuid.UUID `json:"id"`
	StateMachine    string    `json:"state_machine"`     // see EvidenceTraceStateMachine* constants
	StateChangeType string    `json:"state_change_type"` // free-form structured tag e.g. "draft -> submitted"

	RecordedAt time.Time `json:"recorded_at"` // when this node was logged
	OccurredAt time.Time `json:"occurred_at"` // when the underlying state change happened (may differ)

	Actor TraceActor `json:"actor"`

	Inputs           []TraceInput      `json:"inputs,omitempty"`
	ReasoningSummary *ReasoningSummary `json:"reasoning_summary,omitempty"` // nullable
	Outputs          []TraceOutput     `json:"outputs,omitempty"`

	// ResidentRef is nullable: system-only nodes (rule_fire on global config,
	// credential checks not yet bound to a resident) have no resident.
	ResidentRef *uuid.UUID `json:"resident_ref,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// TraceActor identifies who took the recorded action and on what authority.
// All three refs are nullable: system-only nodes may have no person/role
// (e.g. background rule evaluations); authority basis is nullable because
// not every transition is gated by an explicit credential.
type TraceActor struct {
	RoleRef           *uuid.UUID `json:"role_ref,omitempty"`
	PersonRef         *uuid.UUID `json:"person_ref,omitempty"`
	AuthorityBasisRef *uuid.UUID `json:"authority_basis_ref,omitempty"` // Credential / PrescribingAgreement
}

// TraceInput is one input entity that fed into the recorded reasoning step.
// RoleInDecision distinguishes primary evidence from supportive context vs
// counter-evidence that was weighed and rejected.
type TraceInput struct {
	InputType      string    `json:"input_type"`       // see TraceInputType* constants
	InputRef       uuid.UUID `json:"input_ref"`        // reference to the input entity
	RoleInDecision string    `json:"role_in_decision"` // see TraceRoleInDecision* constants
}

// ReasoningSummary captures the rule-engine's reasoning trail at decision
// time. All slice fields are pure rule_id / suppression_id strings; the
// model layer does not interpret them — that's downstream work for the
// rule-engine consumer.
type ReasoningSummary struct {
	Text                          string   `json:"text,omitempty"`
	RuleFires                     []string `json:"rule_fires,omitempty"`
	SuppressionsEvaluated         []string `json:"suppressions_evaluated,omitempty"`
	SuppressionsFired             []string `json:"suppressions_fired,omitempty"`
	AlternativesConsidered        []string `json:"alternatives_considered,omitempty"`
	AlternativeSelectionRationale string   `json:"alternative_selection_rationale,omitempty"`
}

// TraceOutput is one entity the recorded reasoning step produced.
type TraceOutput struct {
	OutputType string    `json:"output_type"` // free-form: Recommendation | MonitoringPlan | RecommendationStateChange | ...
	OutputRef  uuid.UUID `json:"output_ref"`
}

// State-machine identifiers (Layer 2 doc §1.6).
const (
	EvidenceTraceStateMachineAuthorisation  = "Authorisation"
	EvidenceTraceStateMachineRecommendation = "Recommendation"
	EvidenceTraceStateMachineMonitoring     = "Monitoring"
	EvidenceTraceStateMachineClinicalState  = "ClinicalState"
	EvidenceTraceStateMachineConsent        = "Consent"
)

// IsValidEvidenceTraceStateMachine reports whether s is one of the
// recognised state-machine identifiers.
func IsValidEvidenceTraceStateMachine(s string) bool {
	switch s {
	case EvidenceTraceStateMachineAuthorisation,
		EvidenceTraceStateMachineRecommendation,
		EvidenceTraceStateMachineMonitoring,
		EvidenceTraceStateMachineClinicalState,
		EvidenceTraceStateMachineConsent:
		return true
	}
	return false
}

// TraceInput.RoleInDecision values.
const (
	TraceRoleInDecisionSupportive        = "supportive"
	TraceRoleInDecisionPrimaryEvidence   = "primary_evidence"
	TraceRoleInDecisionSecondaryEvidence = "secondary_evidence"
	TraceRoleInDecisionCounterEvidence   = "counter_evidence"
)

// IsValidTraceRoleInDecision reports whether s is recognised.
func IsValidTraceRoleInDecision(s string) bool {
	switch s {
	case TraceRoleInDecisionSupportive,
		TraceRoleInDecisionPrimaryEvidence,
		TraceRoleInDecisionSecondaryEvidence,
		TraceRoleInDecisionCounterEvidence:
		return true
	}
	return false
}

// TraceInput.InputType common values (Layer 2 doc §1.6: "Observation |
// MedicineUse | Event | Condition | Consent | ScopeRule | Rule | other").
// The list is not closed; ValidateEvidenceTraceNode does not enforce a
// specific value, just non-empty.
const (
	TraceInputTypeObservation = "Observation"
	TraceInputTypeMedicineUse = "MedicineUse"
	TraceInputTypeEvent       = "Event"
	TraceInputTypeCondition   = "Condition"
	TraceInputTypeConsent     = "Consent"
	TraceInputTypeScopeRule   = "ScopeRule"
	TraceInputTypeRule        = "Rule"
	TraceInputTypeOther       = "other"
)

// IsSystemEvidenceTraceStateMachine reports whether s is a state machine
// whose nodes typically represent system-level operations (vs clinical
// resource changes). Used by the FHIR mapper to choose AuditEvent vs
// Provenance routing. Authorisation and Consent state changes are
// system-level (security/governance); the others are clinical.
func IsSystemEvidenceTraceStateMachine(s string) bool {
	switch s {
	case EvidenceTraceStateMachineAuthorisation, EvidenceTraceStateMachineConsent:
		return true
	}
	return false
}
