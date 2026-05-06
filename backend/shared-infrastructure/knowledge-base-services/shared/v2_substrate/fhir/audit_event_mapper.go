package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// EvidenceTraceNodeToAuditEvent maps a system-level EvidenceTraceNode
// (state machines: Authorisation, Consent — see
// IsSystemEvidenceTraceStateMachine) to a FHIR R4 AuditEvent resource.
//
// Clinical state machines (Recommendation, Monitoring, ClinicalState)
// should use EvidenceTraceNodeToProvenance instead.
//
// Field mapping:
//   - n.RecordedAt              → AuditEvent.recorded
//   - n.Actor.RoleRef            → AuditEvent.agent[0].who.reference (Role/<id>)
//   - n.Actor.PersonRef          → AuditEvent.agent[0].altId (Practitioner/<id>)
//   - n.Inputs                   → AuditEvent.entity[]
//   - n.Outputs                  → AuditEvent.entity[] (concat'd; outputs
//                                 of system events are also audited)
//   - n.StateMachine             → AuditEvent.type (system + code) +
//                                 Vaidshala extension
//   - StateChangeType, OccurredAt, ReasoningSummary, AuthorityBasisRef,
//     ResidentRef → Vaidshala-namespaced extensions
//
// AuditEvent.type uses a Vaidshala-namespaced Coding because the AU FHIR
// AuditEvent type valueset doesn't cover clinical-reasoning state changes.
func EvidenceTraceNodeToAuditEvent(n models.EvidenceTraceNode) (map[string]interface{}, error) {
	if err := validation.ValidateEvidenceTraceNode(n); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	out := map[string]interface{}{
		"resourceType": "AuditEvent",
		"id":           n.ID.String(),
		"recorded":     n.RecordedAt.UTC().Format(time.RFC3339),
		"type": map[string]interface{}{
			"system":  ExtEvidenceTraceStateMachine,
			"code":    n.StateMachine,
			"display": n.StateMachine,
		},
		// AuditEvent.action is a single character per FHIR R4 spec; "E"
		// (execute) is the safest default for a state-machine transition.
		"action": "E",
		// outcome 0 = success — system events that record an actual
		// transition are by definition successful (failed transitions are
		// recorded as a separate node with a different state_change_type).
		"outcome": "0",
	}

	// agent: actor's role + person.
	if n.Actor.RoleRef != nil || n.Actor.PersonRef != nil {
		agent := map[string]interface{}{
			"requestor": true,
		}
		if n.Actor.RoleRef != nil {
			agent["who"] = map[string]interface{}{
				"reference": "Role/" + n.Actor.RoleRef.String(),
			}
		}
		if n.Actor.PersonRef != nil {
			agent["altId"] = "Practitioner/" + n.Actor.PersonRef.String()
		}
		out["agent"] = []map[string]interface{}{agent}
	}

	// entity: inputs and outputs both ride here. We tag the original kind
	// (input/output) on a Vaidshala extension so round-trip is unambiguous.
	var entities []map[string]interface{}
	for _, in := range n.Inputs {
		entity := map[string]interface{}{
			"what": map[string]interface{}{
				"reference": in.InputType + "/" + in.InputRef.String(),
			},
			// AuditEvent.entity.role uses an integer code per FHIR R4; 4 = "Domain Resource"
			// is the closest fit for a generic referenced resource.
			"role": map[string]interface{}{
				"system":  "http://terminology.hl7.org/CodeSystem/object-role",
				"code":    "4",
				"display": "Domain Resource",
			},
		}
		exts := []map[string]interface{}{
			{"url": ExtEvidenceTraceInputRole, "valueCode": in.RoleInDecision},
			{"url": "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-entity-kind", "valueCode": "input"},
		}
		entity["extension"] = exts
		entities = append(entities, entity)
	}
	for _, o := range n.Outputs {
		entity := map[string]interface{}{
			"what": map[string]interface{}{
				"reference": o.OutputType + "/" + o.OutputRef.String(),
			},
			"role": map[string]interface{}{
				"system":  "http://terminology.hl7.org/CodeSystem/object-role",
				"code":    "4",
				"display": "Domain Resource",
			},
			"extension": []map[string]interface{}{
				{"url": "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-entity-kind", "valueCode": "output"},
			},
		}
		entities = append(entities, entity)
	}
	if len(entities) > 0 {
		out["entity"] = entities
	}

	// Resource-level Vaidshala extensions.
	exts := []map[string]interface{}{
		{"url": ExtEvidenceTraceStateMachine, "valueCode": n.StateMachine},
		{"url": ExtEvidenceTraceStateChangeType, "valueString": n.StateChangeType},
		{"url": ExtEvidenceTraceOccurredAt, "valueDateTime": n.OccurredAt.UTC().Format(time.RFC3339)},
	}
	if n.Actor.AuthorityBasisRef != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtEvidenceTraceAuthorityBasis, "valueString": n.Actor.AuthorityBasisRef.String(),
		})
	}
	if n.ResidentRef != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtEvidenceTraceResidentRef, "valueString": n.ResidentRef.String(),
		})
	}
	if n.ReasoningSummary != nil {
		b, err := json.Marshal(n.ReasoningSummary)
		if err != nil {
			return nil, fmt.Errorf("marshal reasoning_summary: %w", err)
		}
		exts = append(exts, map[string]interface{}{
			"url": ExtEvidenceTraceReasoningSummary, "valueString": string(b),
		})
	}
	out["extension"] = exts
	return out, nil
}

