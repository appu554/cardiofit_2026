package unit

import (
	"math"
	"testing"

	"kb-26-metabolic-digital-twin/internal/services"
)

const epsilon = 1e-6

func TestComputeMAP(t *testing.T) {
	// MAP = DBP + (SBP - DBP) / 3
	// SBP=120, DBP=80 => 80 + 40/3 = 93.333...
	result := services.ComputeMAP(120, 80)
	expected := 80 + 40.0/3.0
	if math.Abs(result-expected) > epsilon {
		t.Errorf("ComputeMAP(120, 80) = %f, want %f", result, expected)
	}

	// SBP=140, DBP=90 => 90 + 50/3 = 106.666...
	result2 := services.ComputeMAP(140, 90)
	expected2 := 90 + 50.0/3.0
	if math.Abs(result2-expected2) > epsilon {
		t.Errorf("ComputeMAP(140, 90) = %f, want %f", result2, expected2)
	}
}

func TestComputeVisceralFatProxy(t *testing.T) {
	// Midpoint waist=100, height=170, TG=150, HDL=50
	result := services.ComputeVisceralFatProxy(100, 170, 150, 50)
	if result < 0 || result > 1 {
		t.Errorf("ComputeVisceralFatProxy result %f out of [0,1]", result)
	}

	// Minimum values should give 0
	resultMin := services.ComputeVisceralFatProxy(60, 170, 0.5*50, 50)
	if resultMin < 0 {
		t.Errorf("ComputeVisceralFatProxy min case = %f, want >= 0", resultMin)
	}

	// HDL=0 should return 0
	resultZero := services.ComputeVisceralFatProxy(100, 170, 150, 0)
	if resultZero != 0 {
		t.Errorf("ComputeVisceralFatProxy with HDL=0 = %f, want 0", resultZero)
	}

	// Height=0 should return 0
	resultZeroH := services.ComputeVisceralFatProxy(100, 0, 150, 50)
	if resultZeroH != 0 {
		t.Errorf("ComputeVisceralFatProxy with height=0 = %f, want 0", resultZeroH)
	}
}

func TestComputeGlycemicVariability(t *testing.T) {
	// Constant readings => CV = 0
	constant := []float64{100, 100, 100, 100}
	cv := services.ComputeGlycemicVariability(constant)
	if cv != 0 {
		t.Errorf("ComputeGlycemicVariability(constant) = %f, want 0", cv)
	}

	// Known values: [100, 200] => mean=150, sample SD=sqrt(5000)=70.71, CV%=47.14...
	readings := []float64{100, 200}
	cv2 := services.ComputeGlycemicVariability(readings)
	sampleSD := math.Sqrt(5000.0) // sqrt(((100-150)^2 + (200-150)^2) / (2-1))
	expected := (sampleSD / 150.0) * 100.0
	if math.Abs(cv2-expected) > 0.01 {
		t.Errorf("ComputeGlycemicVariability([100,200]) = %f, want ~%f", cv2, expected)
	}

	// Single reading => 0
	single := []float64{100}
	cv3 := services.ComputeGlycemicVariability(single)
	if cv3 != 0 {
		t.Errorf("ComputeGlycemicVariability(single) = %f, want 0", cv3)
	}
}

func TestComputeDawnPhenomenon(t *testing.T) {
	// Positive case: 4/5 FBG > PPBG, 4/5 FBG > 130
	fbgs := []float64{140, 135, 145, 125, 138}
	ppbgs := []float64{110, 120, 100, 130, 115}
	if !services.ComputeDawnPhenomenon(fbgs, ppbgs) {
		t.Error("ComputeDawnPhenomenon should be true for classic dawn pattern")
	}

	// Negative case: FBG < PPBG
	fbgsLow := []float64{100, 95, 105, 98, 102}
	ppbgsHigh := []float64{120, 130, 125, 115, 128}
	if services.ComputeDawnPhenomenon(fbgsLow, ppbgsHigh) {
		t.Error("ComputeDawnPhenomenon should be false when FBG < PPBG")
	}

	// Negative case: FBG > PPBG but FBG < 130
	fbgsMild := []float64{120, 125, 128, 122, 119}
	ppbgsMild := []float64{110, 115, 118, 112, 109}
	if services.ComputeDawnPhenomenon(fbgsMild, ppbgsMild) {
		t.Error("ComputeDawnPhenomenon should be false when FBG < 130")
	}

	// Too few readings
	if services.ComputeDawnPhenomenon([]float64{140, 135}, []float64{110, 120}) {
		t.Error("ComputeDawnPhenomenon should be false with <3 readings")
	}
}

func TestComputeTrigHDLRatio(t *testing.T) {
	result := services.ComputeTrigHDLRatio(150, 50)
	if math.Abs(result-3.0) > epsilon {
		t.Errorf("ComputeTrigHDLRatio(150, 50) = %f, want 3.0", result)
	}

	result2 := services.ComputeTrigHDLRatio(200, 40)
	if math.Abs(result2-5.0) > epsilon {
		t.Errorf("ComputeTrigHDLRatio(200, 40) = %f, want 5.0", result2)
	}

	// HDL=0 => 0
	result3 := services.ComputeTrigHDLRatio(150, 0)
	if result3 != 0 {
		t.Errorf("ComputeTrigHDLRatio(150, 0) = %f, want 0", result3)
	}
}

func TestComputeRenalSlope(t *testing.T) {
	// Perfect linear decline: eGFR drops from 90 to 80 over 365.25 days = -10 mL/min/1.73m²/yr
	readings := []services.TimedValue{
		{Value: 90, DaysSinceFirst: 0},
		{Value: 80, DaysSinceFirst: 365.25},
	}
	slope := services.ComputeRenalSlope(readings)
	if math.Abs(slope-(-10.0)) > 0.01 {
		t.Errorf("ComputeRenalSlope(linear decline) = %f, want -10.0", slope)
	}

	// Stable eGFR over 2 years
	stable := []services.TimedValue{
		{Value: 60, DaysSinceFirst: 0},
		{Value: 60, DaysSinceFirst: 365.25},
		{Value: 60, DaysSinceFirst: 730.5},
	}
	slopeStable := services.ComputeRenalSlope(stable)
	if math.Abs(slopeStable) > 0.01 {
		t.Errorf("ComputeRenalSlope(stable) = %f, want ~0", slopeStable)
	}

	// Fewer than 2 readings => 0
	single := []services.TimedValue{{Value: 90, DaysSinceFirst: 0}}
	slopeSingle := services.ComputeRenalSlope(single)
	if slopeSingle != 0 {
		t.Errorf("ComputeRenalSlope(single) = %f, want 0", slopeSingle)
	}

	// Rapid decline: 90 -> 60 in 2 years = -15/yr
	rapid := []services.TimedValue{
		{Value: 90, DaysSinceFirst: 0},
		{Value: 75, DaysSinceFirst: 365.25},
		{Value: 60, DaysSinceFirst: 730.5},
	}
	slopeRapid := services.ComputeRenalSlope(rapid)
	if math.Abs(slopeRapid-(-15.0)) > 0.01 {
		t.Errorf("ComputeRenalSlope(rapid decline) = %f, want -15.0", slopeRapid)
	}
}
