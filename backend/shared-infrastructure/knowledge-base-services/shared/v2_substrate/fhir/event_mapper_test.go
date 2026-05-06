package fhir

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// reMarshal serialises and deserialises through JSON to model the wire-
// format trip the mapper output will take.
func reMarshal(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var rt map[string]interface{}
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return rt
}

func TestEventToAUFHIR_Fall_RoutesToEncounter(t *testing.T) {
	fac := uuid.New()
	in := models.Event{
		ID:                 uuid.New(),
		EventType:          models.EventTypeFall,
		OccurredAt:         time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		OccurredAtFacility: &fac,
		ResidentID:         uuid.New(),
		ReportedByRef:      uuid.New(),
		WitnessedByRefs:    []uuid.UUID{uuid.New()},
		Severity:           models.EventSeverityModerate,
		ReportableUnder:    []string{"QI Program"},
		DescriptionFreeText: "Resident slipped in bathroom",
	}
	fhir, err := EventToAUFHIR(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if fhir["resourceType"] != "Encounter" {
		t.Errorf("Clinical fall should route to Encounter, got %v", fhir["resourceType"])
	}
	out, err := AUFHIRToEvent(reMarshal(t, fhir))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.EventType != models.EventTypeFall || out.Severity != models.EventSeverityModerate {
		t.Errorf("round-trip drift: %+v", out)
	}
	if out.OccurredAtFacility == nil || *out.OccurredAtFacility != fac {
		t.Errorf("OccurredAtFacility lost: got %v", out.OccurredAtFacility)
	}
	if len(out.ReportableUnder) != 1 || out.ReportableUnder[0] != "QI Program" {
		t.Errorf("ReportableUnder lost: got %v", out.ReportableUnder)
	}
	if out.DescriptionFreeText != in.DescriptionFreeText {
		t.Errorf("DescriptionFreeText lost: got %q", out.DescriptionFreeText)
	}
	if out.ReportedByRef != in.ReportedByRef {
		t.Errorf("ReportedByRef lost")
	}
}

func TestEventToAUFHIR_HospitalAdmission_RoutesToEncounter(t *testing.T) {
	in := models.Event{
		ID:                    uuid.New(),
		EventType:             models.EventTypeHospitalAdmission,
		OccurredAt:            time.Now().UTC().Truncate(time.Second),
		ResidentID:            uuid.New(),
		ReportedByRef:         uuid.New(),
		Severity:              models.EventSeverityMajor,
		DescriptionStructured: json.RawMessage(`{"hospital":"acme","reason":"chest_pain"}`),
	}
	fhir, err := EventToAUFHIR(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if fhir["resourceType"] != "Encounter" {
		t.Errorf("hospital_admission should route to Encounter")
	}
	out, err := AUFHIRToEvent(reMarshal(t, fhir))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.EventType != models.EventTypeHospitalAdmission {
		t.Errorf("EventType drift: got %q", out.EventType)
	}
	if !json.Valid(out.DescriptionStructured) || string(out.DescriptionStructured) != string(in.DescriptionStructured) {
		t.Errorf("DescriptionStructured drift: got %s want %s", out.DescriptionStructured, in.DescriptionStructured)
	}
}

func TestEventToAUFHIR_RuleFire_RoutesToCommunication(t *testing.T) {
	in := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeRuleFire,
		OccurredAt:    time.Now().UTC().Truncate(time.Second),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
		DescriptionFreeText: "rule_id=DR-001 fired",
		TriggeredStateChanges: []models.TriggeredStateChange{
			{StateMachine: models.EventStateMachineRecommendation, StateChange: json.RawMessage(`{"to":"submitted"}`)},
		},
	}
	fhir, err := EventToAUFHIR(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if fhir["resourceType"] != "Communication" {
		t.Errorf("System rule_fire should route to Communication, got %v", fhir["resourceType"])
	}
	out, err := AUFHIRToEvent(reMarshal(t, fhir))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.EventType != models.EventTypeRuleFire {
		t.Errorf("EventType drift: got %q", out.EventType)
	}
	if len(out.TriggeredStateChanges) != 1 || out.TriggeredStateChanges[0].StateMachine != models.EventStateMachineRecommendation {
		t.Errorf("TriggeredStateChanges lost: %+v", out.TriggeredStateChanges)
	}
	if out.DescriptionFreeText != in.DescriptionFreeText {
		t.Errorf("DescriptionFreeText (Communication.payload) drift: got %q", out.DescriptionFreeText)
	}
}

func TestEventToAUFHIR_RelatedRefs_RoundTrip(t *testing.T) {
	med1, med2 := uuid.New(), uuid.New()
	obs1 := uuid.New()
	in := models.Event{
		ID:                    uuid.New(),
		EventType:             models.EventTypeAdverseDrugEvent,
		OccurredAt:            time.Now().UTC().Truncate(time.Second),
		ResidentID:            uuid.New(),
		ReportedByRef:         uuid.New(),
		RelatedMedicationUses: []uuid.UUID{med1, med2},
		RelatedObservations:   []uuid.UUID{obs1},
	}
	fhir, err := EventToAUFHIR(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	out, err := AUFHIRToEvent(reMarshal(t, fhir))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if len(out.RelatedMedicationUses) != 2 || out.RelatedMedicationUses[0] != med1 || out.RelatedMedicationUses[1] != med2 {
		t.Errorf("RelatedMedicationUses drift: %v", out.RelatedMedicationUses)
	}
	if len(out.RelatedObservations) != 1 || out.RelatedObservations[0] != obs1 {
		t.Errorf("RelatedObservations drift: %v", out.RelatedObservations)
	}
}

func TestEventToAUFHIR_RejectsInvalid(t *testing.T) {
	bad := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeFall, // missing severity → fails fall rule
		OccurredAt:    time.Now(),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
	if _, err := EventToAUFHIR(bad); err == nil {
		t.Errorf("expected egress validation error")
	}
}

func TestAUFHIRToEvent_WrongResourceType(t *testing.T) {
	if _, err := AUFHIRToEvent(map[string]interface{}{"resourceType": "Patient"}); err == nil {
		t.Errorf("expected error for resourceType=Patient")
	}
}

func TestEventToAUFHIR_WireFormatHasEventTypeExtension(t *testing.T) {
	in := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeGPVisit,
		OccurredAt:    time.Now().UTC(),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
	fhir, err := EventToAUFHIR(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(fhir)
	s := string(b)
	if !strings.Contains(s, ExtEventType) {
		t.Errorf("wire format missing ExtEventType URI; got: %s", s)
	}
	if !strings.Contains(s, `"resourceType":"Encounter"`) {
		t.Errorf("wire format missing Encounter resourceType; got: %s", s)
	}
}
