package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestCATEEstimate_IsActionable_RejectsOverlapFailure(t *testing.T) {
	e := CATEEstimate{
		ID:             uuid.New(),
		PatientID:      "P1",
		InterventionID: "nurse_phone_48h",
		OverlapStatus:  OverlapBelowFloor,
		PointEstimate:  0.15,
		CILower:        0.10,
		CIUpper:        0.20,
	}
	if e.IsActionable() {
		t.Fatal("expected non-actionable when overlap below floor")
	}
}

func TestCATEEstimate_IsActionable_AcceptsPassWithNarrowCI(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: OverlapPass,
		PointEstimate: 0.15,
		CILower:       0.12,
		CIUpper:       0.18,
	}
	if !e.IsActionable() {
		t.Fatal("expected actionable when overlap passes and CI narrow")
	}
}

func TestCATEEstimate_ConfidenceLabel_HighNarrowCI(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: OverlapPass,
		PointEstimate: 0.15, CILower: 0.13, CIUpper: 0.17,
	}
	if got := e.ConfidenceLabel(); got != CATEConfidenceHigh {
		t.Fatalf("want HIGH, got %s", got)
	}
}

func TestCATEEstimate_ConfidenceLabel_LowWideCI(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: OverlapPass,
		PointEstimate: 0.15, CILower: -0.05, CIUpper: 0.35,
	}
	if got := e.ConfidenceLabel(); got != CATEConfidenceLow {
		t.Fatalf("want LOW, got %s", got)
	}
}

func TestCATEEstimate_ConfidenceLabel_MediumBand(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: OverlapPass,
		CILower:       0.00,
		CIUpper:       0.13, // width 0.13 → MEDIUM
	}
	if got := e.ConfidenceLabel(); got != CATEConfidenceMedium {
		t.Fatalf("want MEDIUM, got %s", got)
	}
}

func TestCATEEstimate_ConfidenceLabel_ExactHighBoundary(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: OverlapPass,
		CILower:       0.00,
		CIUpper:       0.06, // width == 0.06 → HIGH (inclusive)
	}
	if got := e.ConfidenceLabel(); got != CATEConfidenceHigh {
		t.Fatalf("want HIGH at boundary 0.06, got %s", got)
	}
}

func TestCATEEstimate_ConfidenceLabel_ExactMediumBoundary(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: OverlapPass,
		CILower:       0.00,
		CIUpper:       0.20, // width == 0.20 → MEDIUM (inclusive)
	}
	if got := e.ConfidenceLabel(); got != CATEConfidenceMedium {
		t.Fatalf("want MEDIUM at boundary 0.20, got %s", got)
	}
}
