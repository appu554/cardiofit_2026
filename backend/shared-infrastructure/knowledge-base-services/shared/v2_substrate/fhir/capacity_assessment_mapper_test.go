package fhir

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// reMarshal is defined in event_mapper_test.go (same package).

func TestCapacityAssessmentToFHIR_IntactMedical_RoundTrip(t *testing.T) {
	in := models.CapacityAssessment{
		ID:                uuid.New(),
		ResidentRef:       uuid.New(),
		AssessedAt:        time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC),
		AssessorRoleRef:   uuid.New(),
		Domain:            models.CapacityDomainMedical,
		Outcome:           models.CapacityOutcomeIntact,
		Duration:          models.CapacityDurationPermanent,
		RationaleFreeText: "Clear orientation, recall 3/3, abstract reasoning intact",
	}
	obs, err := CapacityAssessmentToFHIRObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if obs["resourceType"] != "Observation" {
		t.Errorf("resourceType drift")
	}
	// category=assessment check
	cats, ok := obs["category"].([]map[string]interface{})
	if !ok || len(cats) == 0 {
		t.Fatalf("missing category")
	}
	codings, _ := cats[0]["coding"].([]map[string]interface{})
	if len(codings) == 0 || codings[0]["code"] != "assessment" {
		t.Errorf("category code drift: %+v", codings)
	}

	out, err := FHIRObservationToCapacityAssessment(reMarshal(t, obs))
	if err != nil {
		t.Fatalf("ingress: %v", err)
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
	if !out.AssessedAt.Equal(in.AssessedAt) {
		t.Errorf("AssessedAt drift")
	}
	if out.AssessorRoleRef != in.AssessorRoleRef {
		t.Errorf("AssessorRoleRef drift")
	}
	if out.RationaleFreeText != in.RationaleFreeText {
		t.Errorf("RationaleFreeText drift")
	}
}

func TestCapacityAssessmentToFHIR_ImpairedFinancialWithScore_RoundTrip(t *testing.T) {
	score := 18.5
	in := models.CapacityAssessment{
		ID:                  uuid.New(),
		ResidentRef:         uuid.New(),
		AssessedAt:          time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC),
		AssessorRoleRef:     uuid.New(),
		Domain:              models.CapacityDomainFinancial,
		Instrument:          models.CapacityInstrumentMoCA,
		Score:               &score,
		Outcome:             models.CapacityOutcomeImpaired,
		Duration:            models.CapacityDurationPermanent,
		RationaleStructured: json.RawMessage(`{"snomed":"386807006"}`),
	}
	obs, err := CapacityAssessmentToFHIRObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	out, err := FHIRObservationToCapacityAssessment(reMarshal(t, obs))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.Domain != models.CapacityDomainFinancial {
		t.Errorf("Domain drift: %s", out.Domain)
	}
	if out.Instrument != in.Instrument {
		t.Errorf("Instrument drift: %s", out.Instrument)
	}
	if out.Score == nil || *out.Score != score {
		t.Errorf("Score drift: %v", out.Score)
	}
	if string(out.RationaleStructured) != string(in.RationaleStructured) {
		t.Errorf("RationaleStructured drift: %s", string(out.RationaleStructured))
	}
}

func TestCapacityAssessmentToFHIR_TemporaryWithReviewDate_RoundTrip(t *testing.T) {
	rev := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	in := models.CapacityAssessment{
		ID:                 uuid.New(),
		ResidentRef:        uuid.New(),
		AssessedAt:         time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC),
		AssessorRoleRef:    uuid.New(),
		Domain:             models.CapacityDomainMedical,
		Outcome:            models.CapacityOutcomeImpaired,
		Duration:           models.CapacityDurationTemporary,
		ExpectedReviewDate: &rev,
	}
	obs, err := CapacityAssessmentToFHIRObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	out, err := FHIRObservationToCapacityAssessment(reMarshal(t, obs))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.Duration != models.CapacityDurationTemporary {
		t.Errorf("Duration drift")
	}
	if out.ExpectedReviewDate == nil || !out.ExpectedReviewDate.Equal(rev) {
		t.Errorf("ExpectedReviewDate drift: %v", out.ExpectedReviewDate)
	}
	// Ingress validation: temporary requires expected_review_date strictly
	// after assessed_at. The successful round-trip already proves this; a
	// targeted negative test below confirms the validator runs on ingress.
}

func TestCapacityAssessmentFHIR_RejectsInvalid(t *testing.T) {
	// Missing duration extension causes ingress validation to fail
	// (Duration is a required field).
	bad := map[string]interface{}{
		"resourceType":      "Observation",
		"id":                uuid.New().String(),
		"effectiveDateTime": time.Now().UTC().Format(time.RFC3339),
		"subject":           map[string]interface{}{"reference": "Patient/" + uuid.New().String()},
		"performer":         []interface{}{map[string]interface{}{"reference": "Role/" + uuid.New().String()}},
		"code": map[string]interface{}{"coding": []interface{}{
			map[string]interface{}{"system": SystemCapacityAssessment, "code": models.CapacityDomainMedical},
		}},
		"valueCodeableConcept": map[string]interface{}{"coding": []interface{}{
			map[string]interface{}{"system": SystemCapacityAssessment, "code": models.CapacityOutcomeIntact},
		}},
		// no extension array → no duration → validator rejects
	}
	if _, err := FHIRObservationToCapacityAssessment(bad); err == nil {
		t.Errorf("expected ingress validation failure for missing duration")
	}
}

func TestCapacityAssessmentFHIR_WrongResourceType(t *testing.T) {
	bad := map[string]interface{}{"resourceType": "Patient"}
	if _, err := FHIRObservationToCapacityAssessment(bad); err == nil {
		t.Errorf("expected error for wrong resourceType")
	}
}
