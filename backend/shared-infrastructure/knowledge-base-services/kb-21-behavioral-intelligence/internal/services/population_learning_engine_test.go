package services

import (
	"math"
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestAggregateTechniqueStats_ComputesMean(t *testing.T) {
	engine := NewPopulationLearningEngine(nil, nil, 8)
	records := []models.TechniqueEffectiveness{
		{PatientID: "p1", Technique: models.TechMicroCommitment, Alpha: 5.0, Beta: 2.0, Deliveries: 10},
		{PatientID: "p2", Technique: models.TechMicroCommitment, Alpha: 3.0, Beta: 4.0, Deliveries: 10},
		{PatientID: "p3", Technique: models.TechMicroCommitment, Alpha: 4.0, Beta: 3.0, Deliveries: 10},
	}
	alpha, beta := engine.AggregateTechniqueStats(records)
	if math.Abs(alpha-4.0) > 0.01 {
		t.Errorf("AggregateTechniqueStats alpha: got %f, want 4.0", alpha)
	}
	if math.Abs(beta-3.0) > 0.01 {
		t.Errorf("AggregateTechniqueStats beta: got %f, want 3.0", beta)
	}
}

func TestAggregateTechniqueStats_IgnoresLowDeliveries(t *testing.T) {
	engine := NewPopulationLearningEngine(nil, nil, 8)
	records := []models.TechniqueEffectiveness{
		{PatientID: "p1", Technique: models.TechMicroCommitment, Alpha: 5.0, Beta: 2.0, Deliveries: 10},
		{PatientID: "p2", Technique: models.TechMicroCommitment, Alpha: 100.0, Beta: 1.0, Deliveries: 2},
	}
	alpha, beta := engine.AggregateTechniqueStats(records)
	if math.Abs(alpha-5.0) > 0.01 {
		t.Errorf("AggregateTechniqueStats alpha (ignoring low deliveries): got %f, want 5.0", alpha)
	}
	if math.Abs(beta-2.0) > 0.01 {
		t.Errorf("AggregateTechniqueStats beta (ignoring low deliveries): got %f, want 2.0", beta)
	}
}

func TestValidatePriors_AcceptsImprovement(t *testing.T) {
	engine := NewPopulationLearningEngine(nil, nil, 8)
	ok := engine.ValidateImprovement(0.07)
	if !ok {
		t.Error("ValidateImprovement(0.07): expected true, got false")
	}
}

func TestValidatePriors_RejectsSmallImprovement(t *testing.T) {
	engine := NewPopulationLearningEngine(nil, nil, 8)
	ok := engine.ValidateImprovement(0.03)
	if ok {
		t.Error("ValidateImprovement(0.03): expected false, got true")
	}
}
