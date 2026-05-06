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

// MedicineUseToAUMedicationRequest translates a v2 MedicineUse to an AU FHIR
// MedicationRequest (HL7 AU Base v6.0.0). Returns a generic map[string]interface{}
// suitable for json.Marshal to wire format.
//
// AU-specific extensions used:
//   - ExtMedicineIntent       (Vaidshala-internal; FHIR has no native intent.indication concept)
//   - ExtMedicineTarget       (Vaidshala-internal; encodes Target.Kind + Spec as JSON)
//   - ExtMedicineStopCriteria (Vaidshala-internal; encodes StopCriteria as JSON)
//   - ExtMedicineAMTCode      (Vaidshala-internal; AMT code surface as identifier)
//
// Lossy fields (NOT round-tripped):
//   - CreatedAt / UpdatedAt — managed by canonical store
//   - PrescriberID — encoded as MedicationRequest.requester.reference; reverse parses
//     "Practitioner/<uuid>" but loses non-Practitioner references silently
//
// Egress validates the input via validation.ValidateMedicineUse before
// constructing the FHIR map. Ingress validates the output.
func MedicineUseToAUMedicationRequest(m models.MedicineUse) (map[string]interface{}, error) {
	if err := validation.ValidateMedicineUse(m); err != nil {
		return nil, fmt.Errorf("egress validation: %w", err)
	}

	out := map[string]interface{}{
		"resourceType": "MedicationRequest",
		"id":           m.ID.String(),
		"status":       statusToFHIRStatus(m.Status),
		"intent":       "order", // FHIR MedicationRequest.intent — distinct from Vaidshala Intent.Category
		"subject": map[string]interface{}{
			"reference": "Patient/" + m.ResidentID.String(),
		},
		"authoredOn": m.StartedAt.UTC().Format(time.RFC3339),
	}

	// medicationCodeableConcept
	coding := []map[string]interface{}{}
	if m.AMTCode != "" {
		coding = append(coding, map[string]interface{}{
			"system":  "http://snomed.info/sct", // AMT codes are SNOMED-CT-AU
			"code":    m.AMTCode,
			"display": m.DisplayName,
		})
	}
	out["medicationCodeableConcept"] = map[string]interface{}{
		"coding": coding,
		"text":   m.DisplayName,
	}

	// dosageInstruction (single entry capturing dose/route/frequency)
	if m.Dose != "" || m.Route != "" || m.Frequency != "" {
		parts := []string{}
		for _, p := range []string{m.Dose, m.Route, m.Frequency} {
			if p != "" {
				parts = append(parts, p)
			}
		}
		dosage := map[string]interface{}{
			"text": strings.Join(parts, " "),
		}
		if m.Route != "" {
			dosage["route"] = map[string]interface{}{
				"coding": []map[string]interface{}{
					{"system": SystemRouteCode, "code": m.Route},
				},
			}
		}
		out["dosageInstruction"] = []map[string]interface{}{dosage}
	}

	// requester (PrescriberID → Practitioner reference)
	if m.PrescriberID != nil {
		out["requester"] = map[string]interface{}{
			"reference": "Practitioner/" + m.PrescriberID.String(),
		}
	}

	// dispenseRequest.validityPeriod (StartedAt / EndedAt)
	period := map[string]interface{}{
		"start": m.StartedAt.UTC().Format(time.RFC3339),
	}
	if m.EndedAt != nil {
		period["end"] = m.EndedAt.UTC().Format(time.RFC3339)
	}
	out["dispenseRequest"] = map[string]interface{}{"validityPeriod": period}

	// Vaidshala extensions for v2-distinguishing fields
	exts := []map[string]interface{}{}
	intentBytes, err := json.Marshal(m.Intent)
	if err != nil {
		return nil, fmt.Errorf("marshal intent: %w", err)
	}
	exts = append(exts, map[string]interface{}{
		"url":         ExtMedicineIntent,
		"valueString": string(intentBytes),
	})
	targetBytes, err := json.Marshal(m.Target)
	if err != nil {
		return nil, fmt.Errorf("marshal target: %w", err)
	}
	exts = append(exts, map[string]interface{}{
		"url":         ExtMedicineTarget,
		"valueString": string(targetBytes),
	})
	stopBytes, err := json.Marshal(m.StopCriteria)
	if err != nil {
		return nil, fmt.Errorf("marshal stop criteria: %w", err)
	}
	exts = append(exts, map[string]interface{}{
		"url":         ExtMedicineStopCriteria,
		"valueString": string(stopBytes),
	})
	if m.AMTCode != "" {
		exts = append(exts, map[string]interface{}{
			"url":         ExtMedicineAMTCode,
			"valueString": m.AMTCode,
		})
	}
	out["extension"] = exts

	return out, nil
}

