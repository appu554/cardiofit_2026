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
	// Two improving + two declining domains, deliberately chosen so each pair
	// is in the mechanism map: GLUCOSE↑ vs CARDIO↓ AND BEHAVIORAL↑ vs BODY_COMP↓.
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 1.0, Trend: models.TrendRapidImproving},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -0.8, Trend: models.TrendDeclining},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: 0.6, Trend: models.TrendImproving},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: -0.5, Trend: models.TrendDeclining},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) < 2 {
		t.Fatalf("expected >= 2 divergences, got %d", len(divergences))
	}

	// Build a set of (improving, declining) pairs found.
	type pair struct{ imp, dec models.MHRIDomain }
	found := make(map[pair]bool)
	for _, d := range divergences {
		found[pair{d.ImprovingDomain, d.DecliningDomain}] = true
	}

	// Verify the expected pairs are present (and have specific mechanisms, not the fallback).
	expectGlucoseCardio := pair{models.DomainGlucose, models.DomainCardio}
	expectBehavioralBodyComp := pair{models.DomainBehavioral, models.DomainBodyComp}

	if !found[expectGlucoseCardio] {
		t.Errorf("expected GLUCOSE→CARDIO divergence pair to be detected")
	}
	if !found[expectBehavioralBodyComp] {
		t.Errorf("expected BEHAVIORAL→BODY_COMP divergence pair to be detected")
	}

	// Verify the GLUCOSE→CARDIO mechanism is the specific clinical hypothesis (not the fallback).
	for _, d := range divergences {
		if d.ImprovingDomain == models.DomainGlucose && d.DecliningDomain == models.DomainCardio {
			if !containsSubstring(d.PossibleMechanism, "SGLT2i") {
				t.Errorf("expected GLUCOSE→CARDIO mechanism to mention SGLT2i, got: %s", d.PossibleMechanism)
			}
		}
	}
}

// containsSubstring is a tiny helper for substring checking in divergence tests.
func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
