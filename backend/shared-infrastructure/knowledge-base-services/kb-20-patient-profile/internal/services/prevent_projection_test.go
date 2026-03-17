package services

import "testing"

func TestPREVENTProjection_AssemblesInput(t *testing.T) {
	input := assemblePREVENTInput(
		50,        // age
		SexFemale,
		200,       // total cholesterol
		45,        // HDL
		160,       // SBP
		true,      // on BP treatment
		true,      // on statin
		true,      // diabetes
		false,     // smoking
		90,        // eGFR
		35,        // BMI
		float64Ptr(7.0),  // HbA1c — uses existing float64Ptr from projection_service.go
		float64Ptr(100),  // UACR
		false,     // south asian calibration
		0,         // calibration offset
	)

	if input.ModelVariant != PREVENTModelFull {
		t.Errorf("expected FULL model (both HbA1c and UACR available), got %s", input.ModelVariant)
	}
	if input.TotalCholesterol != 200 {
		t.Errorf("expected TC 200, got %.0f", input.TotalCholesterol)
	}
}

func TestPREVENTProjection_SouthAsianCalibration(t *testing.T) {
	input := assemblePREVENTInput(
		50, SexMale, 200, 45, 140,
		true,  // on BP treatment
		false, // on statin
		true,  // diabetes
		false, // smoking
		90,    // eGFR
		26,    // BMI 26 — in calibration range
		nil, nil,
		true, // south asian calibration enabled
		3.0,  // offset
	)

	if input.BMI != 29.0 {
		t.Errorf("expected calibrated BMI 29.0, got %.1f", input.BMI)
	}
}
