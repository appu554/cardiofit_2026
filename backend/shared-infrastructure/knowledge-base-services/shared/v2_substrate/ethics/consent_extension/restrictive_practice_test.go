package consent_extension

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Plan-verbatim tests (Task 7 §Step 1–3)
// ---------------------------------------------------------------------------

func TestRestrictivePractice_ActiveConsentAllows(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType:                          PracticeChemicalRestraint,
		Status:                                "active",
		MaxDuration:                           12 * 7 * 24 * time.Hour,
		GrantedAt:                             time.Now().Add(-7 * 24 * time.Hour),
		LessRestrictiveAlternativesDocumented: true,
	}
	if !c.Allows(time.Now()) {
		t.Errorf("active consent within max duration should allow")
	}
}

func TestRestrictivePractice_ExpiredDenies(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType: PracticeChemicalRestraint,
		Status:       "active",
		MaxDuration:  12 * 7 * 24 * time.Hour,
		GrantedAt:    time.Now().Add(-100 * 24 * time.Hour),
	}
	if c.Allows(time.Now()) {
		t.Errorf("expired consent should deny")
	}
}

func TestRestrictivePractice_MissingAlternativesDenies(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType:                          PracticeChemicalRestraint,
		Status:                                "active",
		LessRestrictiveAlternativesDocumented: false,
		GrantedAt:                             time.Now(),
		MaxDuration:                           12 * 7 * 24 * time.Hour,
	}
	if c.Allows(time.Now()) {
		t.Errorf("missing less-restrictive-alternatives documentation should deny")
	}
}

// ---------------------------------------------------------------------------
// Augmentation: withdrawn status
// ---------------------------------------------------------------------------

// TestRestrictivePracticeConsent_WithdrawnDenies verifies that a consent
// whose Status is "withdrawn" causes Allows to return false.
// Allows requires Status == "active"; withdrawn is an explicit terminal
// state that must hard-block the practice.
func TestRestrictivePracticeConsent_WithdrawnDenies(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType:                          PracticePhysicalRestraint,
		Status:                                "withdrawn",
		LessRestrictiveAlternativesDocumented: true,
		GrantedAt:                             time.Now().Add(-24 * time.Hour),
		MaxDuration:                           7 * 24 * time.Hour,
	}
	if c.Allows(time.Now()) {
		t.Errorf("withdrawn consent should deny; Allows must return false for Status != \"active\"")
	}
}

// ---------------------------------------------------------------------------
// Augmentation: chemical restraint max-duration cap
// ---------------------------------------------------------------------------

// TestRestrictivePracticeConsent_ChemicalRestraintMaxDurationCap verifies
// that Validate returns an error when MaxDuration exceeds 12 weeks for a
// chemical-restraint consent, and returns nil at or below 12 weeks.
//
// Per Guidelines §6.3.
func TestRestrictivePracticeConsent_ChemicalRestraintMaxDurationCap(t *testing.T) {
	const twelveWeeks = 12 * 7 * 24 * time.Hour

	// At exactly 12 weeks — must be valid.
	atCap := RestrictivePracticeConsent{
		ID:           uuid.New(),
		PracticeType: PracticeChemicalRestraint,
		MaxDuration:  twelveWeeks,
	}
	if err := atCap.Validate(); err != nil {
		t.Errorf("exactly 12-week duration should be valid; got error: %v", err)
	}

	// One nanosecond over 12 weeks — must fail.
	overCap := RestrictivePracticeConsent{
		ID:           uuid.New(),
		PracticeType: PracticeChemicalRestraint,
		MaxDuration:  twelveWeeks + time.Nanosecond,
	}
	if err := overCap.Validate(); err == nil {
		t.Errorf("duration > 12 weeks for chemical restraint should return validation error")
	}

	// Non-chemical practice at > 12 weeks — Validate does not constrain it.
	physicalOverCap := RestrictivePracticeConsent{
		ID:           uuid.New(),
		PracticeType: PracticePhysicalRestraint,
		MaxDuration:  twentyWeeks,
	}
	if err := physicalOverCap.Validate(); err != nil {
		t.Errorf("physical restraint is not subject to 12-week cap; got unexpected error: %v", err)
	}
}

// twentyWeeks is a helper constant used by the cap test.
const twentyWeeks = 20 * 7 * 24 * time.Hour

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestIsValidPracticeType(t *testing.T) {
	valid := []string{
		string(PracticeChemicalRestraint),
		string(PracticePhysicalRestraint),
		string(PracticeEnvironmentalRestraint),
		string(PracticeSeclusion),
	}
	for _, s := range valid {
		if !IsValidPracticeType(s) {
			t.Errorf("IsValidPracticeType(%q) should return true", s)
		}
	}
	if IsValidPracticeType("medication_withholding") {
		t.Errorf("IsValidPracticeType(%q) should return false", "medication_withholding")
	}
}
