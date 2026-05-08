package permissions

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDataAggregationConsent_RevokedDenies(t *testing.T) {
	now := time.Now().UTC()
	revokedAt := now.Add(-1 * time.Hour)
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         now.Add(-30 * 24 * time.Hour),
		ExpiresAt:         now.Add(335 * 24 * time.Hour),
		RevokedAt:         &revokedAt,
	}
	if c.Active(now) {
		t.Error("revoked consent must not be active")
	}
}

func TestDataAggregationConsent_ExpiredDenies(t *testing.T) {
	now := time.Now().UTC()
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeContractRetention,
		GrantedAt:         now.Add(-400 * 24 * time.Hour),
		ExpiresAt:         now.Add(-1 * time.Hour), // already expired
	}
	if c.Active(now) {
		t.Error("expired consent must not be active")
	}
}

func TestDataAggregationConsent_WrongPurposeDenies(t *testing.T) {
	now := time.Now().UTC()
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         now.Add(-30 * 24 * time.Hour),
		ExpiresAt:         now.Add(335 * 24 * time.Hour),
	}
	if c.ActiveForPurpose(PurposeRegulatoryEvidence, now) {
		t.Error("consent granted for workforce_planning must not cover regulatory_evidence")
	}
}

func TestDataAggregationConsent_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         now.Add(-30 * 24 * time.Hour),
		ExpiresAt:         now.Add(335 * 24 * time.Hour),
	}
	if !c.Active(now) {
		t.Error("valid non-revoked non-expired consent must be active")
	}
	if !c.ActiveForPurpose(PurposeWorkforcePlanning, now) {
		t.Error("consent must be active for its declared purpose")
	}
}

func TestIsValidPurpose(t *testing.T) {
	valid := []string{
		PurposeWorkforcePlanning,
		PurposeContractRetention,
		PurposeRegulatoryEvidence,
		PurposePeerDevelopment,
	}
	for _, p := range valid {
		if !IsValidPurpose(p) {
			t.Errorf("expected %q to be a valid purpose", p)
		}
	}
	if IsValidPurpose("not_a_purpose") {
		t.Error("expected \"not_a_purpose\" to be invalid")
	}
}
