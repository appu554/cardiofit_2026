package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCapacityAssessmentJSONRoundTrip(t *testing.T) {
	review := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	supersedes := uuid.New()
	score := 22.0
	in := CapacityAssessment{
		ID:                  uuid.New(),
		ResidentRef:         uuid.New(),
		AssessedAt:          time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		AssessorRoleRef:     uuid.New(),
		Domain:              CapacityDomainMedical,
		Instrument:          CapacityInstrumentMoCA,
		Score:               &score,
		Outcome:             CapacityOutcomeImpaired,
		Duration:            CapacityDurationTemporary,
		ExpectedReviewDate:  &review,
		RationaleStructured: json.RawMessage(`{"snomed":"289253008"}`),
		RationaleFreeText:   "Delirium secondary to UTI; reassess after course",
		SupersedesRef:       &supersedes,
		CreatedAt:           time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out CapacityAssessment
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Domain != in.Domain {
		t.Errorf("Domain drift: got %s want %s", out.Domain, in.Domain)
	}
	if out.Outcome != in.Outcome {
		t.Errorf("Outcome drift: got %s want %s", out.Outcome, in.Outcome)
	}
	if out.Duration != in.Duration {
		t.Errorf("Duration drift: got %s want %s", out.Duration, in.Duration)
	}
	if out.Score == nil || *out.Score != score {
		t.Errorf("Score drift: got %v", out.Score)
	}
	if out.SupersedesRef == nil || *out.SupersedesRef != supersedes {
		t.Errorf("SupersedesRef drift")
	}
	if out.ExpectedReviewDate == nil || !out.ExpectedReviewDate.Equal(review) {
		t.Errorf("ExpectedReviewDate drift")
	}
	if string(out.RationaleStructured) != string(in.RationaleStructured) {
		t.Errorf("RationaleStructured drift")
	}
}

func TestCapacityAssessmentOmitsEmptyOptionalFields(t *testing.T) {
	in := CapacityAssessment{
		ID:              uuid.New(),
		ResidentRef:     uuid.New(),
		AssessedAt:      time.Now().UTC(),
		AssessorRoleRef: uuid.New(),
		Domain:          CapacityDomainMedical,
		Outcome:         CapacityOutcomeIntact,
		Duration:        CapacityDurationPermanent,
	}
	b, _ := json.Marshal(in)
	s := string(b)
	for _, k := range []string{
		`"instrument"`, `"score"`, `"expected_review_date"`,
		`"rationale_structured"`, `"rationale_free_text"`, `"supersedes_ref"`,
	} {
		if strings.Contains(s, k) {
			t.Errorf("expected %s to be omitted; got %s", k, s)
		}
	}
}

func TestIsValidCapacityDomain(t *testing.T) {
	for _, d := range []string{
		CapacityDomainMedical, CapacityDomainFinancial,
		CapacityDomainAccommodation, CapacityDomainRestrictivePractice,
		CapacityDomainMedicationDecisions,
	} {
		if !IsValidCapacityDomain(d) {
			t.Errorf("expected %q valid", d)
		}
	}
	for _, d := range []string{"", "MEDICAL", "medical", "unknown"} {
		if IsValidCapacityDomain(d) {
			t.Errorf("expected %q invalid", d)
		}
	}
}

func TestIsValidCapacityOutcome(t *testing.T) {
	for _, o := range []string{
		CapacityOutcomeIntact, CapacityOutcomeImpaired, CapacityOutcomeUnableToAssess,
	} {
		if !IsValidCapacityOutcome(o) {
			t.Errorf("expected %q valid", o)
		}
	}
	for _, o := range []string{"", "INTACT", "lost", "yes"} {
		if IsValidCapacityOutcome(o) {
			t.Errorf("expected %q invalid", o)
		}
	}
}

func TestIsValidCapacityDuration(t *testing.T) {
	for _, d := range []string{
		CapacityDurationPermanent, CapacityDurationTemporary, CapacityDurationUnableToDetermine,
	} {
		if !IsValidCapacityDuration(d) {
			t.Errorf("expected %q valid", d)
		}
	}
	for _, d := range []string{"", "PERMANENT", "forever", "transient"} {
		if IsValidCapacityDuration(d) {
			t.Errorf("expected %q invalid", d)
		}
	}
}
