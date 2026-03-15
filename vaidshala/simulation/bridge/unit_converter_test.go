package bridge

import (
	"math"
	"testing"
)

func TestGlucoseConversion_PassThrough(t *testing.T) {
	input := 5.5
	if got := GlucoseToProduction(input); got != input {
		t.Errorf("GlucoseToProduction(%v) = %v, want %v", input, got, input)
	}
	if got := GlucoseToSimulation(input); got != input {
		t.Errorf("GlucoseToSimulation(%v) = %v, want %v", input, got, input)
	}
}

func TestCreatinineConversion_PassThrough(t *testing.T) {
	input := 90.0
	if got := CreatinineToProduction(input); got != input {
		t.Errorf("CreatinineToProduction(%v) = %v, want %v", input, got, input)
	}
	if got := CreatinineToSimulation(input); got != input {
		t.Errorf("CreatinineToSimulation(%v) = %v, want %v", input, got, input)
	}
}

func TestConversionConstants_Documented(t *testing.T) {
	mgdl := 180.0
	mmol := mgdl / GlucoseMgDLToMmolL
	if math.Abs(mmol-10.0) > 0.01 {
		t.Errorf("180 mg/dL should be ~10.0 mmol/L, got %v", mmol)
	}

	crMgDL := 1.0
	crUmol := crMgDL * CreatinineMgDLToUmolL
	if math.Abs(crUmol-88.4) > 0.1 {
		t.Errorf("1.0 mg/dL creatinine should be ~88.4 µmol/L, got %v", crUmol)
	}
}
