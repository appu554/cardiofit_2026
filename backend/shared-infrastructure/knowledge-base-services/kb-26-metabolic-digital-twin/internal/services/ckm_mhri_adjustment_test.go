package services

import (
	"testing"
)

func TestMHRI_Stage4a_PreventiveWeighting(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("4a", "", "")
	if adjustment.CardioDomainWeight <= 0.25 {
		t.Errorf("4a should increase cardio weight above default 0.25, got %.2f",
			adjustment.CardioDomainWeight)
	}
}

func TestMHRI_Stage4c_HFrEF_Adjustment(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("4c", "HFrEF", "III")
	if adjustment.CardioDomainWeight < 0.35 {
		t.Errorf("4c-HFrEF should have cardio weight >= 0.35, got %.2f",
			adjustment.CardioDomainWeight)
	}
	if adjustment.ScoreCeiling >= 60 {
		t.Errorf("NYHA III should cap score below 60, got %.1f", adjustment.ScoreCeiling)
	}
}

func TestMHRI_Stage4c_HFpEF_Adjustment(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("4c", "HFpEF", "II")
	if adjustment.BehavioralDomainWeight <= 0.15 {
		t.Errorf("4c-HFpEF should increase behavioral weight above 0.15, got %.2f",
			adjustment.BehavioralDomainWeight)
	}
}

func TestMHRI_Stage2_NoAdjustment(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("2", "", "")
	if adjustment.GlucoseDomainWeight != 0.35 {
		t.Errorf("Stage 2 should have default glucose weight 0.35, got %.2f",
			adjustment.GlucoseDomainWeight)
	}
	if adjustment.CardioDomainWeight != 0.25 {
		t.Errorf("Stage 2 should have default cardio weight 0.25, got %.2f",
			adjustment.CardioDomainWeight)
	}
	if adjustment.ScoreCeiling != 100.0 {
		t.Errorf("Stage 2 should have no ceiling (100), got %.1f", adjustment.ScoreCeiling)
	}
}
