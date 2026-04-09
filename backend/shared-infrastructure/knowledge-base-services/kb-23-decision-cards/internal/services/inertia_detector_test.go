package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

func TestDetectInertia_HbA1c_180Days(t *testing.T) {
	lastIntervention := time.Now().AddDate(0, 0, -250)
	input := InertiaDetectorInput{
		PatientID: "patient-hba1c-180",
		Glycaemic: &DomainInertiaInput{
			AtTarget:            false,
			CurrentValue:        8.2,
			TargetValue:         7.0,
			DaysUncontrolled:    180,
			ConsecutiveReadings: 2,
			DataSource:          "HBA1C",
			LastIntervention:    &lastIntervention,
		},
	}

	report := DetectInertia(input)

	if !report.HasAnyInertia {
		t.Fatal("expected HasAnyInertia=true")
	}

	// Find HBA1C_INERTIA pattern.
	var found *models.InertiaVerdict
	for i := range report.Verdicts {
		if report.Verdicts[i].Pattern == models.PatternHbA1cInertia {
			found = &report.Verdicts[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected HBA1C_INERTIA pattern in verdicts")
	}
	if !found.Detected {
		t.Error("expected verdict Detected=true")
	}

	// 180 days ≈ 25.7 weeks → MODERATE (≥26 weeks threshold).
	// Inertia duration = daysSinceIntervention(250) - gracePeriod(42) = 208 days ≈ 29 weeks → MODERATE.
	if found.Severity != models.SeverityModerate {
		t.Errorf("expected severity MODERATE, got %s", found.Severity)
	}
	if found.InertiaDurationDays < 170 {
		t.Errorf("expected InertiaDurationDays >= 170, got %d", found.InertiaDurationDays)
	}
}

func TestDetectInertia_CGM_14Days(t *testing.T) {
	lastIntervention := time.Now().AddDate(0, 0, -90)
	input := InertiaDetectorInput{
		PatientID: "patient-cgm-14",
		Glycaemic: &DomainInertiaInput{
			AtTarget:            false,
			CurrentValue:        35.0,
			TargetValue:         70.0,
			DaysUncontrolled:    21,
			ConsecutiveReadings: 1,
			DataSource:          "CGM_TIR",
			LastIntervention:    &lastIntervention,
		},
	}

	report := DetectInertia(input)

	if !report.HasAnyInertia {
		t.Fatal("expected HasAnyInertia=true")
	}

	var found *models.InertiaVerdict
	for i := range report.Verdicts {
		if report.Verdicts[i].Pattern == models.PatternCGMInertia {
			found = &report.Verdicts[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected CGM_INERTIA pattern in verdicts")
	}
	if !found.Detected {
		t.Error("expected verdict Detected=true")
	}
	if found.Domain != models.DomainGlycaemic {
		t.Errorf("expected domain GLYCAEMIC, got %s", found.Domain)
	}
}

func TestDetectInertia_DualDomain(t *testing.T) {
	glycaemicIntervention := time.Now().AddDate(0, 0, -120)
	hemodynamicIntervention := time.Now().AddDate(0, 0, -100)

	input := InertiaDetectorInput{
		PatientID: "patient-dual",
		Glycaemic: &DomainInertiaInput{
			AtTarget:            false,
			CurrentValue:        8.5,
			TargetValue:         7.0,
			DaysUncontrolled:    90,
			ConsecutiveReadings: 2,
			DataSource:          "HBA1C",
			LastIntervention:    &glycaemicIntervention,
		},
		Hemodynamic: &DomainInertiaInput{
			AtTarget:            false,
			CurrentValue:        155.0,
			TargetValue:         130.0,
			DaysUncontrolled:    60,
			ConsecutiveReadings: 3,
			DataSource:          "HOME_BP",
			LastIntervention:    &hemodynamicIntervention,
		},
	}

	report := DetectInertia(input)

	if !report.HasDualDomainInertia {
		t.Fatal("expected HasDualDomainInertia=true")
	}
	if report.OverallUrgency != UrgencyImmediate {
		t.Errorf("expected OverallUrgency=IMMEDIATE, got %s", report.OverallUrgency)
	}

	// Should have at least 3 verdicts: glycaemic, hemodynamic, dual-domain.
	if len(report.Verdicts) < 3 {
		t.Errorf("expected at least 3 verdicts, got %d", len(report.Verdicts))
	}

	// Verify DUAL_DOMAIN_INERTIA pattern exists.
	hasDualPattern := false
	for _, v := range report.Verdicts {
		if v.Pattern == models.PatternDualDomainInertia {
			hasDualPattern = true
			break
		}
	}
	if !hasDualPattern {
		t.Error("expected DUAL_DOMAIN_INERTIA pattern in verdicts")
	}
}

func TestDetectInertia_AtTarget_NoDetection(t *testing.T) {
	input := InertiaDetectorInput{
		PatientID: "patient-at-target",
		Glycaemic: &DomainInertiaInput{
			AtTarget:     true,
			CurrentValue: 6.5,
			TargetValue:  7.0,
			DataSource:   "HBA1C",
		},
	}

	report := DetectInertia(input)

	if report.HasAnyInertia {
		t.Error("expected HasAnyInertia=false for at-target patient")
	}
	if len(report.Verdicts) != 0 {
		t.Errorf("expected empty verdicts, got %d", len(report.Verdicts))
	}
}
