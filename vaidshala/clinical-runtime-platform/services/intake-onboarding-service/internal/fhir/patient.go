package fhir

import (
	"encoding/json"
	"time"
)

// NewPatientResource creates a FHIR R4 Patient resource.
// Follows ABDM IG v7.0 PatientIN profile.
func NewPatientResource(givenName, familyName, phone, abhaID string) ([]byte, error) {
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

// UpdatePatientWithDemographics adds demographic observation references to a Patient.
func UpdatePatientWithDemographics(existingPatient []byte, birthDate time.Time, gender string) ([]byte, error) {
	var patient map[string]interface{}
	if err := json.Unmarshal(existingPatient, &patient); err != nil {
		return nil, err
	}

	if !birthDate.IsZero() {
		patient["birthDate"] = birthDate.Format("2006-01-02")
	}
	if gender != "" {
		patient["gender"] = gender
	}

	return json.Marshal(patient)
}
