package services

import (
	"testing"

	"kb-23-decision-cards/internal/models"
)

func TestSort_SafetyFirst(t *testing.T) {
	items := []models.WorklistItem{
		{PatientID: "p-urgent", PAIScore: 95, EscalationTier: "URGENT", PAITrend: "STABLE"},
		{PatientID: "p-safety", PAIScore: 70, EscalationTier: "SAFETY", PAITrend: "STABLE"},
	}

	view := SortAndTierWorklist(items, 10)

	if len(view.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(view.Items))
	}
	if view.Items[0].PatientID != "p-safety" {
		t.Errorf("expected SAFETY item first, got %s", view.Items[0].PatientID)
	}
}

func TestSort_HigherPAIWithinTier(t *testing.T) {
	items := []models.WorklistItem{
		{PatientID: "p-low", PAIScore: 65, EscalationTier: "URGENT", PAITrend: "STABLE"},
		{PatientID: "p-high", PAIScore: 85, EscalationTier: "URGENT", PAITrend: "STABLE"},
	}

	view := SortAndTierWorklist(items, 10)

	if view.Items[0].PatientID != "p-high" {
		t.Errorf("expected higher PAI item first, got %s (PAI %.0f)", view.Items[0].PatientID, view.Items[0].PAIScore)
	}
}

func TestSort_RisingBeatsStable(t *testing.T) {
	items := []models.WorklistItem{
		{PatientID: "p-stable", PAIScore: 73, EscalationTier: "", PAITrend: "STABLE"},
		{PatientID: "p-rising", PAIScore: 70, EscalationTier: "", PAITrend: "RISING"},
	}

	view := SortAndTierWorklist(items, 10)

	// PAI within 5 points (73 vs 70), but RISING gets +50 boost.
	if view.Items[0].PatientID != "p-rising" {
		t.Errorf("expected RISING item first, got %s", view.Items[0].PatientID)
	}
}

func TestSort_TransitionBoost(t *testing.T) {
	items := []models.WorklistItem{
		{PatientID: "p-normal", PAIScore: 50, EscalationTier: "", PAITrend: "STABLE", ContextTags: []string{}},
		{PatientID: "p-discharge", PAIScore: 50, EscalationTier: "", PAITrend: "STABLE", ContextTags: []string{"POST_DISCHARGE"}},
	}

	view := SortAndTierWorklist(items, 10)

	if view.Items[0].PatientID != "p-discharge" {
		t.Errorf("expected post-discharge item first, got %s", view.Items[0].PatientID)
	}
}

func TestSort_TierCounts(t *testing.T) {
	var items []models.WorklistItem

	// 2 CRITICAL (score ≥ 4000): use IMMEDIATE escalation tier (weight 4000).
	for i := 0; i < 2; i++ {
		items = append(items, models.WorklistItem{
			PatientID:      "crit",
			PAIScore:       80,
			EscalationTier: "IMMEDIATE",
			PAITrend:       "STABLE",
		})
	}
	// 3 HIGH (score ≥ 3000): use URGENT escalation tier (weight 3000).
	for i := 0; i < 3; i++ {
		items = append(items, models.WorklistItem{
			PatientID:      "high",
			PAIScore:       60,
			EscalationTier: "URGENT",
			PAITrend:       "STABLE",
		})
	}
	// 5 MODERATE (score ≥ 1000): use ROUTINE escalation tier (weight 1000).
	for i := 0; i < 5; i++ {
		items = append(items, models.WorklistItem{
			PatientID:      "mod",
			PAIScore:       40,
			EscalationTier: "ROUTINE",
			PAITrend:       "STABLE",
		})
	}

	view := SortAndTierWorklist(items, 100)

	if view.CriticalCount != 2 {
		t.Errorf("expected 2 CRITICAL, got %d", view.CriticalCount)
	}
	if view.HighCount != 3 {
		t.Errorf("expected 3 HIGH, got %d", view.HighCount)
	}
	if view.ModerateCount != 5 {
		t.Errorf("expected 5 MODERATE, got %d", view.ModerateCount)
	}
	if view.TotalCount != 10 {
		t.Errorf("expected total 10, got %d", view.TotalCount)
	}
}
