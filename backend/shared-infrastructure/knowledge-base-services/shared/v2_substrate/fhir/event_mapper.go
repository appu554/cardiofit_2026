package fhir

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// FHIR routing for Event (Layer 2 doc §1.5):
//
//   - Clinical events       (fall, pressure_injury, behavioural_incident,
//                            medication_error, adverse_drug_event)
//                          → AU FHIR Encounter
//   - Care transitions      (hospital_admission/_discharge, GP_visit,
//                            specialist_visit, ED_presentation,
//                            end_of_life_recognition, death)
//                          → AU FHIR Encounter
//   - Administrative events (admission_to_facility, transfer_between_facilities,
//                            care_planning_meeting, family_meeting)
//                          → AU FHIR Encounter
//   - System events         (rule_fire, recommendation_*, monitoring_plan_*,
//                            consent_*, credential_*)
//                          → AU FHIR Communication
//
// Vaidshala-specific fields (event_type discriminator, severity,
// description_structured, reportable_under, related_observations,
// related_medication_uses, triggered_state_changes, reported_by_ref,
// witnessed_by_refs) ride as Vaidshala-namespaced extensions on whichever
// resource we routed to. The mappers preserve the routing across round-trip
// because EventType + the Vaidshala extension carry enough information to
// reconstruct the original Event.

// fhirResourceTypeForEvent returns the FHIR resourceType to use for e.
// Defaults to Encounter for unknown event types so future Layer 2 spec
// additions stay safely-routed; explicit System-bucket events go to
// Communication.
func fhirResourceTypeForEvent(eventType string) string {
	if models.IsSystemEventType(eventType) {
		return "Communication"
	}
	return "Encounter"
}

// EventToAUFHIR translates a v2 Event to an AU FHIR Encounter or
// Communication resource (per fhirResourceTypeForEvent) as a generic
// map[string]interface{} suitable for json.Marshal to wire format.
//
// Egress validates the input via validation.ValidateEvent before
// constructing the FHIR map. Ingress validates the output.
//
// Lossy fields (NOT round-tripped):
//   - CreatedAt / UpdatedAt — managed by canonical store, not FHIR shape
func EventToAUFHIR(e models.Event) (map[string]interface{}, error) {
	if err := validation.ValidateEvent(e); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	resourceType := fhirResourceTypeForEvent(e.EventType)
	out := map[string]interface{}{
		"resourceType": resourceType,
		"id":           e.ID.String(),
		"status":       fhirStatusForEvent(resourceType),
		"subject": map[string]interface{}{
			"reference": "Patient/" + e.ResidentID.String(),
		},
	}

	// occurredAt → period.start (Encounter) or sent (Communication)
	occurred := e.OccurredAt.UTC().Format(time.RFC3339)
	switch resourceType {
	case "Encounter":
		out["period"] = map[string]interface{}{"start": occurred}
		if e.OccurredAtFacility != nil {
			out["location"] = []map[string]interface{}{
				{"location": map[string]interface{}{
					"reference": "Location/" + e.OccurredAtFacility.String(),
				}},
			}
		}
	case "Communication":
		out["sent"] = occurred
	}

	// Vaidshala-namespaced extensions encode the v2-distinguishing fields.
	extensions := []map[string]interface{}{
		{"url": ExtEventType, "valueCode": e.EventType},
		{"url": ExtEventReportedBy, "valueString": e.ReportedByRef.String()},
	}
	if e.Severity != "" {
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventSeverity, "valueCode": e.Severity,
		})
	}
	if len(e.WitnessedByRefs) > 0 {
		ids := make([]string, len(e.WitnessedByRefs))
		for i, u := range e.WitnessedByRefs {
			ids[i] = u.String()
		}
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventWitnessedBy, "valueString": strings.Join(ids, ","),
		})
	}
	if len(e.DescriptionStructured) > 0 {
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventDescriptionStructured, "valueString": string(e.DescriptionStructured),
		})
	}
	if len(e.ReportableUnder) > 0 {
		b, err := json.Marshal(e.ReportableUnder)
		if err != nil {
			return nil, fmt.Errorf("marshal reportable_under: %w", err)
		}
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventReportableUnder, "valueString": string(b),
		})
	}
	if len(e.RelatedObservations) > 0 {
		ids := make([]string, len(e.RelatedObservations))
		for i, u := range e.RelatedObservations {
			ids[i] = u.String()
		}
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventRelatedObservations, "valueString": strings.Join(ids, ","),
		})
	}
	if len(e.RelatedMedicationUses) > 0 {
		ids := make([]string, len(e.RelatedMedicationUses))
		for i, u := range e.RelatedMedicationUses {
			ids[i] = u.String()
		}
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventRelatedMedicationUses, "valueString": strings.Join(ids, ","),
		})
	}
	if len(e.TriggeredStateChanges) > 0 {
		b, err := json.Marshal(e.TriggeredStateChanges)
		if err != nil {
			return nil, fmt.Errorf("marshal triggered_state_changes: %w", err)
		}
		extensions = append(extensions, map[string]interface{}{
			"url": ExtEventTriggeredStateChanges, "valueString": string(b),
		})
	}

	// description_free_text rides on the native FHIR Encounter.reasonCode.text
	// or Communication.payload — pick the one that matches the routed
	// resourceType to stay close to spec.
	if e.DescriptionFreeText != "" {
		switch resourceType {
		case "Encounter":
			out["reasonCode"] = []map[string]interface{}{
				{"text": e.DescriptionFreeText},
			}
		case "Communication":
			out["payload"] = []map[string]interface{}{
				{"contentString": e.DescriptionFreeText},
			}
		}
	}

	out["extension"] = extensions
	return out, nil
}

