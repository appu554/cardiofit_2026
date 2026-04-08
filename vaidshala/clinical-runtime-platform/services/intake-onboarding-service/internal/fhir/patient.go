package fhir

import (
	"encoding/json"
	"time"
)

// NewPatientResource creates a FHIR R4 Patient resource.
// Follows ABDM IG v7.0 PatientIN profile.
func NewPatientResource(givenName, familyName, phone, abhaID, email string) ([]byte, error) {
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"meta": map[string]interface{}{
			"profile": []string{
				"https://nrces.in/ndhm/fhir/r4/StructureDefinition/Patient",
			},
		},
		"active": true,
		"name": []map[string]interface{}{
			{
				"use":    "official",
				"family": familyName,
				"given":  []string{givenName},
			},
		},
		"telecom": []map[string]interface{}{
			{
				"system": "phone",
				"value":  phone,
				"use":    "mobile",
			},
		},
	}

	if email != "" {
		patient["telecom"] = append(patient["telecom"].([]map[string]interface{}), map[string]interface{}{
			"system": "email",
			"value":  email,
			"use":    "home",
		})
	}

	if abhaID != "" {
		patient["identifier"] = []map[string]interface{}{
			{
				"system": "https://healthid.abdm.gov.in",
				"value":  abhaID,
				"type": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://terminology.hl7.org/CodeSystem/v2-0203",
							"code":    "MR",
							"display": "ABHA Number",
						},
					},
				},
			},
		}
	}

	return json.Marshal(patient)
}

// PatientDemographics holds the 4 identity fields that live on the Patient resource.
type PatientDemographics struct {
	Age             int    // years → computed to birthDate
	Gender          string // male, female, other, unknown → Patient.gender
	Ethnicity       string // coded value → Patient.extension
	PrimaryLanguage string // coded value → Patient.communication
}

// UpdatePatientDemographics patches an existing Patient resource with identity-level demographics.
// These are fields that belong on the Patient resource per FHIR R4, not as Observations.
func UpdatePatientDemographics(existingPatient []byte, demo PatientDemographics) ([]byte, error) {
	var patient map[string]interface{}
	if err := json.Unmarshal(existingPatient, &patient); err != nil {
		return nil, err
	}

	// Age → birthDate (approximate: today minus age years)
	if demo.Age > 0 {
		birthYear := time.Now().Year() - demo.Age
		patient["birthDate"] = time.Date(birthYear, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}

	// Gender → Patient.gender (FHIR R4 ValueSet: male | female | other | unknown)
	if demo.Gender != "" {
		genderMap := map[string]string{
			"male": "male", "female": "female", "other": "other", "unknown": "unknown",
			"m": "male", "f": "female",
		}
		if mapped, ok := genderMap[demo.Gender]; ok {
			patient["gender"] = mapped
		} else {
			patient["gender"] = demo.Gender
		}
	}

	// Ethnicity → Patient.extension (using HL7 US Core style)
	if demo.Ethnicity != "" {
		extensions, _ := patient["extension"].([]interface{})
		// Remove existing ethnicity extension if present.
		var filtered []interface{}
		for _, ext := range extensions {
			if m, ok := ext.(map[string]interface{}); ok {
				if m["url"] != "http://cardiofit.in/fhir/StructureDefinition/ethnicity" {
					filtered = append(filtered, ext)
				}
			}
		}
		filtered = append(filtered, map[string]interface{}{
			"url": "http://cardiofit.in/fhir/StructureDefinition/ethnicity",
			"valueCodeableConcept": map[string]interface{}{
				"coding": []map[string]interface{}{
					{
						"system":  "http://cardiofit.in/fhir/CodeSystem/ethnicity",
						"code":    demo.Ethnicity,
						"display": demo.Ethnicity,
					},
				},
			},
		})
		patient["extension"] = filtered
	}

	// Primary language → Patient.communication[].language
	if demo.PrimaryLanguage != "" {
		patient["communication"] = []map[string]interface{}{
			{
				"language": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "urn:ietf:bcp:47",
							"code":    demo.PrimaryLanguage,
							"display": demo.PrimaryLanguage,
						},
					},
				},
				"preferred": true,
			},
		}
	}

	return json.Marshal(patient)
}
