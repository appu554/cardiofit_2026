package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

func TestDetectInertia_HbA1c_180Days(t *testing.T) {
	lastIntervention := time.Now().AddDate(0, 0, -250)
	input := InertiaDetectorInput{
		PatientID: "patient-hba1c-190",
		Glycaemic: &DomainInertiaInput{
			AtTarget:            false,
			CurrentValue:        8.2,
			TargetValue:         7.0,
			DaysUncontrolled:    190,
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

	// 190 days ≈ 27 weeks → MODERATE (≥26 weeks = 182 days threshold).
	// Inertia duration = DaysUncontrolled(190) since daysSinceIntervention(250) > 190.
	if found.Severity != models.SeverityModerate {
		t.Errorf("expected severity MODERATE, got %s", found.Severity)
	}
	if found.InertiaDurationDays < 180 {
		t.Errorf("expected InertiaDurationDays >= 180, got %d", found.InertiaDurationDays)
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

// === PATTERN 5: POST-EVENT INERTIA ===

func TestDetectInertia_PostEvent_NoMedChangeAfterHospitalization(t *testing.T) {
	eventDate := time.Now().AddDate(0, 0, -60) // 60 days ago
	input := InertiaDetectorInput{
		PatientID: "patient-post-event",
		PostEvent: &PostEventInput{
			EventType:      "HOSPITALIZATION_CV",
			EventDate:      eventDate,
			DaysSinceEvent: 60,
			Domain:         models.DomainHemodynamic,
			// No LastIntervention — no med change after event
		},
	}

	report := DetectInertia(input)

	if !report.HasAnyInertia {
		t.Fatal("expected HasAnyInertia=true for post-event inertia")
	}
	var found *models.InertiaVerdict
	for i := range report.Verdicts {
		if report.Verdicts[i].Pattern == models.PatternPostEventInertia {
			found = &report.Verdicts[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected POST_EVENT_INERTIA pattern")
	}
	if !found.Detected {
		t.Error("expected Detected=true")
	}
	// 60 days since event - 28 day response window = 32 days of inertia
	if found.InertiaDurationDays < 30 {
		t.Errorf("expected InertiaDurationDays >= 30, got %d", found.InertiaDurationDays)
	}
}

func TestDetectInertia_PostEvent_MedChangeAfterEvent_NoInertia(t *testing.T) {
	eventDate := time.Now().AddDate(0, 0, -60)
	medChange := time.Now().AddDate(0, 0, -45) // med changed 15 days after event
	input := InertiaDetectorInput{
		PatientID: "patient-post-event-responded",
		PostEvent: &PostEventInput{
			EventType:        "HYPOGLYCAEMIA_SEVERE",
			EventDate:        eventDate,
			DaysSinceEvent:   60,
			Domain:           models.DomainGlycaemic,
			LastIntervention: &medChange,
		},
	}

	report := DetectInertia(input)

	for _, v := range report.Verdicts {
		if v.Pattern == models.PatternPostEventInertia {
			t.Error("should NOT detect post-event inertia when med was changed after event")
		}
	}
}

// === PATTERN 6: RENAL PROGRESSION INERTIA ===

func TestDetectInertia_RenalProgression_NoReview(t *testing.T) {
	transitionDate := time.Now().AddDate(0, 0, -90) // stage transition 90 days ago
	input := InertiaDetectorInput{
		PatientID: "patient-renal-progression",
		RenalProgression: &RenalProgressionInput{
			PreviousCKDStage:    "G3a",
			CurrentCKDStage:     "G3b",
			TransitionDate:      transitionDate,
			DaysSinceTransition: 90,
			// No LastIntervention — no renoprotective med review
		},
	}

	report := DetectInertia(input)

	if !report.HasAnyInertia {
		t.Fatal("expected HasAnyInertia=true for renal progression inertia")
	}
	var found *models.InertiaVerdict
	for i := range report.Verdicts {
		if report.Verdicts[i].Pattern == models.PatternRenalProgressionInertia {
			found = &report.Verdicts[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected RENAL_PROGRESSION_INERTIA pattern")
	}
	if found.Domain != models.DomainRenal {
		t.Errorf("expected RENAL domain, got %s", found.Domain)
	}
	// 90 days - 28 day response window = 62 days
	if found.InertiaDurationDays < 60 {
		t.Errorf("expected InertiaDurationDays >= 60, got %d", found.InertiaDurationDays)
	}
}

func TestDetectInertia_RenalProgression_RecentTransition_NoInertia(t *testing.T) {
	transitionDate := time.Now().AddDate(0, 0, -14) // only 14 days ago
	input := InertiaDetectorInput{
		PatientID: "patient-renal-recent",
		RenalProgression: &RenalProgressionInput{
			PreviousCKDStage:    "G3a",
			CurrentCKDStage:     "G3b",
			TransitionDate:      transitionDate,
			DaysSinceTransition: 14,
		},
	}

	report := DetectInertia(input)

	for _, v := range report.Verdicts {
		if v.Pattern == models.PatternRenalProgressionInertia {
			t.Error("should NOT detect renal progression inertia within 28-day response window")
		}
	}
}

// === PATTERN 7: INTENSIFICATION CEILING ===

func TestDetectInertia_Ceiling_AtMaxDoseTargetUnmet(t *testing.T) {
	input := InertiaDetectorInput{
		PatientID: "patient-ceiling",
		Ceiling: &CeilingInput{
			Domain:        models.DomainGlycaemic,
			AtMaxDose:     true,
			CurrentMeds:   []string{"METFORMIN"},
			TargetUnmet:   true,
			DaysAtMaxDose: 100, // >84 days (12 weeks)
			NextStepClass: "SGLT2i",
		},
	}

	report := DetectInertia(input)

	if !report.HasAnyInertia {
		t.Fatal("expected HasAnyInertia=true for ceiling inertia")
	}
	var found *models.InertiaVerdict
	for i := range report.Verdicts {
		if report.Verdicts[i].Pattern == models.PatternIntensificationCeiling {
			found = &report.Verdicts[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected INTENSIFICATION_CEILING pattern")
	}
	if !found.AtMaxDose {
		t.Error("expected AtMaxDose=true")
	}
	if found.NextStepInPathway != "SGLT2i" {
		t.Errorf("expected NextStepInPathway=SGLT2i, got %s", found.NextStepInPathway)
	}
	if found.Domain != models.DomainGlycaemic {
		t.Errorf("expected GLYCAEMIC domain, got %s", found.Domain)
	}
}

func TestDetectInertia_Ceiling_NotAtMaxDose_NoInertia(t *testing.T) {
	input := InertiaDetectorInput{
		PatientID: "patient-not-at-max",
		Ceiling: &CeilingInput{
			Domain:        models.DomainGlycaemic,
			AtMaxDose:     false, // not at max
			TargetUnmet:   true,
			DaysAtMaxDose: 100,
			NextStepClass: "SGLT2i",
		},
	}

	report := DetectInertia(input)

	for _, v := range report.Verdicts {
		if v.Pattern == models.PatternIntensificationCeiling {
			t.Error("should NOT detect ceiling inertia when not at max dose")
		}
	}
}

// === EXISTING TEST: AT TARGET ===

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
