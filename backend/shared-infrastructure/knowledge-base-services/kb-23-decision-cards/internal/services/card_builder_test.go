package services

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// TestDetermineSafetyTier
// ---------------------------------------------------------------------------

func TestDetermineSafetyTier(t *testing.T) {
	log, _ := zap.NewDevelopment()
	builder := NewCardBuilder(nil, nil, nil, nil, nil, log, nil, nil, nil, nil)

	tests := []struct {
		name     string
		flags    []models.SafetyFlagEntry
		expected models.SafetyTier
	}{
		{
			name:     "no flags returns ROUTINE",
			flags:    nil,
			expected: models.SafetyRoutine,
		},
		{
			name:     "empty flags returns ROUTINE",
			flags:    []models.SafetyFlagEntry{},
			expected: models.SafetyRoutine,
		},
		{
			name: "single ROUTINE flag returns ROUTINE",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "ROUTINE"},
			},
			expected: models.SafetyRoutine,
		},
		{
			name: "single URGENT flag returns URGENT",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "URGENT"},
			},
			expected: models.SafetyUrgent,
		},
		{
			name: "single IMMEDIATE flag returns IMMEDIATE",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "IMMEDIATE"},
			},
			expected: models.SafetyImmediate,
		},
		{
			name: "IMMEDIATE takes precedence over URGENT",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "URGENT"},
				{FlagID: "F2", Severity: "IMMEDIATE"},
			},
			expected: models.SafetyImmediate,
		},
		{
			name: "URGENT takes precedence over ROUTINE",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "ROUTINE"},
				{FlagID: "F2", Severity: "URGENT"},
			},
			expected: models.SafetyUrgent,
		},
		{
			name: "IMMEDIATE takes precedence over all others",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "ROUTINE"},
				{FlagID: "F2", Severity: "URGENT"},
				{FlagID: "F3", Severity: "IMMEDIATE"},
			},
			expected: models.SafetyImmediate,
		},
		{
			name: "multiple ROUTINE flags still return ROUTINE",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "ROUTINE"},
				{FlagID: "F2", Severity: "ROUTINE"},
			},
			expected: models.SafetyRoutine,
		},
		{
			name: "unknown severity treated as ROUTINE",
			flags: []models.SafetyFlagEntry{
				{FlagID: "F1", Severity: "LOW"},
			},
			expected: models.SafetyRoutine,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := builder.determineSafetyTier(tc.flags)
			if got != tc.expected {
				t.Errorf("determineSafetyTier(%v) = %q, want %q", tc.flags, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestShouldSuggestChronotherapy
// ---------------------------------------------------------------------------

func TestShouldSuggestChronotherapy(t *testing.T) {
	tests := []struct {
		name          string
		bpPattern     string
		currentTiming DoseTiming
		expected      bool
	}{
		{
			name:          "morning surge with morning dosing triggers chronotherapy",
			bpPattern:     "MORNING_SURGE",
			currentTiming: DoseTimingMorning,
			expected:      true,
		},
		{
			name:          "morning surge with bedtime dosing does not trigger",
			bpPattern:     "MORNING_SURGE",
			currentTiming: DoseTimingBedtime,
			expected:      false,
		},
		{
			name:          "non-surge pattern with morning dosing does not trigger",
			bpPattern:     "NORMAL",
			currentTiming: DoseTimingMorning,
			expected:      false,
		},
		{
			name:          "non-surge pattern with bedtime dosing does not trigger",
			bpPattern:     "NORMAL",
			currentTiming: DoseTimingBedtime,
			expected:      false,
		},
		{
			name:          "empty pattern does not trigger",
			bpPattern:     "",
			currentTiming: DoseTimingMorning,
			expected:      false,
		},
		{
			name:          "dipper pattern with morning dosing does not trigger",
			bpPattern:     "DIPPER",
			currentTiming: DoseTimingMorning,
			expected:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldSuggestChronotherapy(tc.bpPattern, tc.currentTiming)
			if got != tc.expected {
				t.Errorf("ShouldSuggestChronotherapy(%q, %q) = %v, want %v",
					tc.bpPattern, tc.currentTiming, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAddVariabilityNote
// ---------------------------------------------------------------------------

func TestAddVariabilityNote(t *testing.T) {
	t.Run("HIGH variability appends note to all summaries", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base clinician",
			PatientSummaryEn: "base en",
			PatientSummaryHi: "base hi",
		}
		AddVariabilityNote(card, "HIGH", 18.5)

		if card.ClinicianSummary == "base clinician" {
			t.Error("expected clinician summary to be modified for HIGH variability")
		}
		if card.PatientSummaryEn == "base en" {
			t.Error("expected patient summary EN to be modified for HIGH variability")
		}
		if card.PatientSummaryHi == "base hi" {
			t.Error("expected patient summary HI to be modified for HIGH variability")
		}
		// Verify the SD value is embedded
		if !containsSubstring(card.ClinicianSummary, "18.5") {
			t.Error("expected SD value 18.5 in clinician summary")
		}
		if !containsSubstring(card.ClinicianSummary, "[BP Variability]") {
			t.Error("expected [BP Variability] tag in clinician summary")
		}
	})

	t.Run("NORMAL variability does not modify card", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base clinician",
			PatientSummaryEn: "base en",
			PatientSummaryHi: "base hi",
		}
		AddVariabilityNote(card, "NORMAL", 10.0)

		if card.ClinicianSummary != "base clinician" {
			t.Error("expected clinician summary unchanged for NORMAL variability")
		}
		if card.PatientSummaryEn != "base en" {
			t.Error("expected patient summary EN unchanged for NORMAL variability")
		}
	})

	t.Run("LOW variability does not modify card", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "original",
		}
		AddVariabilityNote(card, "LOW", 5.0)

		if card.ClinicianSummary != "original" {
			t.Error("expected clinician summary unchanged for LOW variability")
		}
	})

	t.Run("empty variability status does not modify card", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "original",
		}
		AddVariabilityNote(card, "", 12.0)

		if card.ClinicianSummary != "original" {
			t.Error("expected clinician summary unchanged for empty status")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAddPulsePressureNote
// ---------------------------------------------------------------------------

func TestAddPulsePressureNote(t *testing.T) {
	t.Run("PP above 60 appends arterial stiffness warning", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base",
			PatientSummaryEn: "base en",
			PatientSummaryHi: "base hi",
		}
		AddPulsePressureNote(card, 72.0, "INCREASING")

		if card.ClinicianSummary == "base" {
			t.Error("expected clinician summary to be modified for PP > 60")
		}
		if !containsSubstring(card.ClinicianSummary, "[Pulse Pressure]") {
			t.Error("expected [Pulse Pressure] tag in clinician summary")
		}
		if !containsSubstring(card.ClinicianSummary, "72") {
			t.Error("expected PP value 72 in clinician summary")
		}
		if !containsSubstring(card.ClinicianSummary, "INCREASING") {
			t.Error("expected trend INCREASING in clinician summary")
		}
		if !containsSubstring(card.PatientSummaryEn, "72") {
			t.Error("expected PP value in patient summary EN")
		}
	})

	t.Run("PP exactly 60 does not modify card", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base",
			PatientSummaryEn: "base en",
		}
		AddPulsePressureNote(card, 60.0, "STABLE")

		if card.ClinicianSummary != "base" {
			t.Error("expected clinician summary unchanged for PP == 60")
		}
	})

	t.Run("PP below 60 does not modify card", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base",
		}
		AddPulsePressureNote(card, 45.0, "STABLE")

		if card.ClinicianSummary != "base" {
			t.Error("expected clinician summary unchanged for PP < 60")
		}
	})

	t.Run("PP at boundary 60.1 triggers note", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base",
		}
		AddPulsePressureNote(card, 60.1, "STABLE")

		if card.ClinicianSummary == "base" {
			t.Error("expected clinician summary to be modified for PP = 60.1")
		}
	})
}

