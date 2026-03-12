package fhir

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"kb-patient-profile/internal/models"
)

// FHIRPatientToProfile converts a FHIR Patient resource to a KB-20 PatientProfile.
func FHIRPatientToProfile(patient map[string]interface{}) *models.PatientProfile {
	profile := &models.PatientProfile{
		FHIRPatientID: extractString(patient, "id"),
		Active:        true,
		DMType:        "NONE", // default until clinical assessment
	}

	// Extract patient_id from identifier
	if identifiers, ok := patient["identifier"].([]interface{}); ok {
		for _, ident := range identifiers {
			identMap, ok := ident.(map[string]interface{})
			if !ok {
				continue
			}
			if val := extractString(identMap, "value"); val != "" {
				profile.PatientID = val
				break
			}
		}
	}

	// Extract sex from gender
	gender := extractString(patient, "gender")
	switch gender {
	case "male":
		profile.Sex = "M"
	case "female":
		profile.Sex = "F"
	default:
		profile.Sex = "OTHER"
	}

	// Derive age from birthDate
	if birthDate := extractString(patient, "birthDate"); birthDate != "" {
		if dob, err := time.Parse("2006-01-02", birthDate); err == nil {
			profile.Age = int(time.Since(dob).Hours() / 24 / 365.25)
		}
	}

	return profile
}

// FHIRObservationToLab converts a FHIR Observation resource to a KB-20 LabEntry.
// The KB7Client is used for runtime LOINC→lab type resolution (no hardcoded map).
func FHIRObservationToLab(obs map[string]interface{}, kb7 *KB7Client) *models.LabEntry {
	entry := &models.LabEntry{
		FHIRObservationID: extractString(obs, "id"),
		ValidationStatus:  models.ValidationAccepted,
		Source:            "FHIR",
	}

	// Extract LOINC code from code.coding
	if code, ok := obs["code"].(map[string]interface{}); ok {
		if codings, ok := code["coding"].([]interface{}); ok {
			for _, coding := range codings {
				codingMap, ok := coding.(map[string]interface{})
				if !ok {
					continue
				}
				system := extractString(codingMap, "system")
				if system == "http://loinc.org" {
					entry.LOINCCode = extractString(codingMap, "code")
					// Resolve lab type via KB-7 Terminology Service at runtime
					entry.LabType = kb7.ResolveLabType(entry.LOINCCode)
					break
				}
			}
		}
	}

	// Extract value from valueQuantity
	if vq, ok := obs["valueQuantity"].(map[string]interface{}); ok {
		if val, ok := vq["value"].(float64); ok {
			entry.Value = decimal.NewFromFloat(val)
		}
		entry.Unit = extractString(vq, "unit")
	}

	// Extract effectiveDateTime
	if effectiveDT := extractString(obs, "effectiveDateTime"); effectiveDT != "" {
		if t, err := time.Parse(time.RFC3339, effectiveDT); err == nil {
			entry.MeasuredAt = t
		} else if t, err := time.Parse("2006-01-02", effectiveDT); err == nil {
			entry.MeasuredAt = t
		}
	}

	// Extract patient reference
	if subject, ok := obs["subject"].(map[string]interface{}); ok {
		ref := extractString(subject, "reference")
		entry.PatientID = extractPatientIDFromRef(ref)
	}

	return entry
}

