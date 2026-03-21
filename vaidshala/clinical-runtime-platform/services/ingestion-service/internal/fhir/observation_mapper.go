package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// abdmObservationProfile is the ABDM IG v7.0 profile URL for vitals.
const abdmObservationProfile = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/ObservationVitalSignsIN"

// MapObservation converts a CanonicalObservation to a FHIR R4 Observation resource JSON.
// Conforms to ABDM IG v7.0 ObservationVitalSignsIN profile.
func MapObservation(obs *canonical.CanonicalObservation) ([]byte, error) {
	resource := map[string]interface{}{
		"resourceType": "Observation",
		"meta": map[string]interface{}{
			"profile": []string{abdmObservationProfile},
		},
		"status": "final",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", obs.PatientID.String()),
		},
		"effectiveDateTime": obs.Timestamp.UTC().Format(time.RFC3339),
	}

	// Category
	category := observationCategory(obs)
	resource["category"] = []map[string]interface{}{
		{
			"coding": []map[string]interface{}{
				{
					"system":  "http://terminology.hl7.org/CodeSystem/observation-category",
					"code":    category,
					"display": categoryDisplay(category),
				},
			},
		},
	}

	// Code (LOINC)
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

	// Value
	if obs.ValueString != "" && obs.Value == 0 {
		resource["valueString"] = obs.ValueString
	} else {
		valueQuantity := map[string]interface{}{
			"value": obs.Value,
		}
		if obs.Unit != "" {
			valueQuantity["unit"] = obs.Unit
			valueQuantity["system"] = "http://unitsofmeasure.org"
			valueQuantity["code"] = ucumCode(obs.Unit)
		}
		resource["valueQuantity"] = valueQuantity
	}

	// Interpretation for critical values
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			resource["interpretation"] = []map[string]interface{}{
				{
					"coding": []map[string]interface{}{
						{
							"system":  "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
							"code":    "AA",
							"display": "Critical abnormal",
						},
					},
				},
			}
			break
		}
	}

	// Device reference
	if obs.DeviceContext != nil {
		resource["device"] = map[string]interface{}{
			"display": fmt.Sprintf("%s %s (%s)",
				obs.DeviceContext.Manufacturer,
				obs.DeviceContext.Model,
				obs.DeviceContext.DeviceID,
			),
		}
	}

	// Method
	if obs.ClinicalContext != nil && obs.ClinicalContext.Method != "" {
		if entry, ok := coding.LookupSNOMED(obs.ClinicalContext.Method); ok {
			resource["method"] = map[string]interface{}{
				"coding": []map[string]interface{}{
					{
						"system":  "http://snomed.info/sct",
						"code":    entry.Code,
						"display": entry.Display,
					},
				},
			}
		}
	}

	// Body site
	if obs.ClinicalContext != nil && obs.ClinicalContext.BodySite != "" {
		if entry, ok := coding.LookupSNOMED(obs.ClinicalContext.BodySite); ok {
			resource["bodySite"] = map[string]interface{}{
				"coding": []map[string]interface{}{
					{
						"system":  "http://snomed.info/sct",
						"code":    entry.Code,
						"display": entry.Display,
					},
				},
			}
		}
	}

	return json.Marshal(resource)
}

// observationCategory returns the FHIR observation category string.
func observationCategory(obs *canonical.CanonicalObservation) string {
	switch obs.ObservationType {
	case canonical.ObsVitals, canonical.ObsDeviceData:
		return "vital-signs"
	case canonical.ObsLabs:
		return "laboratory"
	case canonical.ObsPatientReported:
		return "survey"
	default:
		return "laboratory"
	}
}

// categoryDisplay returns the display string for a category code.
func categoryDisplay(code string) string {
	switch code {
	case "vital-signs":
		return "Vital Signs"
	case "laboratory":
		return "Laboratory"
	case "survey":
		return "Survey"
	case "social-history":
		return "Social History"
	case "activity":
		return "Activity"
	default:
		return code
	}
}

// ucumCode maps common unit display strings to UCUM codes.
func ucumCode(unit string) string {
	ucumMap := map[string]string{
		"mg/dL":         "mg/dL",
		"mmol/L":        "mmol/L",
		"mmHg":          "mm[Hg]",
		"bpm":           "/min",
		"%":             "%",
		"degC":          "Cel",
		"degF":          "[degF]",
		"kg":            "kg",
		"lbs":           "[lb_av]",
		"cm":            "cm",
		"kg/m2":         "kg/m2",
		"mL/min/1.73m2": "mL/min/{1.73_m2}",
		"mEq/L":         "meq/L",
		"U/L":           "U/L",
		"g/dL":          "g/dL",
		"mIU/L":         "m[IU]/L",
		"ng/dL":         "ng/dL",
		"mg/L":          "mg/L",
		"mmol/mol":      "mmol/mol",
	}
	if code, ok := ucumMap[unit]; ok {
		return code
	}
	return unit
}
