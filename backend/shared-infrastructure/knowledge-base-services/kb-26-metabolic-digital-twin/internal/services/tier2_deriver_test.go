package services

import (
	"math"
	"testing"
)

func TestComputeLinearTrend_Increasing(t *testing.T) {
	readings := []TimedValue{
		{Value: 130, DaysSinceFirst: 0},
		{Value: 135, DaysSinceFirst: 7},
		{Value: 140, DaysSinceFirst: 14},
		{Value: 145, DaysSinceFirst: 21},
		{Value: 150, DaysSinceFirst: 28},
	}
	slope := ComputeLinearTrend(readings)
	if slope < 0.5 {
		t.Errorf("expected positive slope for increasing trend, got %f", slope)
	}
}

func TestComputeLinearTrend_Decreasing(t *testing.T) {
	readings := []TimedValue{
		{Value: 150, DaysSinceFirst: 0},
		{Value: 145, DaysSinceFirst: 7},
		{Value: 140, DaysSinceFirst: 14},
	}
	slope := ComputeLinearTrend(readings)
	if slope > -0.5 {
		t.Errorf("expected negative slope for decreasing trend, got %f", slope)
	}
}

func TestComputeLinearTrend_InsufficientData(t *testing.T) {
	readings := []TimedValue{{Value: 130, DaysSinceFirst: 0}}
	slope := ComputeLinearTrend(readings)
	if slope != 0 {
		t.Errorf("expected 0 for insufficient data, got %f", slope)
	}
}

func TestComputeWeightTrendPerMonth(t *testing.T) {
	readings := []TimedValue{
		{Value: 80, DaysSinceFirst: 0},
		{Value: 81, DaysSinceFirst: 15},
		{Value: 82, DaysSinceFirst: 30},
	}
	trend := ComputeWeightTrendPerMonth(readings)
	// ~2 kg over 30 days = ~2 kg/month
	if math.Abs(trend-2.0) > 0.5 {
		t.Errorf("expected ~2 kg/month, got %f", trend)
	}
}

func TestComputeHbA1cTrendPerQuarter(t *testing.T) {
	readings := []TimedValue{
		{Value: 7.0, DaysSinceFirst: 0},
		{Value: 7.5, DaysSinceFirst: 90},
	}
	trend := ComputeHbA1cTrendPerQuarter(readings)
	// 0.5% over 90 days = 0.5%/quarter
	if math.Abs(trend-0.5) > 0.1 {
		t.Errorf("expected ~0.5%%/quarter, got %f", trend)
	}
}

func TestClassifyBPDipping_Dipper(t *testing.T) {
	// 15% dip → DIPPER
	class := ClassifyBPDipping(140, 119)
	if class != "DIPPER" {
		t.Errorf("expected DIPPER, got %s", class)
	}
}

func TestClassifyBPDipping_NonDipper(t *testing.T) {
	// 3% dip → NON_DIPPER
	class := ClassifyBPDipping(140, 135.8)
	if class != "NON_DIPPER" {
		t.Errorf("expected NON_DIPPER, got %s", class)
	}
}

func TestClassifyBPDipping_ReverseDipper(t *testing.T) {
	// Morning > evening → REVERSE_DIPPER
	class := ClassifyBPDipping(130, 135)
	if class != "REVERSE_DIPPER" {
		t.Errorf("expected REVERSE_DIPPER, got %s", class)
	}
}
