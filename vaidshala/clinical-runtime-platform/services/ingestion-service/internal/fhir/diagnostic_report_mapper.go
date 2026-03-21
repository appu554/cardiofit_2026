package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// abdmDiagnosticReportProfile is the ABDM IG v7.0 profile URL.
const abdmDiagnosticReportProfile = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/DiagnosticReportLabIN"

// MapDiagnosticReport creates a FHIR DiagnosticReport resource wrapping a lab observation.
// This is the required ABDM wrapper for lab results -- it references the Observation.
func MapDiagnosticReport(obs *canonical.CanonicalObservation, observationID string) ([]byte, error) {
	resource := map[string]interface{}{
		"resourceType": "DiagnosticReport",
		"meta": map[string]interface{}{
			"profile": []string{abdmDiagnosticReportProfile},
		},
		"status": "final",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", obs.PatientID.String()),
		},
		"effectiveDateTime": obs.Timestamp.UTC().Format(time.RFC3339),
		"issued":            time.Now().UTC().Format(time.RFC3339),
		"result": []map[string]interface{}{
			{
				"reference": fmt.Sprintf("Observation/%s", observationID),
			},
		},
	}

	// Category -- always laboratory for DiagnosticReport
	resource["category"] = []map[string]interface{}{
		{
			"coding": []map[string]interface{}{
				{
					"system":  "http://terminology.hl7.org/CodeSystem/v2-0074",
					"code":    "LAB",
					"display": "Laboratory",
				},
			},
		},
	}

	// Code from LOINC
	if obs.LOINCCode != "" {
		codeCoding := []map[string]interface{}{
			{
				"system": "http://loinc.org",
				"code":   obs.LOINCCode,
			},
		}
		if entry, ok := coding.LookupLOINC(obs.LOINCCode); ok {
			codeCoding[0]["display"] = entry.Display
		}
		resource["code"] = map[string]interface{}{
			"coding": codeCoding,
		}
	}

	// Performer (source)
	if obs.SourceID != "" {
		resource["performer"] = []map[string]interface{}{
			{
				"display": obs.SourceID,
			},
		}
	}

	// Conclusion for critical values
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			resource["conclusion"] = "CRITICAL VALUE -- immediate clinical review required"
			break
		}
	}

	return json.Marshal(resource)
}
