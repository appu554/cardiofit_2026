package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// ObservationFromSlot converts a slot fill into a FHIR R4 Observation resource.
func ObservationFromSlot(patientID, encounterID uuid.UUID, slot slots.SlotDefinition, value json.RawMessage) ([]byte, error) {
	obs := map[string]interface{}{
		"resourceType": "Observation",
		"status":       "final",
		"category": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system":  "http://terminology.hl7.org/CodeSystem/observation-category",
						"code":    "intake",
						"display": "Intake Assessment",
					},
				},
			},
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://loinc.org",
					"code":    slot.LOINCCode,
					"display": slot.Label,
				},
			},
			"text": slot.Label,
		},
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"encounter": map[string]interface{}{
			"reference": fmt.Sprintf("Encounter/%s", encounterID),
		},
		"effectiveDateTime": time.Now().UTC().Format(time.RFC3339),
	}

	// Set value based on data type
	switch slot.DataType {
	case slots.DataTypeNumeric:
		var v float64
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse numeric value for slot %s: %w", slot.Name, err)
		}
		obs["valueQuantity"] = map[string]interface{}{
			"value":  v,
			"unit":   slot.Unit,
			"system": "http://unitsofmeasure.org",
			"code":   slot.Unit,
		}
	case slots.DataTypeInteger:
		var v int
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse integer value for slot %s: %w", slot.Name, err)
		}
		obs["valueQuantity"] = map[string]interface{}{
			"value":  v,
			"unit":   slot.Unit,
			"system": "http://unitsofmeasure.org",
			"code":   slot.Unit,
		}
	case slots.DataTypeBoolean:
		var v bool
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse boolean value for slot %s: %w", slot.Name, err)
		}
		obs["valueBoolean"] = v
	case slots.DataTypeCodedChoice:
		var v string
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse coded choice for slot %s: %w", slot.Name, err)
		}
		obs["valueCodeableConcept"] = map[string]interface{}{
			"text": v,
		}
	case slots.DataTypeText:
		var v string
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse text for slot %s: %w", slot.Name, err)
		}
		obs["valueString"] = v
	case slots.DataTypeList:
		obs["valueString"] = string(value)
	case slots.DataTypeDate:
		var v string
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse date for slot %s: %w", slot.Name, err)
		}
		obs["valueDateTime"] = v
	}

	return json.Marshal(obs)
}

// DetectedIssueFromRule converts a safety rule result into a FHIR DetectedIssue resource.
func DetectedIssueFromRule(patientID, encounterID uuid.UUID, rule safety.RuleResult) ([]byte, error) {
	severity := "moderate" // SOFT_FLAG
	if rule.RuleType == safety.RuleTypeHardStop {
		severity = "high"
	}

	di := map[string]interface{}{
		"resourceType": "DetectedIssue",
		"status":       "final",
		"severity":     severity,
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://cardiofit.in/safety-rules",
					"code":    rule.RuleID,
					"display": rule.Reason,
				},
			},
			"text": rule.Reason,
		},
		"patient": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"detail":             rule.Reason,
		"identifiedDateTime": time.Now().UTC().Format(time.RFC3339),
		"implicated": []map[string]interface{}{
			{
				"reference": fmt.Sprintf("Encounter/%s", encounterID),
			},
		},
	}

	return json.Marshal(di)
}
