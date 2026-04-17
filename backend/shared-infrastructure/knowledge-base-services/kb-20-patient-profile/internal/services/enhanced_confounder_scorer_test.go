package services

import (
	"strings"
	"testing"

	"kb-patient-profile/internal/models"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func newScorer() *EnhancedConfounderScorer { return NewEnhancedConfounderScorer() }

func calFactor(name string, category models.ConfounderCategory, weight float64, overlapPct float64, affected []string) models.ConfounderFactor {
	return models.ConfounderFactor{
		Category:         category,
		Name:             name,
		Weight:           weight,
		AffectedOutcomes: affected,
		OverlapPct:       overlapPct,
	}
}

func clinFactor(name string, category models.ConfounderCategory, weight float64, affected []string) models.ConfounderFactor {
	return models.ConfounderFactor{
		Category:         category,
		Name:             name,
		Weight:           weight,
		AffectedOutcomes: affected,
	}
}

func lifeFactor(name string, weight float64) models.ConfounderFactor {
	return models.ConfounderFactor{
		Category: models.ConfounderLifestyle,
		Name:     name,
		Weight:   weight,
	}
}

// ─── tests ──────────────────────────────────────────────────────────────────

func TestEnhancedScorer_NoConfounders(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		OutcomeType: "DELTA_HBA1C",
	})

	if result.CompositeScore != 0.0 {
		t.Errorf("expected composite 0.0, got %f", result.CompositeScore)
	}
	if result.ConfidenceLevel != "HIGH" {
		t.Errorf("expected confidence HIGH, got %s", result.ConfidenceLevel)
	}
	if len(result.ActiveFactors) != 0 {
		t.Errorf("expected 0 active factors, got %d", len(result.ActiveFactors))
	}
	if result.ShouldDefer {
		t.Error("expected ShouldDefer=false")
	}
	if !strings.Contains(result.Narrative, "No significant confounders") {
		t.Errorf("unexpected narrative: %s", result.Narrative)
	}
}

func TestEnhancedScorer_MedicationOnly_BackwardCompatible(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		ConcurrentMedCount: 2,
		AdherenceDrop:      0.15,
		OutcomeType:        "DELTA_HBA1C",
	})

	if result.MedicationScore <= 0 {
		t.Errorf("expected MedicationScore > 0, got %f", result.MedicationScore)
	}
	if result.CompositeScore <= 0 || result.CompositeScore >= 0.5 {
		t.Errorf("expected CompositeScore in (0, 0.5), got %f", result.CompositeScore)
	}
	if result.ConfidenceLevel != "MODERATE" {
		t.Errorf("expected confidence MODERATE, got %s", result.ConfidenceLevel)
	}
}

func TestEnhancedScorer_RamadanDuringWindow(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		CalendarFactors: []models.ConfounderFactor{
			calFactor("RAMADAN", models.ConfounderReligiousFast, 0.25, 80, []string{"DELTA_HBA1C"}),
		},
		OutcomeType: "DELTA_HBA1C",
	})

	if result.CalendarScore <= 0 {
		t.Errorf("expected CalendarScore > 0, got %f", result.CalendarScore)
	}
	if result.CompositeScore < 0.20 {
		t.Errorf("expected CompositeScore >= 0.20, got %f", result.CompositeScore)
	}
	if !strings.Contains(result.Narrative, "RAMADAN") {
		t.Errorf("expected narrative to contain RAMADAN, got: %s", result.Narrative)
	}
}

func TestEnhancedScorer_SteroidCourse_HighConfounder(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		ClinicalEventFactors: []models.ConfounderFactor{
			clinFactor("STEROID_COURSE", models.ConfounderIatrogenic, 0.35, []string{"DELTA_HBA1C", "DELTA_SBP"}),
		},
		OutcomeType: "DELTA_HBA1C",
	})

	if result.ClinicalEventScore < 0.30 {
		t.Errorf("expected ClinicalEventScore >= 0.30, got %f", result.ClinicalEventScore)
	}
	if result.ConfidenceLevel != "LOW" {
		t.Errorf("expected confidence LOW, got %s", result.ConfidenceLevel)
	}
	if !result.ShouldDefer {
		t.Error("expected ShouldDefer=true for high clinical event score")
	}
}

