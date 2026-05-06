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

// PersonToAUPractitioner translates a v2 substrate Person to an AU FHIR
// Practitioner resource. Person has a 1:N relationship with Role; the
// Practitioner resource captures the human, while each Role is a separate
// PractitionerRole resource.
func PersonToAUPractitioner(p models.Person) (map[string]interface{}, error) {
	if err := validation.ValidatePerson(p); err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"resourceType": "Practitioner",
		"id":           p.ID.String(),
		"name": []map[string]interface{}{
			{"use": "official", "given": []string{p.GivenName}, "family": p.FamilyName},
		},
	}

	ids := []map[string]interface{}{}

	if p.HPII != "" {
		ids = append(ids, map[string]interface{}{
			"system": SystemHPII,
			"value":  p.HPII,
		})
	}
	if p.AHPRARegistration != "" {
		ids = append(ids, map[string]interface{}{
			"system": SystemAHPRARegistration,
			"value":  p.AHPRARegistration,
		})
	}
	if len(ids) > 0 {
		out["identifier"] = ids
	}
	return out, nil
}

// AUPractitionerToPerson is the reverse mapper.
func AUPractitionerToPerson(pr map[string]interface{}) (models.Person, error) {
	p := models.Person{}
	if rt, _ := pr["resourceType"].(string); rt != "Practitioner" {
		return p, fmt.Errorf("expected resourceType=Practitioner, got %q", rt)
	}
	if id, _ := pr["id"].(string); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			p.ID = parsed
		}
	}
	if names, ok := pr["name"].([]interface{}); ok && len(names) > 0 {
		if first, ok := names[0].(map[string]interface{}); ok {
			if gs, ok := first["given"].([]interface{}); ok && len(gs) > 0 {
				p.GivenName, _ = gs[0].(string)
			}
			p.FamilyName, _ = first["family"].(string)
		}
	}
	if ids, ok := pr["identifier"].([]interface{}); ok {
		for _, idAny := range ids {
			id, ok := idAny.(map[string]interface{})
			if !ok {
				continue
			}
			switch id["system"] {
			case SystemHPII:
				p.HPII, _ = id["value"].(string)
			case SystemAHPRARegistration:
				p.AHPRARegistration, _ = id["value"].(string)
			}
		}
	}
	// Defence-in-depth ingress validation.
	if err := validation.ValidatePerson(p); err != nil {
		return p, fmt.Errorf("ingress validation: %w", err)
	}
	return p, nil
}

// RoleToAUPractitionerRole translates a v2 substrate Role to an AU FHIR
// PractitionerRole. Role.Qualifications JSONB is preserved via a Vaidshala
// extension (since FHIR's PractitionerRole.qualification[] is a structured
// codeable concept and our shape is freer).
func RoleToAUPractitionerRole(r models.Role) (map[string]interface{}, error) {
	if err := validation.ValidateRole(r); err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"resourceType": "PractitionerRole",
		"id":           r.ID.String(),
		"practitioner": map[string]interface{}{
			"reference": "Practitioner/" + r.PersonID.String(),
		},
		"code": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{"system": SystemRoleKindCodeSystem, "code": r.Kind},
				},
			},
		},
		"period": map[string]interface{}{
			"start": r.ValidFrom.Format(time.RFC3339),
		},
	}

	if r.ValidTo != nil {
		out["period"].(map[string]interface{})["end"] = r.ValidTo.Format(time.RFC3339)
	}
	if r.FacilityID != nil {
		out["organization"] = map[string]interface{}{
			"reference": "Organization/" + r.FacilityID.String(),
		}
	}

	exts := []map[string]interface{}{}
	if len(r.Qualifications) > 0 {
		// Encode as a Vaidshala extension since FHIR's qualification[] is too
		// structured for our freer JSONB shape.
		exts = append(exts, map[string]interface{}{
			"url":         ExtRoleQualifications,
			"valueString": string(r.Qualifications),
		})
	}
	if r.EvidenceURL != "" {
		exts = append(exts, map[string]interface{}{
			"url":      ExtRoleEvidenceURL,
			"valueUri": r.EvidenceURL,
		})
	}
	if len(exts) > 0 {
		out["extension"] = exts
	}
	return out, nil
}

// AUPractitionerRoleToRole is the reverse mapper.
func AUPractitionerRoleToRole(prr map[string]interface{}) (models.Role, error) {
	r := models.Role{}
	if rt, _ := prr["resourceType"].(string); rt != "PractitionerRole" {
		return r, fmt.Errorf("expected resourceType=PractitionerRole, got %q", rt)
	}
	if id, _ := prr["id"].(string); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			r.ID = parsed
		}
	}
	if pract, ok := prr["practitioner"].(map[string]interface{}); ok {
		if ref, _ := pract["reference"].(string); strings.HasPrefix(ref, "Practitioner/") {
			if parsed, err := uuid.Parse(strings.TrimPrefix(ref, "Practitioner/")); err == nil {
				r.PersonID = parsed
			}
		}
	}
	if codes, ok := prr["code"].([]interface{}); ok && len(codes) > 0 {
		if cc, ok := codes[0].(map[string]interface{}); ok {
			if codings, ok := cc["coding"].([]interface{}); ok && len(codings) > 0 {
				if coding, ok := codings[0].(map[string]interface{}); ok {
					r.Kind, _ = coding["code"].(string)
				}
			}
		}
	}
	if period, ok := prr["period"].(map[string]interface{}); ok {
		if start, _ := period["start"].(string); start != "" {
			if t, err := time.Parse(time.RFC3339, start); err == nil {
				r.ValidFrom = t
			}
		}
		if end, _ := period["end"].(string); end != "" {
			if t, err := time.Parse(time.RFC3339, end); err == nil {
				r.ValidTo = &t
			}
		}
	}
	if org, ok := prr["organization"].(map[string]interface{}); ok {
		if ref, _ := org["reference"].(string); strings.HasPrefix(ref, "Organization/") {
			if parsed, err := uuid.Parse(strings.TrimPrefix(ref, "Organization/")); err == nil {
				r.FacilityID = &parsed
			}
		}
	}
	if exts, ok := prr["extension"].([]interface{}); ok {
		for _, extAny := range exts {
			ext, ok := extAny.(map[string]interface{})
			if !ok {
				continue
			}
			switch ext["url"] {
			case ExtRoleQualifications:
				if s, _ := ext["valueString"].(string); s != "" {
					// Guard against malformed JSON from external adapters:
					// only adopt the value if it parses as valid JSON, else
					// leave Qualifications nil and continue.
					if json.Valid([]byte(s)) {
						r.Qualifications = json.RawMessage(s)
					}
				}
			case ExtRoleEvidenceURL:
				r.EvidenceURL, _ = ext["valueUri"].(string)
			}
		}
	}
	// Defence-in-depth ingress validation.
	if err := validation.ValidateRole(r); err != nil {
		return r, fmt.Errorf("ingress validation: %w", err)
	}
	return r, nil
}
