package unit

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"
)

func TestDecisionRule_PrediabetesLifestyleOnly(t *testing.T) {
	rec := services.ApplyDecisionRule(6.0, 0)
	if rec != "LIFESTYLE_ONLY" {
		t.Errorf("HbA1c 6.0 → expected LIFESTYLE_ONLY, got %s", rec)
	}
}

func TestDecisionRule_MildLifestyleFirst(t *testing.T) {
	rec := services.ApplyDecisionRule(7.0, 0)
	if rec != "LIFESTYLE_FIRST" {
		t.Errorf("HbA1c 7.0 → expected LIFESTYLE_FIRST, got %s", rec)
	}
}

func TestDecisionRule_ModerateCombined(t *testing.T) {
	rec := services.ApplyDecisionRule(8.0, 0)
	if rec != "COMBINED" {
		t.Errorf("HbA1c 8.0 → expected COMBINED, got %s", rec)
	}
}

func TestDecisionRule_SevereMedPrimary(t *testing.T) {
	rec := services.ApplyDecisionRule(9.5, 0)
	if rec != "MEDICATION_PRIMARY" {
		t.Errorf("HbA1c 9.5 → expected MEDICATION_PRIMARY, got %s", rec)
	}
}

func TestDecisionRule_BPLifestyleFirst(t *testing.T) {
	rec := services.ApplyDecisionRule(6.5, 140)
	if rec != "LIFESTYLE_FIRST" {
		t.Errorf("SBP 140, HbA1c 6.5 → expected LIFESTYLE_FIRST, got %s", rec)
	}
}

func TestDecisionRule_BPSevereMedPrimary(t *testing.T) {
	rec := services.ApplyDecisionRule(6.5, 165)
	if rec != "MEDICATION_PRIMARY" {
		t.Errorf("SBP 165 → expected MEDICATION_PRIMARY, got %s", rec)
	}
}

func TestRankOptions(t *testing.T) {
	options := []models.ComparedOption{
		{ProjectedEffect: -5.0, EvidenceGrade: "B", SafetyScore: 0.9},
		{ProjectedEffect: -10.0, EvidenceGrade: "A", SafetyScore: 0.95},
		{ProjectedEffect: -8.0, EvidenceGrade: "A", SafetyScore: 0.7},
	}
	ranked := services.RankOptions(options)
	if ranked[0].Rank != 1 || ranked[0].ProjectedEffect != -10.0 {
		t.Errorf("best option should be -10.0 Grade A, got rank %d effect %f", ranked[0].Rank, ranked[0].ProjectedEffect)
	}
}
