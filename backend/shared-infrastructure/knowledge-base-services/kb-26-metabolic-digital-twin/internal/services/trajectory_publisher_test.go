package services

import (
	"context"
	"encoding/json"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestNewDomainTrajectoryComputedEvent_PopulatesAllFields(t *testing.T) {
	driver := models.DomainGlucose
	traj := &models.DecomposedTrajectory{
		PatientID:               "pat-001",
		WindowDays:              13,
		DataPoints:              5,
		CompositeSlope:          -1.42,
		CompositeTrend:          models.TrendDeclining,
		CompositeStartScore:     62.0,
		CompositeEndScore:       42.0,
		DominantDriver:          &driver,
		DriverContribution:      45.3,
		HasDiscordantTrend:      false,
		ConcordantDeterioration: true,
		DomainsDeteriorating:    3,
		DomainSlopes: map[models.MHRIDomain]models.DomainSlope{
			models.DomainGlucose: {
				Domain:      models.DomainGlucose,
				SlopePerDay: -1.67,
				Trend:       models.TrendRapidDeclining,
				Confidence:  models.ConfidenceHigh,
				R2:          0.98,
			},
		},
	}

	event := NewDomainTrajectoryComputedEvent(traj)

	if event.EventType != "DomainTrajectoryComputed" {
		t.Errorf("expected event_type DomainTrajectoryComputed, got %s", event.EventType)
	}
	if event.EventVersion != "v1" {
		t.Errorf("expected v1, got %s", event.EventVersion)
	}
	if event.PatientID != "pat-001" {
		t.Errorf("expected patient_id pat-001, got %s", event.PatientID)
	}
	if !event.ConcordantDeterioration {
		t.Error("expected ConcordantDeterioration true")
	}
	if event.DomainsDeteriorating != 3 {
		t.Errorf("expected DomainsDeteriorating 3, got %d", event.DomainsDeteriorating)
	}

	glucose, ok := event.Domains["GLUCOSE"]
	if !ok {
		t.Fatal("expected GLUCOSE domain in event")
	}
	if glucose.SlopePerDay != -1.67 {
		t.Errorf("expected glucose slope -1.67, got %.2f", glucose.SlopePerDay)
	}
	if glucose.Confidence != models.ConfidenceHigh {
		t.Errorf("expected glucose confidence HIGH, got %s", glucose.Confidence)
	}

	if event.DominantDriver == nil || *event.DominantDriver != "GLUCOSE" {
		t.Errorf("expected dominant_driver GLUCOSE, got %v", event.DominantDriver)
	}
}

func TestDomainTrajectoryComputedEvent_JSONRoundTrip(t *testing.T) {
	traj := &models.DecomposedTrajectory{
		PatientID:      "pat-002",
		WindowDays:     7,
		DataPoints:     3,
		CompositeSlope: 0.5,
		CompositeTrend: models.TrendImproving,
		DomainSlopes: map[models.MHRIDomain]models.DomainSlope{
			models.DomainCardio: {Domain: models.DomainCardio, SlopePerDay: 0.6, Trend: models.TrendImproving, Confidence: models.ConfidenceHigh, R2: 0.92},
		},
	}

	event := NewDomainTrajectoryComputedEvent(traj)
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundtrip DomainTrajectoryComputedEvent
	if err := json.Unmarshal(body, &roundtrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if roundtrip.PatientID != "pat-002" {
		t.Errorf("expected pat-002, got %s", roundtrip.PatientID)
	}
	if roundtrip.Domains["CARDIO"].SlopePerDay != 0.6 {
		t.Errorf("expected cardio slope 0.6, got %.2f", roundtrip.Domains["CARDIO"].SlopePerDay)
	}
}

func TestNoopTrajectoryPublisher_NeverErrors(t *testing.T) {
	noop := NoopTrajectoryPublisher{}
	event := DomainTrajectoryComputedEvent{PatientID: "pat-003"}

	if err := noop.Publish(context.Background(), event); err != nil {
		t.Errorf("noop publisher should not error, got %v", err)
	}
}
