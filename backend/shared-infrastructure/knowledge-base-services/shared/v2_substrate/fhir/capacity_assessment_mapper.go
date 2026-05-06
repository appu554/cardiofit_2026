package fhir

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// FHIR routing for CapacityAssessment (Layer 2 doc §2.5 / Wave 2.5):
//
// CapacityAssessment → AU FHIR Observation with category=assessment and
// a Vaidshala-defined CodeSystem keyed by Domain. Observation is the
// closest native FHIR resource because a capacity assessment is a
// finding about the resident at a point in time, produced by a named
// assessor — exactly the Observation shape — but FHIR has no native
// vocabulary for cognitive-capacity domains, so the code uses the
// Vaidshala CodeSystem `capacity-assessment` (URI in extensions.go).
//
// Mapping table:
//
//   CapacityAssessment field    → FHIR Observation target
//   ----------------------------   ---------------------------------------
//   ID                          → Observation.id
//   ResidentRef                 → Observation.subject.reference (Patient/<uuid>)
//   AssessedAt                  → Observation.effectiveDateTime
//   AssessorRoleRef             → Observation.performer[0].reference (Role/<uuid>)
//   Domain                      → Observation.code.coding[0] (Vaidshala CodeSystem)
//   Outcome                     → Observation.valueCodeableConcept.coding[0].code
//   Instrument                  → Vaidshala extension (valueString)
//   Score                       → Vaidshala extension (valueDecimal)
//   Duration                    → Vaidshala extension (valueCode)
//   ExpectedReviewDate          → Vaidshala extension (valueDateTime)
//   RationaleStructured         → Vaidshala extension (valueString — JSON literal)
//   RationaleFreeText           → Observation.note[0].text (FHIR-native)
//   SupersedesRef               → Vaidshala extension (valueString — UUID)
//
// Observation.category is always [{coding: [{system:
// http://terminology.hl7.org/CodeSystem/observation-category, code:
// assessment}]}] for CapacityAssessment egress.
//
// Lossy fields: CreatedAt is managed by the canonical store and is not
// round-tripped through FHIR.

const observationCategorySystem = "http://terminology.hl7.org/CodeSystem/observation-category"

// CapacityAssessmentToFHIRObservation translates a v2 CapacityAssessment
// to an AU FHIR Observation as a generic map[string]interface{}.
//
// Egress validates the input via validation.ValidateCapacityAssessment
// before constructing the FHIR map.
func CapacityAssessmentToFHIRObservation(c models.CapacityAssessment) (map[string]interface{}, error) {
	if err := validation.ValidateCapacityAssessment(c); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	out := map[string]interface{}{
		"resourceType": "Observation",
		"id":           c.ID.String(),
		"status":       "final",
		"category": []map[string]interface{}{{
			"coding": []map[string]interface{}{{
				"system": observationCategorySystem,
				"code":   "assessment",
			}},
		}},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{{
				"system": SystemCapacityAssessment,
				"code":   c.Domain,
			}},
		},
		"subject": map[string]interface{}{
			"reference": "Patient/" + c.ResidentRef.String(),
		},
		"effectiveDateTime": c.AssessedAt.UTC().Format(time.RFC3339),
		"performer": []map[string]interface{}{{
			"reference": "Role/" + c.AssessorRoleRef.String(),
		}},
		"valueCodeableConcept": map[string]interface{}{
			"coding": []map[string]interface{}{{
				"system": SystemCapacityAssessment,
				"code":   c.Outcome,
			}},
		},
	}

	if c.RationaleFreeText != "" {
		out["note"] = []map[string]interface{}{{"text": c.RationaleFreeText}}
	}

	exts := []map[string]interface{}{
		{"url": ExtCapacityAssessmentDuration, "valueCode": c.Duration},
	}
	if c.Instrument != "" {
		exts = append(exts, map[string]interface{}{
			"url":         ExtCapacityAssessmentInstrument,
			"valueString": c.Instrument,
		})
	}
	if c.Score != nil {
		exts = append(exts, map[string]interface{}{
			"url":          ExtCapacityAssessmentScore,
			"valueDecimal": *c.Score,
		})
	}
	if c.ExpectedReviewDate != nil {
		exts = append(exts, map[string]interface{}{
			"url":           ExtCapacityAssessmentExpectedReviewDate,
			"valueDateTime": c.ExpectedReviewDate.UTC().Format(time.RFC3339),
		})
	}
	if len(c.RationaleStructured) > 0 {
		exts = append(exts, map[string]interface{}{
			"url":         ExtCapacityAssessmentRationaleStructured,
			"valueString": string(c.RationaleStructured),
		})
	}
	if c.SupersedesRef != nil {
		exts = append(exts, map[string]interface{}{
			"url":         ExtCapacityAssessmentSupersedesRef,
			"valueString": c.SupersedesRef.String(),
		})
	}
	out["extension"] = exts
	return out, nil
}

