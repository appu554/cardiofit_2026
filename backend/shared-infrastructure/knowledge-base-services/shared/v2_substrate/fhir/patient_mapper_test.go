package fhir

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestResidentToPatientToResidentRoundTrip(t *testing.T) {
	in := models.Resident{
		ID:               uuid.New(),
		IHI:              "8003608000000570",
		GivenName:        "Margaret",
		FamilyName:       "Brown",
		DOB:              time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
		Sex:              "female",
		IndigenousStatus: "neither",
		FacilityID:       uuid.New(),
		CareIntensity:    models.CareIntensityActive,
		Status:           models.ResidentStatusActive,
	}

	patient, err := ResidentToAUPatient(in)
	if err != nil {
		t.Fatalf("ResidentToAUPatient: %v", err)
	}

	// Verify FHIR shape is sane: must have resourceType=Patient
	if patient["resourceType"] != "Patient" {
		t.Errorf("resourceType: got %v, want Patient", patient["resourceType"])
	}

	// Round-trip back through marshal/unmarshal to simulate wire transport
	b, _ := json.Marshal(patient)
	var rt map[string]interface{}
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := AUPatientToResident(rt)
	if err != nil {
		t.Fatalf("AUPatientToResident: %v", err)
	}

	if out.IHI != in.IHI {
		t.Errorf("IHI: got %q want %q", out.IHI, in.IHI)
	}
	if out.GivenName != in.GivenName || out.FamilyName != in.FamilyName {
		t.Errorf("name: got %q %q, want %q %q", out.GivenName, out.FamilyName, in.GivenName, in.FamilyName)
	}
	if out.Sex != in.Sex {
		t.Errorf("sex: got %q want %q", out.Sex, in.Sex)
	}
	if out.IndigenousStatus != in.IndigenousStatus {
		t.Errorf("indigenous_status: got %q want %q", out.IndigenousStatus, in.IndigenousStatus)
	}
	if out.CareIntensity != in.CareIntensity {
		t.Errorf("care_intensity: got %q want %q (must round-trip via Vaidshala extension)", out.CareIntensity, in.CareIntensity)
	}
	if !out.DOB.Equal(in.DOB) {
		t.Errorf("dob: got %v want %v", out.DOB, in.DOB)
	}
}

func TestResidentToPatientOmitsEmptyIHI(t *testing.T) {
	in := models.Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y", Sex: "male", DOB: time.Now(), CareIntensity: models.CareIntensityActive, Status: models.ResidentStatusActive}
	p, _ := ResidentToAUPatient(in)
	ids, ok := p["identifier"].([]interface{})
	if ok && len(ids) > 0 {
		t.Errorf("identifier should be empty when IHI absent; got %v", ids)
	}
}

// TestAUPatientToResident_WrongResourceType confirms the ingress mapper
// rejects non-Patient FHIR resources.
func TestAUPatientToResident_WrongResourceType(t *testing.T) {
	in := map[string]interface{}{"resourceType": "Observation", "id": uuid.New().String()}
	if _, err := AUPatientToResident(in); err == nil {
		t.Fatalf("expected error for resourceType=Observation, got nil")
	}
}

// TestAUPatientToResident_RejectsMalformed confirms ingress validation runs
// at the end of the mapper and rejects FHIR that round-trips to an invalid
// Resident (e.g. missing required given_name).
func TestAUPatientToResident_RejectsMalformed(t *testing.T) {
	in := map[string]interface{}{
		"resourceType": "Patient",
		"id":           uuid.New().String(),
		// name omitted entirely → GivenName/FamilyName empty → validation fails
		"gender":    "female",
		"birthDate": "1938-04-12",
		"active":    true,
	}
	if _, err := AUPatientToResident(in); err == nil {
		t.Fatalf("expected ingress validation error for missing name, got nil")
	}
}

// TestResidentToAUPatient_WireFormat asserts the FHIR JSON shape produced
// by the egress mapper, not just round-trip equivalence. Guards against
// silent regressions in identifier system, extension URI, or shape changes.
func TestResidentToAUPatient_WireFormat(t *testing.T) {
	r := models.Resident{
		ID:               uuid.New(),
		IHI:              "8003608000000570",
		GivenName:        "Margaret",
		FamilyName:       "Brown",
		DOB:              time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
		Sex:              "female",
		IndigenousStatus: "neither",
		FacilityID:       uuid.New(),
		CareIntensity:    models.CareIntensityActive,
		Status:           models.ResidentStatusActive,
	}
	p, err := ResidentToAUPatient(r)
	if err != nil {
		t.Fatalf("ResidentToAUPatient: %v", err)
	}
	if p["resourceType"] != "Patient" {
		t.Errorf("resourceType: got %v want Patient", p["resourceType"])
	}
	ids, ok := p["identifier"].([]map[string]interface{})
	if !ok || len(ids) != 1 {
		t.Fatalf("expected 1 identifier, got %v", p["identifier"])
	}
	if ids[0]["system"] != SystemIHI {
		t.Errorf("identifier.system: got %q want %q", ids[0]["system"], SystemIHI)
	}
	if ids[0]["value"] != r.IHI {
		t.Errorf("identifier.value: got %q want %q", ids[0]["value"], r.IHI)
	}
	exts, ok := p["extension"].([]map[string]interface{})
	if !ok || len(exts) < 2 {
		t.Fatalf("expected >=2 extensions, got %v", p["extension"])
	}
	seenIndigenous, seenCare := false, false
	for _, e := range exts {
		switch e["url"] {
		case ExtIndigenousStatus:
			if e["valueString"] != r.IndigenousStatus {
				t.Errorf("indigenous valueString: got %q want %q", e["valueString"], r.IndigenousStatus)
			}
			seenIndigenous = true
		case ExtCareIntensity:
			if e["valueString"] != string(r.CareIntensity) {
				t.Errorf("care_intensity valueString: got %q want %q", e["valueString"], r.CareIntensity)
			}
			seenCare = true
		}
	}
	if !seenIndigenous {
		t.Errorf("extension %q not found", ExtIndigenousStatus)
	}
	if !seenCare {
		t.Errorf("extension %q not found", ExtCareIntensity)
	}
}

// TestResidentToAUPatient_RejectsInvalid confirms that the egress mapper
// runs validation and refuses to emit a FHIR Patient for invalid input.
func TestResidentToAUPatient_RejectsInvalid(t *testing.T) {
	// Missing required given_name should fail validation.
	in := models.Resident{
		ID:            uuid.New(),
		FamilyName:    "Brown",
		Sex:           "female",
		DOB:           time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
		CareIntensity: models.CareIntensityActive,
		Status:        models.ResidentStatusActive,
	}
	if _, err := ResidentToAUPatient(in); err == nil {
		t.Fatalf("expected validation error for missing given_name, got nil")
	}
}
