package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestBuildPhenotypeRecords_UsesAchieverPriors(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	records := be.BuildPhenotypeRecords("patient-cs-1", models.PhenotypeAchiever)
	if len(records) != 12 {
		t.Fatalf("expected 12 records, got %d", len(records))
	}

	// Find T-06 Progress Visualization — should have Achiever priors (3.0, 1.5)
	var progressRec *models.TechniqueEffectiveness
	for i, r := range records {
		if r.Technique == models.TechProgressVisualization {
			progressRec = &records[i]
			break
		}
	}
	if progressRec == nil {
		t.Fatal("TechProgressVisualization record not found")
	}
	if progressRec.Alpha != 3.0 {
		t.Errorf("Achiever TechProgressVisualization Alpha: got %.1f, want 3.0", progressRec.Alpha)
	}
	if progressRec.Beta != 1.5 {
		t.Errorf("Achiever TechProgressVisualization Beta: got %.1f, want 1.5", progressRec.Beta)
	}
}

func TestBuildPhenotypeRecords_SupportDependent_SuppressesSocialNorms(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	records := be.BuildPhenotypeRecords("patient-cs-2", models.PhenotypeSupportDependent)

	var socialRec *models.TechniqueEffectiveness
	for i, r := range records {
		if r.Technique == models.TechSocialNorms {
			socialRec = &records[i]
			break
		}
	}
	if socialRec == nil {
		t.Fatal("TechSocialNorms record not found")
	}
	if socialRec.Alpha != 1.0 {
		t.Errorf("SupportDependent TechSocialNorms Alpha: got %.1f, want 1.0 (weak prior)", socialRec.Alpha)
	}
	if socialRec.Beta != 3.0 {
		t.Errorf("SupportDependent TechSocialNorms Beta: got %.1f, want 3.0 (suppressed)", socialRec.Beta)
	}
}

func TestBuildDefaultRecords_StillWorksForUnknownPhenotype(t *testing.T) {
	be := NewBayesianEngine(nil, nil)
	records := be.BuildDefaultRecords("patient-cs-3")
	if len(records) != 12 {
		t.Fatalf("expected 12 records, got %d", len(records))
	}
	// Should use population priors (from TechniqueLibrary)
	for _, r := range records {
		if r.Technique == models.TechMicroCommitment {
			if r.Alpha != 2.0 {
				t.Errorf("MicroCommitment population Alpha: got %.1f, want 2.0", r.Alpha)
			}
			if r.Beta != 2.0 {
				t.Errorf("MicroCommitment population Beta: got %.1f, want 2.0", r.Beta)
			}
		}
	}
}