// FHIRObservationToCapacityAssessment is the inverse mapping. Validates
// the reconstructed CapacityAssessment via
// validation.ValidateCapacityAssessment.
func FHIRObservationToCapacityAssessment(in map[string]interface{}) (*models.CapacityAssessment, error) {
	rt, _ := in["resourceType"].(string)
	if rt != "Observation" {
		return nil, fmt.Errorf("resourceType: got %q want Observation", rt)
	}

	var c models.CapacityAssessment

	if idStr, _ := in["id"].(string); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			c.ID = id
		}
	}

	if subj, ok := in["subject"].(map[string]interface{}); ok {
		if ref, _ := subj["reference"].(string); strings.HasPrefix(ref, "Patient/") {
			if rid, err := uuid.Parse(strings.TrimPrefix(ref, "Patient/")); err == nil {
				c.ResidentRef = rid
			}
		}
	}

	if s, _ := in["effectiveDateTime"].(string); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			c.AssessedAt = t
		}
	}

	if perfArr, ok := in["performer"].([]interface{}); ok && len(perfArr) > 0 {
		if p0, ok := perfArr[0].(map[string]interface{}); ok {
			if ref, _ := p0["reference"].(string); strings.HasPrefix(ref, "Role/") {
				if rid, err := uuid.Parse(strings.TrimPrefix(ref, "Role/")); err == nil {
					c.AssessorRoleRef = rid
				}
			}
		}
	}

	// Domain from code.coding (Vaidshala CodeSystem).
	if code, ok := in["code"].(map[string]interface{}); ok {
		if codings, ok := code["coding"].([]interface{}); ok {
			for _, cc := range codings {
				cm, _ := cc.(map[string]interface{})
				sys, _ := cm["system"].(string)
				if sys == SystemCapacityAssessment {
					if v, _ := cm["code"].(string); v != "" {
						c.Domain = v
					}
					break
				}
			}
		}
	}

	// Outcome from valueCodeableConcept.coding.
	if vcc, ok := in["valueCodeableConcept"].(map[string]interface{}); ok {
		if codings, ok := vcc["coding"].([]interface{}); ok {
			for _, cc := range codings {
				cm, _ := cc.(map[string]interface{})
				sys, _ := cm["system"].(string)
				if sys == SystemCapacityAssessment {
					if v, _ := cm["code"].(string); v != "" {
						c.Outcome = v
					}
					break
				}
			}
		}
	}

	if notes, ok := in["note"].([]interface{}); ok && len(notes) > 0 {
		if n0, ok := notes[0].(map[string]interface{}); ok {
			if t, _ := n0["text"].(string); t != "" {
				c.RationaleFreeText = t
			}
		}
	}

	// Vaidshala extensions.
	if exts, ok := in["extension"].([]interface{}); ok {
		for _, x := range exts {
			em, _ := x.(map[string]interface{})
			url, _ := em["url"].(string)
			switch url {
			case ExtCapacityAssessmentDuration:
				if v, _ := em["valueCode"].(string); v != "" {
					c.Duration = v
				}
			case ExtCapacityAssessmentInstrument:
				if v, _ := em["valueString"].(string); v != "" {
					c.Instrument = v
				}
			case ExtCapacityAssessmentScore:
				// Score arrives as float64 from a fresh egress, but may be
				// re-marshalled through json.Decoder which preserves
				// float64. Accept both float64 and the string form for
				// resilience against intermediate serialisations.
				switch v := em["valueDecimal"].(type) {
				case float64:
					f := v
					c.Score = &f
				case json.Number:
					if f, err := v.Float64(); err == nil {
						c.Score = &f
					}
				case string:
					if f, err := strconv.ParseFloat(v, 64); err == nil {
						c.Score = &f
					}
				}
			case ExtCapacityAssessmentExpectedReviewDate:
				if v, _ := em["valueDateTime"].(string); v != "" {
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						c.ExpectedReviewDate = &t
					}
				}
			case ExtCapacityAssessmentRationaleStructured:
				if v, _ := em["valueString"].(string); v != "" {
					c.RationaleStructured = json.RawMessage(v)
				}
			case ExtCapacityAssessmentSupersedesRef:
				if v, _ := em["valueString"].(string); v != "" {
					if u, err := uuid.Parse(v); err == nil {
						c.SupersedesRef = &u
					}
				}
			}
		}
	}

	if err := validation.ValidateCapacityAssessment(c); err != nil {
		return nil, fmt.Errorf("ingress validation: %w", err)
	}
	return &c, nil
}
