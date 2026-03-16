package services

import (
	"testing"
	"time"

	"kb-patient-profile/internal/models"
)

func TestComputeFBGSlope_Worsening(t *testing.T) {
	readings := []models.TimestampedReading{
		{Value: 6.0, Timestamp: time.Now().Add(-90 * 24 * time.Hour)},
		{Value: 6.5, Timestamp: time.Now().Add(-60 * 24 * time.Hour)},
		{Value: 7.2, Timestamp: time.Now().Add(-30 * 24 * time.Hour)},
		{Value: 8.0, Timestamp: time.Now()},
	}
	slope := computeFBGSlope(readings)
	if slope <= 0 {
		t.Errorf("slope = %.2f, want positive (worsening)", slope)
	}
	trend := classifyFBGTrend(slope)
	if trend != "WORSENING" {
		t.Errorf("trend = %s, want WORSENING", trend)
	}
}

func TestComputeFBGSlope_Improving(t *testing.T) {
	readings := []models.TimestampedReading{
		{Value: 9.0, Timestamp: time.Now().Add(-90 * 24 * time.Hour)},
		{Value: 8.0, Timestamp: time.Now().Add(-60 * 24 * time.Hour)},
		{Value: 7.0, Timestamp: time.Now().Add(-30 * 24 * time.Hour)},
		{Value: 6.5, Timestamp: time.Now()},
	}
	slope := computeFBGSlope(readings)
	if slope >= 0 {
		t.Errorf("slope = %.2f, want negative (improving)", slope)
	}
	trend := classifyFBGTrend(slope)
	if trend != "IMPROVING" {
		t.Errorf("trend = %s, want IMPROVING", trend)
	}
}

func TestComputeGlucoseCV(t *testing.T) {
	readings := []models.TimestampedReading{
		{Value: 4.0, Timestamp: time.Now().Add(-6 * 24 * time.Hour)},
		{Value: 11.0, Timestamp: time.Now().Add(-5 * 24 * time.Hour)},
		{Value: 4.5, Timestamp: time.Now().Add(-4 * 24 * time.Hour)},
		{Value: 12.0, Timestamp: time.Now().Add(-3 * 24 * time.Hour)},
		{Value: 5.0, Timestamp: time.Now().Add(-2 * 24 * time.Hour)},
		{Value: 10.0, Timestamp: time.Now().Add(-1 * 24 * time.Hour)},
		{Value: 4.0, Timestamp: time.Now()},
	}
	cv := computeGlucoseCV(readings)
	if cv < 30.0 {
		t.Errorf("CV = %.1f%%, want >= 30%% (high variability)", cv)
	}
}

func TestComputeGlucoseCV_Stable(t *testing.T) {
	readings := []models.TimestampedReading{
		{Value: 6.0, Timestamp: time.Now().Add(-6 * 24 * time.Hour)},
		{Value: 6.1, Timestamp: time.Now().Add(-5 * 24 * time.Hour)},
		{Value: 5.9, Timestamp: time.Now().Add(-4 * 24 * time.Hour)},
		{Value: 6.0, Timestamp: time.Now().Add(-3 * 24 * time.Hour)},
		{Value: 6.2, Timestamp: time.Now().Add(-2 * 24 * time.Hour)},
		{Value: 5.8, Timestamp: time.Now().Add(-1 * 24 * time.Hour)},
		{Value: 6.0, Timestamp: time.Now()},
	}
	cv := computeGlucoseCV(readings)
	if cv > 10.0 {
		t.Errorf("CV = %.1f%%, want < 10%% (stable glucose)", cv)
	}
}
