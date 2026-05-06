package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestResidentJSONRoundTrip(t *testing.T) {
	sdm := uuid.New()
	in := Resident{
		ID:               uuid.New(),
		IHI:              "8003608000000570",
		GivenName:        "Margaret",
		FamilyName:       "Brown",
		DOB:              time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
		Sex:              "female",
		IndigenousStatus: "neither",
		FacilityID:       uuid.New(),
		CareIntensity:    CareIntensityActive,
		SDMs:             []uuid.UUID{sdm},
		Status:           ResidentStatusActive,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Resident
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.ID != in.ID || out.IHI != in.IHI || out.CareIntensity != in.CareIntensity {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
	if len(out.SDMs) != 1 || out.SDMs[0] != sdm {
		t.Errorf("SDMs round-trip lost: got %v", out.SDMs)
	}
}

func TestResidentOptionalAdmissionDate(t *testing.T) {
	in := Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, present := m["admission_date"]; present {
		t.Errorf("admission_date should be omitted when nil; got: %s", string(b))
	}
}
