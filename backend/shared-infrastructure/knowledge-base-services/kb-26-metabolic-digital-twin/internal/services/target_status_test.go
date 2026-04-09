package services

import (
	"testing"
	"time"
)

func floatPtr(f float64) *float64  { return &f }
func timePtr(t time.Time) *time.Time { return &t }

func TestGlycaemicTargetStatus_Uncontrolled_HbA1c(t *testing.T) {
	now := time.Now()
	prevDate := now.Add(-120 * 24 * time.Hour)
	curDate := now.Add(-30 * 24 * time.Hour)

	result := ComputeGlycaemicTargetStatus(TargetStatusInput{
		HbA1c:         floatPtr(8.2),
		PrevHbA1c:     floatPtr(7.8),
		HbA1cDate:     timePtr(curDate),
		PrevHbA1cDate: timePtr(prevDate),
		HbA1cTarget:   7.0,
	})

	if result.AtTarget {
		t.Error("expected AtTarget=false for HbA1c 8.2 vs target 7.0")
	}
	if result.DataSource != "HBA1C" {
		t.Errorf("expected DataSource HBA1C, got %s", result.DataSource)
	}
	if result.Confidence != "MODERATE" {
		t.Errorf("expected MODERATE confidence, got %s", result.Confidence)
	}
	if result.ConsecutiveReadings != 2 {
		t.Errorf("expected 2 consecutive readings, got %d", result.ConsecutiveReadings)
	}
	if result.DaysUncontrolled < 100 {
		t.Errorf("expected ≥100 days uncontrolled, got %d", result.DaysUncontrolled)
	}
}

func TestGlycaemicTargetStatus_Uncontrolled_CGM(t *testing.T) {
	reportDate := time.Now().Add(-7 * 24 * time.Hour)

	result := ComputeGlycaemicTargetStatus(TargetStatusInput{
		HbA1c:             floatPtr(7.5),
		HbA1cTarget:       7.0,
		CGMAvailable:      true,
		CGMSufficientData: true,
		CGMTIR:            floatPtr(38),
		CGMReportDate:     timePtr(reportDate),
		TIRTarget:         70,
	})

	if result.AtTarget {
		t.Error("expected AtTarget=false for TIR 38 vs target 70")
	}
	if result.DataSource != "CGM_TIR" {
		t.Errorf("expected DataSource CGM_TIR, got %s", result.DataSource)
	}
	if result.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence, got %s", result.Confidence)
	}
	if result.CurrentValue != 38 {
		t.Errorf("expected CurrentValue 38, got %.1f", result.CurrentValue)
	}
}

func TestGlycaemicTargetStatus_Controlled(t *testing.T) {
	curDate := time.Now().Add(-14 * 24 * time.Hour)

	result := ComputeGlycaemicTargetStatus(TargetStatusInput{
		HbA1c:       floatPtr(6.5),
		HbA1cDate:   timePtr(curDate),
		HbA1cTarget: 7.0,
	})

	if !result.AtTarget {
		t.Error("expected AtTarget=true for HbA1c 6.5 vs target 7.0")
	}
	if result.ConsecutiveReadings != 0 {
		t.Errorf("expected 0 consecutive uncontrolled readings, got %d", result.ConsecutiveReadings)
	}
}

func TestHemodynamicTargetStatus_Uncontrolled(t *testing.T) {
	result := ComputeHemodynamicTargetStatus(BPTargetStatusInput{
		MeanSBP7d: floatPtr(155),
		SBPTarget: 130,
	})

	if result.AtTarget {
		t.Error("expected AtTarget=false for SBP 155 vs target 130")
	}
	if result.Domain != "HEMODYNAMIC" {
		t.Errorf("expected HEMODYNAMIC domain, got %s", result.Domain)
	}
	if result.DataSource != "HOME_BP" {
		t.Errorf("expected HOME_BP data source, got %s", result.DataSource)
	}
	if result.CurrentValue != 155 {
		t.Errorf("expected CurrentValue 155, got %.1f", result.CurrentValue)
	}
}
