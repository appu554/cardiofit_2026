package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMonitoringPlanJSONRoundTrip(t *testing.T) {
	due := time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	in := MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: uuid.New(),
		ResidentID:       uuid.New(),
		State:            MonitoringPlanStateActive,
		Obligations: []MonitoringObligation{
			{
				Type:            MonitoringObligationTypeObservation,
				ObservationCode: "blood_pressure",
				FrequencyHours:  24,
				DueAt:           due,
				ThresholdSpec:   "value > 160 OR value < 90",
				FulfilledAt:     nil,
			},
		},
		StartedAt:           time.Now().UTC().Truncate(time.Microsecond),
		ExpectedEndAt:       due,
		EscalateAfterMissed: 2,
		CreatedAt:           time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:           time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out MonitoringPlan
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.State != in.State || out.RecommendationID != in.RecommendationID {
		t.Errorf("scalars mismatch: got %+v", out)
	}
	if len(out.Obligations) != 1 {
		t.Fatalf("obligations lost: got %d", len(out.Obligations))
	}
	got := out.Obligations[0]
	if got.Type != MonitoringObligationTypeObservation ||
		got.ObservationCode != "blood_pressure" ||
		got.FrequencyHours != 24 ||
		!got.DueAt.Equal(due) ||
		got.ThresholdSpec != "value > 160 OR value < 90" {
		t.Errorf("obligation lost detail in round trip: %+v", got)
	}
	if got.FulfilledAt != nil {
		t.Errorf("FulfilledAt should be nil; got %v", *got.FulfilledAt)
	}
}

func TestMonitoringPlanRoundTripFulfilledObligation(t *testing.T) {
	fulfilled := time.Now().UTC().Truncate(time.Microsecond)
	obsID := uuid.New()
	in := MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: uuid.New(),
		ResidentID:       uuid.New(),
		State:            MonitoringPlanStateCompleted,
		Obligations: []MonitoringObligation{
			{
				Type:             MonitoringObligationTypeLab,
				ObservationCode:  "potassium",
				DueAt:            time.Now().UTC().Truncate(time.Microsecond),
				ThresholdSpec:    "value > 5.5",
				FulfilledAt:      &fulfilled,
				FulfilledByObsID: &obsID,
			},
		},
		StartedAt:           time.Now().UTC().Truncate(time.Microsecond),
		ExpectedEndAt:       time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Microsecond),
		EscalateAfterMissed: 1,
		CreatedAt:           time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:           time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, _ := json.Marshal(in)
	var out MonitoringPlan
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Obligations[0].FulfilledAt == nil {
		t.Errorf("FulfilledAt nil after round trip")
	}
	if out.Obligations[0].FulfilledByObsID == nil ||
		*out.Obligations[0].FulfilledByObsID != obsID {
		t.Errorf("FulfilledByObsID lost: %+v", out.Obligations[0].FulfilledByObsID)
	}
}

func TestMonitoringTransitionMatrix(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		// Happy path
		{MonitoringPlanStatePending, MonitoringPlanStateActive, true},
		{MonitoringPlanStatePending, MonitoringPlanStateAbandoned, true},
		{MonitoringPlanStateActive, MonitoringPlanStateCompleted, true},
		{MonitoringPlanStateActive, MonitoringPlanStateEscalated, true},
		{MonitoringPlanStateActive, MonitoringPlanStateAbandoned, true},

		// Forbidden — terminal
		{MonitoringPlanStateCompleted, MonitoringPlanStateActive, false},
		{MonitoringPlanStateEscalated, MonitoringPlanStateActive, false},
		{MonitoringPlanStateAbandoned, MonitoringPlanStateActive, false},

		// Forbidden — skipping active
		{MonitoringPlanStatePending, MonitoringPlanStateCompleted, false},
		{MonitoringPlanStatePending, MonitoringPlanStateEscalated, false},

		// Forbidden — bogus
		{"bogus", MonitoringPlanStateActive, false},
		{MonitoringPlanStateActive, "bogus", false},
	}
	for _, c := range cases {
		if got := IsValidMonitoringTransition(c.from, c.to); got != c.want {
			t.Errorf("IsValidMonitoringTransition(%q,%q)=%v want %v",
				c.from, c.to, got, c.want)
		}
	}
}

func TestIsValidMonitoringPlanState(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{MonitoringPlanStatePending, true},
		{MonitoringPlanStateActive, true},
		{MonitoringPlanStateCompleted, true},
		{MonitoringPlanStateEscalated, true},
		{MonitoringPlanStateAbandoned, true},
		{"bogus", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsValidMonitoringPlanState(c.s); got != c.want {
			t.Errorf("IsValidMonitoringPlanState(%q)=%v want %v", c.s, got, c.want)
		}
	}
}

func TestIsValidMonitoringObligationType(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{MonitoringObligationTypeObservation, true},
		{MonitoringObligationTypeFollowUpReview, true},
		{MonitoringObligationTypeBehaviouralChart, true},
		{MonitoringObligationTypeLab, true},
		{"bogus", false},
	}
	for _, c := range cases {
		if got := IsValidMonitoringObligationType(c.s); got != c.want {
			t.Errorf("IsValidMonitoringObligationType(%q)=%v want %v", c.s, got, c.want)
		}
	}
}
