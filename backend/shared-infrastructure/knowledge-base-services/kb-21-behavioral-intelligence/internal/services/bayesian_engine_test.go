package services

import (
	"math"
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestBayesianEngine_InitializePatient(t *testing.T) {
	be := NewBayesianEngine(nil, nil) // nil db/logger for pure logic test
	records := be.BuildDefaultRecords("patient-1")
	if len(records) != 12 {
		t.Errorf("expected 12 technique records, got %d", len(records))
	}
	for _, r := range records {
		if r.Alpha < 1.0 || r.Beta < 1.0 {
			t.Errorf("technique %s has invalid priors: alpha=%.2f beta=%.2f", r.Technique, r.Alpha, r.Beta)
		}
	}
}

func TestBayesianEngine_PosteriorMean(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	// Beta(3, 2) → mean = 3/5 = 0.6
	mean := be.PosteriorMean(3.0, 2.0)
	if math.Abs(mean-0.6) > 0.001 {
		t.Errorf("posterior mean: got %.4f, want 0.6", mean)
	}
}

func TestBayesianEngine_UpdatePosterior_Success(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	rec := &models.TechniqueEffectiveness{
		Alpha: 2.0, Beta: 2.0, Deliveries: 5, Successes: 3,
	}
	be.UpdatePosterior(rec, true)
	if rec.Alpha != 3.0 {
		t.Errorf("alpha after success: got %.1f, want 3.0", rec.Alpha)
	}
	if rec.Deliveries != 6 {
		t.Errorf("deliveries after update: got %d, want 6", rec.Deliveries)
	}
	if rec.Successes != 4 {
		t.Errorf("successes after success: got %d, want 4", rec.Successes)
	}
}

func TestBayesianEngine_UpdatePosterior_Failure(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	rec := &models.TechniqueEffectiveness{
		Alpha: 2.0, Beta: 2.0, Deliveries: 5, Successes: 3,
	}
	be.UpdatePosterior(rec, false)
	if rec.Beta != 3.0 {
		t.Errorf("beta after failure: got %.1f, want 3.0", rec.Beta)
	}
	if rec.Deliveries != 6 {
		t.Errorf("deliveries after update: got %d, want 6", rec.Deliveries)
	}
	if rec.Successes != 3 {
		t.Errorf("successes unchanged after failure: got %d, want 3", rec.Successes)
	}
}

func TestBayesianEngine_SelectTechnique_ReturnsHighestSample(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	// Create records where T-06 has overwhelming evidence of success
	records := []*models.TechniqueEffectiveness{
		{Technique: models.TechMicroCommitment, Alpha: 1.0, Beta: 10.0},       // very low
		{Technique: models.TechProgressVisualization, Alpha: 50.0, Beta: 1.0}, // very high
		{Technique: models.TechHabitStacking, Alpha: 1.0, Beta: 10.0},         // very low
	}
	// With Alpha=50, Beta=1, T-06 should win nearly every time
	wins := 0
	for i := 0; i < 100; i++ {
		selected := be.ThompsonSelect(records, nil) // nil phase multipliers
		if selected.Technique == models.TechProgressVisualization {
			wins++
		}
	}
	if wins < 90 {
		t.Errorf("T-06 should win >90%% with Alpha=50, got %d%%", wins)
	}
}

// TestBayesianEngine_PhaseMultipliers_Applied requires PhaseMultipliers from phase_engine.go (Task 4).
// Uncomment when Task 4 is implemented.
//
// func TestBayesianEngine_PhaseMultipliers_Applied(t *testing.T) {
// 	be := NewBayesianEngine(nil, nil)
// 	// T-01 (Micro-Commitment) has 1.5x multiplier in INITIATION phase
// 	// T-03 (Loss Aversion) has 0.3x multiplier in INITIATION phase
// 	records := []*models.TechniqueEffectiveness{
// 		{Technique: models.TechMicroCommitment, Alpha: 2.0, Beta: 2.0},
// 		{Technique: models.TechLossAversion, Alpha: 2.0, Beta: 2.0},
// 	}
// 	multipliers := PhaseMultipliers[models.PhaseInitiation]
//
// 	t01Wins, t03Wins := 0, 0
// 	for i := 0; i < 1000; i++ {
// 		selected := be.ThompsonSelect(records, multipliers)
// 		switch selected.Technique {
// 		case models.TechMicroCommitment:
// 			t01Wins++
// 		case models.TechLossAversion:
// 			t03Wins++
// 		}
// 	}
// 	// With 1.5x vs 0.3x multiplier on equal priors, T-01 should dominate
// 	if t01Wins <= t03Wins {
// 		t.Errorf("T-01 (1.5x) should beat T-03 (0.3x) in INITIATION: T-01=%d T-03=%d", t01Wins, t03Wins)
// 	}
// }
