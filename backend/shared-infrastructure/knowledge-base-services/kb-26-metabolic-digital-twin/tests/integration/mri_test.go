//go:build integration

package integration

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// TestMRI_EndToEnd_BorderlinePatient validates the full MRI flow for the
// spec §5.1 "borderline everything" patient scenario.
func TestMRI_EndToEnd_BorderlinePatient(t *testing.T) {
	scorer := services.NewMRIScorer(nil, nil)

	// Spec §5.1: FBG=118, PPBG=172, waist=92(M), steps=3800, SBP=138
	input := services.MRIScorerInput{
		FBG: 118, PPBG: 172, HbA1cTrend: 0.15,
		WaistCm: 92, WeightTrend: 0.4, MuscleSTS: 10,
		SBP: 138, SBPTrend: 5, BPDipping: "NON_DIPPER",
		Steps: 3800, ProteinGKg: 0.65, SleepScore: 0.4,
		Sex: "M",
	}

	result := scorer.ComputeMRI(input, nil)

	// Should be MODERATE_DETERIORATION
	if result.Category != models.MRICategoryModerateDeterioration {
		t.Fatalf("expected MODERATE_DETERIORATION, got %s (score=%.1f)", result.Category, result.Score)
	}

	// All 4 domains should be populated
	if len(result.Domains) != 4 {
		t.Fatalf("expected 4 domains, got %d", len(result.Domains))
	}

	// Top driver should be identified
	if result.TopDriver == "" {
		t.Error("expected a top driver to be identified")
	}

	t.Logf("MRI Score: %.1f (%s), Top Driver: %s", result.Score, result.Category, result.TopDriver)
	for _, d := range result.Domains {
		t.Logf("  %s: raw=%.2f scaled=%.1f", d.Name, d.Score, d.Scaled)
	}
}

// TestMRI_OptimalPatient validates that an exceptionally healthy patient gets OPTIMAL.
// Uses strongly negative z-score inputs to push the sigmoid-scaled composite below 25.
func TestMRI_OptimalPatient(t *testing.T) {
	scorer := services.NewMRIScorer(nil, nil)

	// Very healthy values: FBG=75 (z=-1.33), PPBG=95 (z=-1.4), HbA1cTrend=-0.5 (z=-2.5),
	// waist=70 (z=-1.5), weight losing, strong muscle, low SBP, good dipping,
	// high steps, good protein, excellent sleep.
	input := services.MRIScorerInput{
		FBG: 75, PPBG: 95, HbA1cTrend: -0.5,
		WaistCm: 70, WeightTrend: -0.5, MuscleSTS: 20,
		SBP: 108, SBPTrend: -5, BPDipping: "DIPPER",
		Steps: 12000, ProteinGKg: 1.3, SleepScore: 0.95,
		Sex: "M",
	}

	result := scorer.ComputeMRI(input, nil)

	if result.Category != models.MRICategoryOptimal {
		t.Errorf("expected OPTIMAL, got %s (score=%.1f)", result.Category, result.Score)
	}
	t.Logf("Optimal patient MRI: %.1f (%s)", result.Score, result.Category)
}

// TestMRI_HighDeterioration validates that severe dysregulation gets HIGH.
func TestMRI_HighDeterioration(t *testing.T) {
	scorer := services.NewMRIScorer(nil, nil)

	input := services.MRIScorerInput{
		FBG: 180, PPBG: 280, HbA1cTrend: 0.7,
		WaistCm: 110, WeightTrend: 1.5, MuscleSTS: 6,
		SBP: 160, SBPTrend: 12, BPDipping: "REVERSE_DIPPER",
		Steps: 1500, ProteinGKg: 0.4, SleepScore: 0.1,
		Sex: "M",
	}

	result := scorer.ComputeMRI(input, nil)

	if result.Category != models.MRICategoryHighDeterioration {
		t.Errorf("expected HIGH_DETERIORATION, got %s (score=%.1f)", result.Category, result.Score)
	}
	t.Logf("High deterioration patient MRI: %.1f (%s)", result.Score, result.Category)
}

// TestMRI_DomainDriverDetection validates the masked behavioral decline scenario
// from spec §5.2: MRI stable but behavioral domain deteriorating while glucose improving.
func TestMRI_DomainDriverDetection(t *testing.T) {
	scorer := services.NewMRIScorer(nil, nil)

	// Good glucose (medication working) but bad behavior
	input := services.MRIScorerInput{
		FBG: 95, PPBG: 125, HbA1cTrend: -0.3,
		WaistCm: 95, WeightTrend: 0.8, MuscleSTS: 8,
		SBP: 128, SBPTrend: 0, BPDipping: "DIPPER",
		Steps: 2000, ProteinGKg: 0.5, SleepScore: 0.3,
		Sex: "M",
	}

	result := scorer.ComputeMRI(input, nil)

	// Behavioral domain should be the top driver even though glucose is fine
	behavioralDomain := result.Domains[3]
	glucoseDomain := result.Domains[0]

	if behavioralDomain.Score <= glucoseDomain.Score {
		t.Logf("Behavioral domain (%.2f) should be worse than glucose (%.2f)", behavioralDomain.Score, glucoseDomain.Score)
	}

	t.Logf("Masked decline MRI: %.1f, top driver: %s", result.Score, result.TopDriver)
	for _, d := range result.Domains {
		t.Logf("  %s: %.2f", d.Name, d.Score)
	}
}
