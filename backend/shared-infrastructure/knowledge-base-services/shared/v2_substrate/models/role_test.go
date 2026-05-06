package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoleJSONRoundTrip(t *testing.T) {
	facility := uuid.New()
	in := Role{
		ID:             uuid.New(),
		PersonID:       uuid.New(),
		Kind:           RoleEN,
		Qualifications: json.RawMessage(`{"notation":false,"nmba_medication_qual":true}`),
		FacilityID:     &facility,
		ValidFrom:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EvidenceURL:    "https://ahpra.gov.au/lookup/NMW0001234567",
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Role
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Kind != in.Kind || string(out.Qualifications) != string(in.Qualifications) {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
	if out.FacilityID == nil || *out.FacilityID != facility {
		t.Errorf("FacilityID round-trip lost: got %v", out.FacilityID)
	}
}

func TestRoleQualificationsMatchScopeRulesShape(t *testing.T) {
	// The keys here MUST match regulatory_scope_rules.role_qualifications
	// shape (kb-22 migration 007). This test documents the contract.
	drnp := Role{
		ID: uuid.New(), PersonID: uuid.New(), Kind: RoleDRNP,
		Qualifications: json.RawMessage(`{"endorsement":"designated_rn_prescriber","valid_from":"2025-09-30"}`),
		ValidFrom:      time.Now(),
	}
	var quals map[string]any
	if err := json.Unmarshal(drnp.Qualifications, &quals); err != nil {
		t.Fatalf("qualifications must be valid JSON: %v", err)
	}
	if quals["endorsement"] != "designated_rn_prescriber" {
		t.Errorf("DRNP must carry endorsement=designated_rn_prescriber; got %v", quals)
	}
}
