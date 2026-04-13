package services

import (
	"strings"
	"testing"

	"kb-23-decision-cards/internal/models"
)

func TestEvaluateFourPillars_BPContext_MaskedHTN_Gap(t *testing.T) {
	input := FourPillarInput{
		PatientID: "p1",
		Medication: MedicationPillarInput{
			OnGuidelineMeds: true,
			AdherencePct:    95,
		},
		BPContext: &models.BPContextClassification{
			Phenotype: models.PhenotypeMaskedHTN,
		},
	}

	result := EvaluateFourPillars(input)
	med := findPillar(result, "MEDICATION")
	if med == nil {
		t.Fatal("missing MEDICATION pillar")
	}
	if med.Status != PillarGap {
		t.Errorf("expected PillarGap for masked HTN, got %s", med.Status)
	}
	if !strings.Contains(med.Reason, "masked hypertension") {
		t.Errorf("expected reason to mention masked hypertension, got %q", med.Reason)
	}
}

func TestEvaluateFourPillars_BPContext_MaskedHTN_DM_UrgentGap(t *testing.T) {
	input := FourPillarInput{
		PatientID: "p1",
		Medication: MedicationPillarInput{
			OnGuidelineMeds: true,
			AdherencePct:    95,
		},
		BPContext: &models.BPContextClassification{
			Phenotype:             models.PhenotypeMaskedHTN,
			DiabetesAmplification: true,
		},
	}

	result := EvaluateFourPillars(input)
	med := findPillar(result, "MEDICATION")
	if med == nil {
		t.Fatal("missing MEDICATION pillar")
	}
	if med.Status != PillarUrgentGap {
		t.Errorf("expected PillarUrgentGap for masked HTN + DM, got %s", med.Status)
	}
}

func TestEvaluateFourPillars_BPContext_MaskedHTN_MorningSurge_UrgentGap(t *testing.T) {
	input := FourPillarInput{
		PatientID: "p1",
		Medication: MedicationPillarInput{
			OnGuidelineMeds: true,
			AdherencePct:    95,
		},
		BPContext: &models.BPContextClassification{
			Phenotype:            models.PhenotypeMaskedHTN,
			MorningSurgeCompound: true,
		},
	}

	result := EvaluateFourPillars(input)
	med := findPillar(result, "MEDICATION")
	if med == nil {
		t.Fatal("missing MEDICATION pillar")
	}
	if med.Status != PillarUrgentGap {
		t.Errorf("expected PillarUrgentGap for masked HTN + morning surge, got %s", med.Status)
	}
}

func TestEvaluateFourPillars_BPContext_WhiteCoatHTN_Gap(t *testing.T) {
	input := FourPillarInput{
		PatientID: "p1",
		Medication: MedicationPillarInput{
			OnGuidelineMeds: true,
			AdherencePct:    95,
		},
		BPContext: &models.BPContextClassification{
			Phenotype: models.PhenotypeWhiteCoatHTN,
		},
	}

	result := EvaluateFourPillars(input)
	med := findPillar(result, "MEDICATION")
	if med == nil {
		t.Fatal("missing MEDICATION pillar")
	}
	if med.Status != PillarGap {
		t.Errorf("expected PillarGap for white-coat HTN, got %s", med.Status)
	}
	if !strings.Contains(med.Reason, "white-coat") {
		t.Errorf("expected reason to mention white-coat, got %q", med.Reason)
	}
}

func TestEvaluateFourPillars_BPContext_Nil_NoChange(t *testing.T) {
	// Regression guard: nil BPContext must not affect existing behaviour.
	input := FourPillarInput{
		PatientID: "p1",
		Medication: MedicationPillarInput{
			OnGuidelineMeds: true,
			AdherencePct:    95,
		},
		// BPContext intentionally nil
	}

	result := EvaluateFourPillars(input)
	med := findPillar(result, "MEDICATION")
	if med == nil {
		t.Fatal("missing MEDICATION pillar")
	}
	if med.Status != PillarOnTrack {
		t.Errorf("expected PillarOnTrack with nil BPContext + adequate meds, got %s", med.Status)
	}
}

func findPillar(result FourPillarResult, name string) *PillarResult {
	for i := range result.Pillars {
		if result.Pillars[i].Pillar == name {
			return &result.Pillars[i]
		}
	}
	return nil
}
