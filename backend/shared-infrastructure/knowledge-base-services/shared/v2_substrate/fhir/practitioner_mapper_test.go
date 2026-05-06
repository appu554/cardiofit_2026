package fhir

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestPersonToPractitionerToPersonRoundTrip(t *testing.T) {
	in := models.Person{
		ID:                uuid.New(),
		GivenName:         "Sarah",
		FamilyName:        "Chen",
		HPII:              "8003614900000000",
		AHPRARegistration: "NMW0001234567",
	}

	pr, err := PersonToAUPractitioner(in)
	if err != nil {
		t.Fatalf("PersonToAUPractitioner: %v", err)
	}
	if pr["resourceType"] != "Practitioner" {
		t.Errorf("resourceType: got %v, want Practitioner", pr["resourceType"])
	}

	b, _ := json.Marshal(pr)
	var rt map[string]interface{}
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := AUPractitionerToPerson(rt)
	if err != nil {
		t.Fatalf("AUPractitionerToPerson: %v", err)
	}
	if out.HPII != in.HPII {
		t.Errorf("HPII: got %q want %q", out.HPII, in.HPII)
	}
	if out.AHPRARegistration != in.AHPRARegistration {
		t.Errorf("AHPRA: got %q want %q", out.AHPRARegistration, in.AHPRARegistration)
	}
	if out.GivenName != in.GivenName || out.FamilyName != in.FamilyName {
		t.Errorf("name mismatch: got %q %q want %q %q", out.GivenName, out.FamilyName, in.GivenName, in.FamilyName)
	}
}

func TestRoleToPractitionerRoleRoundTrip(t *testing.T) {
	facility := uuid.New()
	in := models.Role{
		ID:             uuid.New(),
		PersonID:       uuid.New(),
		Kind:           models.RoleEN,
		Qualifications: json.RawMessage(`{"notation":false,"nmba_medication_qual":true}`),
		FacilityID:     &facility,
		ValidFrom:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	prr, err := RoleToAUPractitionerRole(in)
	if err != nil {
		t.Fatalf("RoleToAUPractitionerRole: %v", err)
	}
	if prr["resourceType"] != "PractitionerRole" {
		t.Errorf("resourceType: got %v, want PractitionerRole", prr["resourceType"])
	}

	b, _ := json.Marshal(prr)
	var rt map[string]interface{}
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := AUPractitionerRoleToRole(rt)
	if err != nil {
		t.Fatalf("AUPractitionerRoleToRole: %v", err)
	}
	if out.Kind != in.Kind {
		t.Errorf("Kind: got %q want %q", out.Kind, in.Kind)
	}
	if out.PersonID != in.PersonID {
		t.Errorf("PersonID lost in round-trip")
	}
	if string(out.Qualifications) == "" {
		t.Errorf("Qualifications lost in round-trip")
	}
}

// TestAUPractitionerToPerson_WrongResourceType confirms the ingress mapper
// rejects non-Practitioner FHIR resources.
func TestAUPractitionerToPerson_WrongResourceType(t *testing.T) {
	in := map[string]interface{}{"resourceType": "Patient", "id": uuid.New().String()}
	if _, err := AUPractitionerToPerson(in); err == nil {
		t.Fatalf("expected error for resourceType=Patient, got nil")
	}
}

// TestAUPractitionerRoleToRole_WrongResourceType confirms the ingress mapper
// rejects non-PractitionerRole FHIR resources.
func TestAUPractitionerRoleToRole_WrongResourceType(t *testing.T) {
	in := map[string]interface{}{"resourceType": "Practitioner", "id": uuid.New().String()}
	if _, err := AUPractitionerRoleToRole(in); err == nil {
		t.Fatalf("expected error for resourceType=Practitioner, got nil")
	}
}

// TestAUPractitionerRoleToRole_DropsInvalidJSONQualifications confirms that
// a malformed qualifications extension does not poison the Role: invalid
// JSON is dropped silently and the rest of the mapping proceeds.
func TestAUPractitionerRoleToRole_DropsInvalidJSONQualifications(t *testing.T) {
	in := map[string]interface{}{
		"resourceType": "PractitionerRole",
		"id":           uuid.New().String(),
		"practitioner": map[string]interface{}{"reference": "Practitioner/" + uuid.New().String()},
		"code": []interface{}{
			map[string]interface{}{
				"coding": []interface{}{
					map[string]interface{}{"system": SystemRoleKindCodeSystem, "code": models.RoleEN},
				},
			},
		},
		"period": map[string]interface{}{"start": "2024-01-01T00:00:00Z"},
		"extension": []interface{}{
			map[string]interface{}{
				"url":         ExtRoleQualifications,
				"valueString": "{not valid json",
			},
		},
	}
	out, err := AUPractitionerRoleToRole(in)
	if err != nil {
		t.Fatalf("AUPractitionerRoleToRole: %v", err)
	}
	if len(out.Qualifications) != 0 {
		t.Errorf("expected Qualifications dropped on invalid JSON, got %q", string(out.Qualifications))
	}
}

// TestPersonToAUPractitioner_RejectsInvalid confirms egress validation.
func TestPersonToAUPractitioner_RejectsInvalid(t *testing.T) {
	in := models.Person{
		ID:         uuid.New(),
		FamilyName: "Chen",
		// GivenName missing → validation error.
	}
	if _, err := PersonToAUPractitioner(in); err == nil {
		t.Fatalf("expected validation error for missing given_name, got nil")
	}
}

// TestRoleToAUPractitionerRole_RejectsInvalid confirms egress validation.
func TestRoleToAUPractitionerRole_RejectsInvalid(t *testing.T) {
	in := models.Role{
		ID:        uuid.New(),
		PersonID:  uuid.New(),
		Kind:      "not-a-real-role-kind",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := RoleToAUPractitionerRole(in); err == nil {
		t.Fatalf("expected validation error for invalid kind, got nil")
	}
}
