package fhir

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// FHIR routing for ActiveConcern (Layer 2 doc §2.3 / Wave 2.3):
//
// ActiveConcern → AU FHIR Condition. The Condition resource captures
// "a clinical condition, problem, diagnosis, or other event ... that has
// risen to a level of concern" (FHIR R4 §16.4) — the closest native
// equivalent. Vaidshala-specific fields ride as Vaidshala-namespaced
// extensions on the Condition.
//
// Mapping table:
//
//   ActiveConcern field          → FHIR Condition target
//   ----------------------------   ---------------------------------------
//   ID                          → Condition.id
//   ResidentID                  → Condition.subject.reference (Patient/<uuid>)
//   ConcernType                 → Vaidshala extension + Condition.code.text
//   StartedAt                   → Condition.onsetDateTime
//   StartedByEventRef           → Vaidshala extension (Encounter/<uuid>)
//   ExpectedResolutionAt        → Vaidshala extension (RFC3339 valueDateTime)
//   OwnerRoleRef                → Vaidshala extension (PractitionerRole/<uuid>)
//   RelatedMonitoringPlanRef    → Vaidshala extension
//   ResolutionStatus            → Vaidshala extension + Condition.clinicalStatus
//   ResolvedAt                  → Condition.abatementDateTime + extension
//   ResolutionEvidenceTraceRef  → Vaidshala extension
//   Notes                       → Condition.note[0].text
//
// Lossy fields: CreatedAt / UpdatedAt are managed by the canonical store
// and not round-tripped through FHIR.

// ActiveConcernToFHIRCondition translates a v2 ActiveConcern to an AU FHIR
// Condition resource as a generic map[string]interface{} suitable for
// json.Marshal to wire format.
//
// Egress validates the input via validation.ValidateActiveConcern before
// constructing the FHIR map.
func ActiveConcernToFHIRCondition(c models.ActiveConcern) (map[string]interface{}, error) {
	if err := validation.ValidateActiveConcern(c); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	out := map[string]interface{}{
		"resourceType": "Condition",
		"id":           c.ID.String(),
		"subject": map[string]interface{}{
			"reference": "Patient/" + c.ResidentID.String(),
		},
		"onsetDateTime": c.StartedAt.UTC().Format(time.RFC3339),
		"clinicalStatus": map[string]interface{}{
			"coding": []map[string]interface{}{{
				"system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
				"code":   conditionClinicalStatusFor(c.ResolutionStatus),
			}},
		},
		"code": map[string]interface{}{"text": c.ConcernType},
	}

	if c.ResolvedAt != nil {
		out["abatementDateTime"] = c.ResolvedAt.UTC().Format(time.RFC3339)
	}

	if c.Notes != "" {
		out["note"] = []map[string]interface{}{{"text": c.Notes}}
	}

	exts := []map[string]interface{}{
		{"url": ExtActiveConcernType, "valueCode": c.ConcernType},
		{"url": ExtActiveConcernResolutionStatus, "valueCode": c.ResolutionStatus},
		{"url": ExtActiveConcernExpectedResolutionAt,
			"valueDateTime": c.ExpectedResolutionAt.UTC().Format(time.RFC3339)},
	}
	if c.StartedByEventRef != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtActiveConcernStartedByEventRef,
			"valueString": "Encounter/" + c.StartedByEventRef.String(),
		})
	}
	if c.OwnerRoleRef != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtActiveConcernOwnerRoleRef,
			"valueString": "PractitionerRole/" + c.OwnerRoleRef.String(),
		})
	}
	if c.RelatedMonitoringPlanRef != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtActiveConcernRelatedMonitoringPlanRef,
			"valueString": c.RelatedMonitoringPlanRef.String(),
		})
	}
	if c.ResolvedAt != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtActiveConcernResolvedAt,
			"valueDateTime": c.ResolvedAt.UTC().Format(time.RFC3339),
		})
	}
	if c.ResolutionEvidenceTraceRef != nil {
		exts = append(exts, map[string]interface{}{
			"url": ExtActiveConcernResolutionEvidenceTraceRef,
			"valueString": c.ResolutionEvidenceTraceRef.String(),
		})
	}
	out["extension"] = exts
	return out, nil
}