// fhirStatusForEvent picks a sensible default status code for the routed
// resource. Encounter.status uses "finished" (the event has occurred);
// Communication.status uses "completed".
func fhirStatusForEvent(resourceType string) string {
	switch resourceType {
	case "Communication":
		return "completed"
	default:
		return "finished"
	}
}

// AUFHIRToEvent translates an AU FHIR Encounter or Communication JSON map
// back to a v2 Event. Returns an error if the resourceType is neither
// Encounter nor Communication, or if the resulting Event fails
// ValidateEvent.
func AUFHIRToEvent(in map[string]interface{}) (*models.Event, error) {
	rt, _ := in["resourceType"].(string)
	if rt != "Encounter" && rt != "Communication" {
		return nil, fmt.Errorf("resourceType: got %q want Encounter|Communication", rt)
	}

	var e models.Event

	if idStr, _ := in["id"].(string); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			e.ID = id
		}
	}

	// subject.reference -> ResidentID
	if subj, ok := in["subject"].(map[string]interface{}); ok {
		if ref, _ := subj["reference"].(string); strings.HasPrefix(ref, "Patient/") {
			if rid, err := uuid.Parse(strings.TrimPrefix(ref, "Patient/")); err == nil {
				e.ResidentID = rid
			}
		}
	}

	// occurredAt: Encounter.period.start vs Communication.sent
	switch rt {
	case "Encounter":
		if period, ok := in["period"].(map[string]interface{}); ok {
			if s, _ := period["start"].(string); s != "" {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					e.OccurredAt = t
				}
			}
		}
		if locs, ok := in["location"].([]interface{}); ok && len(locs) > 0 {
			if loc0, ok := locs[0].(map[string]interface{}); ok {
				if locRef, ok := loc0["location"].(map[string]interface{}); ok {
					if ref, _ := locRef["reference"].(string); strings.HasPrefix(ref, "Location/") {
						if fid, err := uuid.Parse(strings.TrimPrefix(ref, "Location/")); err == nil {
							e.OccurredAtFacility = &fid
						}
					}
				}
			}
		}
		// reasonCode[0].text -> description_free_text
		if rc, ok := in["reasonCode"].([]interface{}); ok && len(rc) > 0 {
			if rc0, ok := rc[0].(map[string]interface{}); ok {
				if txt, _ := rc0["text"].(string); txt != "" {
					e.DescriptionFreeText = txt
				}
			}
		}
	case "Communication":
		if s, _ := in["sent"].(string); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				e.OccurredAt = t
			}
		}
		// payload[0].contentString -> description_free_text
		if pl, ok := in["payload"].([]interface{}); ok && len(pl) > 0 {
			if pl0, ok := pl[0].(map[string]interface{}); ok {
				if cs, _ := pl0["contentString"].(string); cs != "" {
					e.DescriptionFreeText = cs
				}
			}
		}
	}

	// extensions
	if exts, ok := in["extension"].([]interface{}); ok {
		for _, x := range exts {
			em, _ := x.(map[string]interface{})
			url, _ := em["url"].(string)
			switch url {
			case ExtEventType:
				if v, _ := em["valueCode"].(string); v != "" {
					e.EventType = v
				}
			case ExtEventSeverity:
				if v, _ := em["valueCode"].(string); v != "" {
					e.Severity = v
				}
			case ExtEventReportedBy:
				if v, _ := em["valueString"].(string); v != "" {
					if rid, err := uuid.Parse(v); err == nil {
						e.ReportedByRef = rid
					}
				}
			case ExtEventWitnessedBy:
				if v, _ := em["valueString"].(string); v != "" {
					e.WitnessedByRefs = parseUUIDList(v)
				}
			case ExtEventDescriptionStructured:
				if v, _ := em["valueString"].(string); v != "" {
					e.DescriptionStructured = json.RawMessage(v)
				}
			case ExtEventReportableUnder:
				if v, _ := em["valueString"].(string); v != "" {
					var list []string
					if err := json.Unmarshal([]byte(v), &list); err == nil {
						e.ReportableUnder = list
					}
				}
			case ExtEventRelatedObservations:
				if v, _ := em["valueString"].(string); v != "" {
					e.RelatedObservations = parseUUIDList(v)
				}
			case ExtEventRelatedMedicationUses:
				if v, _ := em["valueString"].(string); v != "" {
					e.RelatedMedicationUses = parseUUIDList(v)
				}
			case ExtEventTriggeredStateChanges:
				if v, _ := em["valueString"].(string); v != "" {
					var tscs []models.TriggeredStateChange
					if err := json.Unmarshal([]byte(v), &tscs); err == nil {
						e.TriggeredStateChanges = tscs
					}
				}
			}
		}
	}

	if err := validation.ValidateEvent(e); err != nil {
		return nil, fmt.Errorf("ingress validation: %w", err)
	}
	return &e, nil
}

// parseUUIDList parses a comma-separated list of UUID strings into a slice.
// Malformed entries are dropped silently — ingress validation downstream
// catches the case where the resulting slice is empty when not allowed.
func parseUUIDList(csv string) []uuid.UUID {
	parts := strings.Split(csv, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if u, err := uuid.Parse(p); err == nil {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