func TestEnhancedScorer_MultipleFactors_Compound(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		ConcurrentMedCount: 1,
		CalendarFactors: []models.ConfounderFactor{
			calFactor("DIWALI_SEASON", models.ConfounderFestivalDiet, 0.15, 60, []string{"DELTA_HBA1C", "DELTA_WEIGHT"}),
		},
		LifestyleFactors: []models.ConfounderFactor{
			lifeFactor("ENGAGEMENT_COLLAPSE", 0.15),
		},
		OutcomeType: "DELTA_HBA1C",
	})

	if result.CompositeScore <= 0.30 {
		t.Errorf("expected CompositeScore > 0.30, got %f", result.CompositeScore)
	}
	if result.FactorCount < 3 {
		t.Errorf("expected FactorCount >= 3, got %d", result.FactorCount)
	}
	if !strings.Contains(result.Narrative, "DIWALI_SEASON") {
		t.Errorf("narrative should contain DIWALI_SEASON: %s", result.Narrative)
	}
	if !strings.Contains(result.Narrative, "ENGAGEMENT_COLLAPSE") {
		t.Errorf("narrative should contain ENGAGEMENT_COLLAPSE: %s", result.Narrative)
	}
}

func TestEnhancedScorer_IrrelevantConfounder_NotCounted(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		CalendarFactors: []models.ConfounderFactor{
			calFactor("MONSOON_SEASON", models.ConfounderSeasonal, 0.10, 100, []string{"DELTA_WEIGHT", "DELTA_SBP"}),
		},
		OutcomeType: "DELTA_HBA1C",
	})

	if result.CalendarScore != 0.0 {
		t.Errorf("expected CalendarScore 0.0 (monsoon irrelevant for HbA1c), got %f", result.CalendarScore)
	}
}

func TestEnhancedScorer_DeferDuringActiveRamadan(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		CalendarFactors: []models.ConfounderFactor{
			calFactor("RAMADAN", models.ConfounderReligiousFast, 0.25, 90, []string{"DELTA_HBA1C"}),
		},
		OutcomeType:    "DELTA_HBA1C",
		DeferOnRamadan: true,
	})

	if !result.ShouldDefer {
		t.Error("expected ShouldDefer=true with DeferOnRamadan")
	}
	if result.DeferReasonCode != "RAMADAN_ACTIVE" {
		t.Errorf("expected DeferReasonCode RAMADAN_ACTIVE, got %s", result.DeferReasonCode)
	}
	if result.SuggestedRecheckWeeks <= 0 {
		t.Errorf("expected SuggestedRecheckWeeks > 0, got %d", result.SuggestedRecheckWeeks)
	}
}

func TestEnhancedScorer_NarrativeContainsAllFactors(t *testing.T) {
	result := newScorer().Compute(EnhancedConfounderInput{
		ConcurrentMedCount: 2,
		CalendarFactors: []models.ConfounderFactor{
			calFactor("RAMADAN", models.ConfounderReligiousFast, 0.20, 70, []string{"DELTA_HBA1C"}),
		},
		ClinicalEventFactors: []models.ConfounderFactor{
			clinFactor("ACUTE_INFECTION", models.ConfounderAcuteIllness, 0.15, []string{"DELTA_HBA1C"}),
		},
		OutcomeType: "DELTA_HBA1C",
	})

	for _, want := range []string{"concurrent medication", "RAMADAN", "ACUTE_INFECTION"} {
		if !strings.Contains(result.Narrative, want) {
			t.Errorf("expected narrative to contain %q, got: %s", want, result.Narrative)
		}
	}
}
