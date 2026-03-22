package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NewEncounterResource creates a FHIR R4 Encounter resource for an intake session.
func NewEncounterResource(patientID uuid.UUID, encounterType string) ([]byte, error) {
	encounter := map[string]interface{}{
		"resourceType": "Encounter",
		"status":       "in-progress",
		"class": map[string]interface{}{
			"system":  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			"code":    "VR",
			"display": "virtual",
		},
		"type": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system":  "http://cardiofit.in/encounter-types",
						"code":    encounterType,
						"display": encounterType + " session",
					},
				},
			},
		},
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"period": map[string]interface{}{
			"start": time.Now().UTC().Format(time.RFC3339),
		},
	}

	return json.Marshal(encounter)
}

// UpdateEncounterStatus updates the status of an existing Encounter.
func UpdateEncounterStatus(existingEncounter []byte, status string) ([]byte, error) {
	var encounter map[string]interface{}
	if err := json.Unmarshal(existingEncounter, &encounter); err != nil {
		return nil, err
	}

	encounter["status"] = status
	if status == "finished" {
		if period, ok := encounter["period"].(map[string]interface{}); ok {
			period["end"] = time.Now().UTC().Format(time.RFC3339)
		}
	}

	return json.Marshal(encounter)
}
