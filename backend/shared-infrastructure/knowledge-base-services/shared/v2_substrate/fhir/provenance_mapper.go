package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// EvidenceTraceNodeToProvenance maps a clinical EvidenceTraceNode (state
// machines: Recommendation, Monitoring, ClinicalState) to a FHIR R4
// Provenance resource. System-level state machines (Authorisation, Consent)
// should route through EvidenceTraceNodeToAuditEvent instead.
//
// Field mapping:
//   - n.RecordedAt              → Provenance.recorded
//   - n.OccurredAt              → Provenance.occurredDateTime  (also
//                                 mirrored in a Vaidshala extension for
//                                 round-trip safety since Provenance.recorded
//                                 carries different semantics)
//   - n.Actor.RoleRef            → Provenance.agent[0].who.reference (Role/<id>)
//   - n.Actor.PersonRef          → Provenance.agent[0].onBehalfOf.reference (Practitioner/<id>)
//   - n.Actor.AuthorityBasisRef → Vaidshala extension (no FHIR-native field)
//   - n.Inputs                   → Provenance.entity[]  with role=source
//                                 and Vaidshala input-role extension
//   - n.Outputs                  → Provenance.target[]
//   - n.ReasoningSummary         → Vaidshala extension (JSON-encoded)
//   - n.StateMachine             → Vaidshala extension
//   - n.StateChangeType          → Vaidshala extension
//   - n.ResidentRef              → Vaidshala extension (system-only nodes
//                                 may have nil ResidentRef)
//
// The id of the resulting Provenance is the EvidenceTraceNode.ID.
func EvidenceTraceNodeToProvenance(n models.EvidenceTraceNode) (map[string]interface{}, error) {
	if err := validation.ValidateEvidenceTraceNode(n); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	out := map[string]interface{}{
		"resourceType": "Provenance",
		"id":           n.ID.String(),
		"recorded":     n.RecordedAt.UTC().Format(time.RFC3339),
	}
	if !n.OccurredAt.IsZero() {
		out["occurredDateTime"] = n.OccurredAt.UTC().Format(time.RFC3339)
	}

	// Targets: outputs become Provenance.target.
	if len(n.Outputs) > 0 {
		targets := make([]map[string]interface{}, 0, len(n.Outputs))
		for _, o := range n.Outputs {
			targets = append(targets, map[string]interface{}{
				"reference": o.OutputType + "/" + o.OutputRef.String(),
			})
		}
		out["target"] = targets
	}

	// Agent: actor's role + person.
	if n.Actor.RoleRef != nil || n.Actor.PersonRef != nil {
		agent := map[string]interface{}{}
		if n.Actor.RoleRef != nil {
			agent["who"] = map[string]interface{}{
				"reference": "Role/" + n.Actor.RoleRef.String(),
			}
		}
		if n.Actor.PersonRef != nil {
			agent["onBehalfOf"] = map[string]interface{}{
				"reference": "Practitioner/" + n.Actor.PersonRef.String(),
			}
		}
		out["agent"] = []map[string]interface{}{agent}
	}

	// Entities: inputs become Provenance.entity, each with a Vaidshala
	// input-role extension carrying RoleInDecision.
	if len(n.Inputs) > 0 {
		entities := make([]map[string]interface{}, 0, len(n.Inputs))
		for _, in := range n.Inputs {
			entity := map[string]interface{}{
				"role": "source",
				"what": map[string]interface{}{
					"reference": in.InputType + "/" + in.InputRef.String(),
				},
			}
			if in.RoleInDecision != "" {
				entity["extension"] = []map[string]interface{}{
					{"url": ExtEvidenceTraceInputRole, "valueCode": in.RoleInDecision},
				}
			}
			entities = append(entities, entity)
		}
		out["entity"] = entities
	}

	// Vaidshala extensions on the resource (state machine, state change
	// type, occurred_at mirror, reasoning summary, authority basis,
	// resident ref).
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

// ProvenanceToEvidenceTraceNode is the inverse mapper.
//
// Lossy fields (NOT round-tripped):
//   - CreatedAt — managed by canonical store, not on FHIR shape
func ProvenanceToEvidenceTraceNode(in map[string]interface{}) (*models.EvidenceTraceNode, error) {
	if rt, _ := in["resourceType"].(string); rt != "Provenance" {
		return nil, fmt.Errorf("resourceType: got %q want Provenance", rt)
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
	if s, _ := in["occurredDateTime"].(string); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			n.OccurredAt = t
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
			if onb, ok := a0["onBehalfOf"].(map[string]interface{}); ok {
				if ref, _ := onb["reference"].(string); ref != "" {
					if u := parsePrefixedRef(ref, "Practitioner/"); u != nil {
						n.Actor.PersonRef = u
					}
				}
			}
		}
	}

	// target[] → outputs
	if targets, ok := in["target"].([]interface{}); ok {
		for _, t := range targets {
			tm, _ := t.(map[string]interface{})
			ref, _ := tm["reference"].(string)
			if ref == "" {
				continue
			}
			ot, oid := splitFHIRRef(ref)
			if oid == uuid.Nil {
				continue
			}
			n.Outputs = append(n.Outputs, models.TraceOutput{OutputType: ot, OutputRef: oid})
		}
	}

	// entity[] → inputs (with Vaidshala input-role extension)
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
			it, iid := splitFHIRRef(whatRef)
			if iid == uuid.Nil {
				continue
			}
			role := ""
			if exts, ok := em["extension"].([]interface{}); ok {
				for _, x := range exts {
					xm, _ := x.(map[string]interface{})
					if u, _ := xm["url"].(string); u == ExtEvidenceTraceInputRole {
						role, _ = xm["valueCode"].(string)
					}
				}
			}
			n.Inputs = append(n.Inputs, models.TraceInput{
				InputType: it, InputRef: iid, RoleInDecision: role,
			})
		}
	}

	// resource-level extensions
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
					if t, err := time.Parse(time.RFC3339, v); err == nil && n.OccurredAt.IsZero() {
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

	if err := validation.ValidateEvidenceTraceNode(n); err != nil {
		return nil, fmt.Errorf("ingress validation: %w", err)
	}
	return &n, nil
}

// parsePrefixedRef returns the UUID portion of "Prefix/<uuid>" if and only
// if the string starts with prefix and the suffix parses cleanly. Returns
// nil otherwise.
func parsePrefixedRef(ref, prefix string) *uuid.UUID {
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		return nil
	}
	u, err := uuid.Parse(ref[len(prefix):])
	if err != nil {
		return nil
	}
	return &u
}

// splitFHIRRef splits "ResourceType/<uuid>" into ("ResourceType", uuid).
// Returns ("", uuid.Nil) on malformed input.
func splitFHIRRef(ref string) (string, uuid.UUID) {
	for i := 0; i < len(ref); i++ {
		if ref[i] == '/' {
			rt := ref[:i]
			if u, err := uuid.Parse(ref[i+1:]); err == nil {
				return rt, u
			}
			return "", uuid.Nil
		}
	}
	return "", uuid.Nil
}
