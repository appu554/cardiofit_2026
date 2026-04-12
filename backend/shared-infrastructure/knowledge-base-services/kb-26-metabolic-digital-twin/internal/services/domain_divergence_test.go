package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ---------------------------------------------------------------------------
// TestDetectDivergence_GlucoseImproving_CardioDeclining
// ---------------------------------------------------------------------------

func TestDetectDivergence_GlucoseImproving_CardioDeclining(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 0.8, Trend: models.TrendImproving},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -1.2, Trend: models.TrendRapidDeclining},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: 0.1, Trend: models.TrendStable},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: 0.0, Trend: models.TrendStable},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) != 1 {
		t.Fatalf("expected 1 divergence, got %d", len(divergences))
	}
	if divergences[0].ImprovingDomain != models.DomainGlucose {
		t.Errorf("expected improving domain GLUCOSE, got %s", divergences[0].ImprovingDomain)
	}
	if divergences[0].DecliningDomain != models.DomainCardio {
		t.Errorf("expected declining domain CARDIO, got %s", divergences[0].DecliningDomain)
	}
	if divergences[0].DivergenceRate < 1.5 {
		t.Errorf("expected divergence rate >= 1.5, got %.3f", divergences[0].DivergenceRate)
	}
	if divergences[0].ClinicalConcern == "" {
		t.Error("expected non-empty ClinicalConcern")
	}
}

// ---------------------------------------------------------------------------
// TestDetectDivergence_NoDivergence_AllStable
// ---------------------------------------------------------------------------

func TestDetectDivergence_NoDivergence_AllStable(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {SlopePerDay: 0.1},
		models.DomainCardio:     {SlopePerDay: -0.1},
		models.DomainBodyComp:   {SlopePerDay: 0.05},
		models.DomainBehavioral: {SlopePerDay: -0.05},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) != 0 {
		t.Errorf("expected 0 divergences for all-stable, got %d", len(divergences))
	}
}

// ---------------------------------------------------------------------------
// TestDetectDivergence_MultiplePairs
// ---------------------------------------------------------------------------

func TestDetectDivergence_MultiplePairs(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 1.0, Trend: models.TrendRapidImproving},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -0.8, Trend: models.TrendDeclining},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: 0.5, Trend: models.TrendImproving},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: -0.6, Trend: models.TrendDeclining},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) < 2 {
		t.Errorf("expected >= 2 divergences (glucose/cardio + bodycomp/behavioral), got %d", len(divergences))
	}
}

// ---------------------------------------------------------------------------
// TestDivergence_ClinicalConcernText
// ---------------------------------------------------------------------------

func TestDivergence_ClinicalConcernText(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 0.8, Trend: models.TrendImproving},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -0.9, Trend: models.TrendDeclining},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: 0.0, Trend: models.TrendStable},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: 0.0, Trend: models.TrendStable},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) != 1 {
		t.Fatalf("expected 1 divergence, got %d", len(divergences))
	}

	if divergences[0].ClinicalConcern == "" {
		t.Error("expected non-empty ClinicalConcern")
	}

	if divergences[0].PossibleMechanism == "" {
		t.Error("expected non-empty PossibleMechanism")
	}
}
