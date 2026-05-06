package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestPersonJSONRoundTrip(t *testing.T) {
	in := Person{
		ID:                uuid.New(),
		GivenName:         "Sarah",
		FamilyName:        "Chen",
		HPII:              "8003614900000000",
		AHPRARegistration: "NMW0001234567",
		ContactDetails:    json.RawMessage(`{"email":"sarah.chen@example.com"}`),
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Person
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.ID != in.ID || out.GivenName != in.GivenName || out.HPII != in.HPII {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func TestPersonOptionalHPII(t *testing.T) {
	in := Person{ID: uuid.New(), GivenName: "X", FamilyName: "Y"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, present := m["hpii"]; present {
		t.Errorf("hpii should be omitted when empty; got: %s", string(b))
	}
}

// TestPersonContactDetailsJSONBShape documents the contact details JSONB
// contract — analogous to Role.Qualifications. Persisted as a freeform
// JSONB blob; downstream consumers (KB-20 patient-profile) interpret
// shapes like {"email":"...", "phone":"..."}.
func TestPersonContactDetailsJSONBShape(t *testing.T) {
	in := Person{
		ID:             uuid.New(),
		GivenName:      "X",
		FamilyName:     "Y",
		ContactDetails: json.RawMessage(`{"email":"x@y.com","phone":"+61400000000"}`),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Person
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var shape map[string]any
	if err := json.Unmarshal(out.ContactDetails, &shape); err != nil {
		t.Fatalf("contact details must be valid JSON object: %v", err)
	}
	if shape["email"] != "x@y.com" {
		t.Errorf("contact.email lost in round-trip; got %v", shape["email"])
	}
	if shape["phone"] != "+61400000000" {
		t.Errorf("contact.phone lost in round-trip; got %v", shape["phone"])
	}
}
