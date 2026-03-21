package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// abdmMedicationStatementProfile is the ABDM IG v7.0 profile URL.
const abdmMedicationStatementProfile = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/MedicationStatementIN"

// MapMedicationStatement creates a FHIR MedicationStatement resource from
// a medication adherence observation.
func MapMedicationStatement(obs *canonical.CanonicalObservation) ([]byte, error) {
	resource := map[string]interface{}{
		"resourceType": "MedicationStatement",
		"meta": map[string]interface{}{
			"profile": []string{abdmMedicationStatementProfile},
		},
		"status": "active",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", obs.PatientID.String()),
		},
		"effectiveDateTime": obs.Timestamp.UTC().Format(time.RFC3339),
		"dateAsserted":      time.Now().UTC().Format(time.RFC3339),
	}

	// Medication reference from value string (drug name or code)
	if obs.ValueString != "" {
		resource["medicationCodeableConcept"] = map[string]interface{}{
			"text": obs.ValueString,
		}
	}

	// SNOMED code if available
	if obs.SNOMEDCode != "" {
		resource["medicationCodeableConcept"] = map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://snomed.info/sct",
					"code":    obs.SNOMEDCode,
					"display": obs.ValueString,
				},
			},
			"text": obs.ValueString,
		}
	}

	// Category -- patient-reported vs clinician
	categoryCode := "patientreported"
	categoryDisplay := "Patient Reported"
	if obs.SourceType == canonical.SourceEHR || obs.SourceType == canonical.SourceABDM {
		categoryCode = "inpatient"
		categoryDisplay = "Inpatient"
	}
	resource["category"] = map[string]interface{}{
		"coding": []map[string]interface{}{
			{
				"system":  "http://terminology.hl7.org/CodeSystem/medication-statement-category",
				"code":    categoryCode,
				"display": categoryDisplay,
			},
		},
	}

	return json.Marshal(resource)
}
