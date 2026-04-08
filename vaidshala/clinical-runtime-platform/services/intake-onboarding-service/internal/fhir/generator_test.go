package fhir

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

func TestObservationFromSlot_Numeric(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	slot := slots.SlotDefinition{
		Name: "fbg", Domain: "glycemic", LOINCCode: "1558-6",
		DataType: slots.DataTypeNumeric, Unit: "mg/dL", Label: "Fasting blood glucose",
	}

	raw, err := ObservationFromSlot(patientID, encounterID, slot, json.RawMessage(`178`))
	if err != nil {
		t.Fatalf("ObservationFromSlot failed: %v", err)
	}

	var obs map[string]interface{}
	if err := json.Unmarshal(raw, &obs); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if obs["resourceType"] != "Observation" {
		t.Errorf("expected resourceType=Observation, got %v", obs["resourceType"])
	}
	if obs["status"] != "final" {
		t.Errorf("expected status=final, got %v", obs["status"])
	}

	// Check LOINC code
	code := obs["code"].(map[string]interface{})
	codings := code["coding"].([]interface{})
	coding := codings[0].(map[string]interface{})
	if coding["code"] != "1558-6" {
		t.Errorf("expected LOINC 1558-6, got %v", coding["code"])
	}

	// Check value
	vq := obs["valueQuantity"].(map[string]interface{})
	if vq["value"].(float64) != 178 {
		t.Errorf("expected value=178, got %v", vq["value"])
	}
	if vq["unit"] != "mg/dL" {
		t.Errorf("expected unit=mg/dL, got %v", vq["unit"])
	}
}

func TestObservationFromSlot_Boolean(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	slot := slots.SlotDefinition{
		Name: "pregnant", Domain: "demographics", LOINCCode: "82810-3",
		DataType: slots.DataTypeBoolean, Label: "Currently pregnant",
	}

	raw, err := ObservationFromSlot(patientID, encounterID, slot, json.RawMessage(`true`))
	if err != nil {
		t.Fatalf("ObservationFromSlot failed: %v", err)
	}

	var obs map[string]interface{}
	json.Unmarshal(raw, &obs)
	if obs["valueBoolean"] != true {
		t.Errorf("expected valueBoolean=true, got %v", obs["valueBoolean"])
	}
}

func TestDetectedIssueFromSafetyResult_HardStop(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	rule := safety.RuleResult{
		RuleID:   "H1",
		RuleType: safety.RuleTypeHardStop,
		Reason:   "Type 1 DM",
	}

	raw, err := DetectedIssueFromRule(patientID, encounterID, rule)
	if err != nil {
		t.Fatalf("DetectedIssueFromRule failed: %v", err)
	}

	var di map[string]interface{}
	json.Unmarshal(raw, &di)
	if di["resourceType"] != "DetectedIssue" {
		t.Errorf("expected resourceType=DetectedIssue")
	}
	if di["severity"] != "high" {
		t.Errorf("expected severity=high for HARD_STOP, got %v", di["severity"])
	}
	if di["status"] != "final" {
		t.Errorf("expected status=final, got %v", di["status"])
	}
}

func TestDetectedIssueFromSafetyResult_SoftFlag(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	rule := safety.RuleResult{
		RuleID:   "SF-01",
		RuleType: safety.RuleTypeSoftFlag,
		Reason:   "Elderly patient",
	}

	raw, err := DetectedIssueFromRule(patientID, encounterID, rule)
	if err != nil {
		t.Fatalf("DetectedIssueFromRule failed: %v", err)
	}

	var di map[string]interface{}
	json.Unmarshal(raw, &di)
	if di["severity"] != "moderate" {
		t.Errorf("expected severity=moderate for SOFT_FLAG, got %v", di["severity"])
	}
}

func TestPatientResource(t *testing.T) {
	raw, err := NewPatientResource("John", "Doe", "+919876543210", "", "")
	if err != nil {
		t.Fatalf("NewPatientResource failed: %v", err)
	}

	var pat map[string]interface{}
	json.Unmarshal(raw, &pat)
	if pat["resourceType"] != "Patient" {
		t.Errorf("expected resourceType=Patient")
	}
	names := pat["name"].([]interface{})
	name := names[0].(map[string]interface{})
	if name["family"] != "Doe" {
		t.Errorf("expected family=Doe, got %v", name["family"])
	}
}

func TestEncounterResource(t *testing.T) {
	patientID := uuid.New()
	raw, err := NewEncounterResource(patientID, "intake")
	if err != nil {
		t.Fatalf("NewEncounterResource failed: %v", err)
	}

	var enc map[string]interface{}
	json.Unmarshal(raw, &enc)
	if enc["resourceType"] != "Encounter" {
		t.Errorf("expected resourceType=Encounter")
	}
	if enc["status"] != "in-progress" {
		t.Errorf("expected status=in-progress, got %v", enc["status"])
	}
}