// AuditEventToEvidenceTraceNode is the inverse mapper.
func AuditEventToEvidenceTraceNode(in map[string]interface{}) (*models.EvidenceTraceNode, error) {
	if rt, _ := in["resourceType"].(string); rt != "AuditEvent" {
		return nil, fmt.Errorf("resourceType: got %q want AuditEvent", rt)
	}

	var n models.EvidenceTraceNode

	if idStr, _ := in["id"].(string); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			n.ID = id
		}
	}
	if s, _ := in["recorded"].(string); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			n.RecordedAt = t
		}
	}

	// agent[0] → actor
	if agents, ok := in["agent"].([]interface{}); ok && len(agents) > 0 {
		if a0, ok := agents[0].(map[string]interface{}); ok {
			if who, ok := a0["who"].(map[string]interface{}); ok {
				if ref, _ := who["reference"].(string); ref != "" {
					if u := parsePrefixedRef(ref, "Role/"); u != nil {
						n.Actor.RoleRef = u
					}
				}
			}
			if altID, _ := a0["altId"].(string); altID != "" {
				if u := parsePrefixedRef(altID, "Practitioner/"); u != nil {
					n.Actor.PersonRef = u
				}
			}
		}
	}

	// entity[] → split into inputs vs outputs by the entity-kind extension
	if entities, ok := in["entity"].([]interface{}); ok {
		for _, e := range entities {
			em, _ := e.(map[string]interface{})
			whatRef := ""
			if what, ok := em["what"].(map[string]interface{}); ok {
				whatRef, _ = what["reference"].(string)
			}
			if whatRef == "" {
				continue
			}
			rt, refUUID := splitFHIRRef(whatRef)
			if refUUID == uuid.Nil {
				continue
			}

			kind, role := "", ""
			if exts, ok := em["extension"].([]interface{}); ok {
				for _, x := range exts {
					xm, _ := x.(map[string]interface{})
					switch u, _ := xm["url"].(string); u {
					case "https://vaidshala.health/fhir/StructureDefinition/evidence-trace-entity-kind":
						kind, _ = xm["valueCode"].(string)
					case ExtEvidenceTraceInputRole:
						role, _ = xm["valueCode"].(string)
					}
				}
			}

			switch kind {
			case "output":
				n.Outputs = append(n.Outputs, models.TraceOutput{OutputType: rt, OutputRef: refUUID})
			default: // "input" or unspecified — default to input (safer for round-trip)
				n.Inputs = append(n.Inputs, models.TraceInput{
					InputType: rt, InputRef: refUUID, RoleInDecision: role,
				})
			}
		}
	}

	// resource-level Vaidshala extensions
	if exts, ok := in["extension"].([]interface{}); ok {
		for _, x := range exts {
			xm, _ := x.(map[string]interface{})
			url, _ := xm["url"].(string)
			switch url {
			case ExtEvidenceTraceStateMachine:
				if v, _ := xm["valueCode"].(string); v != "" {
					n.StateMachine = v
				}
			case ExtEvidenceTraceStateChangeType:
				if v, _ := xm["valueString"].(string); v != "" {
					n.StateChangeType = v
				}
			case ExtEvidenceTraceOccurredAt:
				if v, _ := xm["valueDateTime"].(string); v != "" {
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						n.OccurredAt = t
					}
				}
			case ExtEvidenceTraceAuthorityBasis:
				if v, _ := xm["valueString"].(string); v != "" {
					if u, err := uuid.Parse(v); err == nil {
						n.Actor.AuthorityBasisRef = &u
					}
				}
			case ExtEvidenceTraceResidentRef:
				if v, _ := xm["valueString"].(string); v != "" {
					if u, err := uuid.Parse(v); err == nil {
						n.ResidentRef = &u
					}
				}
			case ExtEvidenceTraceReasoningSummary:
				if v, _ := xm["valueString"].(string); v != "" {
					var rs models.ReasoningSummary
					if err := json.Unmarshal([]byte(v), &rs); err == nil {
						n.ReasoningSummary = &rs
					}
				}
			}
		}
	}

	// Fallback: if state_machine wasn't carried in extensions, try the
	// AuditEvent.type.code (which we set on egress).
	if n.StateMachine == "" {
		if t, ok := in["type"].(map[string]interface{}); ok {
			if c, _ := t["code"].(string); c != "" {
				n.StateMachine = c
			}
		}
	}

	if err := validation.ValidateEvidenceTraceNode(n); err != nil {
		return nil, fmt.Errorf("ingress validation: %w", err)
	}
	return &n, nil
}
