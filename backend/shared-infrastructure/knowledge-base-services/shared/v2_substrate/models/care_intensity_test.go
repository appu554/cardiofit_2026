package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCareIntensityJSONRoundTrip(t *testing.T) {
	review := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	supersedes := uuid.New()
	in := CareIntensity{
		ID:                  uuid.New(),
		ResidentRef:         uuid.New(),
		Tag:                 CareIntensityTagPalliative,
		EffectiveDate:       time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		DocumentedByRoleRef: uuid.New(),
		ReviewDueDate:       &review,
		RationaleStructured: json.RawMessage(`{"snomed":"428361000124107"}`),
		RationaleFreeText:   "Family meeting outcome — focus on comfort",
		SupersedesRef:       &supersedes,
		CreatedAt:           time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out CareIntensity
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Tag != in.Tag {
		t.Errorf("Tag drift: got %s want %s", out.Tag, in.Tag)
	}
	if out.SupersedesRef == nil || *out.SupersedesRef != supersedes {
		t.Errorf("SupersedesRef drift")
	}
	if out.ReviewDueDate == nil || !out.ReviewDueDate.Equal(review) {
		t.Errorf("ReviewDueDate drift: got %v want %v", out.ReviewDueDate, review)
	}
	if string(out.RationaleStructured) != string(in.RationaleStructured) {
		t.Errorf("RationaleStructured drift: got %s", string(out.RationaleStructured))
	}
}

func TestCareIntensityOmitsEmptyOptionalFields(t *testing.T) {
	in := CareIntensity{
		ID:                  uuid.New(),
		ResidentRef:         uuid.New(),
		Tag:                 CareIntensityTagActiveTreatment,
		EffectiveDate:       time.Now().UTC(),
		DocumentedByRoleRef: uuid.New(),
	}
	b, _ := json.Marshal(in)
	s := string(b)
	for _, k := range []string{
		`"review_due_date"`, `"rationale_structured"`,
		`"rationale_free_text"`, `"supersedes_ref"`,
	} {
		if strings.Contains(s, k) {
			t.Errorf("expected %s to be omitted; got %s", k, s)
		}
	}
}

func TestIsValidCareIntensityTag(t *testing.T) {
	for _, tag := range []string{
		CareIntensityTagActiveTreatment,
		CareIntensityTagRehabilitation,
		CareIntensityTagComfortFocused,
		CareIntensityTagPalliative,
	} {
		if !IsValidCareIntensityTag(tag) {
			t.Errorf("expected %q to be valid", tag)
		}
	}
	for _, tag := range []string{"", "active", "comfort", "unknown", "ACTIVE_TREATMENT"} {
		if IsValidCareIntensityTag(tag) {
			t.Errorf("expected %q to be invalid", tag)
		}
	}
}

func TestIsValidCareIntensityTransition(t *testing.T) {
	// Empty 'from' is valid for a resident's first row.
	if !IsValidCareIntensityTransition("", CareIntensityTagActiveTreatment) {
		t.Errorf("expected empty→active_treatment to be valid (first row)")
	}
	// All four-by-four valid tag transitions are allowed in MVP.
	tags := []string{
		CareIntensityTagActiveTreatment,
		CareIntensityTagRehabilitation,
		CareIntensityTagComfortFocused,
		CareIntensityTagPalliative,
	}
	for _, from := range tags {
		for _, to := range tags {
			if !IsValidCareIntensityTransition(from, to) {
				t.Errorf("expected %s→%s to be valid", from, to)
			}
		}
	}
	// Invalid 'to' is rejected even with an empty 'from'.
	if IsValidCareIntensityTransition("", "bogus") {
		t.Errorf("expected empty→bogus to be invalid")
	}
	// Invalid 'from' (non-empty, unknown) is rejected.
	if IsValidCareIntensityTransition("bogus", CareIntensityTagActiveTreatment) {
		t.Errorf("expected bogus→active_treatment to be invalid")
	}
}

func TestLegacyCareIntensityForTag(t *testing.T) {
	cases := map[string]string{
		CareIntensityTagActiveTreatment: CareIntensityActive,
		CareIntensityTagRehabilitation:  CareIntensityRehabilitation,
		CareIntensityTagComfortFocused:  CareIntensityComfort,
		CareIntensityTagPalliative:      CareIntensityPalliative,
		"bogus":                         "",
		"":                              "",
	}
	for in, want := range cases {
		if got := LegacyCareIntensityForTag(in); got != want {
			t.Errorf("LegacyCareIntensityForTag(%q) = %q, want %q", in, got, want)
		}
	}
}
