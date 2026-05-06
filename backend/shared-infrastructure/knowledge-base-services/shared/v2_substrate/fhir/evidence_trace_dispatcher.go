package fhir

// Wave 5.3 — EvidenceTrace → FHIR resource dispatcher.
//
// Layer 2 doc §1.6 dual-resource pattern: every EvidenceTraceNode produces
// EXACTLY ONE FHIR resource, either a Provenance OR an AuditEvent. The
// previous Wave 1R.2 mappers shipped both halves but routing was implicit
// (clinical state machines → Provenance, system state machines →
// AuditEvent). Wave 5.3 codifies the rule explicitly so:
//
//   - The dispatch decision is observable (callers can see which resource
//     a given node will become without invoking the mapper).
//   - System-event subtypes (rule_fire / credential_check) inside an
//     otherwise-clinical state machine route correctly.
//
// Routing rule (this is the Wave 5.3 hardening):
//
//   1. If the node's state_change_type is one of the system-event tags
//      (rule_fire, credential_check, query_recorded, login_propagated)
//      → AuditEvent. Regardless of state_machine.
//   2. Otherwise dispatch by state_machine:
//      - Recommendation, Monitoring, ClinicalState   → Provenance
//      - Authorisation, Consent                       → Provenance
//        (NOTE: Wave 5.3 plan task says these route to Provenance because
//         a normal credential-grant or capacity-affirmation transition is
//         a clinical-relevant state change. Only the system-event subtypes
//         above route to AuditEvent.)
//
// Mutually exclusive: this function returns exactly one (resourceType,
// resource) pair per node.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// SystemEventStateChangeTypes is the closed set of state_change_type tags
// that always route to AuditEvent regardless of state_machine. Add to this
// list (and update the AuditEvent mapper test) when introducing a new
// system-event subtype.
var SystemEventStateChangeTypes = map[string]struct{}{
	"rule_fire":         {},
	"credential_check":  {},
	"query_recorded":    {},
	"login_propagated":  {},
}

// IsSystemEventStateChange reports whether the StateChangeType marks a
// system event (routes to AuditEvent).
func IsSystemEventStateChange(stateChangeType string) bool {
	_, ok := SystemEventStateChangeTypes[strings.ToLower(strings.TrimSpace(stateChangeType))]
	return ok
}

// ResourceTypeProvenance / ResourceTypeAuditEvent are the dispatcher's
// return values for clarity at call sites.
const (
	ResourceTypeProvenance = "Provenance"
	ResourceTypeAuditEvent = "AuditEvent"
)

// RouteEvidenceTrace returns the FHIR resourceType ("Provenance" or
// "AuditEvent") for the given node WITHOUT actually mapping it. Useful
// for routing decisions in egress pipelines and for testing the rule
// in isolation.
func RouteEvidenceTrace(n models.EvidenceTraceNode) (string, error) {
	if !models.IsValidEvidenceTraceStateMachine(n.StateMachine) {
		return "", fmt.Errorf("evidence_trace dispatcher: unrecognised state_machine %q", n.StateMachine)
	}
	if IsSystemEventStateChange(n.StateChangeType) {
		return ResourceTypeAuditEvent, nil
	}
	switch n.StateMachine {
	case models.EvidenceTraceStateMachineRecommendation,
		models.EvidenceTraceStateMachineMonitoring,
		models.EvidenceTraceStateMachineClinicalState,
		models.EvidenceTraceStateMachineAuthorisation,
		models.EvidenceTraceStateMachineConsent:
		return ResourceTypeProvenance, nil
	}
	// Defensive: IsValidEvidenceTraceStateMachine above should make this
	// unreachable.
	return "", errors.New("evidence_trace dispatcher: unroutable node")
}

// MapEvidenceTrace dispatches one EvidenceTraceNode to either a Provenance
// or an AuditEvent resource per the Wave 5.3 routing rule. The returned
// (resourceType, resource) pair is the single FHIR resource the egress
// pipeline should emit for this node.
//
// Mutually exclusive: never returns both; never returns neither.
func MapEvidenceTrace(n models.EvidenceTraceNode) (resourceType string, resource map[string]interface{}, err error) {
	rt, err := RouteEvidenceTrace(n)
	if err != nil {
		return "", nil, err
	}
	switch rt {
	case ResourceTypeProvenance:
		res, err := EvidenceTraceNodeToProvenance(n)
		if err != nil {
			return "", nil, err
		}
		return rt, res, nil
	case ResourceTypeAuditEvent:
		res, err := EvidenceTraceNodeToAuditEvent(n)
		if err != nil {
			return "", nil, err
		}
		return rt, res, nil
	}
	return "", nil, errors.New("evidence_trace dispatcher: internal routing inconsistency")
}
