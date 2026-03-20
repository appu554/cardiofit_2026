package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestSeasonCoach_GetMaxNudges_DecreasesPerSeason(t *testing.T) {
	coach := NewSeasonCoach(nil, nil)
	tests := []struct {
		season models.EngagementSeason
		want   int
	}{
		{models.SeasonCorrection, 5},
		{models.SeasonConsolidation, 3},
		{models.SeasonIndependence, 2},
		{models.SeasonStability, 1},
		{models.SeasonPartnership, 1},
	}
	for _, tt := range tests {
		t.Run(string(tt.season), func(t *testing.T) {
			got := coach.GetMaxNudgesPerDay(tt.season)
			if got != tt.want {
				t.Errorf("max nudges = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSeasonCoach_IsEventTriggered(t *testing.T) {
	coach := NewSeasonCoach(nil, nil)

	if coach.IsEventTriggered(models.SeasonCorrection) {
		t.Error("S1 should be calendar-triggered, not event-triggered")
	}
	if coach.IsEventTriggered(models.SeasonConsolidation) {
		t.Error("S2 should be calendar-triggered, not event-triggered")
	}
	if !coach.IsEventTriggered(models.SeasonIndependence) {
		t.Error("S3 should be event-triggered")
	}
	if !coach.IsEventTriggered(models.SeasonPartnership) {
		t.Error("S5 should be event-triggered")
	}
}

func TestSeasonCoach_ShouldContactPatient_EventTriggered(t *testing.T) {
	coach := NewSeasonCoach(nil, nil)

	if coach.ShouldContactPatient(models.SeasonPartnership, false) {
		t.Error("S5 without trigger event should not contact patient")
	}
	if !coach.ShouldContactPatient(models.SeasonPartnership, true) {
		t.Error("S5 with trigger event should contact patient")
	}
	if !coach.ShouldContactPatient(models.SeasonCorrection, false) {
		t.Error("S1 should always contact (calendar-triggered)")
	}
}