// FHIRMedicationRequestToState converts a FHIR MedicationRequest to a KB-20 MedicationState.
func FHIRMedicationRequestToState(req map[string]interface{}) *models.MedicationState {
	state := &models.MedicationState{
		FHIRMedicationRequestID: extractString(req, "id"),
		IsActive:                true,
		Route:                   "ORAL",
	}

	// Extract drug name from medicationCodeableConcept
	if medCC, ok := req["medicationCodeableConcept"].(map[string]interface{}); ok {
		state.DrugName = extractString(medCC, "text")
		if codings, ok := medCC["coding"].([]interface{}); ok {
			for _, coding := range codings {
				codingMap, ok := coding.(map[string]interface{})
				if !ok {
					continue
				}
				system := extractString(codingMap, "system")
				if system == "http://www.whocc.no/atc" {
					state.ATCCode = extractString(codingMap, "code")
				}
			}
		}
	}

	// Extract status
	status := extractString(req, "status")
	state.IsActive = (status == "active")

	// Extract patient reference
	if subject, ok := req["subject"].(map[string]interface{}); ok {
		ref := extractString(subject, "reference")
		state.PatientID = extractPatientIDFromRef(ref)
	}

	// Extract dosage
	if dosages, ok := req["dosageInstruction"].([]interface{}); ok && len(dosages) > 0 {
		dosage, ok := dosages[0].(map[string]interface{})
		if ok {
			// Route
			if route, ok := dosage["route"].(map[string]interface{}); ok {
				if text := extractString(route, "text"); text != "" {
					state.Route = text
				}
			}
			// Dose quantity
			if doseAndRate, ok := dosage["doseAndRate"].([]interface{}); ok && len(doseAndRate) > 0 {
				dr, ok := doseAndRate[0].(map[string]interface{})
				if ok {
					if doseQty, ok := dr["doseQuantity"].(map[string]interface{}); ok {
						if val, ok := doseQty["value"].(float64); ok {
							state.DoseMg = decimal.NewFromFloat(val)
						}
					}
				}
			}
			// Frequency from timing.code.text
			if timing, ok := dosage["timing"].(map[string]interface{}); ok {
				if code, ok := timing["code"].(map[string]interface{}); ok {
					state.Frequency = extractString(code, "text")
				}
			}
		}
	}

	return state
}

// CKDStatusToFHIRCondition builds a FHIR Condition resource for CKD diagnosis.
func CKDStatusToFHIRCondition(profile *models.PatientProfile) map[string]interface{} {
	condition := map[string]interface{}{
		"resourceType": "Condition",
		"clinicalStatus": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
					"code":   "active",
				},
			},
		},
		"verificationStatus": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
					"code":   verificationStatusFromCKD(profile.CKDStatus),
				},
			},
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://snomed.info/sct",
					"code":    "709044004",
					"display": "Chronic kidney disease",
				},
			},
			"text": fmt.Sprintf("CKD Stage %s", profile.CKDStage),
		},
		"subject": map[string]interface{}{
			"reference": "Patient/" + profile.FHIRPatientID,
		},
		"recordedDate": time.Now().UTC().Format(time.RFC3339),
	}

	// Add stage if available
	if profile.CKDStage != "" {
		condition["stage"] = []map[string]interface{}{
			{
				"summary": map[string]interface{}{
					"text": "CKD Stage " + profile.CKDStage,
				},
			},
		}
	}

	return condition
}

// ThresholdCrossingToDetectedIssue builds a FHIR DetectedIssue for medication threshold events.
func ThresholdCrossingToDetectedIssue(patientFHIRID string, payload *models.MedicationThresholdCrossedPayload) map[string]interface{} {
	detail := fmt.Sprintf("%s crossed threshold %.0f (%.1f → %.1f)",
		payload.Lab, payload.ThresholdCrossed, payload.OldValue, payload.NewValue)

	issue := map[string]interface{}{
		"resourceType": "DetectedIssue",
		"status":       "final",
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
					"code":    "DOSEHINDA",
					"display": "Dose High Non-Adjusted",
				},
			},
		},
		"severity": "high",
		"patient": map[string]interface{}{
			"reference": "Patient/" + patientFHIRID,
		},
		"detail":         detail,
		"identifiedDateTime": time.Now().UTC().Format(time.RFC3339),
	}

	// Add implicated medications
	var implicated []map[string]interface{}
	for _, med := range payload.AffectedMedications {
		implicated = append(implicated, map[string]interface{}{
			"display": fmt.Sprintf("%s — %s", med.DrugClass, med.RequiredAction),
		})
	}
	if len(implicated) > 0 {
		issue["implicated"] = implicated
	}

	return issue
}

// --- helpers ---

func extractString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func extractPatientIDFromRef(ref string) string {
	// "Patient/abc123" → "abc123"
	if len(ref) > 8 && ref[:8] == "Patient/" {
		return ref[8:]
	}
	return ref
}

func verificationStatusFromCKD(status string) string {
	switch status {
	case "CONFIRMED":
		return "confirmed"
	case "SUSPECTED":
		return "provisional"
	default:
		return "unconfirmed"
	}
}