// AUMedicationRequestToMedicineUse is the reverse mapper.
func AUMedicationRequestToMedicineUse(mr map[string]interface{}) (models.MedicineUse, error) {
	var m models.MedicineUse
	if rt, _ := mr["resourceType"].(string); rt != "MedicationRequest" {
		return m, fmt.Errorf("expected resourceType=MedicationRequest, got %q", rt)
	}
	if id, _ := mr["id"].(string); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			m.ID = parsed
		}
	}
	if s, _ := mr["status"].(string); s != "" {
		m.Status = fhirStatusToStatus(s)
	}

	// subject → ResidentID
	if subj, ok := mr["subject"].(map[string]interface{}); ok {
		if ref, _ := subj["reference"].(string); strings.HasPrefix(ref, "Patient/") {
			if parsed, err := uuid.Parse(strings.TrimPrefix(ref, "Patient/")); err == nil {
				m.ResidentID = parsed
			}
		}
	}

	// medicationCodeableConcept → DisplayName + AMTCode (from coding system snomed-ct)
	if mcc, ok := mr["medicationCodeableConcept"].(map[string]interface{}); ok {
		if txt, _ := mcc["text"].(string); txt != "" {
			m.DisplayName = txt
		}
		if codes, ok := mcc["coding"].([]interface{}); ok {
			for _, codeAny := range codes {
				code, ok := codeAny.(map[string]interface{})
				if !ok {
					continue
				}
				if code["system"] == "http://snomed.info/sct" {
					if c, _ := code["code"].(string); c != "" {
						m.AMTCode = c
					}
				}
			}
		}
	}

	// dosageInstruction → Dose / Route / Frequency
	if di, ok := mr["dosageInstruction"].([]interface{}); ok && len(di) > 0 {
		if first, ok := di[0].(map[string]interface{}); ok {
			// Dose/Frequency are reconstructed from text; this is intentionally lossy.
			// For richer ingress, parse FHIR Timing/DoseAndRate properly.
			if route, ok := first["route"].(map[string]interface{}); ok {
				if coding, ok := route["coding"].([]interface{}); ok && len(coding) > 0 {
					if c, ok := coding[0].(map[string]interface{}); ok {
						m.Route, _ = c["code"].(string)
					}
				}
			}
		}
	}

	// requester → PrescriberID
	if req, ok := mr["requester"].(map[string]interface{}); ok {
		if ref, _ := req["reference"].(string); strings.HasPrefix(ref, "Practitioner/") {
			if parsed, err := uuid.Parse(strings.TrimPrefix(ref, "Practitioner/")); err == nil {
				m.PrescriberID = &parsed
			}
		}
	}

	// dispenseRequest.validityPeriod → StartedAt / EndedAt
	if dr, ok := mr["dispenseRequest"].(map[string]interface{}); ok {
		if vp, ok := dr["validityPeriod"].(map[string]interface{}); ok {
			if s, _ := vp["start"].(string); s != "" {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					m.StartedAt = t
				}
			}
			if e, _ := vp["end"].(string); e != "" {
				if t, err := time.Parse(time.RFC3339, e); err == nil {
					m.EndedAt = &t
				}
			}
		}
	}

	// Extensions → Intent / Target / StopCriteria
	if exts, ok := mr["extension"].([]interface{}); ok {
		for _, extAny := range exts {
			ext, ok := extAny.(map[string]interface{})
			if !ok {
				continue
			}
			url, _ := ext["url"].(string)
			v, _ := ext["valueString"].(string)
			switch url {
			case ExtMedicineIntent:
				if v != "" && json.Valid([]byte(v)) {
					_ = json.Unmarshal([]byte(v), &m.Intent)
				}
			case ExtMedicineTarget:
				if v != "" && json.Valid([]byte(v)) {
					_ = json.Unmarshal([]byte(v), &m.Target)
				}
			case ExtMedicineStopCriteria:
				if v != "" && json.Valid([]byte(v)) {
					_ = json.Unmarshal([]byte(v), &m.StopCriteria)
				}
			case ExtMedicineAMTCode:
				if v != "" {
					m.AMTCode = v
				}
			}
		}
	}

	// Defense-in-depth: validate the output before returning to caller.
	if err := validation.ValidateMedicineUse(m); err != nil {
		return m, fmt.Errorf("ingress validation: %w", err)
	}
	return m, nil
}

// statusToFHIRStatus maps Vaidshala MedicineUseStatus to FHIR MedicationRequest.status.
func statusToFHIRStatus(s string) string {
	switch s {
	case models.MedicineUseStatusActive:
		return "active"
	case models.MedicineUseStatusPaused:
		return "on-hold"
	case models.MedicineUseStatusCeased:
		return "stopped"
	case models.MedicineUseStatusCompleted:
		return "completed"
	}
	return "unknown"
}

// fhirStatusToStatus is the reverse mapping. FHIR has additional values
// (draft, cancelled, entered-in-error) that have no Vaidshala analog —
// these collapse to "ceased" as the most conservative interpretation.
func fhirStatusToStatus(fs string) string {
	switch fs {
	case "active":
		return models.MedicineUseStatusActive
	case "on-hold":
		return models.MedicineUseStatusPaused
	case "stopped":
		return models.MedicineUseStatusCeased
	case "completed":
		return models.MedicineUseStatusCompleted
	}
	return models.MedicineUseStatusCeased
}
