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

	// Code (LOINC) — required 1..1 in FHIR R4 Observation
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
	} else {
		// WhatsApp NLU and other unmapped sources may lack a LOINC code.
		// FHIR R4 requires Observation.code (1..1), so we emit a text-only
		// code element to satisfy the Google Healthcare API constraint.
		codeElement := map[string]interface{}{
			"text": "Unmapped observation",
		}
		if obs.ValueString != "" {
			codeElement["text"] = obs.ValueString
		} else if obs.Unit != "" {
			codeElement["text"] = fmt.Sprintf("Observation (%s)", obs.Unit)
		}
		resource["code"] = codeElement
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

	// V4 Signal Schema Extensions — mapped as FHIR R4 extensions
	// Per §7.1–7.3 of the Flink Architecture spec.
	var v4Extensions []map[string]interface{}

	if obs.DataTier != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/data-tier", obs.DataTier,
		))
	}
	if obs.BPDeviceType != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/bp-device-type", obs.BPDeviceType,
		))
	}
	if obs.MeasurementMethod != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/measurement-method", obs.MeasurementMethod,
		))
	}
	if obs.PreparationMethod != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/preparation-method", obs.PreparationMethod,
		))
	}
	if obs.FoodNameLocal != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/food-name-local", obs.FoodNameLocal,
		))
	}
	if obs.SodiumEstimatedMg != 0 {
		v4Extensions = append(v4Extensions, fhirDecimalExtension(
			"https://vaidshala.in/fhir/extension/sodium-estimated-mg", obs.SodiumEstimatedMg,
		))
	}
	if obs.LinkedMealID != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/linked-meal-id", obs.LinkedMealID,
		))
	}
	if obs.LinkedSeatedReadingID != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/linked-seated-reading-id", obs.LinkedSeatedReadingID,
		))
	}
	if obs.SymptomAwareness != nil {
		v4Extensions = append(v4Extensions, fhirBoolExtension(
			"https://vaidshala.in/fhir/extension/symptom-awareness", *obs.SymptomAwareness,
		))
	}
	if obs.ClinicalGrade != nil {
		v4Extensions = append(v4Extensions, fhirBoolExtension(
			"https://vaidshala.in/fhir/extension/clinical-grade", *obs.ClinicalGrade,
		))
	}
	if obs.WakingTime != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/waking-time", obs.WakingTime,
		))
	}
	if obs.SleepTime != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/sleep-time", obs.SleepTime,
		))
	}
	if obs.SourceProtocol != "" {
		v4Extensions = append(v4Extensions, fhirStringExtension(
			"https://vaidshala.in/fhir/extension/source-protocol", obs.SourceProtocol,
		))
	}

	if len(v4Extensions) > 0 {
		resource["extension"] = v4Extensions
	}

	return json.Marshal(resource)
}

// fhirStringExtension creates a FHIR R4 extension element with a valueString.
func fhirStringExtension(url, value string) map[string]interface{} {
	return map[string]interface{}{
		"url":         url,
		"valueString": value,
	}
}

// fhirDecimalExtension creates a FHIR R4 extension element with a valueDecimal.
func fhirDecimalExtension(url string, value float64) map[string]interface{} {
	return map[string]interface{}{
		"url":          url,
		"valueDecimal": value,
	}
}

// fhirBoolExtension creates a FHIR R4 extension element with a valueBoolean.
func fhirBoolExtension(url string, value bool) map[string]interface{} {
	return map[string]interface{}{
		"url":          url,
		"valueBoolean": value,
	}
}

// observationCategory returns the FHIR observation category string.
func observationCategory(obs *canonical.CanonicalObservation) string {
	switch obs.ObservationType {
	case canonical.ObsVitals, canonical.ObsDeviceData, canonical.ObsCGMRaw,
		canonical.ObsWaistCircumference:
		return "vital-signs"
	case canonical.ObsLabs:
		return "laboratory"
	case canonical.ObsPatientReported, canonical.ObsSodiumEstimate,
		canonical.ObsMoodStress:
		return "survey"
	case canonical.ObsExerciseSession, canonical.ObsWearableAggregates:
		return "activity"
	case canonical.ObsInterventionEvent, canonical.ObsPhysicianFeedback:
		return "social-history"
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