// ---------------------------------------------------------------------------
// TestShouldPrioritizeDietaryIntervention
// ---------------------------------------------------------------------------

func TestShouldPrioritizeDietaryIntervention(t *testing.T) {
	tests := []struct {
		name               string
		sodiumEstimate     string
		reductionPotential float64
		expected           bool
	}{
		{
			name:               "HIGH sodium with reduction potential >= 0.6 returns true",
			sodiumEstimate:     "HIGH",
			reductionPotential: 0.6,
			expected:           true,
		},
		{
			name:               "HIGH sodium with reduction potential > 0.6 returns true",
			sodiumEstimate:     "HIGH",
			reductionPotential: 0.8,
			expected:           true,
		},
		{
			name:               "HIGH sodium with reduction potential < 0.6 returns false",
			sodiumEstimate:     "HIGH",
			reductionPotential: 0.5,
			expected:           false,
		},
		{
			name:               "NORMAL sodium with high reduction potential returns false",
			sodiumEstimate:     "NORMAL",
			reductionPotential: 0.9,
			expected:           false,
		},
		{
			name:               "LOW sodium returns false regardless of potential",
			sodiumEstimate:     "LOW",
			reductionPotential: 1.0,
			expected:           false,
		},
		{
			name:               "empty sodium returns false",
			sodiumEstimate:     "",
			reductionPotential: 0.8,
			expected:           false,
		},
		{
			name:               "HIGH sodium with exactly 0.59 returns false",
			sodiumEstimate:     "HIGH",
			reductionPotential: 0.59,
			expected:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldPrioritizeDietaryIntervention(tc.sodiumEstimate, tc.reductionPotential)
			if got != tc.expected {
				t.Errorf("ShouldPrioritizeDietaryIntervention(%q, %f) = %v, want %v",
					tc.sodiumEstimate, tc.reductionPotential, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAddThiazideKPlusCausalContext
// ---------------------------------------------------------------------------

func TestAddThiazideKPlusCausalContext(t *testing.T) {
	ptrFloat := func(v float64) *float64 { return &v }

	t.Run("thiazide active and K+ below 3.5 appends note", func(t *testing.T) {
		card := &models.DecisionCard{
			ClinicianSummary: "base",
			PatientSummaryEn: "base en",
			PatientSummaryHi: "base hi",
		}
		AddThiazideKPlusCausalContext(card, true, ptrFloat(3.2))

		if card.ClinicianSummary == "base" {
			t.Error("expected clinician summary to be modified for low K+ with thiazide")
		}
		if !containsSubstring(card.ClinicianSummary, "[Thiazide K+]") {
			t.Error("expected [Thiazide K+] tag in clinician summary")
		}
		if !containsSubstring(card.ClinicianSummary, "3.2") {
			t.Error("expected K+ value 3.2 in clinician summary")
		}
	})

	t.Run("thiazide active and K+ exactly 3.5 does not trigger", func(t *testing.T) {
		card := &models.DecisionCard{ClinicianSummary: "base"}
		AddThiazideKPlusCausalContext(card, true, ptrFloat(3.5))

		if card.ClinicianSummary != "base" {
			t.Error("expected clinician summary unchanged for K+ == 3.5")
		}
	})

	t.Run("thiazide active and K+ above 3.5 does not trigger", func(t *testing.T) {
		card := &models.DecisionCard{ClinicianSummary: "base"}
		AddThiazideKPlusCausalContext(card, true, ptrFloat(4.0))

		if card.ClinicianSummary != "base" {
			t.Error("expected clinician summary unchanged for K+ > 3.5")
		}
	})

	t.Run("thiazide not active does not trigger even with low K+", func(t *testing.T) {
		card := &models.DecisionCard{ClinicianSummary: "base"}
		AddThiazideKPlusCausalContext(card, false, ptrFloat(2.8))

		if card.ClinicianSummary != "base" {
			t.Error("expected clinician summary unchanged when thiazide inactive")
		}
	})

	t.Run("nil potassium level does not trigger", func(t *testing.T) {
		card := &models.DecisionCard{ClinicianSummary: "base"}
		AddThiazideKPlusCausalContext(card, true, nil)

		if card.ClinicianSummary != "base" {
			t.Error("expected clinician summary unchanged for nil potassium")
		}
	})

	t.Run("borderline K+ 3.49 triggers note", func(t *testing.T) {
		card := &models.DecisionCard{ClinicianSummary: "base"}
		AddThiazideKPlusCausalContext(card, true, ptrFloat(3.49))

		if card.ClinicianSummary == "base" {
			t.Error("expected clinician summary to be modified for K+ = 3.49")
		}
	})
}

// ---------------------------------------------------------------------------
// TestGetReEscalationSpec
// ---------------------------------------------------------------------------

func TestGetReEscalationSpec(t *testing.T) {
	t.Run("THIAZIDE dose reduction failure restores full dose", func(t *testing.T) {
		spec := GetReEscalationSpec("THIAZIDE", "DOSE_REDUCTION")
		if spec.DrugClass != "THIAZIDE" {
			t.Errorf("expected DrugClass THIAZIDE, got %s", spec.DrugClass)
		}
		if spec.RestartDose != "Restore full dose" {
			t.Errorf("expected 'Restore full dose', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 26 {
			t.Errorf("expected ReassessWeeks 26, got %d", spec.ReassessWeeks)
		}
	})

	t.Run("THIAZIDE removal failure restarts at 12.5 mg", func(t *testing.T) {
		spec := GetReEscalationSpec("THIAZIDE", "REMOVAL")
		if spec.RestartDose != "Restart at 12.5 mg (not full dose)" {
			t.Errorf("expected 'Restart at 12.5 mg (not full dose)', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 13 {
			t.Errorf("expected ReassessWeeks 13, got %d", spec.ReassessWeeks)
		}
	})

	t.Run("CCB dose reduction failure restores full dose", func(t *testing.T) {
		spec := GetReEscalationSpec("CCB", "DOSE_REDUCTION")
		if spec.RestartDose != "Restore full dose" {
			t.Errorf("expected 'Restore full dose', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 26 {
			t.Errorf("expected ReassessWeeks 26, got %d", spec.ReassessWeeks)
		}
	})

	t.Run("CCB removal failure restarts at lowest effective dose", func(t *testing.T) {
		spec := GetReEscalationSpec("CCB", "REMOVAL")
		if spec.RestartDose != "Restart at lowest effective dose" {
			t.Errorf("expected 'Restart at lowest effective dose', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 13 {
			t.Errorf("expected ReassessWeeks 13, got %d", spec.ReassessWeeks)
		}
	})

	t.Run("BETA_BLOCKER always restarts at half-dose regardless of phase", func(t *testing.T) {
		for _, phase := range []string{"DOSE_REDUCTION", "REMOVAL"} {
			spec := GetReEscalationSpec("BETA_BLOCKER", phase)
			if spec.RestartDose != "Restart at half-dose, taper up" {
				t.Errorf("BETA_BLOCKER phase=%s: expected 'Restart at half-dose, taper up', got %q", phase, spec.RestartDose)
			}
			if spec.ReassessWeeks != 26 {
				t.Errorf("BETA_BLOCKER phase=%s: expected ReassessWeeks 26, got %d", phase, spec.ReassessWeeks)
			}
			if !containsSubstring(spec.CardNote, "Rebound tachycardia") {
				t.Errorf("BETA_BLOCKER phase=%s: expected rebound tachycardia warning in CardNote", phase)
			}
		}
	})

	t.Run("ACE_INHIBITOR restores full RAAS dose with 6-week recheck", func(t *testing.T) {
		spec := GetReEscalationSpec("ACE_INHIBITOR", "DOSE_REDUCTION")
		if spec.RestartDose != "Restore full RAAS dose" {
			t.Errorf("expected 'Restore full RAAS dose', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 6 {
			t.Errorf("expected ReassessWeeks 6, got %d", spec.ReassessWeeks)
		}
		if !containsSubstring(spec.CardNote, "ACR") {
			t.Error("expected ACR mention in ACE_INHIBITOR CardNote")
		}
	})

	t.Run("ARB restores full RAAS dose same as ACE_INHIBITOR", func(t *testing.T) {
		spec := GetReEscalationSpec("ARB", "REMOVAL")
		if spec.RestartDose != "Restore full RAAS dose" {
			t.Errorf("expected 'Restore full RAAS dose', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 6 {
			t.Errorf("expected ReassessWeeks 6, got %d", spec.ReassessWeeks)
		}
	})

	t.Run("unknown drug class returns generic fallback", func(t *testing.T) {
		spec := GetReEscalationSpec("STATIN", "DOSE_REDUCTION")
		if spec.DrugClass != "STATIN" {
			t.Errorf("expected DrugClass STATIN, got %s", spec.DrugClass)
		}
		if spec.RestartDose != "Restore previous dose" {
			t.Errorf("expected 'Restore previous dose', got %q", spec.RestartDose)
		}
		if spec.ReassessWeeks != 26 {
			t.Errorf("expected ReassessWeeks 26, got %d", spec.ReassessWeeks)
		}
	})
}

// ---------------------------------------------------------------------------
// TestComputeSLADeadline
// ---------------------------------------------------------------------------

func TestComputeSLADeadline(t *testing.T) {
	log, _ := zap.NewDevelopment()
	cfg := DefaultSLAConfig()
	scanner := NewSLAScanner(nil, nil, log, cfg)

	baseTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		gate     models.MCUGate
		expected time.Time
	}{
		{
			name:     "HALT gate gets 15 minute SLA",
			gate:     models.GateHalt,
			expected: baseTime.Add(15 * time.Minute),
		},
		{
			name:     "PAUSE gate gets 1 hour SLA",
			gate:     models.GatePause,
			expected: baseTime.Add(1 * time.Hour),
		},
		{
			name:     "MODIFY gate gets 4 hour SLA",
			gate:     models.GateModify,
			expected: baseTime.Add(4 * time.Hour),
		},
		{
			name:     "SAFE gate defaults to 24 hour SLA",
			gate:     models.GateSafe,
			expected: baseTime.Add(24 * time.Hour),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scanner.ComputeSLADeadline(tc.gate, baseTime)
			if !got.Equal(tc.expected) {
				t.Errorf("ComputeSLADeadline(%s, %v) = %v, want %v",
					tc.gate, baseTime, got, tc.expected)
			}
		})
	}

	t.Run("custom SLA config is respected", func(t *testing.T) {
		customCfg := SLAConfig{
			HaltSLA:   5 * time.Minute,
			PauseSLA:  30 * time.Minute,
			ModifySLA: 2 * time.Hour,
		}
		customScanner := NewSLAScanner(nil, nil, log, customCfg)

		got := customScanner.ComputeSLADeadline(models.GateHalt, baseTime)
		expected := baseTime.Add(5 * time.Minute)
		if !got.Equal(expected) {
			t.Errorf("custom HALT SLA: got %v, want %v", got, expected)
		}

		got = customScanner.ComputeSLADeadline(models.GatePause, baseTime)
		expected = baseTime.Add(30 * time.Minute)
		if !got.Equal(expected) {
			t.Errorf("custom PAUSE SLA: got %v, want %v", got, expected)
		}
	})
}

// ---------------------------------------------------------------------------
// TestEvaluateGuidelineConditions
// ---------------------------------------------------------------------------

func TestEvaluateGuidelineConditions(t *testing.T) {
	log, _ := zap.NewDevelopment()
	builder := NewCardBuilder(nil, nil, nil, nil, nil, log, nil, nil, nil, nil)

	ptrCondition := func(c models.ConditionStatus) *models.ConditionStatus { return &c }

	t.Run("empty recommendations returns nil", func(t *testing.T) {
		result := builder.evaluateGuidelineConditions(nil)
		if result != nil {
			t.Errorf("expected nil for empty recommendations, got %v", *result)
		}
	})

	t.Run("all MET returns CRITERIA_MET", func(t *testing.T) {
		recs := []models.CardRecommendation{
			{ConditionStatus: ptrCondition(models.ConditionMet)},
			{ConditionStatus: ptrCondition(models.ConditionMet)},
		}
		result := builder.evaluateGuidelineConditions(recs)
		if result == nil || *result != models.ConditionMet {
			t.Errorf("expected CRITERIA_MET, got %v", result)
		}
	})

	t.Run("any NOT_MET returns CRITERIA_NOT_MET immediately", func(t *testing.T) {
		recs := []models.CardRecommendation{
			{ConditionStatus: ptrCondition(models.ConditionMet)},
			{ConditionStatus: ptrCondition(models.ConditionNotMet)},
			{ConditionStatus: ptrCondition(models.ConditionMet)},
		}
		result := builder.evaluateGuidelineConditions(recs)
		if result == nil || *result != models.ConditionNotMet {
			t.Errorf("expected CRITERIA_NOT_MET, got %v", result)
		}
	})

	t.Run("PARTIAL without NOT_MET returns CRITERIA_PARTIAL", func(t *testing.T) {
		recs := []models.CardRecommendation{
			{ConditionStatus: ptrCondition(models.ConditionMet)},
			{ConditionStatus: ptrCondition(models.ConditionPartial)},
			{ConditionStatus: ptrCondition(models.ConditionMet)},
		}
		result := builder.evaluateGuidelineConditions(recs)
		if result == nil || *result != models.ConditionPartial {
			t.Errorf("expected CRITERIA_PARTIAL, got %v", result)
		}
	})

	t.Run("NOT_MET takes precedence over PARTIAL", func(t *testing.T) {
		recs := []models.CardRecommendation{
			{ConditionStatus: ptrCondition(models.ConditionPartial)},
			{ConditionStatus: ptrCondition(models.ConditionNotMet)},
		}
		result := builder.evaluateGuidelineConditions(recs)
		if result == nil || *result != models.ConditionNotMet {
			t.Errorf("expected CRITERIA_NOT_MET to take precedence, got %v", result)
		}
	})

	t.Run("nil ConditionStatus entries are skipped", func(t *testing.T) {
		recs := []models.CardRecommendation{
			{ConditionStatus: nil},
			{ConditionStatus: ptrCondition(models.ConditionMet)},
			{ConditionStatus: nil},
		}
		result := builder.evaluateGuidelineConditions(recs)
		if result == nil || *result != models.ConditionMet {
			t.Errorf("expected CRITERIA_MET (skipping nils), got %v", result)
		}
	})

	t.Run("all nil ConditionStatus returns MET default", func(t *testing.T) {
		recs := []models.CardRecommendation{
			{ConditionStatus: nil},
			{ConditionStatus: nil},
		}
		result := builder.evaluateGuidelineConditions(recs)
		if result == nil || *result != models.ConditionMet {
			t.Errorf("expected CRITERIA_MET as default when all nil, got %v", result)
		}
	})
}

// ---------------------------------------------------------------------------
// TestHypoglycaemiaSeverityClassification
// ---------------------------------------------------------------------------

func TestHypoglycaemiaSeverityClassification(t *testing.T) {
	cfg := &config.Config{
		HypoglycaemiaSevereThreshold:   3.0,
		HypoglycaemiaModerateThreshold: 3.9,
	}
	handler := &HypoglycaemiaHandler{cfg: cfg}

	tests := []struct {
		name         string
		glucoseMmolL float64
		severity     models.HypoglycaemiaSeverity
		gate         models.MCUGate
	}{
		{
			name:         "glucose 2.5 is SEVERE -> HALT",
			glucoseMmolL: 2.5,
			severity:     models.HypoSevere,
			gate:         models.GateHalt,
		},
		{
			name:         "glucose 3.0 (at threshold) is SEVERE -> HALT",
			glucoseMmolL: 3.0,
			severity:     models.HypoSevere,
			gate:         models.GateHalt,
		},
		{
			name:         "glucose 3.1 is MODERATE -> PAUSE",
			glucoseMmolL: 3.1,
			severity:     models.HypoModerate,
			gate:         models.GatePause,
		},
		{
			name:         "glucose 3.9 (at threshold) is MODERATE -> PAUSE",
			glucoseMmolL: 3.9,
			severity:     models.HypoModerate,
			gate:         models.GatePause,
		},
		{
			name:         "glucose 4.0 is MILD -> MODIFY",
			glucoseMmolL: 4.0,
			severity:     models.HypoMild,
			gate:         models.GateModify,
		},
		{
			name:         "glucose 5.0 is MILD -> MODIFY",
			glucoseMmolL: 5.0,
			severity:     models.HypoMild,
			gate:         models.GateModify,
		},
		{
			name:         "glucose 1.5 deep severe is SEVERE -> HALT",
			glucoseMmolL: 1.5,
			severity:     models.HypoSevere,
			gate:         models.GateHalt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotSeverity := handler.classifySeverity(tc.glucoseMmolL)
			if gotSeverity != tc.severity {
				t.Errorf("classifySeverity(%f) = %q, want %q", tc.glucoseMmolL, gotSeverity, tc.severity)
			}
			gotGate := handler.severityToGate(gotSeverity)
			if gotGate != tc.gate {
				t.Errorf("severityToGate(%q) = %q, want %q", gotSeverity, gotGate, tc.gate)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
