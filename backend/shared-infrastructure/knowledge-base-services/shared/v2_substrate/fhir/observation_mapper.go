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

// ObservationToAUObservation translates a v2 Observation to an AU FHIR
// Observation (HL7 AU Base v6.0.0) as a generic map[string]interface{}
// suitable for json.Marshal to wire format.
//
// AU-specific extensions used:
//   - ExtObservationKind     (Vaidshala-internal; FHIR has no native discriminator)
//   - ExtObservationDelta    (Vaidshala-internal; encodes Delta as JSON)
//   - ExtObservationSourceID (Vaidshala-internal; UUID reference to kb-22.clinical_sources)
//
// Lossy fields (NOT round-tripped):
//   - CreatedAt — managed by canonical store, not part of FHIR shape
//
// Egress validates the input via validation.ValidateObservation before
// constructing the FHIR map. Ingress validates the output.
func ObservationToAUObservation(o models.Observation) (map[string]interface{}, error) {
	if err := validation.ValidateObservation(o); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	out := map[string]interface{}{
		"resourceType": "Observation",
		"id":           o.ID.String(),
		"status":       "final",
		"subject": map[string]interface{}{
			"reference": "Patient/" + o.ResidentID.String(),
		},
		"effectiveDateTime": o.ObservedAt.UTC().Format(time.RFC3339),
	}

	// code.coding from LOINC + SNOMED (whichever present)
	coding := []map[string]interface{}{}
	if o.LOINCCode != "" {
		coding = append(coding, map[string]interface{}{
			"system": "http://loinc.org",
			"code":   o.LOINCCode,
		})
	}
	if o.SNOMEDCode != "" {
		coding = append(coding, map[string]interface{}{
			"system": "http://snomed.info/sct",
			"code":   o.SNOMEDCode,
		})
	}
	out["code"] = map[string]interface{}{"coding": coding}

	// valueQuantity OR valueString
	if o.Value != nil {
		vq := map[string]interface{}{"value": *o.Value}
		if o.Unit != "" {
			vq["unit"] = o.Unit
		}
		out["valueQuantity"] = vq
	} else if o.ValueText != "" {
		out["valueString"] = o.ValueText
	}

	// Vaidshala extensions
	extensions := []map[string]interface{}{
		{"url": ExtObservationKind, "valueCode": o.Kind},
	}
	if o.SourceID != nil {
		extensions = append(extensions, map[string]interface{}{
			"url": ExtObservationSourceID, "valueString": o.SourceID.String(),
		})
	}
	if o.Delta != nil {
		deltaJSON, err := json.Marshal(o.Delta)
		if err != nil {
			return nil, fmt.Errorf("marshal delta: %w", err)
		}
		extensions = append(extensions, map[string]interface{}{
			"url": ExtObservationDelta, "valueString": string(deltaJSON),
		})
	}
	out["extension"] = extensions

	return out, nil
}

// AUObservationToObservation translates an AU FHIR Observation JSON map back
// to a v2 Observation. Returns an error if resourceType != "Observation"
// or if the resulting Observation fails ValidateObservation.
func AUObservationToObservation(in map[string]interface{}) (*models.Observation, error) {
	if rt, _ := in["resourceType"].(string); rt != "Observation" {
		return nil, fmt.Errorf("resourceType: got %q want Observation", rt)
	}

	var o models.Observation

	if idStr, _ := in["id"].(string); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			o.ID = id
		}
	}

	// subject.reference -> ResidentID
	if subj, ok := in["subject"].(map[string]interface{}); ok {
		if ref, _ := subj["reference"].(string); strings.HasPrefix(ref, "Patient/") {
			if rid, err := uuid.Parse(strings.TrimPrefix(ref, "Patient/")); err == nil {
				o.ResidentID = rid
			}
		}
	}

	// effectiveDateTime
	if eff, _ := in["effectiveDateTime"].(string); eff != "" {
		if t, err := time.Parse(time.RFC3339, eff); err == nil {
			o.ObservedAt = t
		}
	}

	// code.coding -> LOINC / SNOMED
	if code, ok := in["code"].(map[string]interface{}); ok {
		if codings, ok := code["coding"].([]interface{}); ok {
			for _, c := range codings {
				cm, _ := c.(map[string]interface{})
				sys, _ := cm["system"].(string)
				val, _ := cm["code"].(string)
				switch sys {
				case "http://loinc.org":
					o.LOINCCode = val
				case "http://snomed.info/sct":
					o.SNOMEDCode = val
				}
			}
		}
	}

	// valueQuantity OR valueString
	if vq, ok := in["valueQuantity"].(map[string]interface{}); ok {
		if v, ok := vq["value"].(float64); ok {
			o.Value = &v
		}
		if u, _ := vq["unit"].(string); u != "" {
			o.Unit = u
		}
	} else if vs, ok := in["valueString"].(string); ok {
		o.ValueText = vs
	}

	// extensions
	if exts, ok := in["extension"].([]interface{}); ok {
		for _, e := range exts {
			em, _ := e.(map[string]interface{})
			url, _ := em["url"].(string)
			switch url {
			case ExtObservationKind:
				if k, _ := em["valueCode"].(string); k != "" {
					o.Kind = k
				}
			case ExtObservationSourceID:
				if s, _ := em["valueString"].(string); s != "" {
					if sid, err := uuid.Parse(s); err == nil {
						o.SourceID = &sid
					}
				}
			case ExtObservationDelta:
				if s, _ := em["valueString"].(string); s != "" {
					var d models.Delta
					if err := json.Unmarshal([]byte(s), &d); err == nil {
						o.Delta = &d
					}
				}
			}
		}
	}

	if err := validation.ValidateObservation(o); err != nil {
		return nil, fmt.Errorf("ingress validation: %w", err)
	}
	return &o, nil
}
