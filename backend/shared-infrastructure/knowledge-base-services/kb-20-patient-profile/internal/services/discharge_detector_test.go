package services

import (
	"testing"
	"time"

	"kb-patient-profile/internal/models"
)

func TestDischarge_ValidFHIR_CreatesTransition(t *testing.T) {
	det := NewDischargeDetector()

	input := DischargeInput{
		PatientID:        "91-1001-2001-3001",
		DischargeDate:    time.Now().Add(-2 * 24 * time.Hour), // 2 days ago
		Source:           models.SourceFHIREncounter,
		FacilityName:     "Royal Melbourne Hospital",
		FacilityType:     "ACUTE_HOSPITAL",
		PrimaryDiagnosis: "Acute decompensated heart failure",
		LengthOfStayDays: 5,
		Disposition:      "HOME",
	}

	ct, err := det.DetectDischarge(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ct == nil {
		t.Fatal("expected CareTransition, got nil")
	}
	if ct.SourceConfidence != "HIGH" {
		t.Errorf("expected SourceConfidence HIGH, got %s", ct.SourceConfidence)
	}
	if ct.TransitionState != string(models.TransitionActive) {
		t.Errorf("expected TransitionState ACTIVE, got %s", ct.TransitionState)
	}
	if !ct.HeightenedSurveillanceActive {
		t.Error("expected HeightenedSurveillanceActive true")
	}
	if ct.WindowDays != 30 {
		t.Errorf("expected WindowDays 30, got %d", ct.WindowDays)
	}
	if ct.PatientID != input.PatientID {
		t.Errorf("expected PatientID %s, got %s", input.PatientID, ct.PatientID)
	}
	if ct.ReconciliationStatus != string(models.ReconciliationPending) {
		t.Errorf("expected ReconciliationStatus PENDING, got %s", ct.ReconciliationStatus)
	}
}

func TestDischarge_ManualEntry_CreatesTransition(t *testing.T) {
	det := NewDischargeDetector()

	input := DischargeInput{
		PatientID:     "91-2002-3003-4004",
		DischargeDate: time.Now().Add(-1 * 24 * time.Hour), // 1 day ago
		Source:        models.SourceManual,
		FacilityName:  "St Vincent's Hospital",
		FacilityType:  "ACUTE_HOSPITAL",
	}

	ct, err := det.DetectDischarge(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ct == nil {
		t.Fatal("expected CareTransition, got nil")
	}
	if ct.SourceConfidence != "HIGH" {
		t.Errorf("expected SourceConfidence HIGH for MANUAL source, got %s", ct.SourceConfidence)
	}
	if ct.FacilityName != "St Vincent's Hospital" {
		t.Errorf("expected FacilityName 'St Vincent's Hospital', got %s", ct.FacilityName)
	}
}

func TestDischarge_PatientReported_LowConfidence(t *testing.T) {
	det := NewDischargeDetector()

	input := DischargeInput{
		PatientID:     "91-3003-4004-5005",
		DischargeDate: time.Now().Add(-3 * 24 * time.Hour),
		Source:        models.SourcePatientReported,
		FacilityName:  "Local Clinic",
		FacilityType:  "ACUTE_HOSPITAL",
	}

	ct, err := det.DetectDischarge(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ct.SourceConfidence != "LOW" {
		t.Errorf("expected SourceConfidence LOW for PATIENT_REPORTED source, got %s", ct.SourceConfidence)
	}
}

func TestDischarge_TooOld_Rejected(t *testing.T) {
	det := NewDischargeDetector()

	input := DischargeInput{
		PatientID:     "91-4004-5005-6006",
		DischargeDate: time.Now().Add(-20 * 24 * time.Hour), // 20 days ago
		Source:        models.SourceFHIREncounter,
		FacilityName:  "Alfred Hospital",
		FacilityType:  "ACUTE_HOSPITAL",
	}

	ct, err := det.DetectDischarge(input)
	if ct != nil {
		t.Error("expected nil CareTransition for too-old discharge")
	}
	if err == nil {
		t.Fatal("expected error for too-old discharge, got nil")
	}
	if !containsSubstring(err.Error(), "discharge too old") {
		t.Errorf("expected error containing 'discharge too old', got: %s", err.Error())
	}
}

func TestDischarge_Duplicate_Detected(t *testing.T) {
	det := NewDischargeDetector()

	existing := &models.CareTransition{
		PatientID:     "91-5005-6006-7007",
		DischargeDate: time.Now().Add(-2 * 24 * time.Hour),
	}

	// New input within 24h of existing discharge
	newInput := DischargeInput{
		PatientID:     "91-5005-6006-7007",
		DischargeDate: existing.DischargeDate.Add(12 * time.Hour),
		Source:        models.SourceManual,
	}

	if !det.IsDuplicate(existing, newInput) {
		t.Error("expected IsDuplicate=true for same patient within 24h")
	}

	// Different patient should not be duplicate
	differentPatient := DischargeInput{
		PatientID:     "91-9999-0000-1111",
		DischargeDate: existing.DischargeDate.Add(1 * time.Hour),
		Source:        models.SourceManual,
	}
	if det.IsDuplicate(existing, differentPatient) {
		t.Error("expected IsDuplicate=false for different patient")
	}

	// Same patient but >24h apart should not be duplicate
	farApart := DischargeInput{
		PatientID:     "91-5005-6006-7007",
		DischargeDate: existing.DischargeDate.Add(48 * time.Hour),
		Source:        models.SourceManual,
	}
	if det.IsDuplicate(existing, farApart) {
		t.Error("expected IsDuplicate=false for discharge >24h apart")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