// conditionClinicalStatusFor maps a v2 ResolutionStatus to the FHIR R4
// condition-clinical CodeSystem (active | recurrence | relapse | inactive
// | remission | resolved). Open → active; resolved_stop_criteria →
// resolved; escalated → active (the concern is still on the radar, just
// transferred to a higher-acuity workflow); expired_unresolved → inactive
// (no longer being monitored).
func conditionClinicalStatusFor(s string) string {
	switch s {
	case models.ResolutionStatusOpen:
		return "active"
	case models.ResolutionStatusResolvedStopCriteria:
		return "resolved"
	case models.ResolutionStatusEscalated:
		return "active"
	case models.ResolutionStatusExpiredUnresolved:
		return "inactive"
	default:
		return "active"
	}
}

// FHIRConditionToActiveConcern is the inverse mapping. Validates the
// reconstructed ActiveConcern via validation.ValidateActiveConcern.
func FHIRConditionToActiveConcern(in map[string]interface{}) (*models.ActiveConcern, error) {
	rt, _ := in["resourceType"].(string)
	if rt != "Condition" {
		return nil, fmt.Errorf("resourceType: got %q want Condition", rt)
	}

	var c models.ActiveConcern

	if idStr, _ := in["id"].(string); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			c.ID = id
		}
	}

	if subj, ok := in["subject"].(map[string]interface{}); ok {
		if ref, _ := subj["reference"].(string); strings.HasPrefix(ref, "Patient/") {
			if rid, err := uuid.Parse(strings.TrimPrefix(ref, "Patient/")); err == nil {
				c.ResidentID = rid
			}
		}
	}

	if s, _ := in["onsetDateTime"].(string); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			c.StartedAt = t
		}
	}
	if s, _ := in["abatementDateTime"].(string); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			c.ResolvedAt = &t
		}
	}

	if notes, ok := in["note"].([]interface{}); ok && len(notes) > 0 {
		if n0, ok := notes[0].(map[string]interface{}); ok {
			if t, _ := n0["text"].(string); t != "" {
				c.Notes = t
			}
		}
	}

	// Vaidshala extensions carry the v2-distinguishing fields.
	if exts, ok := in["extension"].([]interface{}); ok {
		for _, x := range exts {
			em, _ := x.(map[string]interface{})
			url, _ := em["url"].(string)
			switch url {
			case ExtActiveConcernType:
				if v, _ := em["valueCode"].(string); v != "" {
					c.ConcernType = v
				}
			case ExtActiveConcernResolutionStatus:
				if v, _ := em["valueCode"].(string); v != "" {
					c.ResolutionStatus = v
				}
			case ExtActiveConcernExpectedResolutionAt:
				if v, _ := em["valueDateTime"].(string); v != "" {
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						c.ExpectedResolutionAt = t
					}
				}
			case ExtActiveConcernStartedByEventRef:
				if v, _ := em["valueString"].(string); v != "" {
					if u, err := uuid.Parse(strings.TrimPrefix(v, "Encounter/")); err == nil {
						c.StartedByEventRef = &u
					}
				}
			case ExtActiveConcernOwnerRoleRef:
				if v, _ := em["valueString"].(string); v != "" {
					if u, err := uuid.Parse(strings.TrimPrefix(v, "PractitionerRole/")); err == nil {
						c.OwnerRoleRef = &u
					}
				}
			case ExtActiveConcernRelatedMonitoringPlanRef:
				if v, _ := em["valueString"].(string); v != "" {
					if u, err := uuid.Parse(v); err == nil {
						c.RelatedMonitoringPlanRef = &u
					}
				}
			case ExtActiveConcernResolvedAt:
				// Prefer the extension over abatementDateTime when both
				// are present; semantics are the same but the extension
				// is the source-of-truth on egress.
				if v, _ := em["valueDateTime"].(string); v != "" {
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						c.ResolvedAt = &t
					}
				}
			case ExtActiveConcernResolutionEvidenceTraceRef:
				if v, _ := em["valueString"].(string); v != "" {
					if u, err := uuid.Parse(v); err == nil {
						c.ResolutionEvidenceTraceRef = &u
					}
				}
			}
		}
	}

	if err := validation.ValidateActiveConcern(c); err != nil {
		return nil, fmt.Errorf("ingress validation: %w", err)
	}
	return &c, nil
}
